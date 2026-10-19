package main

import (
	"crypto/rand"
	"encoding/hex"
	"fmt"
	"math/big"
	"path/filepath"
	"regexp"
	"strings"

	b "github.com/TrebuchetDynamics/research-forge/internal/benchmarks"
)

type BlindAnswer struct {
	Label      string `json:"label"`
	QuestionID string `json:"question_id"`
	Answer     Answer `json:"answer"`
}
type BlindBundle struct {
	Version     string        `json:"version"`
	SetChecksum string        `json:"set_checksum"`
	Answers     []BlindAnswer `json:"answers"`
}
type Mapping struct {
	AnswersSHA256    string         `json:"answers_sha256"`
	ExperimentSHA256 string         `json:"experiment_sha256"`
	Labels           map[string]int `json:"labels"`
}
type BlindScore struct {
	Label string `json:"label"`
	Run   b.Run  `json:"metrics"`
}
type Scores struct {
	Version       string       `json:"version"`
	SetChecksum   string       `json:"set_checksum"`
	AnswersSHA256 string       `json:"answers_sha256"`
	Scores        []BlindScore `json:"scores"`
}
type Freeze struct {
	ScoresSHA256  string `json:"scores_sha256"`
	AnswersSHA256 string `json:"answers_sha256"`
}

// Treat underscores as separators too: Go regexp's \\b treats them as word
// characters, which would let condition_with_rforge and rforge_search through.
var markerRE = regexp.MustCompile(`(?i)(^|[^a-z0-9])(rforge|researchforge|pi|codex|with_rforge|without_rforge|provenance|run_id|run-id|native_search|web_search|web_read|mcp)([^a-z0-9]|$)`)
var labelRE = regexp.MustCompile(`^[0-9a-f]{32}$`)

func validateAnswer(a Answer, runID string) error {
	// Raw transcripts/artifacts are discarded, not regex-edited into an answer.
	// Ambiguous attribution in substantive material requires blind extraction;
	// never silently alter a claim, quote, source identifier or rank to hide it.
	data, err := b.CanonicalJSON(a)
	if err != nil {
		return err
	}
	if markerRE.Match(data) || (runID != "" && strings.Contains(string(data), runID)) {
		return fmt.Errorf("protocol_violation: attribution in substantive answer; extract before blinding")
	}
	return nil
}
func blind(private, out string) error {
	p, e, err := loadPrivate(private)
	if err != nil {
		return err
	}
	if len(e.Runs) == 0 {
		return fmt.Errorf("protocol_violation: empty experiment")
	}
	bundle := BlindBundle{scorerVersion, p.SetChecksum, []BlindAnswer{}}
	mapping := Mapping{ExperimentSHA256: hash(e), Labels: map[string]int{}}
	for i, r := range e.Runs {
		if err = validateAnswer(r.Answer, r.RunID); err != nil {
			return err
		}
		id := make([]byte, 16)
		if _, err = rand.Read(id); err != nil {
			return err
		}
		label := hex.EncodeToString(id)
		if _, exists := mapping.Labels[label]; exists {
			return fmt.Errorf("opaque label collision")
		}
		mapping.Labels[label] = i
		bundle.Answers = append(bundle.Answers, BlindAnswer{label, r.Slot.QuestionID, r.Answer})
	}
	for i := len(bundle.Answers) - 1; i > 0; i-- {
		j, err := rand.Int(rand.Reader, big.NewInt(int64(i+1)))
		if err != nil {
			return err
		}
		n := int(j.Int64())
		bundle.Answers[i], bundle.Answers[n] = bundle.Answers[n], bundle.Answers[i]
	}
	mapping.AnswersSHA256 = hash(bundle)
	if filepath.Clean(private) == filepath.Clean(out) {
		return fmt.Errorf("blind output must be a separate directory")
	}
	if err = makeNewDir(out); err != nil {
		return err
	}
	if err = save(filepath.Join(private, "mapping.json"), mapping); err != nil {
		return err
	}
	return save(filepath.Join(out, "answers.json"), bundle)
}
func loadBlind(dir string) (BlindBundle, error) {
	var bundle BlindBundle
	err := b.ReadAgentArtifact(filepath.Join(dir, "answers.json"), &bundle)
	if err != nil {
		return bundle, err
	}
	if bundle.Version != scorerVersion || len(bundle.Answers) == 0 {
		return bundle, fmt.Errorf("protocol_violation: invalid blind bundle")
	}
	seen := map[string]bool{}
	for _, a := range bundle.Answers {
		if !labelRE.MatchString(a.Label) || seen[a.Label] {
			return bundle, fmt.Errorf("protocol_violation: invalid/duplicate opaque label")
		}
		seen[a.Label] = true
		if err = validateAnswer(a.Answer, ""); err != nil {
			return bundle, err
		}
	}
	return bundle, nil
}
func scoreAnswers(setDir, blindDir, out string) error {
	s, err := b.LoadAgentSet(setDir)
	if err != nil {
		return err
	}
	bundle, err := loadBlind(blindDir)
	if err != nil {
		return err
	}
	if bundle.SetChecksum != s.SetChecksum {
		return fmt.Errorf("checksum_mismatch")
	}
	result := Scores{scorerVersion, s.SetChecksum, hash(bundle), []BlindScore{}}
	questions := map[string]b.Question{}
	for _, q := range s.Questions {
		if q.Split == "held_out" {
			questions[q.QuestionID] = q
		}
	}
	for _, a := range bundle.Answers {
		q, ok := questions[a.QuestionID]
		if !ok {
			return fmt.Errorf("protocol_violation: unknown question")
		}
		bibliography := [][]b.Identifier{}
		for _, paper := range a.Answer.Bibliography {
			bibliography = append(bibliography, paper.Identifiers)
		}
		text := []string{}
		for _, claim := range a.Answer.Claims {
			text = append(text, claim.Text, claim.Quote, claim.Locator)
		}
		if err = b.CheckAgentLeak(s, bibliography, text); err != nil {
			return err
		}
		row, err := b.ScoreAgentBibliography(q, s.K, bibliography)
		if err != nil {
			return err
		}
		result.Scores = append(result.Scores, BlindScore{a.Label, row})
	}
	if err = save(out, result); err != nil {
		return err
	}
	// The scorer freezes scores before the custodian can join labels/counters.
	return save(out+".freeze.json", Freeze{hash(result), hash(bundle)})
}
