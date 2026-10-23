package benchmarks

import (
	"context"
	"math"
	"reflect"
	"strings"
	"testing"
)

// Start with the real tiny replay, then hand-score three synthetic question
// outcomes. No evaluation/held-out outcomes are used to tune retrieval.
func comparisonFixture(t *testing.T) EngineReport {
	t.Helper()
	result, err := RunEngine(context.Background(), EngineOptions{SetDir: "testdata/fixture-tiny", Version: "old", Revision: "old-revision"})
	if err != nil {
		t.Fatal(err)
	}
	var report EngineReport
	if _, err = decodeStrict(result.Report, &report); err != nil {
		t.Fatal(err)
	}
	template := report.Runs[0]
	report.Runs = nil
	for i, q := range []struct {
		id, domain string
		gold, hits int64
	}{
		{"q-one", "artificial-photosynthesis", 3, 1},
		{"q-two", "artificial-photosynthesis", 1, 1},
		{"q-three", "other", 2, 0},
	} {
		r := template
		r.QuestionID, r.Domain, r.GoldCount = q.id, q.domain, int(q.gold)
		r.Returned, r.Recall = count(3), fraction(q.hits, q.gold)
		r.AtK = []AtK{{K: 1, Recall: fraction(q.hits, q.gold), Precision: fraction(q.hits, 1)}, {K: 5, Recall: fraction(q.hits, q.gold), Precision: fraction(q.hits, 5)}}
		r.HTTPRequests = count(i + 1)
		report.Runs = append(report.Runs, r)
	}
	report.Aggregates = aggregate(report.Runs, report.K)
	return report
}

func TestEngineReportComparison(t *testing.T) {
	baseline := comparisonFixture(t)
	candidate := comparisonFixture(t)
	candidate.RforgeVersion, candidate.Revision = "new", "new-revision"
	// q-one gains a second hit at rank 2; q-two loses its single hit;
	// q-three gains a hit after rank 1. Macro recall falls while micro rises.
	candidate.Runs[0].Recall = fraction(2, 3)
	candidate.Runs[0].AtK[1].Recall, candidate.Runs[0].AtK[1].Precision = fraction(2, 3), fraction(2, 5)
	candidate.Runs[1].Recall = fraction(0, 1)
	for i, k := range candidate.K {
		candidate.Runs[1].AtK[i] = AtK{K: k, Recall: fraction(0, 1), Precision: fraction(0, int64(k))}
	}
	candidate.Runs[2].Recall = fraction(1, 2)
	candidate.Runs[2].AtK[1].Recall, candidate.Runs[2].AtK[1].Precision = fraction(1, 2), fraction(1, 5)
	for i := range candidate.Runs {
		candidate.Runs[i].HTTPRequests = nil
	} // unavailable side => null delta
	candidate.Aggregates = aggregate(candidate.Runs, candidate.K)
	before, _ := CanonicalJSON(candidate)
	got, err := CompareEngineReports(baseline, candidate)
	if err != nil {
		t.Fatal(err)
	}
	if got.Baseline.Revision != "old-revision" || got.Candidate.Revision != "new-revision" || got.Baseline.SetChecksum != baseline.SetChecksum || got.Candidate.SnapshotsSHA256 == nil || len(got.Limitations) < 3 {
		t.Fatalf("missing identities/limitations: %+v", got)
	}
	for _, tc := range []struct {
		domain, metric           string
		k                        int
		n                        int
		base, cand, macro, micro float64
	}{
		{"", "recall", 0, 3, 4.0 / 9, 7.0 / 18, -1.0 / 18, 1.0 / 6},
		{"artificial-photosynthesis", "recall", 0, 2, 2.0 / 3, 1.0 / 3, -1.0 / 3, 0},
		{"other", "recall", 0, 1, 0, .5, .5, .5},
		{"", "recall_at_k", 1, 3, 4.0 / 9, 1.0 / 9, -1.0 / 3, -1.0 / 6},
		{"", "precision_at_k", 5, 3, 2.0 / 15, 1.0 / 5, 1.0 / 15, 1.0 / 15},
	} {
		a := findComparison(t, got, tc.domain, tc.metric, tc.k)
		if a.BaselineQuestionCount != tc.n || a.CandidateQuestionCount != tc.n {
			t.Fatalf("counts: %+v", a)
		}
		for _, pair := range [][2]*float64{{a.Macro.Baseline, &tc.base}, {a.Macro.Candidate, &tc.cand}, {a.Macro.Delta, &tc.macro}, {a.Micro.Delta, &tc.micro}} {
			if pair[0] == nil || math.Abs(*pair[0]-*pair[1]) > 1e-12 {
				t.Fatalf("%s/%s/%d: %+v", tc.domain, tc.metric, tc.k, a)
			}
		}
	}
	for _, metric := range []string{"citation_correctness", "cost_usd"} {
		a := findComparison(t, got, "", metric, 0)
		if a.Macro.Delta != nil || a.Macro.Baseline != nil || a.BaselineQuestionCount != 0 || a.Micro != nil && a.Micro.Delta != nil {
			t.Fatalf("unavailable became measured: %+v", a)
		}
	}
	http := findComparison(t, got, "", "http_requests", 0)
	if http.BaselineQuestionCount != 3 || http.CandidateQuestionCount != 0 || http.Macro.Delta != nil || http.Micro != nil {
		t.Fatalf("availability: %+v", http)
	}
	after, _ := CanonicalJSON(candidate)
	if !reflect.DeepEqual(before, after) {
		t.Fatal("comparison modified candidate")
	}
	self, err := CompareEngineReports(baseline, baseline)
	if err != nil {
		t.Fatal(err)
	}
	for _, a := range self.Aggregates {
		if a.Macro.Baseline != nil && (a.Macro.Delta == nil || *a.Macro.Delta != 0) {
			t.Fatalf("nonzero self delta: %+v", a)
		}
		if a.Micro != nil && a.Micro.Baseline != nil && (a.Micro.Delta == nil || *a.Micro.Delta != 0) {
			t.Fatalf("nonzero micro self delta: %+v", a)
		}
	}
}

func TestEngineReportComparisonRejectsIncompatibleInputs(t *testing.T) {
	tests := []struct {
		name        string
		mutate      func(*EngineReport)
		reaggregate bool
	}{
		{"schema", func(r *EngineReport) { r.SchemaVersion = 2 }, false},
		{"tier", func(r *EngineReport) { r.Tier = "agent" }, false},
		{"record", func(r *EngineReport) { r.Mode = "record" }, false},
		{"demo", func(r *EngineReport) { r.Purpose = "demo" }, false},
		{"purpose", func(r *EngineReport) { r.Purpose = "evaluation"; r.SetID = "other" }, false},
		{"invalid", func(r *EngineReport) { r.Status = "invalid" }, false},
		{"errors", func(r *EngineReport) { r.Errors = []RunError{{Code: "source_error"}} }, false},
		{"paired", func(r *EngineReport) { r.Paired = []any{1} }, false},
		{"set", func(r *EngineReport) { r.SetID = "fixture-other" }, false},
		{"checksum", func(r *EngineReport) { r.SetChecksum = strings.Repeat("a", 64) }, false},
		{"snapshot", func(r *EngineReport) { v := strings.Repeat("b", 64); r.SnapshotsSHA256 = &v }, false},
		{"no snapshot", func(r *EngineReport) { r.SnapshotsSHA256 = nil }, false},
		{"protocol", func(r *EngineReport) { r.ProtocolSHA256 = strings.Repeat("c", 64) }, false},
		{"malformed hash", func(r *EngineReport) { r.SetChecksum = "bad" }, false},
		{"split", func(r *EngineReport) { r.Split = "dev" }, false},
		{"K", func(r *EngineReport) { r.K = []int{1, 10} }, false},
		{"duplicate K", func(r *EngineReport) { r.K = []int{1, 1} }, false},
		{"empty version", func(r *EngineReport) { r.RforgeVersion = " " }, false},
		{"missing question", func(r *EngineReport) { r.Runs = r.Runs[1:] }, true},
		{"duplicate question", func(r *EngineReport) { r.Runs = append(r.Runs, r.Runs[0]) }, true},
		{"domain", func(r *EngineReport) { r.Runs[0].Domain = "changed" }, true},
		{"gold count", func(r *EngineReport) {
			r.Runs[2].GoldCount = 3
			r.Runs[2].Recall = fraction(0, 3)
			for i := range r.Runs[2].AtK {
				r.Runs[2].AtK[i].Recall = fraction(0, 3)
			}
		}, true},
		{"condition", func(r *EngineReport) { r.Runs[0].Condition = "with_rforge" }, false},
		{"repetition", func(r *EngineReport) { r.Runs[0].Repetition = 2 }, false},
		{"stop reason", func(r *EngineReport) { r.Runs[0].StopReason = "invented" }, false},
		{"negative count", func(r *EngineReport) { r.Runs[0].HTTPRequests = count(-1) }, true},
		{"missing retrieval", func(r *EngineReport) { r.Runs[0].Recall = Ratio{} }, true},
		{"ratio value", func(r *EngineReport) { v := .5; r.Runs[0].Recall.Value = &v }, true},
		{"ratio count", func(r *EngineReport) { r.Runs[0].Recall.Numerator = nil }, false},
		{"ratio bounds", func(r *EngineReport) { r.Runs[0].Recall = fraction(4, 3) }, true},
		{"ratio gold denominator", func(r *EngineReport) { r.Runs[0].Recall = fraction(1, 2) }, true},
		{"missing at K", func(r *EngineReport) { r.Runs[0].AtK = r.Runs[0].AtK[:1] }, false},
		{"duplicate at K", func(r *EngineReport) { r.Runs[0].AtK[1] = r.Runs[0].AtK[0] }, false},
		{"precision denominator", func(r *EngineReport) { r.Runs[0].AtK[1].Precision = fraction(1, 3) }, true},
		{"precision hits", func(r *EngineReport) { r.Runs[0].AtK[1].Precision = fraction(0, 5) }, true},
		{"rank consistency", func(r *EngineReport) {
			r.Runs[0].AtK[0].Recall = fraction(0, 3)
			r.Runs[0].AtK[0].Precision = fraction(0, 1)
			r.Runs[0].Returned = count(1)
		}, true},
		{"dedupe status", func(r *EngineReport) { r.Runs[0].DedupeStatus = "invalid" }, false},
		{"missing dedupe count", func(r *EngineReport) { r.Runs[0].TrueMerges = nil }, true},
		{"citation status", func(r *EngineReport) { r.Runs[0].CitationStatus = "measured" }, false},
		{"citation counts", func(r *EngineReport) { r.Runs[0].CitationCorrectness = fraction(1, 1) }, true},
		{"engine tokens", func(r *EngineReport) { r.Runs[0].InputTokens = count(1) }, true},
		{"engine tool calls", func(r *EngineReport) { r.Runs[0].ToolCalls = count(1) }, true},
		{"missing aggregate", func(r *EngineReport) { r.Aggregates = r.Aggregates[1:] }, false},
		{"duplicate aggregate", func(r *EngineReport) { r.Aggregates = append(r.Aggregates, r.Aggregates[0]) }, false},
		{"unknown aggregate", func(r *EngineReport) { r.Aggregates[0].Metric = "invented" }, false},
		{"aggregate count", func(r *EngineReport) { r.Aggregates[0].QuestionCount = 1 }, false},
		{"aggregate macro", func(r *EngineReport) { v := 1.; r.Aggregates[0].MacroMean = &v }, false},
		{"aggregate micro", func(r *EngineReport) { r.Aggregates[0].Micro = &Ratio{}; r.Aggregates[0].Micro.Value = new(float64) }, false},
		{"different contributing counts", func(r *EngineReport) { r.Runs[0].HTTPRequests = nil }, true},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			baseline, candidate := comparisonFixture(t), comparisonFixture(t)
			tc.mutate(&candidate)
			if tc.reaggregate {
				candidate.Aggregates = aggregate(candidate.Runs, candidate.K)
			}
			if _, err := CompareEngineReports(baseline, candidate); err == nil {
				t.Fatal("accepted incompatible/inconsistent candidate")
			}
			if _, err := CompareEngineReports(candidate, baseline); err == nil {
				t.Fatal("accepted incompatible/inconsistent baseline")
			}
		})
	}
	t.Run("same count different contributors", func(t *testing.T) {
		baseline, candidate := comparisonFixture(t), comparisonFixture(t)
		baseline.Runs[0].HTTPRequests = nil
		candidate.Runs[1].HTTPRequests = nil
		baseline.Aggregates = aggregate(baseline.Runs, baseline.K)
		candidate.Aggregates = aggregate(candidate.Runs, candidate.K)
		if _, err := CompareEngineReports(baseline, candidate); err == nil {
			t.Fatal("subtracted different measured question populations")
		}
	})
}

func findComparison(t *testing.T, c EngineComparison, domain, metric string, k int) EngineAggregateComparison {
	t.Helper()
	for _, a := range c.Aggregates {
		if (a.Domain == nil && domain == "" || a.Domain != nil && *a.Domain == domain) && a.Metric == metric && (a.K == nil && k == 0 || a.K != nil && *a.K == k) {
			return a
		}
	}
	t.Fatalf("missing aggregate %s/%s/%d", domain, metric, k)
	return EngineAggregateComparison{}
}
