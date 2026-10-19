package search

import (
	"context"
	"fmt"
	"strings"

	"github.com/TrebuchetDynamics/research-forge/internal/jobs"
	"github.com/TrebuchetDynamics/research-forge/internal/library"
	"github.com/TrebuchetDynamics/research-forge/internal/sources"
)

// SearchBudget uses MaxAPICalls like sources.GraphExpansionBudget. One unit is
// one connector Search invocation, including a failure or interrupted attempt.
type SearchBudget struct {
	Depth                string `json:"depth"`
	MaxQueryVariants     int    `json:"max_query_variants"`
	MaxSources           int    `json:"max_sources"`
	MaxAPICalls          int    `json:"max_api_calls"`
	LowYieldThreshold    int    `json:"low_yield_threshold"`
	SaturationQueries    int    `json:"saturation_queries"`
	MaxConsecutiveErrors int    `json:"max_consecutive_errors"`
}

func DepthBudget(depth string) (SearchBudget, error) {
	switch depth {
	case "quick":
		return SearchBudget{depth, 3, 2, 6, 1, 2, 3}, nil
	case "", "standard":
		return SearchBudget{"standard", 10, 5, 50, 2, 3, 5}, nil
	case "comprehensive":
		return SearchBudget{depth, 30, 20, 600, 1, 5, 10}, nil
	default:
		return SearchBudget{}, fmt.Errorf("unknown search depth %q", depth)
	}
}

type QueryCoverage struct {
	Query     string `json:"query"`
	NewUnique int    `json:"new_unique"`
	Complete  bool   `json:"complete"`
}
type Coverage struct {
	UniqueRecords       int             `json:"unique_records"`
	Queries             []QueryCoverage `json:"queries"`
	ConsecutiveLowYield int             `json:"consecutive_low_yield"`
}
type BatchProgress struct {
	Budget            SearchBudget       `json:"budget"`
	StopReason        sources.StopReason `json:"stop_reason"`
	RequestsUsed      int                `json:"requests_used"`
	Coverage          Coverage           `json:"coverage"`
	ConsecutiveErrors int                `json:"consecutive_errors"`
}

// BatchHooks keep output transactions and rendering in callers, while the
// service owns dispatch, resume, deduplication, coverage and stopping policy.
type BatchHooks struct {
	// Job attaches durable progress to the same lease used by CLI and MCP hosts.
	Job        *jobs.Run
	Search     func(context.Context, string, string) ([]library.PaperRecord, error)
	Checkpoint func() error
	Completed  func(source, query string, queryIndex int, papers []library.PaperRecord) (rawFile string, err error)
	Failed     func(source, query string, err error) error
	onlyPairs  map[string]bool
}

// RunBatch executes a budgeted sweep. Hooks persist artifacts; Completed must
// durably append results before returning their raw-file path. Checkpoint then
// commits that result prefix with pair identity and coverage. Supply a budget
// from DepthBudget and load prior records with LoadCheckpointResults on resume.
func RunBatch(ctx context.Context, state *BatchState, prior []library.PaperRecord, configurationErrors map[string]string, continueOnError bool, h BatchHooks) (resumedPairs int, err error) {
	if h.Job != nil {
		state.JobID = h.Job.Job().ID
		persist := h.Checkpoint
		h.Checkpoint = func() error {
			if err := persist(); err != nil {
				return err
			}
			reason := string(state.StopReason)
			// The pre-dispatch checkpoint reserves budget for a possible kill;
			// a live host has not actually interrupted until finish says so.
			if state.StopReason == sources.StopInterrupted && !state.Interrupted {
				reason = ""
			}
			return h.Job.Checkpoint(BatchJobProgress(state), reason)
		}
	}
	b := state.Budget
	papers := append([]library.PaperRecord{}, prior...)
	state.Coverage.UniqueRecords = len(DedupePapers(papers))
	count := len(papers)
	state.ResultCount = &count
	state.refreshCoverage(configurationErrors)
	state.StopReason = sources.StopCompleted
	state.Interrupted = false
	finish := func(reason sources.StopReason) (int, error) {
		state.StopReason = reason
		state.Interrupted = reason == sources.StopInterrupted
		return resumedPairs, h.Checkpoint()
	}
	for qi, query := range state.Queries {
		if ctx.Err() != nil {
			return finish(sources.StopInterrupted)
		}
		if qi >= b.MaxQueryVariants {
			return finish(sources.StopBudgetExhausted)
		}
		attempted := false
		for si, source := range state.Sources {
			if si >= b.MaxSources {
				break
			}
			if configurationErrors[source] != "" || (h.onlyPairs != nil && !h.onlyPairs[PairKey(source, query)]) {
				continue
			}
			if state.HasCompleted(source, query) {
				resumedPairs++
				continue
			}
			if ctx.Err() != nil {
				return finish(sources.StopInterrupted)
			}
			// Only preceding queries may stop this pair. Later saved coverage must
			// not prevent retrying an earlier gap or a newly introduced source.
			state.Coverage.ConsecutiveLowYield = state.lowYieldStreak(state.Queries[:qi])
			if state.Coverage.ConsecutiveLowYield >= b.SaturationQueries {
				return finish(sources.StopCoverageSaturated)
			}
			if state.RequestsUsed >= b.MaxAPICalls {
				return finish(sources.StopBudgetExhausted)
			}
			state.RequestsUsed++
			// An abrupt kill during the call must leave both usage and an interrupted checkpoint.
			state.StopReason = sources.StopInterrupted
			if err := h.Checkpoint(); err != nil {
				return resumedPairs, err
			}
			records, requestErr := h.Search(ctx, source, query)
			if ctx.Err() != nil {
				return finish(sources.StopInterrupted)
			}
			attempted = true
			index := state.coverageIndex(query)
			if requestErr != nil {
				state.Coverage.Queries[index].Complete = false
				state.Coverage.ConsecutiveLowYield = 0
				state.ConsecutiveErrors++
				state.RecordFailure(BatchFailure{Source: source, Query: query, Error: requestErr.Error()})
				if err := h.Failed(source, query, requestErr); err != nil {
					return resumedPairs, err
				}
				if !continueOnError {
					if _, err := finish(sources.StopErrorThreshold); err != nil {
						return resumedPairs, err
					}
					return resumedPairs, requestErr
				}
				if err := h.Checkpoint(); err != nil {
					return resumedPairs, err
				}
				if state.ConsecutiveErrors >= b.MaxConsecutiveErrors {
					return finish(sources.StopErrorThreshold)
				}
				continue
			}
			state.ConsecutiveErrors = 0
			before := state.Coverage.UniqueRecords
			papers = append(papers, records...)
			state.Coverage.UniqueRecords = len(DedupePapers(papers))
			state.Coverage.Queries[index].NewUnique += state.Coverage.UniqueRecords - before
			rawFile, err := h.Completed(source, query, qi, records)
			if err != nil {
				return resumedPairs, err
			}
			state.RecordCompleted(BatchPair{Source: source, Query: query, RawFile: rawFile, Records: len(records)})
			state.ClearFailure(source, query)
			count := len(papers)
			state.ResultCount = &count
			if err := h.Checkpoint(); err != nil {
				return resumedPairs, err
			}
		}
		state.refreshCoverage(configurationErrors)
		if attempted {
			if err := h.Checkpoint(); err != nil {
				return resumedPairs, err
			}
		}
	}
	if len(state.Sources) > b.MaxSources {
		return finish(sources.StopBudgetExhausted)
	}
	return finish(sources.StopCompleted)
}

// RetryBatch retries only the supplied failures through the same budgeted
// service as a full sweep. Other pending pairs remain for batch --resume.
func RetryBatch(ctx context.Context, s *BatchState, prior []library.PaperRecord, pending []BatchFailure, h BatchHooks) (int, error) {
	add := func(list []string, value string) []string {
		for _, v := range list {
			if v == value {
				return list
			}
		}
		return append(list, value)
	}
	h.onlyPairs = map[string]bool{}
	for _, f := range pending {
		s.Queries = add(s.Queries, f.Query)
		s.Sources = add(s.Sources, f.Source)
		h.onlyPairs[PairKey(f.Source, f.Query)] = true
	}
	s.Resume(s.Queries, s.Sources, s.Limit)
	if s.Budget.Depth == "" {
		s.Budget, _ = DepthBudget("")
	}
	return RunBatch(ctx, s, prior, nil, true, h)
}

// Only fully successful queries signal saturation. Reconcile against
// the active worklist so changed queries or added sources cannot inherit it.
func (s *BatchState) refreshCoverage(configurationErrors map[string]string) {
	active := []QueryCoverage{}
	for _, q := range s.Queries {
		for _, c := range s.Coverage.Queries {
			if c.Query != q {
				continue
			}
			complete := false
			for si, source := range s.Sources {
				if si >= s.Budget.MaxSources {
					break
				}
				if configurationErrors[source] != "" {
					continue
				}
				complete = true
				if !s.HasCompleted(source, q) {
					complete = false
					break
				}
			}
			c.Complete = complete
			active = append(active, c)
			break
		}
	}
	s.Coverage.Queries = active
	s.Coverage.ConsecutiveLowYield = s.lowYieldStreak(s.Queries)
}

func (s *BatchState) lowYieldStreak(queries []string) int {
	byQuery := map[string]QueryCoverage{}
	for _, c := range s.Coverage.Queries {
		byQuery[c.Query] = c
	}
	streak := 0
	for _, q := range queries {
		c, known := byQuery[q]
		// Unknown legacy yield is neutral, not a permanent barrier to later data.
		if !known || !c.Complete || c.NewUnique >= s.Budget.LowYieldThreshold {
			streak = 0
		} else {
			streak++
		}
	}
	return streak
}

func (s *BatchState) coverageIndex(query string) int {
	for i, c := range s.Coverage.Queries {
		if c.Query == query {
			return i
		}
	}
	s.Coverage.Queries = append(s.Coverage.Queries, QueryCoverage{Query: query})
	return len(s.Coverage.Queries) - 1
}

// DedupePapers is the same DOI/title identity policy used by aggregate output
// and marginal yield, so duplicate source hits never inflate coverage.
func DedupePapers(papers []library.PaperRecord) []library.PaperRecord {
	seen := map[string]int{}
	out := []library.PaperRecord{}
	for _, paper := range papers {
		key := strings.ToLower(strings.TrimSpace(paper.Identifiers.DOI))
		if key == "" {
			key = strings.ToLower(strings.Join(strings.Fields(paper.Title), " "))
		}
		if key == "" {
			key = fmt.Sprintf("record-%d", len(out))
		}
		if existing, ok := seen[key]; ok {
			out[existing] = library.MergeDuplicate(out[existing], paper)
			continue
		}
		seen[key] = len(out)
		out = append(out, paper)
	}
	return out
}
