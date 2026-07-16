package forge

import (
	"bytes"
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/TrebuchetDynamics/research-forge/internal/reviewpkg"
)

func TestGuidedWorkflowCapturesSourceToolChoicesAndPrivacyPreview(t *testing.T) {
	projectPath := t.TempDir()
	state, err := Init(projectPath, InitOptions{Question: "Do catalysts improve hydrogen evolution?", SourceChoices: []string{"openalex", "semantic-scholar"}, ToolChoices: []string{"grobid", "qdrant"}, Actor: "tester"})
	if err != nil {
		t.Fatalf("init: %v", err)
	}
	if len(state.SourceChoices) != 2 || len(state.ToolChoices) != 2 || len(state.PrivacyLegalPreview) == 0 {
		t.Fatalf("state missing guided choices/privacy preview: %#v", state)
	}
	if !strings.Contains(strings.Join(state.PrivacyLegalPreview, " "), "reviewer approval") {
		t.Fatalf("preview missing reviewer approval warning: %#v", state.PrivacyLegalPreview)
	}
}

func TestInitDoesNotOverwriteHardLinkedForgeStateTarget(t *testing.T) {
	projectPath := t.TempDir()
	if _, err := Init(projectPath, InitOptions{Question: "Initial question", Actor: "tester"}); err != nil {
		t.Fatalf("seed forge state: %v", err)
	}
	path := statePath(projectPath)
	if err := os.Remove(path); err != nil {
		t.Fatalf("remove seeded forge state: %v", err)
	}
	outsidePath := filepath.Join(t.TempDir(), "outside-state.json")
	outsideBefore := []byte("outside state must remain unchanged\n")
	if err := os.WriteFile(outsidePath, outsideBefore, 0o600); err != nil {
		t.Fatalf("write outside state: %v", err)
	}
	if err := os.Link(outsidePath, path); err != nil {
		t.Skipf("hard links unavailable: %v", err)
	}

	state, err := Init(projectPath, InitOptions{Question: "Replacement question", Actor: "tester"})
	if err != nil {
		t.Fatalf("reinitialize forge state: %v", err)
	}
	if state.Question != "Replacement question" {
		t.Fatalf("state question = %q, want replacement question", state.Question)
	}
	outsideAfter, err := os.ReadFile(outsidePath)
	if err != nil {
		t.Fatalf("read outside state: %v", err)
	}
	if !bytes.Equal(outsideAfter, outsideBefore) {
		t.Fatalf("outside hard link changed: got %q, want %q", outsideAfter, outsideBefore)
	}
	info, err := os.Stat(path)
	if err != nil {
		t.Fatalf("stat replaced forge state: %v", err)
	}
	if got := info.Mode().Perm(); got != 0o600 {
		t.Fatalf("replaced forge state mode = %o, want 600", got)
	}
}

func TestInitRollbackDoesNotRewriteHardLinkedForgeStateWhenSaveFails(t *testing.T) {
	if os.Geteuid() == 0 {
		t.Skip("root can create staging files in a read-only directory")
	}
	projectPath := t.TempDir()
	if _, err := Init(projectPath, InitOptions{Question: "Initial question", Actor: "tester"}); err != nil {
		t.Fatalf("seed forge state: %v", err)
	}
	path := statePath(projectPath)
	if err := os.Remove(path); err != nil {
		t.Fatalf("remove seeded forge state: %v", err)
	}
	outsidePath := filepath.Join(t.TempDir(), "outside-state.json")
	outsideBefore := []byte("outside state must not be rewritten\n")
	if err := os.WriteFile(outsidePath, outsideBefore, 0o600); err != nil {
		t.Fatalf("write outside state: %v", err)
	}
	if err := os.Link(outsidePath, path); err != nil {
		t.Skipf("hard links unavailable: %v", err)
	}
	oldTime := time.Unix(1_600_000_000, 0)
	if err := os.Chtimes(outsidePath, oldTime, oldTime); err != nil {
		t.Fatalf("set outside timestamp: %v", err)
	}
	dataDir := filepath.Dir(path)
	if err := os.Chmod(dataDir, 0o555); err != nil {
		t.Fatalf("make forge data directory read-only: %v", err)
	}
	t.Cleanup(func() { _ = os.Chmod(dataDir, 0o755) })

	if _, err := Init(projectPath, InitOptions{Question: "Replacement question", Actor: "tester"}); err == nil {
		t.Fatal("Init succeeded despite a read-only forge data directory")
	}
	outsideAfter, err := os.ReadFile(outsidePath)
	if err != nil {
		t.Fatalf("read outside state: %v", err)
	}
	if !bytes.Equal(outsideAfter, outsideBefore) {
		t.Fatalf("outside hard link changed: got %q, want %q", outsideAfter, outsideBefore)
	}
	info, err := os.Stat(outsidePath)
	if err != nil {
		t.Fatalf("stat outside state: %v", err)
	}
	if !info.ModTime().Equal(oldTime) {
		t.Fatalf("outside hard link was rewritten: mtime=%s, want %s", info.ModTime(), oldTime)
	}
}

func TestStatusRejectsInvalidPersistedState(t *testing.T) {
	projectPath := t.TempDir()
	state, err := Init(projectPath, InitOptions{Question: "Do catalysts improve hydrogen evolution?", Actor: "tester"})
	if err != nil {
		t.Fatalf("init: %v", err)
	}
	state.CurrentState = StateID("corrupt_state")
	writeStateForTest(t, projectPath, state)

	_, err = Status(projectPath)
	if err == nil || !strings.Contains(err.Error(), `invalid forge state "corrupt_state"`) {
		t.Fatalf("status error = %v, want invalid persisted state rejection", err)
	}
}

func TestStatusRejectsUnsupportedPersistedSchemaVersion(t *testing.T) {
	projectPath := t.TempDir()
	state, err := Init(projectPath, InitOptions{Question: "Do catalysts improve hydrogen evolution?", Actor: "tester"})
	if err != nil {
		t.Fatalf("init: %v", err)
	}
	state.SchemaVersion = "999"
	writeStateForTest(t, projectPath, state)

	_, err = Status(projectPath)
	if err == nil || !strings.Contains(err.Error(), `unsupported forge schema version "999"`) {
		t.Fatalf("status error = %v, want unsupported schema rejection", err)
	}
}

func writeStateForTest(t *testing.T, projectPath string, state State) {
	t.Helper()
	data, err := json.MarshalIndent(state, "", "  ")
	if err != nil {
		t.Fatalf("marshal forge state: %v", err)
	}
	if err := os.WriteFile(statePath(projectPath), append(data, '\n'), 0o600); err != nil {
		t.Fatalf("write forge state: %v", err)
	}
}

func TestForgeStateRollsBackWhenTransitionProvenanceFails(t *testing.T) {
	t.Run("removes new init state", func(t *testing.T) {
		projectPath := t.TempDir()
		if _, err := Init(projectPath, InitOptions{Question: "Initial question", Actor: "tester"}); err != nil {
			t.Fatalf("seed project: %v", err)
		}
		if err := os.Remove(statePath(projectPath)); err != nil {
			t.Fatalf("remove seeded state: %v", err)
		}
		blockForgeProvenance(t, projectPath)

		if _, err := Init(projectPath, InitOptions{Question: "Replacement question", Actor: "tester"}); err == nil {
			t.Fatal("Init returned nil error, want provenance failure")
		}
		if _, err := os.Stat(statePath(projectPath)); !os.IsNotExist(err) {
			t.Fatalf("failed init left state file: %v", err)
		}
	})

	for _, tt := range []struct {
		name string
		run  func(string) error
	}{
		{
			name: "restores state after approval",
			run: func(projectPath string) error {
				_, err := Approve(projectPath, ApprovalInput{Gate: "question approval", Note: "approved", Actor: "reviewer"})
				return err
			},
		},
		{
			name: "restores state after reopen",
			run: func(projectPath string) error {
				_, err := Reopen(projectPath, StateSourcePlan, "revisit sources", "reviewer")
				return err
			},
		},
	} {
		t.Run(tt.name, func(t *testing.T) {
			projectPath := t.TempDir()
			if _, err := Init(projectPath, InitOptions{Question: "Initial question", Actor: "tester"}); err != nil {
				t.Fatalf("seed forge state: %v", err)
			}
			before, err := os.ReadFile(statePath(projectPath))
			if err != nil {
				t.Fatalf("read prior state: %v", err)
			}
			blockForgeProvenance(t, projectPath)

			if err := tt.run(projectPath); err == nil {
				t.Fatal("state transition returned nil error, want provenance failure")
			}
			after, err := os.ReadFile(statePath(projectPath))
			if err != nil {
				t.Fatalf("read state after failure: %v", err)
			}
			if !bytes.Equal(after, before) {
				t.Errorf("state changed after provenance failure:\n got: %s\nwant: %s", after, before)
			}
		})
	}
}

func TestFixtureSourceImportRollsBackArtifactsWhenTransitionProvenanceFails(t *testing.T) {
	projectPath := t.TempDir()
	state, err := Init(projectPath, InitOptions{Question: "Initial question", Actor: "tester"})
	if err != nil {
		t.Fatalf("seed forge state: %v", err)
	}
	state.CurrentState = StateImportPlan
	state.refresh()
	writeStateForTest(t, projectPath, state)
	stateBefore, err := os.ReadFile(statePath(projectPath))
	if err != nil {
		t.Fatalf("read prior state: %v", err)
	}
	relativePaths := reviewpkg.ArtificialPhotosynthesisFixtureSourceImportPaths()
	wantPaths := []string{
		"data/connector-capabilities.json",
		"data/import-receipts/fake-sources.json",
		"data/library.json",
		"data/source-cache/fake-arxiv-artificial-photosynthesis.json",
		"data/source-cache/fake-openalex-artificial-photosynthesis.json",
		"data/source-plans/artificial-photosynthesis.json",
	}
	if strings.Join(relativePaths, "\n") != strings.Join(wantPaths, "\n") {
		t.Fatalf("source-import artifact paths = %#v, want %#v", relativePaths, wantPaths)
	}
	previous := make(map[string][]byte, len(relativePaths))
	for _, relativePath := range relativePaths {
		path := filepath.Join(projectPath, filepath.FromSlash(relativePath))
		if relativePath == "data/library.json" {
			previous[path] = []byte("[{\"sentinel\":true}]\n")
			if err := os.WriteFile(path, previous[path], 0o640); err != nil {
				t.Fatalf("write prior library: %v", err)
			}
		} else {
			previous[path] = nil
		}
	}
	blockForgeProvenance(t, projectPath)

	if _, err := CompleteFixtureSourceImport(projectPath, "tester"); err == nil {
		t.Fatal("CompleteFixtureSourceImport returned nil error, want provenance failure")
	}
	for path, want := range previous {
		got, err := os.ReadFile(path)
		if want == nil {
			if !os.IsNotExist(err) {
				t.Errorf("failed source import left artifact %s: %v", path, err)
			}
			continue
		}
		if err != nil {
			t.Fatalf("read restored artifact %s: %v", path, err)
		}
		if !bytes.Equal(got, want) {
			t.Errorf("restored artifact %s = %q, want %q", path, got, want)
		}
	}
	stateAfter, err := os.ReadFile(statePath(projectPath))
	if err != nil {
		t.Fatalf("read restored state: %v", err)
	}
	if !bytes.Equal(stateAfter, stateBefore) {
		t.Errorf("state changed after failed source import:\n got: %s\nwant: %s", stateAfter, stateBefore)
	}
}

func TestFixtureReferenceManagerRollsBackArtifactsWhenTransitionProvenanceFails(t *testing.T) {
	projectPath := t.TempDir()
	state, err := Init(projectPath, InitOptions{Question: "Initial question", Actor: "tester"})
	if err != nil {
		t.Fatalf("seed forge state: %v", err)
	}
	state.CurrentState = StateDedupeReview
	state.refresh()
	writeStateForTest(t, projectPath, state)
	stateBefore, err := os.ReadFile(statePath(projectPath))
	if err != nil {
		t.Fatalf("read prior state: %v", err)
	}
	wantPaths := []string{
		"data/connector-capabilities.json",
		"data/import-receipts/fake-sources.json",
		"data/library.json",
		"data/privacy-licensing-review.json",
		"data/reference-manager/fidelity.json",
		"data/reference-manager/interchange-matrix.json",
		"data/source-cache/fake-arxiv-artificial-photosynthesis.json",
		"data/source-cache/fake-openalex-artificial-photosynthesis.json",
		"data/source-cache/jabref-artificial-photosynthesis.bib",
		"data/source-cache/zotero-rdf-artificial-photosynthesis.xml",
		"data/source-plans/artificial-photosynthesis.json",
	}
	relativePaths := reviewpkg.ArtificialPhotosynthesisReferenceManagerFixturePaths()
	if strings.Join(relativePaths, "\n") != strings.Join(wantPaths, "\n") {
		t.Fatalf("reference-manager artifact paths = %#v, want %#v", relativePaths, wantPaths)
	}
	previous := make(map[string][]byte, len(relativePaths))
	for _, relativePath := range relativePaths {
		path := filepath.Join(projectPath, filepath.FromSlash(relativePath))
		switch relativePath {
		case "data/library.json":
			previous[path] = []byte("[{\"sentinel\":\"library\"}]\n")
		case "data/privacy-licensing-review.json":
			previous[path] = []byte("{\"sentinel\":\"privacy\"}\n")
		default:
			previous[path] = nil
			continue
		}
		if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
			t.Fatalf("create prior artifact directory: %v", err)
		}
		if err := os.WriteFile(path, previous[path], 0o640); err != nil {
			t.Fatalf("write prior artifact %s: %v", path, err)
		}
	}
	blockForgeProvenance(t, projectPath)

	if _, err := CompleteFixtureReferenceManager(projectPath, "tester"); err == nil {
		t.Fatal("CompleteFixtureReferenceManager returned nil error, want provenance failure")
	}
	for path, want := range previous {
		got, err := os.ReadFile(path)
		if want == nil {
			if !os.IsNotExist(err) {
				t.Errorf("failed reference-manager import left artifact %s: %v", path, err)
			}
			continue
		}
		if err != nil {
			t.Fatalf("read restored artifact %s: %v", path, err)
		}
		if !bytes.Equal(got, want) {
			t.Errorf("restored artifact %s = %q, want %q", path, got, want)
		}
	}
	stateAfter, err := os.ReadFile(statePath(projectPath))
	if err != nil {
		t.Fatalf("read restored state: %v", err)
	}
	if !bytes.Equal(stateAfter, stateBefore) {
		t.Errorf("state changed after failed reference-manager import:\n got: %s\nwant: %s", stateAfter, stateBefore)
	}
}

func TestFixtureAcquisitionRollsBackArtifactsWhenTransitionProvenanceFails(t *testing.T) {
	projectPath := t.TempDir()
	state, err := Init(projectPath, InitOptions{Question: "Initial question", Actor: "tester"})
	if err != nil {
		t.Fatalf("seed forge state: %v", err)
	}
	state.CurrentState = StateFullTextAcquisition
	state.refresh()
	writeStateForTest(t, projectPath, state)
	stateBefore, err := os.ReadFile(statePath(projectPath))
	if err != nil {
		t.Fatalf("read prior state: %v", err)
	}
	wantPaths := []string{
		"data/connector-capabilities.json",
		"data/document-assets.json",
		"data/import-receipts/fake-sources.json",
		"data/legal-acquisition-queue.json",
		"data/library.json",
		"data/privacy-licensing-review.json",
		"data/reference-manager/fidelity.json",
		"data/reference-manager/interchange-matrix.json",
		"data/source-cache/fake-arxiv-artificial-photosynthesis.json",
		"data/source-cache/fake-openalex-artificial-photosynthesis.json",
		"data/source-cache/jabref-artificial-photosynthesis.bib",
		"data/source-cache/zotero-rdf-artificial-photosynthesis.xml",
		"data/source-plans/artificial-photosynthesis.json",
		"documents/open-access/ap-fixture.txt",
	}
	relativePaths := reviewpkg.ArtificialPhotosynthesisAcquisitionFixturePaths()
	if strings.Join(relativePaths, "\n") != strings.Join(wantPaths, "\n") {
		t.Fatalf("acquisition artifact paths = %#v, want %#v", relativePaths, wantPaths)
	}
	previous := make(map[string][]byte, len(relativePaths))
	for _, relativePath := range relativePaths {
		path := filepath.Join(projectPath, filepath.FromSlash(relativePath))
		switch relativePath {
		case "data/library.json":
			previous[path] = []byte("[{\"sentinel\":\"library\"}]\n")
		case "data/legal-acquisition-queue.json":
			previous[path] = []byte("{\"sentinel\":\"queue\"}\n")
		case "documents/open-access/ap-fixture.txt":
			previous[path] = []byte("existing document text\n")
		default:
			previous[path] = nil
			continue
		}
		if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
			t.Fatalf("create prior artifact directory: %v", err)
		}
		if err := os.WriteFile(path, previous[path], 0o640); err != nil {
			t.Fatalf("write prior artifact %s: %v", path, err)
		}
	}
	blockForgeProvenance(t, projectPath)

	if _, err := CompleteFixtureAcquisition(projectPath, "tester"); err == nil {
		t.Fatal("CompleteFixtureAcquisition returned nil error, want provenance failure")
	}
	for path, want := range previous {
		got, err := os.ReadFile(path)
		if want == nil {
			if !os.IsNotExist(err) {
				t.Errorf("failed acquisition left artifact %s: %v", path, err)
			}
			continue
		}
		if err != nil {
			t.Fatalf("read restored artifact %s: %v", path, err)
		}
		if !bytes.Equal(got, want) {
			t.Errorf("restored artifact %s = %q, want %q", path, got, want)
		}
	}
	stateAfter, err := os.ReadFile(statePath(projectPath))
	if err != nil {
		t.Fatalf("read restored state: %v", err)
	}
	if !bytes.Equal(stateAfter, stateBefore) {
		t.Errorf("state changed after failed acquisition:\n got: %s\nwant: %s", stateAfter, stateBefore)
	}
}

func TestFixturePackageRollsBackProjectAndPackageWhenTransitionProvenanceFails(t *testing.T) {
	projectPath := t.TempDir()
	state, err := Init(projectPath, InitOptions{Question: "Initial question", Actor: "tester"})
	if err != nil {
		t.Fatalf("seed forge state: %v", err)
	}
	state.CurrentState = StatePackageExport
	state.refresh()
	writeStateForTest(t, projectPath, state)
	stateBefore, err := os.ReadFile(statePath(projectPath))
	if err != nil {
		t.Fatalf("read prior state: %v", err)
	}
	wantFixturePaths := []string{
		"analysis/forest-plot.txt",
		"analysis/run1-artifact-manifest.json",
		"data/claim-trace.json",
		"data/connector-capabilities.json",
		"data/document-assets.json",
		"data/evidence.items.json",
		"data/evidence.schemas.json",
		"data/forge-state.json",
		"data/identity-decisions.jsonl",
		"data/import-receipts/fake-sources.json",
		"data/legal-acquisition-queue.json",
		"data/library.json",
		"data/parser-manifests/fake-parser.json",
		"data/privacy-licensing-review.json",
		"data/provenance.jsonl",
		"data/reference-manager/fidelity.json",
		"data/reference-manager/interchange-matrix.json",
		"data/screening-audit.jsonl",
		"data/source-cache/fake-arxiv-artificial-photosynthesis.json",
		"data/source-cache/fake-openalex-artificial-photosynthesis.json",
		"data/source-cache/jabref-artificial-photosynthesis.bib",
		"data/source-cache/zotero-rdf-artificial-photosynthesis.xml",
		"data/source-plans/artificial-photosynthesis.json",
		"documents/open-access/ap-fixture.txt",
		"parsed/artificial-photosynthesis-passages.json",
		"reports/report.md",
		"rforge.lock.json",
		"rforge.project.toml",
	}
	fixturePaths := reviewpkg.ArtificialPhotosynthesisFixtureProjectPaths()
	if strings.Join(fixturePaths, "\n") != strings.Join(wantFixturePaths, "\n") {
		t.Fatalf("complete fixture artifact paths = %#v, want %#v", fixturePaths, wantFixturePaths)
	}
	previous := make(map[string][]byte, len(fixturePaths))
	for _, relativePath := range fixturePaths {
		path := filepath.Join(projectPath, filepath.FromSlash(relativePath))
		switch relativePath {
		case "data/forge-state.json":
			previous[path] = stateBefore
			continue
		case "data/library.json":
			previous[path] = []byte("[{\"sentinel\":\"library\"}]\n")
		case "reports/report.md":
			previous[path] = []byte("# Existing report\n")
		default:
			data, err := os.ReadFile(path)
			if err == nil {
				previous[path] = data
			} else if os.IsNotExist(err) {
				previous[path] = nil
			} else {
				t.Fatalf("read prior artifact %s: %v", path, err)
			}
			continue
		}
		if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
			t.Fatalf("create prior artifact directory: %v", err)
		}
		if err := os.WriteFile(path, previous[path], 0o640); err != nil {
			t.Fatalf("write prior artifact %s: %v", path, err)
		}
	}

	packagePath := filepath.Join(t.TempDir(), "review.rforgepkg")
	if err := os.MkdirAll(packagePath, 0o750); err != nil {
		t.Fatalf("create prior package: %v", err)
	}
	packageSentinel := []byte("existing package content\n")
	if err := os.WriteFile(filepath.Join(packagePath, "prior-package.txt"), packageSentinel, 0o640); err != nil {
		t.Fatalf("write prior package: %v", err)
	}
	blockForgeProvenance(t, projectPath)

	if _, err := CompleteFixturePackage(projectPath, packagePath, "tester"); err == nil {
		t.Fatal("CompleteFixturePackage returned nil error, want provenance failure")
	}
	for path, want := range previous {
		got, err := os.ReadFile(path)
		if want == nil {
			if !os.IsNotExist(err) {
				t.Errorf("failed package completion left project artifact %s: %v", path, err)
			}
			continue
		}
		if err != nil {
			t.Fatalf("read restored project artifact %s: %v", path, err)
		}
		if !bytes.Equal(got, want) {
			t.Errorf("restored project artifact %s = %q, want %q", path, got, want)
		}
	}
	entries, err := os.ReadDir(packagePath)
	if err != nil {
		t.Fatalf("read restored package: %v", err)
	}
	if len(entries) != 1 || entries[0].Name() != "prior-package.txt" {
		t.Fatalf("restored package entries = %#v, want only prior-package.txt", entries)
	}
	gotPackageSentinel, err := os.ReadFile(filepath.Join(packagePath, "prior-package.txt"))
	if err != nil {
		t.Fatalf("read restored package content: %v", err)
	}
	if !bytes.Equal(gotPackageSentinel, packageSentinel) {
		t.Errorf("restored package content = %q, want %q", gotPackageSentinel, packageSentinel)
	}
}

func TestFixturePackageRemovesNewOutputDirectoriesWhenTransitionProvenanceFails(t *testing.T) {
	projectPath := t.TempDir()
	state, err := Init(projectPath, InitOptions{Question: "Initial question", Actor: "tester"})
	if err != nil {
		t.Fatalf("seed forge state: %v", err)
	}
	state.CurrentState = StatePackageExport
	state.refresh()
	writeStateForTest(t, projectPath, state)
	stateBefore, err := os.ReadFile(statePath(projectPath))
	if err != nil {
		t.Fatalf("read prior state: %v", err)
	}
	outputRoot := t.TempDir()
	createdParent := filepath.Join(outputRoot, "new", "nested")
	packagePath := filepath.Join(createdParent, "review.rforgepkg")
	blockForgeProvenance(t, projectPath)

	if _, err := CompleteFixturePackage(projectPath, packagePath, "tester"); err == nil {
		t.Fatal("CompleteFixturePackage returned nil error, want provenance failure")
	}
	if _, err := os.Stat(filepath.Join(outputRoot, "new")); !os.IsNotExist(err) {
		t.Fatalf("failed package completion left new output directories: %v", err)
	}
	stateAfter, err := os.ReadFile(statePath(projectPath))
	if err != nil {
		t.Fatalf("read restored state: %v", err)
	}
	if !bytes.Equal(stateAfter, stateBefore) {
		t.Errorf("state changed after failed package completion:\n got: %s\nwant: %s", stateAfter, stateBefore)
	}
}

func blockForgeProvenance(t *testing.T, projectPath string) {
	t.Helper()
	path := filepath.Join(projectPath, "provenance")
	if err := os.RemoveAll(path); err != nil {
		t.Fatalf("remove provenance directory: %v", err)
	}
	if err := os.WriteFile(path, []byte("not a directory"), 0o644); err != nil {
		t.Fatalf("create provenance blocker: %v", err)
	}
}

func TestGuidedWorkflowRequiresReviewGatesBeforeIrreversibleTransitions(t *testing.T) {
	projectPath := t.TempDir()
	state, err := Init(projectPath, InitOptions{Question: "Do catalysts improve hydrogen evolution?", Actor: "tester"})
	if err != nil {
		t.Fatalf("init: %v", err)
	}
	if state.CurrentState != StateQuestionDraft || !state.BlockedBy("question approval") {
		t.Fatalf("init state = %#v", state)
	}
	if _, err := Next(projectPath, "tester"); err == nil || !strings.Contains(err.Error(), "blocked review gate") {
		t.Fatalf("next before approval err = %v", err)
	}

	state, err = Approve(projectPath, ApprovalInput{Gate: "question approval", Note: "canonical question accepted", Actor: "reviewer"})
	if err != nil {
		t.Fatalf("approve question: %v", err)
	}
	if state.CurrentState != StateProtocolPlan || !state.BlockedBy("protocol approval") {
		t.Fatalf("after question approval = %#v", state)
	}

	state, err = Approve(projectPath, ApprovalInput{Gate: "protocol approval", Note: "criteria acceptable", Actor: "reviewer"})
	if err != nil {
		t.Fatalf("approve protocol: %v", err)
	}
	if state.CurrentState != StateSourcePlan || !state.BlockedBy("network/API approval") {
		t.Fatalf("after protocol approval = %#v", state)
	}

	status, err := Status(projectPath)
	if err != nil {
		t.Fatalf("status: %v", err)
	}
	if len(status.NextSafeActions) == 0 || !strings.Contains(status.NextSafeActions[0].CLI, "rforge protocol") {
		t.Fatalf("next actions = %#v", status.NextSafeActions)
	}
	events, err := ProvenanceEvents(projectPath)
	if err != nil {
		t.Fatalf("events: %v", err)
	}
	transitions := 0
	for _, event := range events {
		if event.Action == "forge.state.transition" {
			transitions++
		}
	}
	if transitions < 3 {
		t.Fatalf("want provenance transitions, got %d events=%#v", transitions, events)
	}
}

func TestReviewGateCatalogCoversIrreversibleScientificAndSharingDecisions(t *testing.T) {
	gates := ReviewGates()
	want := []string{"protocol approval", "network/API approval", "identity approval", "legal acquisition approval", "parser arbitration approval", "screening approval", "evidence approval", "analysis approval", "claim approval", "package approval"}
	for _, gate := range want {
		if _, ok := gates[gate]; !ok {
			t.Fatalf("missing gate %q in %#v", gate, gates)
		}
	}
}
