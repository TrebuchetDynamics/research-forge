package main

import (
	"fmt"
	"path/filepath"
	"sort"
	"strings"

	b "github.com/TrebuchetDynamics/research-forge/internal/benchmarks"
)

func report(setDir, private, blindDir, scoreFile, out string) (resultErr error) {
	s, err := b.LoadAgentSet(setDir)
	if err != nil {
		return err
	}
	// A validated registration is needed to identify the protocol and expected
	// matrix without fabricating either when the input itself is unreadable.
	var p Protocol
	if err = b.ReadAgentArtifact(filepath.Join(private, "protocol.json"), &p); err != nil {
		return err
	}
	if err = validateProtocol(p, s); err != nil {
		return err
	}
	defer func() {
		if resultErr != nil {
			invalid := AgentReport{EngineReport: reportHeader(s, p), ProviderAvailability: []ProviderAvailability{}}
			invalid.Status = "invalid"
			invalid.Errors = []b.RunError{{Code: errorCode(resultErr)}}
			for _, q := range s.Questions {
				if q.Split != "held_out" {
					continue
				}
				for _, condition := range conditions {
					for repetition := 1; repetition <= p.Repetitions; repetition++ {
						row := b.Run{QuestionID: q.QuestionID, Domain: q.Domain, Condition: condition, Repetition: repetition, StopReason: "error_threshold", GoldCount: len(q.Gold), AtK: []b.AtK{}, DedupeStatus: "invalid", CitationStatus: "invalid"}
						for _, k := range s.K {
							row.AtK = append(row.AtK, b.AtK{K: k})
						}
						invalid.Runs = append(invalid.Runs, row)
					}
				}
			}
			if err := save(out, invalid); err != nil {
				resultErr = fmt.Errorf("%w; invalid report could not be written: %v", resultErr, err)
			}
		}
	}()
	_, e, err := loadPrivate(private)
	if err != nil {
		return err
	}
	bundle, err := loadBlind(blindDir)
	if err != nil {
		return err
	}
	var mapping Mapping
	var scores Scores
	var freeze Freeze
	for _, x := range []struct {
		path string
		dst  any
	}{{filepath.Join(private, "mapping.json"), &mapping}, {scoreFile, &scores}, {scoreFile + ".freeze.json", &freeze}} {
		if err = b.ReadAgentArtifact(x.path, x.dst); err != nil {
			return err
		}
	}
	ah := hash(bundle)
	if bundle.SetChecksum != s.SetChecksum || scores.Version != scorerVersion || scores.SetChecksum != s.SetChecksum || scores.AnswersSHA256 != ah || mapping.AnswersSHA256 != ah || mapping.ExperimentSHA256 != hash(e) || freeze.AnswersSHA256 != ah || freeze.ScoresSHA256 != hash(scores) {
		return fmt.Errorf("checksum_mismatch: frozen artifacts changed")
	}
	if len(mapping.Labels) != len(e.Runs) || len(scores.Scores) != len(e.Runs) || len(bundle.Answers) != len(e.Runs) {
		return fmt.Errorf("protocol_violation: incomplete matrix")
	}
	answers := map[string]BlindAnswer{}
	for _, a := range bundle.Answers {
		answers[a.Label] = a
	}
	seen := map[int]bool{}
	runs := []b.Run{}
	for _, score := range scores.Scores {
		index, ok := mapping.Labels[score.Label]
		if !ok || index < 0 || index >= len(e.Runs) || seen[index] {
			return fmt.Errorf("protocol_violation: missing/duplicate score")
		}
		seen[index] = true
		raw := e.Runs[index]
		a, ok := answers[score.Label]
		if !ok || a.QuestionID != raw.Slot.QuestionID || hash(a.Answer) != hash(raw.Answer) || score.Run.QuestionID != raw.Slot.QuestionID || score.Run.Condition != "" || score.Run.Repetition != 0 {
			return fmt.Errorf("protocol_violation: answer attribution mismatch")
		}
		row := score.Run
		row.Condition = raw.Slot.Condition
		row.Repetition = raw.Slot.Repetition
		row.StopReason = raw.StopReason
		u := raw.Usage
		row.InputTokens = &u.InputTokens
		row.OutputTokens = &u.OutputTokens
		row.CostUSD = &u.CostUSD
		row.SearchCalls = &u.SearchCalls
		row.ToolCalls = &u.ToolCalls
		row.HTTPRequests = &u.HTTPRequests
		row.WallClockMS = &u.WallClockMS
		excluded := raw.Gate.DateExcluded
		row.DateExcluded = &excluded
		runs = append(runs, row)
	}
	sort.Slice(runs, func(i, j int) bool {
		a, c := runs[i], runs[j]
		if a.QuestionID != c.QuestionID {
			return a.QuestionID < c.QuestionID
		}
		if a.Condition != c.Condition {
			return a.Condition < c.Condition
		}
		return a.Repetition < c.Repetition
	})
	aggregates, paired := summaries(runs, s.K, p.Repetitions)
	result := AgentReport{EngineReport: reportHeader(s, p), ProviderAvailability: availability(e.Runs)}
	result.Runs, result.Aggregates, result.Paired = runs, aggregates, paired
	// Publish the complete report last: missing audit must not leave an
	// apparently successful standalone report behind.
	if err = save(out+".audit.json", map[string]string{"protocol_sha256": hash(p), "experiment_sha256": hash(e), "answers_sha256": ah, "scores_sha256": hash(scores), "mapping_sha256": hash(mapping), "report_sha256": hash(result)}); err != nil {
		return err
	}
	return save(out, result)
}

func reportHeader(s b.Set, p Protocol) b.EngineReport {
	purpose, mode := "evaluation", "live"
	if strings.HasPrefix(s.SetID, "fixture-") {
		purpose = "fixture"
	}
	if p.Model.Provider == "stub" {
		mode = "replay"
	}
	return b.EngineReport{SchemaVersion: 1, Purpose: purpose, Tier: "agent", Mode: mode, Status: "complete", SetID: s.SetID, SetChecksum: s.SetChecksum, SnapshotsSHA256: s.SnapshotsSHA256, Split: "held_out", RforgeVersion: p.RforgeVersion, Revision: p.Revision, ProtocolSHA256: hash(p), K: s.K, Runs: []b.Run{}, Paired: []any{}, Errors: []b.RunError{}}
}
