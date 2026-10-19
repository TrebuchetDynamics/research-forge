package main

import (
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"math"
	"math/rand"
	"os"
	"reflect"
	"strings"

	b "github.com/TrebuchetDynamics/research-forge/internal/benchmarks"
)

const scorerVersion = "agent-v1"

var conditions = []string{"with_rforge", "without_rforge"}
var metrics = []string{"citation_correctness", "cost_usd", "dedupe_false_merges", "dedupe_missed_merges", "dedupe_true_merges", "dedupe_unjudged_merges", "http_requests", "input_tokens", "output_tokens", "precision_at_k", "recall", "recall_at_k", "search_calls", "tool_calls", "wall_clock_ms"}

type Host struct {
	Name    string `json:"name"`
	Version string `json:"version"`
}
type Model struct {
	Provider string `json:"provider"`
	Version  string `json:"version"`
}
type Sampling struct {
	Temperature float64 `json:"temperature"`
	TopP        float64 `json:"top_p"`
	PairedSeeds bool    `json:"paired_seeds"`
}
type Caps struct {
	Tokens       int64 `json:"tokens"`
	SearchCalls  int64 `json:"search_calls"`
	ToolCalls    int64 `json:"tool_calls"`
	HTTPRequests int64 `json:"http_requests"`
	WallClockMS  int64 `json:"wall_clock_ms"`
}
type Prompts struct {
	System  string `json:"system"`
	Task    string `json:"task"`
	With    string `json:"with_rforge"`
	Without string `json:"without_rforge"`
}
type Protocol struct {
	Version         int      `json:"schema_version"`
	Host            Host     `json:"host"`
	Model           Model    `json:"model"`
	Sampling        Sampling `json:"sampling"`
	Prompts         Prompts  `json:"prompts"`
	RforgeVersion   string   `json:"rforge_version"`
	Revision        string   `json:"revision"`
	SetChecksum     string   `json:"set_checksum"`
	QuestionIDs     []string `json:"question_ids"`
	Repetitions     int      `json:"repetitions"`
	Caps            Caps     `json:"caps"`
	RunOrderSeed    int64    `json:"run_order_seed"`
	ScorerVersion   string   `json:"scorer_version"`
	CitationVersion string   `json:"citation_metric_version"`
	Primary         []string `json:"primary_metrics"`
	Secondary       []string `json:"secondary_metrics"`
}
type Slot struct {
	QuestionID string `json:"question_id"`
	Repetition int    `json:"repetition"`
	Condition  string `json:"condition"`
}
type Paper struct {
	Identifiers []b.Identifier `json:"identifiers"`
}
type Claim struct {
	Text    string `json:"text"`
	Quote   string `json:"quote"`
	Locator string `json:"locator"`
}
type Answer struct {
	Bibliography []Paper `json:"bibliography"`
	Claims       []Claim `json:"claims"`
}
type Usage struct {
	InputTokens  int64   `json:"input_tokens"`
	OutputTokens int64   `json:"output_tokens"`
	CostUSD      float64 `json:"cost_usd"`
	SearchCalls  int64   `json:"search_calls"`
	ToolCalls    int64   `json:"tool_calls"`
	HTTPRequests int64   `json:"http_requests"`
	WallClockMS  int64   `json:"wall_clock_ms"`
}
type Gate struct {
	ReferenceReviewLeak bool  `json:"reference_review_leak"`
	DateCapEnforced     bool  `json:"date_cap_enforced"`
	CompleteAccounting  bool  `json:"complete_accounting"`
	BudgetChecked       bool  `json:"budget_checked"`
	DateExcluded        int64 `json:"date_excluded"`
}
type RawRun struct {
	Slot       Slot     `json:"slot"`
	Model      Model    `json:"model"`
	Caps       Caps     `json:"caps"`
	RunID      string   `json:"run_id"`
	Transcript string   `json:"transcript"`
	Artifacts  []string `json:"workspace_artifacts"`
	StopReason string   `json:"stop_reason"`
	Usage      Usage    `json:"usage"`
	Gate       Gate     `json:"gate"`
	Answer     Answer   `json:"answer"`
}
type Experiment struct {
	ProtocolSHA256 string   `json:"protocol_sha256"`
	Schedule       []Slot   `json:"schedule"`
	Runs           []RawRun `json:"runs"`
}

func hashBytes(data []byte) string { h := sha256.Sum256(data); return hex.EncodeToString(h[:]) }
func hash(v any) string {
	data, err := b.CanonicalJSON(v)
	if err != nil {
		panic(err)
	}
	return hashBytes(data[:len(data)-1])
}
func save(path string, v any) error {
	data, err := b.CanonicalJSON(v)
	if err != nil {
		return err
	}
	f, err := os.OpenFile(path, os.O_WRONLY|os.O_CREATE|os.O_EXCL, 0600)
	if err != nil {
		return err
	}
	_, err = f.Write(data)
	closeErr := f.Close()
	if err != nil {
		return err
	}
	return closeErr
}
func schedule(p Protocol) []Slot {
	// Balance first-condition assignment globally, then shuffle pair order while
	// retaining adjacent members to limit temporal service/model drift.
	type pair struct {
		q     string
		r     int
		first int
	}
	pairs := []pair{}
	for _, q := range p.QuestionIDs {
		for r := 1; r <= p.Repetitions; r++ {
			pairs = append(pairs, pair{q: q, r: r})
		}
	}
	rng := rand.New(rand.NewSource(p.RunOrderSeed))
	rng.Shuffle(len(pairs), func(i, j int) { pairs[i], pairs[j] = pairs[j], pairs[i] })
	// Assign the balanced vector AFTER shuffling pair identities. Attaching
	// parity before shuffling would randomize presentation but leave first
	// condition confounded with the original question/repetition index.
	flip := rng.Intn(2)
	for i := range pairs {
		pairs[i].first = (i + flip) % 2
	}
	out := []Slot{}
	for _, x := range pairs {
		for i := 0; i < 2; i++ {
			out = append(out, Slot{x.q, x.r, conditions[(x.first+i)%2]})
		}
	}
	return out
}
func validateProtocol(p Protocol, s b.Set) error {
	ids := []string{}
	for _, q := range s.Questions {
		if q.Split == "held_out" {
			ids = append(ids, q.QuestionID)
		}
	}
	if p.Version != 1 || (p.Host.Name != "pi" && p.Host.Name != "codex") || p.SetChecksum != s.SetChecksum || !reflect.DeepEqual(ids, p.QuestionIDs) || p.Repetitions < 1 || p.Repetitions > 100 || p.ScorerVersion != scorerVersion || p.CitationVersion != "not_implemented" || !reflect.DeepEqual(p.Primary, []string{"recall"}) {
		return fmt.Errorf("protocol_violation: incompatible protocol")
	}
	secondary := []string{}
	for _, m := range metrics {
		if m != "recall" {
			secondary = append(secondary, m)
		}
	}
	if !reflect.DeepEqual(p.Secondary, secondary) {
		return fmt.Errorf("protocol_violation: missing secondary metrics")
	}
	for _, v := range []string{p.Host.Version, p.Model.Provider, p.Model.Version, p.RforgeVersion, p.Revision, p.Prompts.System, p.Prompts.Task, p.Prompts.With, p.Prompts.Without} {
		if strings.TrimSpace(v) == "" {
			return fmt.Errorf("protocol_violation: missing identity/prompt")
		}
	}
	if p.Sampling.Temperature < 0 || p.Sampling.TopP <= 0 || p.Sampling.TopP > 1 || p.Caps.Tokens < 1 || p.Caps.SearchCalls < 1 || p.Caps.ToolCalls < 1 || p.Caps.HTTPRequests < 1 || p.Caps.WallClockMS < 1 {
		return fmt.Errorf("protocol_violation: invalid sampling/caps")
	}
	if !strings.HasPrefix(s.SetID, "fixture-") && (len(ids) < 8 || p.Repetitions != 3 || p.Model.Provider == "stub") {
		return fmt.Errorf("protocol_violation: not a registered pilot")
	}
	return nil
}
func loadPrivate(dir string) (Protocol, Experiment, error) {
	var p Protocol
	var e Experiment
	if err := b.ReadAgentArtifact(dir+"/protocol.json", &p); err != nil {
		return p, e, err
	}
	if err := b.ReadAgentArtifact(dir+"/experiment.json", &e); err != nil {
		return p, e, err
	}
	if e.ProtocolSHA256 != hash(p) || !reflect.DeepEqual(e.Schedule, schedule(p)) || len(e.Runs) != len(e.Schedule) {
		return p, e, fmt.Errorf("protocol_violation: protocol/schedule/matrix mismatch")
	}
	for i, r := range e.Runs {
		if r.Slot != e.Schedule[i] || r.Model != p.Model || r.Caps != p.Caps {
			return p, e, fmt.Errorf("protocol_violation: unmatched pair")
		}
		u := r.Usage
		if !r.Gate.DateCapEnforced || !r.Gate.CompleteAccounting || !r.Gate.BudgetChecked || r.Gate.DateExcluded < 0 {
			return p, e, fmt.Errorf("protocol_violation: unqualified gateway")
		}
		if r.Gate.ReferenceReviewLeak {
			return p, e, fmt.Errorf("reference_review_leak")
		}
		if u.InputTokens < 0 || u.OutputTokens < 0 || u.CostUSD < 0 || math.IsNaN(u.CostUSD) || math.IsInf(u.CostUSD, 0) || u.SearchCalls < 0 || u.ToolCalls < 0 || u.HTTPRequests < 0 || u.WallClockMS < 0 {
			return p, e, fmt.Errorf("protocol_violation: invalid usage")
		}
		if u.InputTokens > p.Caps.Tokens-u.OutputTokens || u.SearchCalls > p.Caps.SearchCalls || u.ToolCalls > p.Caps.ToolCalls || u.HTTPRequests > p.Caps.HTTPRequests || u.WallClockMS > p.Caps.WallClockMS {
			if r.StopReason != "budget_exhausted" {
				return p, e, fmt.Errorf("protocol_violation: unreported budget overrun")
			}
		}
		switch r.StopReason {
		case "completed", "budget_exhausted", "coverage_saturated", "interrupted", "error_threshold":
		default:
			return p, e, fmt.Errorf("protocol_violation: invalid stop")
		}
	}
	return p, e, nil
}
