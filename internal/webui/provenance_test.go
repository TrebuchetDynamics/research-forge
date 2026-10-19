package webui

import (
	"os"
	"path/filepath"
	"slices"
	"testing"

	"github.com/TrebuchetDynamics/research-forge/internal/provenance"
	"github.com/TrebuchetDynamics/research-forge/internal/reviewpkg"
)

func TestProjectProvenanceReachesReviewPackageAndViewModels(t *testing.T) {
	for _, tt := range []struct {
		name      string
		canonical bool
		legacy    bool
		wantPath  string
		wantID    string
	}{
		{name: "appended event", canonical: true, wantPath: "provenance/events.jsonl", wantID: "evt-project"},
		{name: "legacy only", legacy: true, wantPath: "data/provenance.jsonl", wantID: "evt-legacy"},
		{name: "canonical takes precedence", canonical: true, legacy: true, wantPath: "provenance/events.jsonl", wantID: "evt-project"},
	} {
		t.Run(tt.name, func(t *testing.T) {
			project := t.TempDir()
			mustWrite(t, filepath.Join(project, "rforge.project.toml"), "title = \"Provenance\"\n")
			mustWrite(t, filepath.Join(project, "rforge.lock.json"), "{}\n")
			event := provenance.Event{SchemaVersion: "1", ID: "evt-legacy", Timestamp: "2026-01-01T00:00:00Z", Actor: "tester", Action: "project.create", Target: "project", Inputs: map[string]any{}, Outputs: map[string]any{}, Warnings: []string{}}
			if tt.legacy {
				if err := provenance.Append(project, event); err != nil {
					t.Fatal(err)
				}
				if err := os.MkdirAll(filepath.Join(project, "data"), 0o755); err != nil {
					t.Fatal(err)
				}
				if err := os.Rename(filepath.Join(project, "provenance/events.jsonl"), filepath.Join(project, "data/provenance.jsonl")); err != nil {
					t.Fatal(err)
				}
			}
			if tt.canonical {
				event.ID = "evt-project"
				if err := provenance.Append(project, event); err != nil {
					t.Fatal(err)
				}
			}

			home, err := BuildForgeHomeState(project)
			if err != nil {
				t.Fatal(err)
			}
			if len(home.ProvenanceEvents) != 1 || home.ProvenanceEvents[0].ID != tt.wantID {
				t.Errorf("home provenance = %#v, want %s", home.ProvenanceEvents, tt.wantID)
			}
			preview := BuildPackageExportCenterState(project)
			if !slices.Contains(preview.PackageContents, tt.wantPath) {
				t.Errorf("package preview = %v, missing %s", preview.PackageContents, tt.wantPath)
			}
			out := filepath.Join(t.TempDir(), "review.rforgepkg")
			pkg, err := reviewpkg.Create(project, out, reviewpkg.Options{})
			if err != nil {
				t.Fatal(err)
			}
			if pkg.Manifest.ProvenanceRef != "project/"+tt.wantPath {
				t.Errorf("package provenance ref = %q, want project/%s", pkg.Manifest.ProvenanceRef, tt.wantPath)
			}
			events, err := provenance.Read(filepath.Join(out, "project"))
			if err != nil || len(events) != 1 || events[0].ID != tt.wantID {
				t.Errorf("packaged provenance = %#v, err = %v, want %s", events, err, tt.wantID)
			}
		})
	}
}
