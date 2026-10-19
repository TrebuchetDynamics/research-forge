package reviewpkg

import (
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/TrebuchetDynamics/research-forge/internal/provenance"
)

// Real projects record absolute local paths in provenance events (for example
// `project create <path>`). The packaged ledger must not carry them, wherever
// the project lives, so the package passes its own content-privacy audit.
func TestCreateRedactsLocalPathsInPackagedProvenance(t *testing.T) {
	project := t.TempDir()
	write(t, filepath.Join(project, "rforge.project.toml"), "title='Review'\n")
	write(t, filepath.Join(project, "rforge.lock.json"), `{"version":"1"}`)
	homePath := "/home/alice/research/notes.md"
	if err := provenance.Append(project, provenance.Event{
		SchemaVersion: "1", ID: "evt_test_project_create", Timestamp: "2026-10-19T00:00:00Z",
		Actor: "rforge", Action: "project.create", Target: project, Warnings: []string{},
		Inputs:  map[string]any{"path": filepath.Join(project, "data", "library.json"), "note": homePath},
		Outputs: map[string]any{"report": `C:\Users\alice\review.md`},
	}); err != nil {
		t.Fatalf("append event: %v", err)
	}
	out := filepath.Join(t.TempDir(), "review.rforgepkg")

	if _, err := Create(project, out, Options{}); err != nil {
		t.Fatalf("Create: %v", err)
	}
	packaged, err := os.ReadFile(filepath.Join(out, "project", filepath.FromSlash(provenance.EventsRelativePath)))
	if err != nil {
		t.Fatalf("read packaged ledger: %v", err)
	}
	text := string(packaged)
	for _, leaked := range []string{project, "/home/alice", `Users\\alice`} {
		if strings.Contains(text, leaked) {
			t.Fatalf("packaged ledger still contains %q:\n%s", leaked, text)
		}
	}
	var event map[string]any
	if err := json.Unmarshal([]byte(strings.TrimSpace(text)), &event); err != nil {
		t.Fatalf("packaged ledger is not valid JSONL: %v\n%s", err, text)
	}
	if event["target"] != "project" {
		t.Fatalf("project path not rewritten to package-relative form: target=%v", event["target"])
	}
	if got := event["inputs"].(map[string]any)["path"]; got != "project/data/library.json" {
		t.Fatalf("nested project path = %v, want project/data/library.json", got)
	}
	report, err := Audit(out)
	if err != nil {
		t.Fatalf("Audit: %v", err)
	}
	for _, check := range report.Checks {
		if check.Code == "content_privacy" && !check.OK {
			t.Fatalf("content_privacy failed after redaction: %s", check.Message)
		}
	}
	redaction, err := os.ReadFile(filepath.Join(out, "redaction-report.json"))
	if err != nil {
		t.Fatalf("read redaction report: %v", err)
	}
	if !strings.Contains(string(redaction), provenance.EventsRelativePath) {
		t.Fatalf("redaction report does not record the rewritten ledger:\n%s", redaction)
	}
}
