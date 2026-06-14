package cli

import (
	"bytes"
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/TrebuchetDynamics/research-forge/internal/analysis"
	"github.com/TrebuchetDynamics/research-forge/internal/library"
	"github.com/TrebuchetDynamics/research-forge/internal/provenance"
	"github.com/TrebuchetDynamics/research-forge/internal/screening"
)

// mustRunCLI executes an rforge command end-to-end and fails the test on a
// non-zero exit code. When the command is run with --json it also asserts the
// response envelope reports ok:true. It returns raw stdout for typed decoding.
func mustRunCLI(t *testing.T, args ...string) []byte {
	t.Helper()
	stdout := new(bytes.Buffer)
	stderr := new(bytes.Buffer)
	if code := Execute(args, stdout, stderr); code != 0 {
		t.Fatalf("rforge %s: exit=%d stderr=%s stdout=%s", strings.Join(args, " "), code, stderr.String(), stdout.String())
	}
	if len(args) > 0 && args[0] == "--json" {
		var envelope struct {
			OK bool `json:"ok"`
		}
		if err := json.Unmarshal(stdout.Bytes(), &envelope); err != nil {
			t.Fatalf("rforge %s: stdout is not JSON: %v\n%s", strings.Join(args, " "), err, stdout.String())
		}
		if !envelope.OK {
			t.Fatalf("rforge %s: ok != true:\n%s", strings.Join(args, " "), stdout.String())
		}
	}
	return stdout.Bytes()
}

// TestE2EFullResearchPipelineThroughCLI drives the whole research workflow that
// the MVP acceptance checklist describes (create -> import -> dedupe -> screen
// -> PRISMA -> extract -> meta-analysis -> report) end-to-end through the CLI,
// offline and deterministically, asserting each stage hands real state to the
// next and that the pipeline records an append-only provenance trail.
func TestE2EFullResearchPipelineThroughCLI(t *testing.T) {
	proj := filepath.Join(t.TempDir(), "research")

	// 1. Create the project workspace.
	mustRunCLI(t, "--json", "project", "create", proj, "--title", "Artificial Photosynthesis Review")
	if _, err := os.Stat(filepath.Join(proj, "rforge.project.toml")); err != nil {
		t.Fatalf("project manifest missing after create: %v", err)
	}

	// 2. Import a JSON library where two records are a fuzzy duplicate pair
	// (same title/author/year, distinct DOIs) plus one distinct record.
	libFile := filepath.Join(t.TempDir(), "library.json")
	fixture := `[
  {"Title":"Artificial photosynthesis catalyst A","Identifiers":{"DOI":"10.1000/ap-1"},"Authors":[{"Family":"Smith","Given":"Jane"}],"Year":2026},
  {"Title":"Artificial photosynthesis catalyst A","Identifiers":{"DOI":"10.1000/ap-1-preprint"},"Authors":[{"Family":"Smith","Given":"Jane"}],"Year":2026},
  {"Title":"Artificial photosynthesis catalyst B","Identifiers":{"DOI":"10.1000/ap-2"},"Authors":[{"Family":"Doe","Given":"John"}],"Year":2026}
]
`
	if err := os.WriteFile(libFile, []byte(fixture), 0o644); err != nil {
		t.Fatalf("write json library fixture: %v", err)
	}
	var imp struct {
		Data struct {
			Imported int `json:"imported"`
		} `json:"data"`
	}
	if err := json.Unmarshal(mustRunCLI(t, "--json", "--project", proj, "import", "json", libFile), &imp); err != nil {
		t.Fatalf("decode import: %v", err)
	}
	if imp.Data.Imported != 3 {
		t.Fatalf("imported = %d, want 3", imp.Data.Imported)
	}

	// 3. List the library and confirm every imported record is present.
	var lib struct {
		Data struct {
			Papers []library.PaperRecord `json:"papers"`
		} `json:"data"`
	}
	if err := json.Unmarshal(mustRunCLI(t, "--json", "--project", proj, "library", "list"), &lib); err != nil {
		t.Fatalf("decode library list: %v", err)
	}
	if len(lib.Data.Papers) != 3 {
		t.Fatalf("library list = %d papers, want 3", len(lib.Data.Papers))
	}

	// 4. Deduplicate: the exact-DOI pair must be reported.
	var dup struct {
		Data struct {
			Matches []json.RawMessage `json:"matches"`
		} `json:"data"`
	}
	if err := json.Unmarshal(mustRunCLI(t, "--json", "--project", proj, "duplicate", "report"), &dup); err != nil {
		t.Fatalf("decode duplicate report: %v", err)
	}
	if len(dup.Data.Matches) == 0 {
		t.Fatalf("duplicate report found no matches for the exact-DOI duplicate")
	}

	// 5. Configure screening and record decisions across the workflow.
	mustRunCLI(t, "--project", proj, "screen", "configure", "--reason", "off-topic", "--reason", "wrong outcome")
	mustRunCLI(t, "--project", proj, "screen", "decide", "--paper", "10.1000/ap-1", "--stage", "title_abstract", "--decision", "include", "--reviewer", "alice")
	mustRunCLI(t, "--project", proj, "screen", "decide", "--paper", "10.1000/ap-2", "--stage", "title_abstract", "--decision", "exclude", "--reason", "off-topic", "--reviewer", "alice")

	// 6. PRISMA counts must reflect the recorded decisions.
	var prisma struct {
		Data struct {
			Counts screening.PRISMACounts `json:"counts"`
		} `json:"data"`
	}
	if err := json.Unmarshal(mustRunCLI(t, "--json", "--project", proj, "prisma", "counts"), &prisma); err != nil {
		t.Fatalf("decode prisma counts: %v", err)
	}
	if prisma.Data.Counts.Included != 1 || prisma.Data.Counts.Excluded != 1 {
		t.Fatalf("prisma counts = %+v, want included=1 excluded=1", prisma.Data.Counts)
	}

	// 7. Define an extraction schema and extract accepted evidence with effect-size inputs.
	mustRunCLI(t, "--project", proj, "extraction", "schema", "add", "ap-schema",
		"--field", "mean_treatment:number", "--field", "mean_control:number",
		"--field", "sd_pooled:number", "--field", "n_treatment:number", "--field", "n_control:number")
	for _, ev := range []struct{ id, meanTreatment string }{
		{"10.1000/ap-1", "6"},
		{"10.1000/ap-2", "5"},
	} {
		mustRunCLI(t, "--project", proj, "extract", "add",
			"--paper", ev.id, "--schema", "ap-schema",
			"--value", "mean_treatment="+ev.meanTreatment, "--value", "mean_control=3",
			"--value", "sd_pooled=2", "--value", "n_treatment=12", "--value", "n_control=12",
			"--support", "passage:results-1", "--status", "accepted")
	}

	// 8. Evidence audit must be clean for accepted, supported evidence.
	var evAudit struct {
		Data struct {
			Issues []json.RawMessage `json:"issues"`
		} `json:"data"`
	}
	if err := json.Unmarshal(mustRunCLI(t, "--json", "--project", proj, "evidence", "audit"), &evAudit); err != nil {
		t.Fatalf("decode evidence audit: %v", err)
	}
	if len(evAudit.Data.Issues) != 0 {
		t.Fatalf("evidence audit reported %d issues for supported accepted evidence", len(evAudit.Data.Issues))
	}

	// 9. Prepare, run, and export the meta-analysis (deterministic fake R runner).
	var prep struct {
		Data struct {
			Run analysis.AnalysisRun `json:"run"`
		} `json:"data"`
	}
	if err := json.Unmarshal(mustRunCLI(t, "--json", "--project", proj, "analysis", "prepare", "run1"), &prep); err != nil {
		t.Fatalf("decode analysis prepare: %v", err)
	}
	if len(prep.Data.Run.InputRows) != 2 {
		t.Fatalf("analysis prepare InputRows = %d, want 2 accepted evidence rows", len(prep.Data.Run.InputRows))
	}
	mustRunCLI(t, "--json", "--project", proj, "analysis", "run", "run1")
	exportPath := filepath.Join(proj, "exports", "run1.json")
	mustRunCLI(t, "--json", "--project", proj, "analysis", "export", "run1", exportPath)
	if _, err := os.Stat(exportPath); err != nil {
		t.Fatalf("analysis export missing: %v", err)
	}

	// 10. Build and audit the reproducible report.
	reportPath := filepath.Join(proj, "exports", "report.md")
	mustRunCLI(t, "--json", "--project", proj, "report", "build", "--out", reportPath)
	reportBytes, err := os.ReadFile(reportPath)
	if err != nil {
		t.Fatalf("read built report: %v", err)
	}
	if !strings.Contains(string(reportBytes), "Artificial Photosynthesis Review") {
		t.Fatalf("report missing project title:\n%s", string(reportBytes))
	}
	mustRunCLI(t, "--json", "--project", proj, "report", "audit")

	// 11. The whole pipeline must have recorded an append-only provenance trail.
	events, err := provenance.Read(proj)
	if err != nil {
		t.Fatalf("read provenance: %v", err)
	}
	if len(events) == 0 {
		t.Fatalf("no provenance events recorded across the pipeline")
	}
}

// TestE2EReportReproducibleAcrossRebuilds asserts that rebuilding the report
// from the same stored project state produces byte-identical output, which the
// MVP acceptance checklist requires for reproducibility.
func TestE2EReportReproducibleAcrossRebuilds(t *testing.T) {
	proj := filepath.Join(t.TempDir(), "research")
	mustRunCLI(t, "--json", "project", "create", proj, "--title", "Reproducible Review")

	first := filepath.Join(proj, "exports", "report-1.md")
	second := filepath.Join(proj, "exports", "report-2.md")
	mustRunCLI(t, "--json", "--project", proj, "report", "build", "--out", first)
	mustRunCLI(t, "--json", "--project", proj, "report", "build", "--out", second)

	a, err := os.ReadFile(first)
	if err != nil {
		t.Fatalf("read first report: %v", err)
	}
	b, err := os.ReadFile(second)
	if err != nil {
		t.Fatalf("read second report: %v", err)
	}
	if !bytes.Equal(a, b) {
		t.Fatalf("report rebuild is not reproducible:\n--- first ---\n%s\n--- second ---\n%s", a, b)
	}
	if !strings.Contains(string(a), "Reproducible Review") {
		t.Fatalf("report missing project title:\n%s", a)
	}
}

// TestE2EImportExportRoundTripPreservesLibrary imports a JSON library, exports
// it to BibTeX, and re-imports that export into a fresh project, asserting the
// scholarly identifiers survive the round trip across formats and projects.
func TestE2EImportExportRoundTripPreservesLibrary(t *testing.T) {
	source := filepath.Join(t.TempDir(), "source")
	mustRunCLI(t, "--json", "project", "create", source, "--title", "Round Trip Source")

	libFile := filepath.Join(t.TempDir(), "library.json")
	fixture := `[
  {"Title":"Artificial photosynthesis catalyst A","Identifiers":{"DOI":"10.1000/rt-a"},"Year":2026},
  {"Title":"Artificial photosynthesis catalyst B","Identifiers":{"DOI":"10.1000/rt-b"},"Year":2025}
]
`
	if err := os.WriteFile(libFile, []byte(fixture), 0o644); err != nil {
		t.Fatalf("write json library fixture: %v", err)
	}
	mustRunCLI(t, "--json", "--project", source, "import", "json", libFile)

	exported := filepath.Join(t.TempDir(), "exported.bib")
	mustRunCLI(t, "--json", "--project", source, "export", "bibtex", exported)

	dest := filepath.Join(t.TempDir(), "dest")
	mustRunCLI(t, "--json", "project", "create", dest, "--title", "Round Trip Dest")
	mustRunCLI(t, "--json", "--project", dest, "import", "bibtex", exported)

	var lib struct {
		Data struct {
			Papers []library.PaperRecord `json:"papers"`
		} `json:"data"`
	}
	if err := json.Unmarshal(mustRunCLI(t, "--json", "--project", dest, "library", "list"), &lib); err != nil {
		t.Fatalf("decode library list: %v", err)
	}
	got := map[string]string{}
	for _, p := range lib.Data.Papers {
		got[p.Identifiers.DOI] = p.Title
	}
	for doi, title := range map[string]string{
		"10.1000/rt-a": "Artificial photosynthesis catalyst A",
		"10.1000/rt-b": "Artificial photosynthesis catalyst B",
	} {
		if got[doi] != title {
			t.Fatalf("round-trip lost %s: got title %q, want %q (library=%+v)", doi, got[doi], title, got)
		}
	}
}

// TestE2EProjectArchiveRestoreRoundTrip archives a populated project and
// restores it into a fresh location, asserting the manifest and library survive
// archive/restore so a project can be moved or backed up reproducibly.
func TestE2EProjectArchiveRestoreRoundTrip(t *testing.T) {
	proj := filepath.Join(t.TempDir(), "research")
	mustRunCLI(t, "--json", "project", "create", proj, "--title", "Archive Review")

	libFile := filepath.Join(t.TempDir(), "library.json")
	if err := os.WriteFile(libFile, []byte(`[{"Title":"Artificial photosynthesis archive fixture","Identifiers":{"DOI":"10.1000/arc-1"},"Year":2026}]`), 0o644); err != nil {
		t.Fatalf("write json library fixture: %v", err)
	}
	mustRunCLI(t, "--json", "--project", proj, "import", "json", libFile)

	archivePath := filepath.Join(t.TempDir(), "project.rforge.tar")
	mustRunCLI(t, "--json", "archive", "create", proj, archivePath)

	restored := filepath.Join(t.TempDir(), "restored")
	mustRunCLI(t, "--json", "archive", "restore", archivePath, restored)

	manifest, err := os.ReadFile(filepath.Join(restored, "rforge.project.toml"))
	if err != nil {
		t.Fatalf("restored manifest missing: %v", err)
	}
	if !strings.Contains(string(manifest), `title = "Archive Review"`) {
		t.Fatalf("restored manifest lost title:\n%s", manifest)
	}

	var lib struct {
		Data struct {
			Papers []library.PaperRecord `json:"papers"`
		} `json:"data"`
	}
	if err := json.Unmarshal(mustRunCLI(t, "--json", "--project", restored, "library", "list"), &lib); err != nil {
		t.Fatalf("decode restored library list: %v", err)
	}
	if len(lib.Data.Papers) != 1 || lib.Data.Papers[0].Identifiers.DOI != "10.1000/arc-1" {
		t.Fatalf("restored library lost records: %+v", lib.Data.Papers)
	}
}
