package cli

import (
	"bytes"
	"encoding/json"
	"os"
	"path/filepath"
	"testing"

	"github.com/TrebuchetDynamics/research-forge/internal/analysis"
)

func TestAnalysisPrepareRawContinuousWithModerators(t *testing.T) {
	project := filepath.Join(t.TempDir(), "demo")
	if code := Execute([]string{"project", "create", project, "--title", "STH Benchmarking"}, ioDiscard{}, ioDiscard{}); code != 0 {
		t.Fatalf("project create code = %d", code)
	}
	items := `[
		{"PaperID":"paper-a","Values":{"value_pct":"12.5","device_type":"pec","auxiliary_bias":"unassisted"},"Support":{"Kind":"passage","Ref":"p1"},"Status":"accepted"},
		{"PaperID":"paper-b","Values":{"value_pct":"8.0","device_type":"pv-electrolysis","auxiliary_bias":"unassisted"},"Support":{"Kind":"passage","Ref":"p2"},"Status":"accepted"}
	]`
	if err := os.WriteFile(filepath.Join(project, "data", "evidence.items.json"), []byte(items), 0o644); err != nil {
		t.Fatalf("write evidence: %v", err)
	}
	var stdout, stderr bytes.Buffer
	code := Execute([]string{
		"--json", "--project", project,
		"analysis", "prepare", "sth-run",
		"--effect", "raw-continuous",
		"--moderator", "device_type",
		"--moderator", "auxiliary_bias",
	}, &stdout, &stderr)
	if code != 0 {
		t.Fatalf("analysis prepare code=%d stderr=%s", code, stderr.String())
	}
	var env struct {
		Data struct {
			Run analysis.AnalysisRun `json:"run"`
		} `json:"data"`
	}
	if err := json.Unmarshal(stdout.Bytes(), &env); err != nil {
		t.Fatalf("parse output: %v\n%s", err, stdout.String())
	}
	if len(env.Data.Run.InputRows) != 2 {
		t.Fatalf("rows = %d, want 2", len(env.Data.Run.InputRows))
	}
	row := env.Data.Run.InputRows[0]
	if row.Moderators == nil {
		t.Fatalf("row[0] Moderators is nil — --moderator flag not wired to PrepareRawContinuous")
	}
	if row.Moderators["device_type"] != "pec" {
		t.Fatalf("row[0] device_type = %q, want pec", row.Moderators["device_type"])
	}
	if env.Data.Run.InputRows[1].Moderators["device_type"] != "pv-electrolysis" {
		t.Fatalf("row[1] device_type = %q, want pv-electrolysis", env.Data.Run.InputRows[1].Moderators["device_type"])
	}
}

func TestAnalysisPrepareModeratorFlagRejectsArmPairEffect(t *testing.T) {
	project := filepath.Join(t.TempDir(), "demo")
	if code := Execute([]string{"project", "create", project, "--title", "Demo"}, ioDiscard{}, ioDiscard{}); code != 0 {
		t.Fatalf("project create code = %d", code)
	}
	var stdout, stderr bytes.Buffer
	code := Execute([]string{
		"--json", "--project", project,
		"analysis", "prepare", "run-1",
		"--effect", "smd",
		"--moderator", "region",
	}, &stdout, &stderr)
	if code == 0 {
		t.Fatal("expected non-zero exit code when --moderator used with arm-pair effect, got 0")
	}
}

func TestAnalysisPrepareRawContinuousNoModeratorFlagStillWorks(t *testing.T) {
	project := filepath.Join(t.TempDir(), "demo")
	if code := Execute([]string{"project", "create", project, "--title", "Demo"}, ioDiscard{}, ioDiscard{}); code != 0 {
		t.Fatalf("project create code = %d", code)
	}
	items := `[{"PaperID":"p1","Values":{"value_pct":"5.0"},"Support":{"Kind":"passage","Ref":"p1"},"Status":"accepted"}]`
	if err := os.WriteFile(filepath.Join(project, "data", "evidence.items.json"), []byte(items), 0o644); err != nil {
		t.Fatalf("write evidence: %v", err)
	}
	var stdout, stderr bytes.Buffer
	code := Execute([]string{
		"--json", "--project", project,
		"analysis", "prepare", "run-1",
		"--effect", "raw-continuous",
	}, &stdout, &stderr)
	if code != 0 {
		t.Fatalf("prepare without --moderator should still succeed: code=%d stderr=%s", code, stderr.String())
	}
}
