package benchmarks

import (
	"fmt"
	"math"
	"reflect"
	"sort"
	"strings"
)

// EngineReportIdentity keeps both implementation and input identities visible.
// It describes report assertions, not authenticated input custody or approval.
type EngineReportIdentity struct {
	SchemaVersion   int     `json:"schema_version"`
	Purpose         string  `json:"purpose"`
	Tier            string  `json:"tier"`
	Mode            string  `json:"mode"`
	RforgeVersion   string  `json:"rforge_version"`
	Revision        string  `json:"revision"`
	SetID           string  `json:"set_id"`
	SetChecksum     string  `json:"set_checksum"`
	SnapshotsSHA256 *string `json:"snapshots_sha256"`
	ProtocolSHA256  string  `json:"protocol_sha256"`
	Split           string  `json:"split"`
	K               []int   `json:"k"`
}

// MeasurementDelta is signed candidate minus baseline. Missing measurements
// remain nil, including when only one side is measured.
type MeasurementDelta struct {
	Baseline  *float64 `json:"baseline"`
	Candidate *float64 `json:"candidate"`
	Delta     *float64 `json:"delta"`
}

type EngineAggregateComparison struct {
	Condition              string            `json:"condition"`
	Domain                 *string           `json:"domain"`
	Metric                 string            `json:"metric"`
	K                      *int              `json:"k"`
	BaselineQuestionCount  int               `json:"baseline_question_count"`
	CandidateQuestionCount int               `json:"candidate_question_count"`
	Macro                  MeasurementDelta  `json:"macro"`
	Micro                  *MeasurementDelta `json:"micro,omitempty"`
}

type EngineComparison struct {
	SchemaVersion int                         `json:"schema_version"`
	Baseline      EngineReportIdentity        `json:"baseline"`
	Candidate     EngineReportIdentity        `json:"candidate"`
	Aggregates    []EngineAggregateComparison `json:"aggregates"`
	Limitations   []string                    `json:"limitations"`
}

func reportIdentity(r EngineReport) EngineReportIdentity {
	return EngineReportIdentity{r.SchemaVersion, r.Purpose, r.Tier, r.Mode, r.RforgeVersion, r.Revision, r.SetID, r.SetChecksum, r.SnapshotsSHA256, r.ProtocolSHA256, r.Split, r.K}
}

func measurementDelta(b, c *float64) MeasurementDelta {
	d := MeasurementDelta{Baseline: b, Candidate: c}
	if b != nil && c != nil {
		v := *c - *b
		d.Delta = &v
	}
	return d
}

// CompareEngineReports computes qualified offline deltas without executing the
// engine, modifying its reports, or making a release/agent-quality judgment.
func CompareEngineReports(baseline, candidate EngineReport) (EngineComparison, error) {
	b, bp, err := validateComparisonReport(baseline)
	if err != nil {
		return EngineComparison{}, fmt.Errorf("baseline: %w", err)
	}
	c, cp, err := validateComparisonReport(candidate)
	if err != nil {
		return EngineComparison{}, fmt.Errorf("candidate: %w", err)
	}
	bi, ci := reportIdentity(baseline), reportIdentity(candidate)
	bi.RforgeVersion, bi.Revision = "", ""
	ci.RforgeVersion, ci.Revision = "", ""
	if !reflect.DeepEqual(bi, ci) {
		return EngineComparison{}, fmt.Errorf("incompatible report identities")
	}
	bm, cm := questionMatrix(baseline.Runs), questionMatrix(candidate.Runs)
	if !reflect.DeepEqual(bm, cm) {
		return EngineComparison{}, fmt.Errorf("incompatible question/domain/gold-count matrix")
	}
	for i, a := range b {
		if a.MacroMean != nil && c[i].MacroMean != nil && !reflect.DeepEqual(bp[comparisonKey(a)], cp[comparisonKey(c[i])]) {
			return EngineComparison{}, fmt.Errorf("different contributing questions for %s", a.Metric)
		}
	}
	out := EngineComparison{SchemaVersion: 1, Baseline: reportIdentity(baseline), Candidate: reportIdentity(candidate), Aggregates: []EngineAggregateComparison{}, Limitations: []string{
		"Replay latency is captured timing, not current implementation speed.",
		"Reports alone do not prove input custody, set approval, or engine verification.",
		"Differences are not a release, superiority, or agent-improvement verdict.",
		"Precision is gold-set precision; non-gold papers are not necessarily irrelevant.",
	}}
	for i, a := range b {
		other := c[i]
		row := EngineAggregateComparison{Condition: a.Condition, Domain: a.Domain, Metric: a.Metric, K: a.K, BaselineQuestionCount: a.QuestionCount, CandidateQuestionCount: other.QuestionCount, Macro: measurementDelta(a.MacroMean, other.MacroMean)}
		if a.Micro != nil {
			d := measurementDelta(a.Micro.Value, other.Micro.Value)
			row.Micro = &d
		}
		out.Aggregates = append(out.Aggregates, row)
	}
	return out, nil
}

// Keys use a separate domain-presence bit so null and an actual domain cannot
// collide. Valid K values are positive; zero represents null.
type aggregateComparisonKey struct {
	condition, domain, metric string
	hasDomain                 bool
	k                         int
}

func comparisonKey(a Aggregate) aggregateComparisonKey {
	key := aggregateComparisonKey{condition: a.Condition, metric: a.Metric}
	if a.Domain != nil {
		key.domain, key.hasDomain = *a.Domain, true
	}
	if a.K != nil {
		key.k = *a.K
	}
	return key
}

type comparisonQuestion struct {
	domain string
	gold   int
}

func questionMatrix(runs []Run) map[string]comparisonQuestion {
	m := map[string]comparisonQuestion{}
	for _, r := range runs {
		m[r.QuestionID] = comparisonQuestion{r.Domain, r.GoldCount}
	}
	return m
}

// validateComparisonReport recomputes rather than trusting supplied aggregates.
// Single-question aggregates expose contributor identities using the very same
// availability rules, without maintaining a second metric-selection formula.
func validateComparisonReport(r EngineReport) ([]Aggregate, map[aggregateComparisonKey][]string, error) {
	fail := func(why string) ([]Aggregate, map[aggregateComparisonKey][]string, error) {
		return nil, nil, fmt.Errorf("invalid engine replay report: %s", why)
	}
	if r.SchemaVersion != 1 || r.Tier != "engine" || r.Mode != "replay" || r.Status != "complete" || len(r.Errors) != 0 || len(r.Paired) != 0 {
		return fail("requires complete, error-free schema-v1 engine replay")
	}
	if (r.Purpose != "fixture" && r.Purpose != "evaluation") || strings.TrimSpace(r.SetID) == "" || (r.Purpose == "fixture") != strings.HasPrefix(r.SetID, "fixture-") {
		return fail("purpose/set ID")
	}
	if !hashRE.MatchString(r.SetChecksum) || r.SnapshotsSHA256 == nil || !hashRE.MatchString(*r.SnapshotsSHA256) || !hashRE.MatchString(r.ProtocolSHA256) {
		return fail("input digests")
	}
	if strings.TrimSpace(r.RforgeVersion) == "" || strings.TrimSpace(r.Revision) == "" || (r.Split != "dev" && r.Split != "held_out") {
		return fail("implementation identity/split")
	}
	if len(r.K) == 0 || len(r.Runs) == 0 {
		return fail("empty K/question matrix")
	}
	for i, k := range r.K {
		if k <= 0 || i > 0 && k <= r.K[i-1] {
			return fail("K must be positive, unique and ascending")
		}
	}
	runs := append([]Run(nil), r.Runs...)
	sort.Slice(runs, func(i, j int) bool { return runs[i].QuestionID < runs[j].QuestionID })
	populations := map[aggregateComparisonKey][]string{}
	// Guard the int64 additions performed by aggregate for micro ratios. All
	// metric denominators are bounded by this conservative per-report total.
	totals := map[aggregateComparisonKey]int64{}
	for i, row := range runs {
		if i > 0 && row.QuestionID == runs[i-1].QuestionID {
			return fail("duplicate question")
		}
		if err := validateComparisonRun(row, r.K); err != nil {
			return fail(fmt.Sprintf("question %q: %v", row.QuestionID, err))
		}
		for _, a := range aggregate([]Run{row}, r.K) {
			key := comparisonKey(a)
			if a.QuestionCount > 0 {
				populations[key] = append(populations[key], row.QuestionID)
			}
			if a.Micro != nil && a.Micro.Denominator != nil {
				d := *a.Micro.Denominator
				if totals[key] > math.MaxInt64-d {
					return fail("micro denominator overflow")
				}
				totals[key] += d
			}
		}
	}
	expected := aggregate(runs, r.K)
	if len(r.Aggregates) != len(expected) {
		return fail("missing/extra aggregates")
	}
	supplied := map[aggregateComparisonKey]Aggregate{}
	for _, a := range r.Aggregates {
		key := comparisonKey(a)
		if _, ok := supplied[key]; ok {
			return fail("duplicate aggregate")
		}
		supplied[key] = a
	}
	for _, a := range expected {
		if actual, ok := supplied[comparisonKey(a)]; !ok || !reflect.DeepEqual(a, actual) {
			return fail("aggregate disagrees with rows")
		}
	}
	return expected, populations, nil
}

func validateComparisonRatio(r Ratio) error {
	if r.Numerator == nil && r.Denominator == nil && r.Value == nil {
		return nil
	}
	if r.Numerator == nil || r.Denominator == nil || *r.Numerator < 0 || *r.Denominator < 0 || *r.Numerator > *r.Denominator {
		return fmt.Errorf("invalid ratio counts")
	}
	if *r.Denominator == 0 {
		if r.Value != nil {
			return fmt.Errorf("zero-denominator ratio must be null")
		}
	} else if r.Value == nil || *r.Value != float64(*r.Numerator)/float64(*r.Denominator) {
		return fmt.Errorf("ratio value disagrees with counts")
	}
	return nil
}

func validateComparisonRun(r Run, ks []int) error {
	fail := func(why string) error { return fmt.Errorf("%s", why) }
	if strings.TrimSpace(r.QuestionID) == "" || strings.TrimSpace(r.Domain) == "" || r.Condition != "engine" || r.Repetition != 1 || r.GoldCount <= 0 {
		return fail("question identity/condition/gold count")
	}
	switch r.StopReason {
	case "completed", "budget_exhausted", "coverage_saturated", "interrupted", "error_threshold":
	default:
		return fail("stop reason")
	}
	for _, n := range []*int64{r.Returned, r.TrueMerges, r.FalseMerges, r.UnjudgedMerges, r.MissedMerges, r.HTTPRequests, r.SearchCalls, r.ToolCalls, r.WallClockMS, r.InputTokens, r.OutputTokens, r.DateExcluded} {
		if n != nil && *n < 0 {
			return fail("negative count")
		}
	}
	if r.Returned == nil || r.ToolCalls == nil || *r.ToolCalls != 0 || r.InputTokens != nil || r.OutputTokens != nil || r.CostUSD != nil {
		return fail("engine output/usage")
	}
	if r.DedupeStatus != "measured" || r.TrueMerges == nil || r.FalseMerges == nil || r.UnjudgedMerges == nil || r.MissedMerges == nil {
		return fail("engine dedupe must be measured")
	}
	if err := validateComparisonRatio(r.Recall); err != nil {
		return err
	}
	if r.Recall.Value == nil || *r.Recall.Denominator != int64(r.GoldCount) || *r.Recall.Numerator > *r.Returned {
		return fail("retrieval counts")
	}
	if len(r.AtK) != len(ks) {
		return fail("missing/extra at-K entries")
	}
	previousHits, previousK := int64(0), int64(0)
	for i, at := range r.AtK {
		if at.K != ks[i] {
			return fail("at-K matrix")
		}
		if err := validateComparisonRatio(at.Recall); err != nil {
			return err
		}
		if err := validateComparisonRatio(at.Precision); err != nil {
			return err
		}
		if at.Recall.Value == nil || at.Precision.Value == nil || *at.Recall.Denominator != int64(r.GoldCount) || *at.Precision.Denominator != int64(at.K) || *at.Recall.Numerator != *at.Precision.Numerator {
			return fail("at-K ratio counts")
		}
		hits, k := *at.Recall.Numerator, int64(at.K)
		ranks := min(k, *r.Returned)
		if hits < previousHits || hits-previousHits > k-previousK || hits > ranks || hits > *r.Recall.Numerator || *r.Recall.Numerator-hits > *r.Returned-ranks {
			return fail("inconsistent ranked hits")
		}
		previousHits, previousK = hits, k
	}
	if err := validateComparisonRatio(r.CitationCorrectness); err != nil {
		return err
	}
	switch r.CitationStatus {
	case "not_implemented":
		if r.CitationCorrectness != (Ratio{}) {
			return fail("unimplemented citation counts must be null")
		}
	case "measured":
		if r.CitationCorrectness.Value == nil {
			return fail("missing measured citations")
		}
	case "no_citations":
		c := r.CitationCorrectness
		if c.Numerator == nil || c.Denominator == nil || *c.Numerator != 0 || *c.Denominator != 0 || c.Value != nil {
			return fail("no-citation ratio must be 0/0/null")
		}
	default:
		return fail("citation status")
	}
	return nil
}
