package main

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io/fs"
	"math"
	"math/big"
	"os"
	"path/filepath"
	"reflect"
	"sort"
	"testing"

	b "github.com/TrebuchetDynamics/research-forge/internal/benchmarks"
)

// Only measured floating values get absolute tolerance 1e-9. Counts, digests,
// booleans and nulls are exact, including counts inside otherwise float rows.
const pilotFloatTolerance = 1e-9

func pilotJSON(value any) ([]byte, error) { return b.CanonicalJSON(value) }

func unchangedPilotInputs(t *testing.T) {
	t.Helper()
	for _, root := range []string{publishedPilot, pilotSet} {
		before := map[string][]byte{}
		err := filepath.WalkDir(root, func(path string, entry fs.DirEntry, err error) error {
			if err != nil {
				return err
			}
			if entry.IsDir() {
				return nil
			}
			data, err := os.ReadFile(path)
			before[path] = data
			return err
		})
		if err != nil {
			t.Fatal(err)
		}
		t.Cleanup(func() {
			after := map[string][]byte{}
			err := filepath.WalkDir(root, func(path string, entry fs.DirEntry, err error) error {
				if err != nil {
					return err
				}
				if entry.IsDir() {
					return nil
				}
				data, err := os.ReadFile(path)
				after[path] = data
				return err
			})
			if err != nil || !reflect.DeepEqual(before, after) {
				t.Errorf("inputs changed under %s: %v", root, err)
			}
		})
	}
}

func validatePublishedPilot(dir, output string) error {
	s, err := b.LoadAgentSet(pilotSet)
	if err != nil {
		return err
	}
	var p Protocol
	var slots []Slot
	var report AgentReport
	var scores Scores
	var freeze Freeze
	var audit map[string]string
	var registration, execution, analysis map[string]any
	for _, input := range []struct {
		name string
		dst  any
	}{
		{"protocol.json", &p}, {"schedule.json", &slots},
		{"scores.json", &scores}, {"scores.json.freeze.json", &freeze},
		{"report.json.audit.json", &audit}, {"registration.json", &registration},
		{"execution.json", &execution}, {"analysis.json", &analysis},
	} {
		if err := b.ReadAgentArtifact(filepath.Join(dir, input.name), input.dst); err != nil {
			return fmt.Errorf("%s: %w", input.name, err)
		}
	}
	// ReadAgentArtifact's shape checker does not flatten embedded structs.
	// Parse strictly into an open map first, then reject unknown report fields.
	var reportObject map[string]any
	if err := b.ReadAgentArtifact(filepath.Join(dir, "report.json"), &reportObject); err != nil {
		return err
	}
	reportBytes, err := json.Marshal(reportObject)
	if err != nil {
		return err
	}
	decoder := json.NewDecoder(bytes.NewReader(reportBytes))
	decoder.DisallowUnknownFields()
	if err := decoder.Decode(&report); err != nil {
		return err
	}
	// Typed decoding alone would turn a missing nullable measurement into
	// the same nil as an explicitly published null. Require the full shape.
	if err := pilotEqual(reportObject, report); err != nil {
		return fmt.Errorf("report shape: %w", err)
	}
	if err := validateProtocol(p, s); err != nil {
		return err
	}
	if len(p.QuestionIDs) != 8 || p.Repetitions != 3 || len(slots) != 48 || !reflect.DeepEqual(slots, schedule(p)) {
		return fmt.Errorf("schedule: expected registered 8 x 2 x 3 slots")
	}
	header := reportHeader(s, p)
	header.Runs, header.Aggregates, header.Paired = report.Runs, report.Aggregates, report.Paired
	if err := pilotEqual(report.EngineReport, header); err != nil {
		return fmt.Errorf("report header: %w", err)
	}

	expected := map[Slot]bool{}
	for _, slot := range slots {
		expected[slot] = true
	}
	rows := map[Slot]b.Run{}
	native, budget := 0, 0
	for _, row := range report.Runs {
		slot := Slot{row.QuestionID, row.Repetition, row.Condition}
		if !expected[slot] {
			return fmt.Errorf("report matrix: unknown slot %v", slot)
		}
		if _, exists := rows[slot]; exists {
			return fmt.Errorf("report matrix: duplicate slot %v", slot)
		}
		rows[slot] = row
		if row.DedupeStatus != "unavailable" || row.CitationStatus != "not_implemented" || row.TrueMerges != nil || row.FalseMerges != nil || row.MissedMerges != nil || row.UnjudgedMerges != nil || row.CitationCorrectness != (b.Ratio{}) {
			return fmt.Errorf("unavailable facets: fabricated measurement at %v", slot)
		}
		if row.InputTokens == nil || row.OutputTokens == nil || row.HTTPRequests == nil || row.SearchCalls == nil || row.ToolCalls == nil || row.WallClockMS == nil || row.CostUSD == nil || row.DateExcluded == nil || row.Returned == nil {
			return fmt.Errorf("counters: missing public counter at %v", slot)
		}
		if !validStop(row.StopReason) {
			return fmt.Errorf("stop classification: invalid stop at %v", slot)
		}
		over := *row.InputTokens+*row.OutputTokens > p.Caps.Tokens || *row.HTTPRequests > p.Caps.HTTPRequests || *row.SearchCalls > p.Caps.SearchCalls || *row.ToolCalls > p.Caps.ToolCalls || *row.WallClockMS > p.Caps.WallClockMS
		if over && row.StopReason != "budget_exhausted" {
			return fmt.Errorf("stop classification: hidden overrun at %v", slot)
		}
		if row.Condition == "without_rforge" {
			native++
			if row.StopReason != "completed" || *row.Returned != 0 || row.Recall.Value == nil || *row.Recall.Value != 0 {
				return fmt.Errorf("native classification: expected completed empty answer")
			}
		} else if row.StopReason == "budget_exhausted" {
			if !over {
				return fmt.Errorf("stop classification: missing recorded overrun")
			}
			budget++
		}
	}
	if len(rows) != 48 || native != 24 {
		return fmt.Errorf("report matrix: missing slots (%d rows, %d native)", len(rows), native)
	}
	if budget != 23 {
		return fmt.Errorf("stop classification: expected 23 with-tools budget stops, got %d", budget)
	}

	// The report's public event rows are the only availability source here. This
	// does not rejoin opaque score labels or assert private execution/billing facts.
	raw := []RawRun{}
	seen := map[Slot]bool{}
	nativeAttempts, nativeFailures := 0, 0
	for _, available := range report.ProviderAvailability {
		row, ok := rows[available.Slot]
		if !ok || seen[available.Slot] {
			return fmt.Errorf("availability matrix: unknown/duplicate slot")
		}
		seen[available.Slot] = true
		run := RawRun{Slot: available.Slot, ProviderEvents: available.Events, Usage: Usage{HTTPRequests: *row.HTTPRequests, SearchCalls: *row.SearchCalls}}
		if err := validateProviderEvents(run, p.ProviderPolicy); err != nil {
			return fmt.Errorf("provider counters: %w", err)
		}
		// This pilot has one research HTTP attempt per search page. Local skips
		// remain events but must not be counted as attempts or pages.
		if *row.SearchCalls != *row.HTTPRequests {
			return fmt.Errorf("provider counters: search/HTTP mismatch")
		}
		if row.Condition == "without_rforge" {
			failed := false
			for _, event := range available.Events {
				if event.Action != "failed" || event.Provider != "duckduckgo" || event.HTTPStatus != 202 {
					return fmt.Errorf("native classification: expected failed HTTP-202 native attempts")
				}
				nativeAttempts++
				failed = true
			}
			if failed {
				nativeFailures++
			}
		}
		raw = append(raw, run)
	}
	if len(seen) != 48 {
		return fmt.Errorf("availability matrix: missing slots")
	}
	if nativeFailures != 24 || nativeAttempts != 48 {
		return fmt.Errorf("native classification: expected 24 failed native observations / 48 attempts")
	}
	if err := pilotEqual(report.ProviderAvailability, availability(raw)); err != nil {
		return fmt.Errorf("availability flags: %w", err)
	}
	for _, row := range report.ProviderAvailability {
		if !row.ProviderLost || !row.PairProviderLost {
			return fmt.Errorf("availability flags: every pilot observation/pair lost a provider")
		}
	}

	aggregates, paired := summaries(report.Runs, s.K, p.Repetitions)
	if err := pilotEqual(report.Aggregates, aggregates); err != nil {
		return fmt.Errorf("aggregates: %w", err)
	}
	if err := pilotEqual(report.Paired, paired); err != nil {
		return fmt.Errorf("contrasts: %w", err)
	}
	if err := validatePilotAnalysis(analysis, registration, p, rows); err != nil {
		return err
	}

	// scoreAnswers reads only the locked set and blinded answers, writing only
	// fresh temporary outputs. This is NOT an OS-isolated historical scorer audit.
	scoreFile := filepath.Join(output, "scores.json")
	if err := scoreAnswers(pilotSet, dir, scoreFile); err != nil {
		return fmt.Errorf("rescore: %w", err)
	}
	for _, name := range []string{"scores.json", "scores.json.freeze.json"} {
		want, err := os.ReadFile(filepath.Join(dir, name))
		if err != nil {
			return err
		}
		got, err := os.ReadFile(filepath.Join(output, name))
		if err != nil {
			return err
		}
		if !bytes.Equal(got, want) {
			return fmt.Errorf("frozen bytes: %s differs", name)
		}
	}
	bundle, err := loadBlind(dir)
	if err != nil {
		return err
	}
	if len(bundle.Answers) != 48 || len(scores.Scores) != 48 {
		return fmt.Errorf("blinded matrix: expected 48 answers/scores")
	}
	counts := map[string]int{}
	for _, answer := range bundle.Answers {
		counts[answer.QuestionID]++
	}
	for _, q := range p.QuestionIDs {
		if counts[q] != 6 {
			return fmt.Errorf("blinded matrix: expected six answers for %s", q)
		}
	}
	for name, want := range map[string]string{"protocol_sha256": hash(p), "answers_sha256": hash(bundle), "scores_sha256": hash(scores), "report_sha256": hash(report)} {
		if audit[name] != want {
			return fmt.Errorf("digest links: audit %s mismatch", name)
		}
	}
	// Registration hashes the schedule file including LF, unlike the four
	// canonical value digests above (which exclude the terminating LF).
	scheduleBytes, err := os.ReadFile(filepath.Join(dir, "schedule.json"))
	if err != nil {
		return err
	}
	if scores.AnswersSHA256 != hash(bundle) || freeze.AnswersSHA256 != hash(bundle) || freeze.ScoresSHA256 != hash(scores) || registration["protocol_sha256"] != hash(p) || registration["schedule_sha256"] != hashBytes(scheduleBytes) || execution["protocol_sha256"] != hash(p) {
		return fmt.Errorf("digest links: public protocol/answers/scores/schedule receipt mismatch")
	}
	return nil
}

func validatePilotAnalysis(analysis, registration map[string]any, p Protocol, rows map[Slot]b.Run) error {
	plan, ok := registration["analysis_plan"].(map[string]any)
	if !ok {
		return fmt.Errorf("analysis: missing registered plan")
	}
	delta, okDelta := plan["desired_recall_delta"].(float64)
	power, okPower := plan["power"].(float64)
	floor, okFloor := plan["minimum_expanded_questions"].(float64)
	if !okDelta || !okPower || !okFloor || delta <= 0 || power <= 0 || power >= 1 || floor < 1 || floor != math.Trunc(floor) {
		return fmt.Errorf("analysis: invalid registered sizing parameters")
	}
	differences := []float64{}
	questions := []any{}
	for _, q := range p.QuestionIDs {
		means := map[string]float64{}
		for _, condition := range conditions {
			for repetition := 1; repetition <= p.Repetitions; repetition++ {
				recall := rows[Slot{q, repetition, condition}].Recall.Value
				if recall == nil {
					return fmt.Errorf("analysis: missing recall")
				}
				means[condition] += *recall / float64(p.Repetitions)
			}
		}
		difference := means[conditions[0]] - means[conditions[1]]
		differences = append(differences, difference)
		questions = append(questions, map[string]any{"question_id": q, "with_rforge_recall": means[conditions[0]], "without_rforge_recall": means[conditions[1]], "difference": difference})
	}
	mean := 0.0
	for _, d := range differences {
		mean += d / float64(len(differences))
	}
	variance := 0.0
	for _, d := range differences {
		variance += (d - mean) * (d - mean) / float64(len(differences)-1)
	}
	sd := math.Sqrt(variance)
	// Alpha .05 is registered in the published analysis-plan prose. Inverse CDF
	// is computed from erf, independently of the rounded values in the report.
	const alpha = .05
	quantile := func(probability float64) float64 { return math.Sqrt2 * math.Erfinv(2*probability-1) }
	raw := math.Pow((quantile(1-alpha/2)+quantile(power))*sd/delta, 2)
	rounded := int(math.Ceil(raw))
	planned := max(int(floor), rounded)
	expected := map[string]any{
		"question_count": len(differences), "repetitions_per_condition": p.Repetitions,
		"question_recall_differences": questions, "recall_mean_difference": mean,
		"recall_sample_sd": sd, "recall_standardized_paired_effect_dz": mean / sd,
		"two_sided_alpha": alpha, "power": power, "prospective_absolute_recall_difference": delta,
		"minimum_independent_review_families":           int(floor),
		"normal_approximation_question_count_unrounded": raw,
		"normal_approximation_question_count_ceil":      rounded,
		"planned_independent_review_families":           planned,
		"planned_observations_at_N3":                    planned * len(conditions) * p.Repetitions,
	}
	for key, want := range expected {
		got, exists := analysis[key]
		if !exists {
			return fmt.Errorf("analysis: missing %s", key)
		}
		if err := pilotEqual(map[string]any{key: got}, map[string]any{key: want}); err != nil {
			return fmt.Errorf("analysis: %w", err)
		}
	}
	return nil
}

// Canonical serialization normalizes typed structs and the []any contrast
// decoder to a single tree. UseNumber avoids rounding integer counters.
func pilotEqual(got, want any) error {
	normalize := func(value any) (any, error) {
		data, err := pilotJSON(value)
		if err != nil {
			return nil, err
		}
		decoder := json.NewDecoder(bytes.NewReader(data))
		decoder.UseNumber()
		var tree any
		err = decoder.Decode(&tree)
		return tree, err
	}
	a, err := normalize(got)
	if err != nil {
		return err
	}
	c, err := normalize(want)
	if err != nil {
		return err
	}
	return comparePilotTree("", "", a, c)
}

func comparePilotTree(path, field string, got, want any) error {
	switch w := want.(type) {
	case map[string]any:
		g, ok := got.(map[string]any)
		if !ok || len(g) != len(w) {
			return fmt.Errorf("%s: object shape differs", path)
		}
		keys := []string{}
		for key := range w {
			keys = append(keys, key)
		}
		sort.Strings(keys)
		for _, key := range keys {
			value, exists := g[key]
			if !exists {
				return fmt.Errorf("%s: missing %s", path, key)
			}
			if err := comparePilotTree(path+"."+key, key, value, w[key]); err != nil {
				return err
			}
		}
	case []any:
		g, ok := got.([]any)
		if !ok || len(g) != len(w) {
			return fmt.Errorf("%s: array length differs", path)
		}
		for i := range w {
			if err := comparePilotTree(fmt.Sprintf("%s[%d]", path, i), field, g[i], w[i]); err != nil {
				return err
			}
		}
	case json.Number:
		g, ok := got.(json.Number)
		if !ok {
			return fmt.Errorf("%s: number/null differs", path)
		}
		floating := map[string]bool{"value": true, "macro_mean": true, "mean_difference": true, "ci95_low": true, "ci95_high": true, "difference": true, "with_rforge_recall": true, "without_rforge_recall": true, "recall_mean_difference": true, "recall_sample_sd": true, "recall_standardized_paired_effect_dz": true, "two_sided_alpha": true, "power": true, "prospective_absolute_recall_difference": true, "normal_approximation_question_count_unrounded": true}
		if floating[field] {
			gf, err := g.Float64()
			if err != nil {
				return err
			}
			wf, err := w.Float64()
			if err != nil {
				return err
			}
			if math.Abs(gf-wf) <= pilotFloatTolerance {
				return nil
			}
		} else {
			gr, gok := new(big.Rat).SetString(string(g))
			wr, wok := new(big.Rat).SetString(string(w))
			if gok && wok && gr.Cmp(wr) == 0 {
				return nil
			}
		}
		return fmt.Errorf("%s: got %s, want %s", path, g, w)
	default:
		if !reflect.DeepEqual(got, want) {
			return fmt.Errorf("%s: got %v, want %v", path, got, want)
		}
	}
	return nil
}
