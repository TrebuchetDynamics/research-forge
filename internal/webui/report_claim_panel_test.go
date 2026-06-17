package webui

import (
	"net/http/httptest"
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
