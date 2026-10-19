package benchmarks

import (
	"sort"

	"github.com/TrebuchetDynamics/research-forge/internal/sources"
)

func count(n int) *int64 { x := int64(n); return &x }
func fraction(n, d int64) Ratio {
	r := Ratio{Numerator: &n, Denominator: &d}
	if d > 0 {
		v := float64(n) / float64(d)
		r.Value = &v
	}
	return r
}
func score(r *Run, q Question, ks []int, clusters [][]string, raw map[string]sources.SourceRecord, labels map[string]string) {
	goldAliases := map[string]string{}
	for _, g := range q.Gold {
		for a := range aliases(g.Identifiers) {
			goldAliases[a] = g.PaperID
		}
	}
	hits := []int{}
	credited := map[string]bool{}
	membership := map[string]int{}
	trueM, falseM, unjudged, missed := 0, 0, 0, 0
	for rank, cluster := range clusters {
		entities := map[string]bool{}
		gold := map[string]bool{}
		for _, id := range cluster {
			membership[id] = rank
			if labels[id] != "" {
				entities[labels[id]] = true
			}
			for a := range sourceAliases(raw[id]) {
				if g := goldAliases[a]; g != "" {
					gold[g] = true
				}
			}
		}
		if len(entities) <= 1 && len(gold) == 1 {
			for g := range gold {
				conflict := false
				for entity := range entities {
					conflict = conflict || entity != g
				}
				if !conflict && !credited[g] {
					credited[g] = true
					hits = append(hits, rank+1)
				}
			}
		}
		for i, a := range cluster {
			for _, b := range cluster[i+1:] {
				if labels[a] == "" || labels[b] == "" {
					unjudged++
				} else if labels[a] == labels[b] {
					trueM++
				} else {
					falseM++
				}
			}
		}
	}
	ids := []string{}
	for id := range membership {
		ids = append(ids, id)
	}
	sort.Strings(ids)
	for i, a := range ids {
		for _, b := range ids[i+1:] {
			if membership[a] != membership[b] && labels[a] != "" && labels[a] == labels[b] {
				missed++
			}
		}
	}
	r.TrueMerges = count(trueM)
	r.FalseMerges = count(falseM)
	r.UnjudgedMerges = count(unjudged)
	r.MissedMerges = count(missed)
	r.Recall = fraction(int64(len(hits)), int64(len(q.Gold)))
	for _, k := range ks {
		n := int64(0)
		for _, rank := range hits {
			if rank <= k {
				n++
			}
		}
		r.AtK = append(r.AtK, AtK{K: k, Recall: fraction(n, int64(len(q.Gold))), Precision: fraction(n, int64(k))})
	}
}
func invalidRow(r Run, ks []int) Run {
	out := Run{QuestionID: r.QuestionID, Domain: r.Domain, Condition: r.Condition, Repetition: r.Repetition, StopReason: r.StopReason, GoldCount: r.GoldCount, DedupeStatus: "invalid", CitationStatus: "invalid", AtK: []AtK{}}
	for _, k := range ks {
		out.AtK = append(out.AtK, AtK{K: k})
	}
	return out
}

// Engine rows have exactly one repetition. The shared agent protocol must first
// reduce equal-N repetitions within question before using these aggregates.
func aggregate(runs []Run, ks []int) []Aggregate {
	domains := []*string{nil}
	seen := map[string]bool{}
	names := []string{}
	for _, r := range runs {
		if !seen[r.Domain] {
			seen[r.Domain] = true
			names = append(names, r.Domain)
		}
	}
	sort.Strings(names)
	for _, d := range names {
		x := d
		domains = append(domains, &x)
	}
	metrics := []string{"recall", "recall_at_k", "precision_at_k", "dedupe_true_merges", "dedupe_false_merges", "dedupe_unjudged_merges", "dedupe_missed_merges", "http_requests", "search_calls", "tool_calls", "wall_clock_ms", "input_tokens", "output_tokens", "cost_usd", "citation_correctness"}
	sort.Strings(metrics)
	out := []Aggregate{}
	for _, domain := range domains {
		for _, m := range metrics {
			atks := []*int{nil}
			if m == "recall_at_k" || m == "precision_at_k" {
				atks = nil
				for _, k := range ks {
					x := k
					atks = append(atks, &x)
				}
			}
			for _, k := range atks {
				a := Aggregate{Condition: "engine", Domain: domain, Metric: m, K: k}
				sum := float64(0)
				num, den := int64(0), int64(0)
				ratioMetric := m == "recall" || m == "recall_at_k" || m == "precision_at_k" || m == "citation_correctness"
				for _, r := range runs {
					if domain != nil && r.Domain != *domain {
						continue
					}
					var v *float64
					if ratioMetric {
						rat := r.Recall
						if m == "citation_correctness" {
							rat = r.CitationCorrectness
						}
						if k != nil {
							for _, at := range r.AtK {
								if at.K == *k {
									if m == "recall_at_k" {
										rat = at.Recall
									} else {
										rat = at.Precision
									}
								}
							}
						}
						v = rat.Value
						if v != nil {
							num += *rat.Numerator
							den += *rat.Denominator
						}
					} else {
						counts := map[string]*int64{"dedupe_true_merges": r.TrueMerges, "dedupe_false_merges": r.FalseMerges, "dedupe_unjudged_merges": r.UnjudgedMerges, "dedupe_missed_merges": r.MissedMerges, "http_requests": r.HTTPRequests, "search_calls": r.SearchCalls, "tool_calls": r.ToolCalls, "wall_clock_ms": r.WallClockMS, "input_tokens": r.InputTokens, "output_tokens": r.OutputTokens}
						if c := counts[m]; c != nil {
							x := float64(*c)
							v = &x
						}
						if m == "cost_usd" {
							v = r.CostUSD
						}
					}
					if v != nil {
						a.QuestionCount++
						sum += *v
					}
				}
				if a.QuestionCount > 0 {
					v := sum / float64(a.QuestionCount)
					a.MacroMean = &v
				}
				if ratioMetric {
					r := fraction(num, den)
					if a.QuestionCount == 0 {
						r = Ratio{}
					}
					a.Micro = &r
				}
				out = append(out, a)
			}
		}
	}
	return out
}
