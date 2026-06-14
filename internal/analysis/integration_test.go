package analysis

import (
	"os"
	"os/exec"
	"testing"
)

// TestOptInRMetaforIntegration runs a real R/metafor meta-analysis through the
// RscriptRunner, complementing the deterministic FakeRunner coverage. It is
// opt-in (requires RFORGE_RUN_R_METAFOR_INTEGRATION=1) and skips cleanly when
// Rscript is not installed, so the normal, network-free suite is unaffected.
func TestOptInRMetaforIntegration(t *testing.T) {
	if os.Getenv("RFORGE_RUN_R_METAFOR_INTEGRATION") != "1" {
		t.Skip("set RFORGE_RUN_R_METAFOR_INTEGRATION=1 to run real R/metafor integration")
	}
	if _, err := exec.LookPath("Rscript"); err != nil {
		t.Skip("Rscript not found in PATH; install R and metafor to run this integration")
	}

	run := AnalysisRun{
		ID:            "metafor-integration",
		SchemaVersion: "1",
		InputRows: []InputRow{
			{PaperID: "p1", EffectSize: 0.42, Variance: 0.05},
			{PaperID: "p2", EffectSize: 0.30, Variance: 0.04},
			{PaperID: "p3", EffectSize: 0.55, Variance: 0.06},
		},
	}
	result, err := RunMetafor(t.TempDir(), run, RscriptRunner{})
	if err != nil {
		t.Fatalf("real R/metafor run failed (is the metafor package installed?): %v", err)
	}
	if len(result.Versions) == 0 {
		t.Fatalf("expected detected tool versions from the real Rscript run")
	}
	if result.ScriptChecksum == "" || result.OutputChecksum == "" {
		t.Fatalf("expected script/output checksums from the real run, got %+v", result)
	}
}
