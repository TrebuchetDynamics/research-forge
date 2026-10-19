package main

import (
	"math"
	"sort"

	b "github.com/TrebuchetDynamics/research-forge/internal/benchmarks"
)

type Contrast struct {
	Metric         string   `json:"metric"`
	K              *int     `json:"k"`
	QuestionCount  int      `json:"question_count"`
	Repetitions    int      `json:"repetitions"`
	MeanDifference *float64 `json:"mean_difference"`
	Low            *float64 `json:"ci95_low"`
	High           *float64 `json:"ci95_high"`
	Method         string   `json:"method"`
}

func betaFraction(a, c, x float64) float64 {
	// Continued fraction for the regularized incomplete beta (modified Lentz).
	const tiny = 1e-300
	qab, qap, qam := a+c, a+1, a-1
	d := 1 - qab*x/qap
	if math.Abs(d) < tiny {
		d = tiny
	}
	d = 1 / d
	h, v := d, 1.0
	for m := 1; m <= 300; m++ {
		mf := float64(m)
		m2 := 2 * mf
		aa := mf * (c - mf) * x / ((qam + m2) * (a + m2))
		d = 1 + aa*d
		if math.Abs(d) < tiny {
			d = tiny
		}
		v = 1 + aa/v
		if math.Abs(v) < tiny {
			v = tiny
		}
		d = 1 / d
		h *= d * v
		aa = -(a + mf) * (qab + mf) * x / ((a + m2) * (qap + m2))
		d = 1 + aa*d
		if math.Abs(d) < tiny {
			d = tiny
		}
		v = 1 + aa/v
		if math.Abs(v) < tiny {
			v = tiny
		}
		d = 1 / d
		delta := d * v
		h *= delta
		if math.Abs(delta-1) < 3e-14 {
			break
		}
	}
	return h
}
func incompleteBeta(a, c, x float64) float64 {
	if x <= 0 {
		return 0
	}
	if x >= 1 {
		return 1
	}
	la, _ := math.Lgamma(a)
	lc, _ := math.Lgamma(c)
	lac, _ := math.Lgamma(a + c)
	factor := math.Exp(lac - la - lc + a*math.Log(x) + c*math.Log1p(-x))
	if x < (a+1)/(a+c+2) {
		return factor * betaFraction(a, c, x) / a
	}
	return 1 - factor*betaFraction(c, a, 1-x)/c
}
func studentCritical(df float64) float64 {
	lo, hi := 0.0, 1.0
	tail := func(t float64) float64 { return .5 * incompleteBeta(df/2, .5, df/(df+t*t)) }
	for tail(hi) > .025 {
		hi *= 2
	}
	for i := 0; i < 100; i++ {
		mid := (lo + hi) / 2
		if tail(mid) > .025 {
			lo = mid
		} else {
			hi = mid
		}
	}
	return (lo + hi) / 2
}
func questionInterval(d []float64) (mean, low, high *float64) {
	if len(d) == 0 {
		return
	}
	m := 0.0
	for _, v := range d {
		m += v
	}
	m /= float64(len(d))
	mean = &m
	if len(d) < 2 {
		return
	}
	variance := 0.0
	for _, v := range d {
		variance += (v - m) * (v - m)
	}
	variance /= float64(len(d) - 1)
	margin := studentCritical(float64(len(d)-1)) * math.Sqrt(variance/float64(len(d)))
	l, h := m-margin, m+margin
	return mean, &l, &h
}
func metricValue(r b.Run, m string, k *int) (*float64, *b.Ratio) {
	if m == "recall" {
		return r.Recall.Value, &r.Recall
	}
	if m == "citation_correctness" {
		return r.CitationCorrectness.Value, &r.CitationCorrectness
	}
	if m == "recall_at_k" || m == "precision_at_k" {
		for _, at := range r.AtK {
			if k != nil && at.K == *k {
				rat := at.Recall
				if m == "precision_at_k" {
					rat = at.Precision
				}
				return rat.Value, &rat
			}
		}
		return nil, nil
	}
	counts := map[string]*int64{"dedupe_true_merges": r.TrueMerges, "dedupe_false_merges": r.FalseMerges, "dedupe_unjudged_merges": r.UnjudgedMerges, "dedupe_missed_merges": r.MissedMerges, "http_requests": r.HTTPRequests, "search_calls": r.SearchCalls, "tool_calls": r.ToolCalls, "wall_clock_ms": r.WallClockMS, "input_tokens": r.InputTokens, "output_tokens": r.OutputTokens}
	if c := counts[m]; c != nil {
		v := float64(*c)
		return &v, nil
	}
	if m == "cost_usd" {
		return r.CostUSD, nil
	}
	return nil, nil
}
func metricKs(m string, ks []int) []*int {
	if m != "recall_at_k" && m != "precision_at_k" {
		return []*int{nil}
	}
	out := []*int{}
	for _, k := range ks {
		v := k
		out = append(out, &v)
	}
	return out
}
func summaries(runs []b.Run, ks []int, n int) ([]b.Aggregate, []any) {
	byQuestion := map[string]map[string][]b.Run{}
	domainNames := map[string]bool{}
	for _, r := range runs {
		if byQuestion[r.QuestionID] == nil {
			byQuestion[r.QuestionID] = map[string][]b.Run{}
		}
		byQuestion[r.QuestionID][r.Condition] = append(byQuestion[r.QuestionID][r.Condition], r)
		domainNames[r.Domain] = true
	}
	names := []string{}
	for d := range domainNames {
		names = append(names, d)
	}
	sort.Strings(names)
	domains := []*string{nil}
	for _, d := range names {
		v := d
		domains = append(domains, &v)
	}
	questions := []string{}
	for q := range byQuestion {
		questions = append(questions, q)
	}
	sort.Strings(questions)
	// Complete-N availability is evaluated before either macro or micro sums.
	measure := func(rows []b.Run, m string, k *int) (*float64, int64, int64) {
		if len(rows) != n {
			return nil, 0, 0
		}
		sum := 0.0
		num, den := int64(0), int64(0)
		for _, r := range rows {
			v, ratio := metricValue(r, m, k)
			if v == nil {
				return nil, 0, 0
			}
			sum += *v
			if ratio != nil {
				num += *ratio.Numerator
				den += *ratio.Denominator
			}
		}
		mean := sum / float64(n)
		return &mean, num, den
	}
	aggregates := []b.Aggregate{}
	paired := []any{}
	for _, condition := range conditions {
		for _, domain := range domains {
			for _, m := range metrics {
				for _, k := range metricKs(m, ks) {
					a := b.Aggregate{Condition: condition, Domain: domain, Metric: m, K: k}
					sum := 0.0
					num, den := int64(0), int64(0)
					ratioMetric := m == "recall" || m == "recall_at_k" || m == "precision_at_k" || m == "citation_correctness"
					for _, q := range questions {
						rows := byQuestion[q][condition]
						if len(rows) == 0 || (domain != nil && rows[0].Domain != *domain) {
							continue
						}
						v, qn, qd := measure(rows, m, k)
						if v != nil {
							a.QuestionCount++
							sum += *v
							num += qn
							den += qd
						}
					}
					if a.QuestionCount > 0 {
						v := sum / float64(a.QuestionCount)
						a.MacroMean = &v
					}
					if ratioMetric {
						a.Micro = &b.Ratio{}
						if a.QuestionCount > 0 {
							v := float64(num) / float64(den)
							a.Micro = &b.Ratio{Numerator: &num, Denominator: &den, Value: &v}
						}
					}
					aggregates = append(aggregates, a)
				}
			}
		}
	}
	for _, m := range metrics {
		for _, k := range metricKs(m, ks) {
			differences := []float64{}
			for _, q := range questions {
				with, _, _ := measure(byQuestion[q][conditions[0]], m, k)
				without, _, _ := measure(byQuestion[q][conditions[1]], m, k)
				if with != nil && without != nil {
					differences = append(differences, *with-*without)
				}
			}
			mean, low, high := questionInterval(differences)
			paired = append(paired, Contrast{m, k, len(differences), n, mean, low, high, "paired-question-student-t"})
		}
	}
	return aggregates, paired
}
