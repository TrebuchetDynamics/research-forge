package analysis

import (
	"path/filepath"
	"testing"
)

// RscriptRunner must satisfy the Runner interface used by RunMetafor.
var _ Runner = RscriptRunner{}

func TestRscriptRunnerErrorsWhenBinaryMissing(t *testing.T) {
	runner := RscriptRunner{Path: filepath.Join(t.TempDir(), "no-such-rscript")}
	if _, err := runner.Run("cat('hello')\n"); err == nil {
		t.Fatalf("expected error executing a missing Rscript binary")
	}
}

func TestRscriptRunnerToolVersionsEmptyWhenBinaryMissing(t *testing.T) {
	runner := RscriptRunner{Path: filepath.Join(t.TempDir(), "no-such-rscript")}
	if versions := runner.ToolVersions(); len(versions) != 0 {
		t.Fatalf("expected no detected versions for missing binary, got %v", versions)
	}
}

func TestRunMetaforPropagatesRunnerError(t *testing.T) {
	run := AnalysisRun{ID: "run1", SchemaVersion: "1", InputRows: []InputRow{{PaperID: "p1", EffectSize: 0.5, Variance: 0.1}}}
	runner := RscriptRunner{Path: filepath.Join(t.TempDir(), "no-such-rscript")}
	if _, err := RunMetafor(t.TempDir(), run, runner); err == nil {
		t.Fatalf("expected RunMetafor to propagate the runner error")
	}
}
