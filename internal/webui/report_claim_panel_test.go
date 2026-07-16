package webui

import (
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/TrebuchetDynamics/research-forge/internal/evidence"
	reportpkg "github.com/TrebuchetDynamics/research-forge/internal/report"
)

func TestReportClaimPanelHandlerBlocksWeakUnresolvedClaims(t *testing.T) {
	project := t.TempDir()
	panel := reportpkg.BuildClaimTraceabilityPanel(reportpkg.CitationEvidenceTraceView{SchemaVersion: "1", Claims: []reportpkg.ClaimTraceView{{ClaimID: "weak", PaperID: "p1", ClaimText: "generated figure", ClaimStatus: evidence.StatusAccepted}}})
	writeJSON(t, filepath.Join(project, "data", "claim-panel.json"), panel)
	rec := httptest.NewRecorder()
	newReportClaimPanelHandler(func() string { return project }).ServeHTTP(rec, httptest.NewRequest("GET", "/report", nil))
	body := rec.Body.String()
	for _, want := range []string{"Claim traceability panel", "paragraph/table/figure", "accepted evidence", "unresolved", "weak", "block final export", "weak", "rforge report final-export"} {
		if !strings.Contains(body, want) {
			t.Fatalf("missing %q:\n%s", want, body)
		}
	}
}

func TestReportClaimPanelHandlerDoesNotReadSymlinkedPanel(t *testing.T) {
	projectPath := t.TempDir()
	dataDir := filepath.Join(projectPath, "data")
	if err := os.MkdirAll(dataDir, 0o755); err != nil {
		t.Fatalf("mkdir project data: %v", err)
	}
	externalPanel := reportpkg.BuildClaimTraceabilityPanel(reportpkg.CitationEvidenceTraceView{SchemaVersion: "1", Claims: []reportpkg.ClaimTraceView{{
		ClaimID: "external-private-claim", PaperID: "external-paper", ClaimText: "external private finding", ClaimStatus: evidence.StatusAccepted,
	}}})
	externalPath := filepath.Join(t.TempDir(), "claim-panel.json")
	writeJSON(t, externalPath, externalPanel)
	panelPath := filepath.Join(dataDir, "claim-panel.json")
	if err := os.Symlink(externalPath, panelPath); err != nil {
		t.Skipf("symlinks unavailable: %v", err)
	}

	rec := httptest.NewRecorder()
	newReportClaimPanelHandler(func() string { return projectPath }).ServeHTTP(rec, httptest.NewRequest("GET", "/report", nil))
	if body := rec.Body.String(); strings.Contains(body, "external-private-claim") || strings.Contains(body, "external private finding") {
		t.Fatalf("report disclosed symlinked claim panel: %s", body)
	}
	if info, err := os.Lstat(panelPath); err != nil || info.Mode()&os.ModeSymlink == 0 {
		t.Fatalf("claim panel symlink changed: info=%v err=%v", info, err)
	}
}
