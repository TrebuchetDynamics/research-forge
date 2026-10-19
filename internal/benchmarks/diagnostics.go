package benchmarks

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
)

// CaptureFailure is private operational evidence, never an agent/scorer input.
// HTTPStatus is null when no response was received (or no call was attempted).
type CaptureFailure struct {
	QuestionID string `json:"question_id"`
	PlanCall   int    `json:"plan_call"`
	Provider   string `json:"provider"`
	Sequence   int    `json:"sequence"`
	HTTPStatus *int   `json:"http_status"`
	Check      string `json:"check"`
	Why        string `json:"why"`
}

func (s *session) trace(check, why string, status *int, sequence int) {
	if !s.record {
		return
	}
	provider := ""
	if s.call > 0 && s.call <= len(s.q.SearchPlan) {
		provider = s.q.SearchPlan[s.call-1].Source
	}
	if status == nil && sequence > 0 && sequence <= len(s.snap.Exchanges) {
		if response := s.snap.Exchanges[sequence-1].Response; response != nil {
			status = intPtr(response.Status)
		}
	}
	s.failures = append(s.failures, CaptureFailure{QuestionID: s.q.QuestionID, PlanCall: s.call, Provider: provider, Sequence: sequence, HTTPStatus: status, Check: check, Why: sanitizeText(why)})
}

func writeCaptureDiagnostics(setDir string, idx SnapshotIndex, bodies map[string][]byte, failures []CaptureFailure, report, audit []byte) (string, error) {
	// TMPDIR can be supplied by an embedding caller. Never put diagnostics in
	// the evaluated set or in benchmarks/sets, even if TMPDIR points there.
	base, err := filepath.Abs(os.TempDir())
	if err != nil {
		return "", err
	}
	setPath, err := filepath.Abs(setDir)
	if err != nil {
		return "", err
	}
	if resolved, e := filepath.EvalSymlinks(base); e == nil {
		base = resolved
	}
	if resolved, e := filepath.EvalSymlinks(setPath); e == nil {
		setPath = resolved
	}
	rel, _ := filepath.Rel(setPath, base)
	if rel == "." || rel != ".." && !strings.HasPrefix(rel, ".."+string(filepath.Separator)) || strings.Contains(filepath.ToSlash(base)+"/", "/benchmarks/sets/") {
		return "", fmt.Errorf("unsafe diagnostics temporary directory outside set required: %s", base)
	}
	dir, err := os.MkdirTemp(base, "rforge-benchmark-diagnostics-*")
	if err != nil {
		return "", err
	}
	// Keep even a partially written diagnostics directory on disk failure.
	save := func(name string, b []byte) error { return os.WriteFile(filepath.Join(dir, name), b, 0600) }
	if err = os.MkdirAll(filepath.Join(dir, "snapshots/bodies"), 0700); err != nil {
		return dir, err
	}
	for hash, b := range bodies {
		if err = save("snapshots/bodies/"+hash+".bin", b); err != nil {
			return dir, err
		}
	}
	for name, v := range map[string]any{"snapshots/index.json": idx, "failures.json": failures} {
		b, e := CanonicalJSON(v)
		if e != nil {
			return dir, e
		}
		if err = save(name, b); err != nil {
			return dir, err
		}
	}
	for _, name := range []string{"set.json", "review.md"} {
		if b, e := readSetFile(setDir, name); e == nil {
			if err = save(name, b); err != nil {
				return dir, err
			}
		}
	}
	if len(audit) > 0 {
		if err = save("audit.json", audit); err != nil {
			return dir, err
		}
	}
	if err = save("report.json", report); err != nil {
		return dir, err
	}
	return dir, nil
}
