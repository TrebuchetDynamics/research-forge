package cli

import (
	"bytes"
	"io"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"regexp"
	"strings"
	"testing"
	"time"

	"github.com/TrebuchetDynamics/research-forge/internal/library"
	"github.com/TrebuchetDynamics/research-forge/internal/parsing"
)

func TestExecuteWatchRunReportsProvenanceFailure(t *testing.T) {
	for _, tt := range []struct {
		name          string
		previousInbox []byte
	}{
		{name: "removes new inbox"},
		{name: "restores existing inbox", previousInbox: []byte("[{\"sentinel\":true}]\n")},
	} {
		t.Run(tt.name, func(t *testing.T) {
			project := t.TempDir()
			opts := globalOptions{Project: project, JSON: true}
			var stdout, stderr bytes.Buffer
			if code := executeWatch([]string{"add", "ap", "--source", "openalex", "--query", "artificial photosynthesis"}, &stdout, &stderr, opts); code != 0 {
				t.Fatalf("watch add code = %d, stdout = %s, stderr = %s", code, stdout.String(), stderr.String())
			}
			inboxPath := filepath.Join(project, "data", "inbox.json")
			if tt.previousInbox != nil {
				if err := os.WriteFile(inboxPath, tt.previousInbox, 0o640); err != nil {
					t.Fatalf("write prior inbox: %v", err)
				}
			}
			if err := os.WriteFile(filepath.Join(project, "provenance"), []byte("not a directory"), 0o644); err != nil {
				t.Fatalf("create provenance blocker: %v", err)
			}
			stdout.Reset()
			stderr.Reset()

			code := executeWatch([]string{"run", "ap"}, &stdout, &stderr, opts)
			if code != 1 || !strings.Contains(stdout.String(), "watch_provenance_failed") {
				t.Errorf("watch run code = %d, stdout = %s, stderr = %s, want provenance failure", code, stdout.String(), stderr.String())
			}
			if tt.previousInbox == nil {
				if _, err := os.Stat(inboxPath); !os.IsNotExist(err) {
					t.Errorf("new inbox remains after provenance failure: %v", err)
				}
				return
			}
			gotInbox, err := os.ReadFile(inboxPath)
			if err != nil {
				t.Fatalf("read restored inbox: %v", err)
			}
			if !bytes.Equal(gotInbox, tt.previousInbox) {
				t.Errorf("inbox after provenance failure = %q, want prior contents %q", gotInbox, tt.previousInbox)
			}
		})
	}
}

func TestParseArbitrationRestoresReportAfterProvenanceFailure(t *testing.T) {
	project := t.TempDir()
	left := filepath.Join(project, "left.json")
	right := filepath.Join(project, "right.json")
	out := filepath.Join(project, "arbitration.json")
	writeParsedFixture(t, left, parsing.ParsedDocument{
		SchemaVersion: "1",
		PaperID:       "paper-1",
		ParserName:    "grobid",
		Title:         "Title",
		Abstract:      "Abstract",
	})
	writeParsedFixture(t, right, parsing.ParsedDocument{
		SchemaVersion: "1",
		PaperID:       "paper-1",
		ParserName:    "s2orc",
		Title:         "Title",
	})
	previous := []byte("{\"sentinel\":\"prior arbitration\"}\n")
	if err := os.WriteFile(out, previous, 0o600); err != nil {
		t.Fatalf("write prior arbitration report: %v", err)
	}
	blockProvenance(t, project)
	var stdout, stderr bytes.Buffer

	code := Execute([]string{
		"--json", "--project", project, "parse", "arbitrate",
		"--left", left, "--right", right, "--out", out,
		"--accept", "grobid", "--reason", "best field coverage", "--reviewer", "reviewer-a",
	}, &stdout, &stderr)
	if code != 1 || !strings.Contains(stdout.String(), "parse_arbitrate_provenance_failed") {
		t.Fatalf("parse arbitrate code = %d, stdout = %s, stderr = %s, want provenance failure", code, stdout.String(), stderr.String())
	}
	assertFileRestored(t, out, previous)
	info, err := os.Stat(out)
	if err != nil {
		t.Fatalf("stat restored arbitration report: %v", err)
	}
	if got := info.Mode().Perm(); got != 0o600 {
		t.Fatalf("restored arbitration report mode = %o, want 600", got)
	}
}

func TestReferenceAdjudicationRestoresLogAfterProvenanceFailure(t *testing.T) {
	project := t.TempDir()
	parsed := filepath.Join(project, "parsed.json")
	writeParsedFixture(t, parsed, parsing.ParsedDocument{
		SchemaVersion: "1",
		PaperID:       "paper-1",
		References:    []parsing.Reference{{Title: "Original"}},
	})
	logPath := filepath.Join(project, "data", "reference-adjudications.jsonl")
	if err := os.MkdirAll(filepath.Dir(logPath), 0o755); err != nil {
		t.Fatalf("create adjudication log directory: %v", err)
	}
	previous := []byte("{\"sentinel\":\"prior adjudication\"}\n")
	if err := os.WriteFile(logPath, previous, 0o600); err != nil {
		t.Fatalf("write prior adjudication log: %v", err)
	}
	blockProvenance(t, project)
	var stdout, stderr bytes.Buffer

	code := Execute([]string{
		"--json", "--project", project, "parse", "adjudicate-ref",
		"--parsed", parsed, "--index", "0", "--decision", "accept",
		"--reviewer", "reviewer-a", "--reason", "verified reference",
	}, &stdout, &stderr)
	if code != 1 || !strings.Contains(stdout.String(), "parse_ref_adjudication_provenance_failed") {
		t.Fatalf("parse adjudicate-ref code = %d, stdout = %s, stderr = %s, want provenance failure", code, stdout.String(), stderr.String())
	}
	assertFileRestored(t, logPath, previous)
	info, err := os.Stat(logPath)
	if err != nil {
		t.Fatalf("stat restored adjudication log: %v", err)
	}
	if got := info.Mode().Perm(); got != 0o600 {
		t.Fatalf("restored adjudication log mode = %o, want 600", got)
	}
}

func TestSearchImportRestoresLibraryAndResumeStateAfterProvenanceFailure(t *testing.T) {
	project := t.TempDir()
	libraryPath := filepath.Join(project, "data", "library.json")
	if err := os.MkdirAll(filepath.Dir(libraryPath), 0o755); err != nil {
		t.Fatalf("create library directory: %v", err)
	}
	previousLibrary := []byte("[]\n")
	if err := os.WriteFile(libraryPath, previousLibrary, 0o600); err != nil {
		t.Fatalf("write prior library: %v", err)
	}
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/works" {
			t.Fatalf("unexpected path: %s", r.URL.Path)
		}
		_, _ = w.Write([]byte(`{"meta":{"next_cursor":"page-2"},"results":[{"id":"https://openalex.org/W1","title":"Imported paper","doi":"https://doi.org/10.1000/imported"}]}`))
	}))
	defer server.Close()
	t.Setenv("RFORGE_OPENALEX_URL", server.URL)
	resumeStatePath := filepath.Join(project, "openalex-state.json")
	blockProvenance(t, project)
	var stdout, stderr bytes.Buffer

	code := Execute([]string{
		"--json", "--project", project, "search", "import",
		"--source", "openalex", "--query", "machine learning",
		"--pages", "1", "--limit", "1", "--resume-state", resumeStatePath,
	}, &stdout, &stderr)
	if code != 1 || !strings.Contains(stdout.String(), "search_import_provenance_failed") {
		t.Fatalf("search import code=%d stdout=%s stderr=%s, want provenance failure", code, stdout.String(), stderr.String())
	}
	assertFileRestored(t, libraryPath, previousLibrary)
	info, err := os.Stat(libraryPath)
	if err != nil {
		t.Fatalf("stat restored library: %v", err)
	}
	if got := info.Mode().Perm(); got != 0o600 {
		t.Fatalf("restored library mode = %o, want 600", got)
	}
	assertFileRestored(t, resumeStatePath, nil)
}

func TestApprovalCommandsRestoreJSONAfterProvenanceFailure(t *testing.T) {
	for _, tt := range []struct {
		name      string
		path      func(string) string
		original  []byte
		execute   func(io.Writer, io.Writer, globalOptions) int
		errorCode string
	}{
		{
			name:     "privacy review",
			path:     privacyReviewPath,
			original: []byte("{\"schemaVersion\":\"1\",\"issues\":[],\"blocked\":true,\"approved\":false}\n"),
			execute: func(stdout, stderr io.Writer, opts globalOptions) int {
				return executePrivacyApprove([]string{"--reviewer", "Ada", "--reason", "reviewed"}, stdout, stderr, opts)
			},
			errorCode: "privacy_review_provenance_failed",
		},
		{
			name:     "acquisition queue",
			path:     acquisitionQueuePath,
			original: []byte("{\"schemaVersion\":\"1\",\"items\":[{\"id\":\"acq_1\",\"paperTitle\":\"Paper\",\"source\":\"openalex\",\"sourceUrl\":\"https://example.test/paper\",\"expectedLocalPath\":\"papers/paper.pdf\",\"restricted\":true,\"shareable\":false,\"reviewerApprovalRequired\":true,\"reviewerApproved\":false,\"provenance\":\"openalex\"}]}\n"),
			execute: func(stdout, stderr io.Writer, opts globalOptions) int {
				return executeAcquisitionApprove([]string{"acq_1", "--reviewer", "Ada", "--reason", "reviewed"}, stdout, stderr, opts)
			},
			errorCode: "acquisition_provenance_failed",
		},
	} {
		t.Run(tt.name, func(t *testing.T) {
			project := t.TempDir()
			path := tt.path(project)
			if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
				t.Fatalf("create data directory: %v", err)
			}
			if err := os.WriteFile(path, tt.original, 0o640); err != nil {
				t.Fatalf("write original approval JSON: %v", err)
			}
			if err := os.WriteFile(filepath.Join(project, "provenance"), []byte("not a directory"), 0o644); err != nil {
				t.Fatalf("create provenance blocker: %v", err)
			}
			var stdout, stderr bytes.Buffer

			code := tt.execute(&stdout, &stderr, globalOptions{Project: project, JSON: true})
			if code != 1 || !strings.Contains(stdout.String(), tt.errorCode) {
				t.Errorf("approval code = %d, stdout = %s, stderr = %s, want %s", code, stdout.String(), stderr.String(), tt.errorCode)
			}
			got, err := os.ReadFile(path)
			if err != nil {
				t.Fatalf("read approval JSON: %v", err)
			}
			if !bytes.Equal(got, tt.original) {
				t.Errorf("approval JSON changed after provenance failure:\n got: %s\nwant: %s", got, tt.original)
			}
			info, err := os.Stat(path)
			if err != nil {
				t.Fatalf("stat restored approval JSON: %v", err)
			}
			if gotMode := info.Mode().Perm(); gotMode != 0o640 {
				t.Errorf("approval JSON mode after provenance failure = %o, want 640", gotMode)
			}
		})
	}
}

func TestCitationCommandsRestoreOutputsAfterProvenanceFailure(t *testing.T) {
	t.Run("domain map", func(t *testing.T) {
		for _, previous := range [][]byte{nil, []byte("{\"sentinel\":\"domain-map\"}\n")} {
			name := "removes new output"
			if previous != nil {
				name = "restores existing output"
			}
			t.Run(name, func(t *testing.T) {
				project := t.TempDir()
				parsedDir := filepath.Join(project, "parsed")
				if err := os.MkdirAll(parsedDir, 0o755); err != nil {
					t.Fatalf("create parsed directory: %v", err)
				}
				writeParsedFixture(t, filepath.Join(parsedDir, "paper.json"), parsing.ParsedDocument{
					PaperID: "paper-1",
					Title:   "Solar catalyst study",
					Sections: []parsing.Section{{ID: "s1", Passages: []parsing.Passage{{
						ID: "p1", PaperID: "paper-1", SectionID: "s1", Text: "Solar catalyst passage.",
					}}}},
				})
				outPath := filepath.Join(project, "data", "domain-map.json")
				if previous != nil {
					if err := os.MkdirAll(filepath.Dir(outPath), 0o755); err != nil {
						t.Fatalf("create output directory: %v", err)
					}
					if err := os.WriteFile(outPath, previous, 0o640); err != nil {
						t.Fatalf("write prior domain map: %v", err)
					}
				}
				blockProvenance(t, project)
				var stdout, stderr bytes.Buffer

				code := executeCitations([]string{"domain-map", "--parsed-dir", parsedDir, "--out", outPath}, &stdout, &stderr, globalOptions{Project: project, JSON: true})
				if code != 1 || !strings.Contains(stdout.String(), "domain_map_provenance_failed") {
					t.Errorf("domain-map code = %d, stdout = %s, stderr = %s, want provenance failure", code, stdout.String(), stderr.String())
				}
				assertFileRestored(t, outPath, previous)
			})
		}
	})

	t.Run("bibliography import", func(t *testing.T) {
		for _, tt := range []struct {
			name           string
			previousGraph  []byte
			previousReport []byte
		}{
			{name: "restores graph and removes report", previousGraph: []byte("{\"sentinel\":\"graph\"}\n")},
			{name: "removes graph and restores report", previousReport: []byte("{\"sentinel\":\"report\"}\n")},
		} {
			t.Run(tt.name, func(t *testing.T) {
				project := t.TempDir()
				parsedPath := filepath.Join(project, "parsed.json")
				writeParsedFixture(t, parsedPath, parsing.EnrichParsedDocumentModel(parsing.ParsedDocument{
					PaperID:    "paper-1",
					References: []parsing.Reference{{Title: "Reference", DOI: "10.1000/ref"}},
					Sections: []parsing.Section{{ID: "s1", Passages: []parsing.Passage{{
						ID: "p1", PaperID: "paper-1", SectionID: "s1", Text: "Known [1].",
					}}}},
				}))
				graphPath := filepath.Join(project, "data", "citation-graph.json")
				reportPath := filepath.Join(project, "data", "bibliography-import.json")
				if err := os.MkdirAll(filepath.Dir(graphPath), 0o755); err != nil {
					t.Fatalf("create output directory: %v", err)
				}
				if tt.previousGraph != nil {
					if err := os.WriteFile(graphPath, tt.previousGraph, 0o640); err != nil {
						t.Fatalf("write prior graph: %v", err)
					}
				}
				if tt.previousReport != nil {
					if err := os.WriteFile(reportPath, tt.previousReport, 0o640); err != nil {
						t.Fatalf("write prior report: %v", err)
					}
				}
				blockProvenance(t, project)
				var stdout, stderr bytes.Buffer

				code := executeCitations([]string{"import-bibliography", "--parsed", parsedPath, "--out", graphPath, "--report", reportPath}, &stdout, &stderr, globalOptions{Project: project, JSON: true})
				if code != 1 || !strings.Contains(stdout.String(), "citation_bibliography_provenance_failed") {
					t.Errorf("import-bibliography code = %d, stdout = %s, stderr = %s, want provenance failure", code, stdout.String(), stderr.String())
				}
				assertFileRestored(t, graphPath, tt.previousGraph)
				assertFileRestored(t, reportPath, tt.previousReport)
			})
		}
	})
}

func TestCitationExpansionRestoresAllOutputsAfterProvenanceFailure(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		_, _ = w.Write([]byte(`{"data":[{"citedPaper":{"paperId":"R1","title":"Reference"}}]}`))
	}))
	defer server.Close()
	t.Setenv("RFORGE_SEMANTIC_SCHOLAR_URL", server.URL)

	for _, tt := range []struct {
		name            string
		previousGraph   []byte
		previousRun     []byte
		previousLibrary []byte
		hardLinkGraph   bool
	}{
		{
			name:            "restores graph and library and removes run state",
			previousGraph:   []byte("{\"sentinel\":\"graph\"}\n"),
			previousLibrary: []byte("[]\n"),
			hardLinkGraph:   true,
		},
		{
			name:        "removes graph and library and restores run state",
			previousRun: []byte("{\"sentinel\":\"run\"}\n"),
		},
	} {
		t.Run(tt.name, func(t *testing.T) {
			project := t.TempDir()
			graphPath := filepath.Join(project, "citation-graph.json")
			runPath := filepath.Join(project, "semantic-run.json")
			libraryPath := filepath.Join(project, "data", "library.json")
			var outsideGraphPath string
			var outsideGraphBefore os.FileInfo
			for path, previous := range map[string][]byte{
				graphPath:   tt.previousGraph,
				runPath:     tt.previousRun,
				libraryPath: tt.previousLibrary,
			} {
				if previous == nil {
					continue
				}
				if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
					t.Fatalf("create output directory: %v", err)
				}
				if path == graphPath && tt.hardLinkGraph {
					outsideGraphPath = filepath.Join(t.TempDir(), "outside-citation-graph.json")
					if err := os.WriteFile(outsideGraphPath, previous, 0o640); err != nil {
						t.Fatalf("write outside citation graph: %v", err)
					}
					if err := os.Link(outsideGraphPath, graphPath); err != nil {
						t.Skipf("hard links are unavailable: %v", err)
					}
					fixedTime := time.Unix(1_600_000_000, 0)
					if err := os.Chtimes(outsideGraphPath, fixedTime, fixedTime); err != nil {
						t.Fatalf("set outside citation graph timestamps: %v", err)
					}
					info, err := os.Stat(outsideGraphPath)
					if err != nil {
						t.Fatalf("stat outside citation graph before expansion: %v", err)
					}
					outsideGraphBefore = info
					continue
				}
				if err := os.WriteFile(path, previous, 0o640); err != nil {
					t.Fatalf("write prior output %s: %v", path, err)
				}
			}
			blockProvenance(t, project)
			var stdout, stderr bytes.Buffer

			code := executeCitations([]string{
				"expand", "--source", "semantic-scholar", "--paper", "S1", "--direction", "references",
				"--out", graphPath, "--run-state", runPath, "--import-library",
			}, &stdout, &stderr, globalOptions{Project: project, JSON: true})
			if code != 1 || !strings.Contains(stdout.String(), "citation_provenance_failed") {
				t.Errorf("citation expand code = %d, stdout = %s, stderr = %s, want provenance failure", code, stdout.String(), stderr.String())
			}
			assertFileRestored(t, graphPath, tt.previousGraph)
			assertFileRestored(t, runPath, tt.previousRun)
			assertFileRestored(t, libraryPath, tt.previousLibrary)
			if outsideGraphPath != "" {
				outsideGraph, err := os.ReadFile(outsideGraphPath)
				if err != nil {
					t.Fatalf("read outside citation graph: %v", err)
				}
				if !bytes.Equal(outsideGraph, tt.previousGraph) {
					t.Fatalf("outside citation graph changed: got %q, want %q", outsideGraph, tt.previousGraph)
				}
				outsideGraphAfter, err := os.Stat(outsideGraphPath)
				if err != nil {
					t.Fatalf("stat outside citation graph after expansion: %v", err)
				}
				if !outsideGraphAfter.ModTime().Equal(outsideGraphBefore.ModTime()) {
					t.Fatalf("outside citation graph mtime changed: got %s, want %s", outsideGraphAfter.ModTime(), outsideGraphBefore.ModTime())
				}
			}
		})
	}
}

func TestCitationExpansionRejectsMissingProjectBeforeSideEffects(t *testing.T) {
	requests := 0
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		requests++
		_, _ = w.Write([]byte(`{"data":[]}`))
	}))
	defer server.Close()
	t.Setenv("RFORGE_SEMANTIC_SCHOLAR_URL", server.URL)
	outPath := filepath.Join(t.TempDir(), "citation-graph.json")
	var stdout, stderr bytes.Buffer

	code := executeCitations([]string{
		"expand", "--source", "semantic-scholar", "--paper", "S1", "--direction", "references",
		"--out", outPath, "--import-library",
	}, &stdout, &stderr, globalOptions{JSON: true})
	if code != 2 || !strings.Contains(stdout.String(), "missing_project") {
		t.Errorf("citation expand code = %d, stdout = %s, stderr = %s, want missing project", code, stdout.String(), stderr.String())
	}
	if requests != 0 {
		t.Errorf("citation expansion made %d request(s) before rejecting missing project", requests)
	}
	assertFileRestored(t, outPath, nil)
}

func TestParseRestoresArtifactsAfterProvenanceFailure(t *testing.T) {
	project := t.TempDir()
	parsedDir := filepath.Join(project, "parsed")
	if err := os.MkdirAll(parsedDir, 0o755); err != nil {
		t.Fatalf("create parsed directory: %v", err)
	}
	parsedPath := filepath.Join(parsedDir, "paper-1.json")
	manifestPath := filepath.Join(parsedDir, "paper-1.manifest.json")
	parsedBefore := []byte("prior parsed document\n")
	manifestBefore := []byte("prior parser manifest\n")
	if err := os.WriteFile(parsedPath, parsedBefore, 0o600); err != nil {
		t.Fatalf("write prior parsed document: %v", err)
	}
	if err := os.WriteFile(manifestPath, manifestBefore, 0o640); err != nil {
		t.Fatalf("write prior parser manifest: %v", err)
	}
	s2orcPath := filepath.Join(t.TempDir(), "paper.json")
	fixture := `{"title":"S2ORC Fixture","abstract":"Abstract text.","body_text":[{"section":"Intro","text":"Body text."}],"bib_entries":{}}`
	if err := os.WriteFile(s2orcPath, []byte(fixture), 0o644); err != nil {
		t.Fatalf("write S2ORC fixture: %v", err)
	}
	blockProvenance(t, project)
	var stdout, stderr bytes.Buffer

	code := Execute([]string{"--json", "--project", project, "parse", "--paper", "paper-1", "--parser", "s2orc", "--s2orc", s2orcPath}, &stdout, &stderr)
	if code != 1 || !strings.Contains(stdout.String(), "parse_provenance_failed") {
		t.Fatalf("parse code = %d, stdout = %s, stderr = %s, want provenance failure", code, stdout.String(), stderr.String())
	}
	assertFileRestored(t, parsedPath, parsedBefore)
	assertFileRestored(t, manifestPath, manifestBefore)
	for path, wantMode := range map[string]os.FileMode{parsedPath: 0o600, manifestPath: 0o640} {
		info, err := os.Stat(path)
		if err != nil {
			t.Fatalf("stat restored parser artifact %s: %v", path, err)
		}
		if got := info.Mode().Perm(); got != wantMode {
			t.Fatalf("restored parser artifact %s mode = %o, want %o", path, got, wantMode)
		}
	}
}

func TestDuplicateMergeRestoresLibraryAfterProvenanceFailure(t *testing.T) {
	project := t.TempDir()
	libraryPath := filepath.Join(project, "data", "library.json")
	store, err := library.OpenStore(libraryPath)
	if err != nil {
		t.Fatalf("open library: %v", err)
	}
	left, _ := library.NewPaperRecord(library.PaperRecordInput{Title: "A catalyst review", Identifiers: library.Identifiers{DOI: "10.1000/left"}})
	right, _ := library.NewPaperRecord(library.PaperRecordInput{Title: "B catalyst review", Identifiers: library.Identifiers{ArXivID: "2401.00001"}})
	if err := store.Create(left); err != nil {
		t.Fatalf("create left record: %v", err)
	}
	if err := store.Create(right); err != nil {
		t.Fatalf("create right record: %v", err)
	}
	if err := os.Chmod(libraryPath, 0o600); err != nil {
		t.Fatalf("chmod library: %v", err)
	}
	libraryBefore, err := os.ReadFile(libraryPath)
	if err != nil {
		t.Fatalf("read prior library: %v", err)
	}
	blockProvenance(t, project)
	var stdout, stderr bytes.Buffer

	code := Execute([]string{"--json", "--project", project, "duplicate", "merge", "0", "1"}, &stdout, &stderr)
	if code != 1 || !strings.Contains(stdout.String(), "duplicate_provenance_failed") {
		t.Fatalf("duplicate merge code = %d, stdout = %s, stderr = %s, want provenance failure", code, stdout.String(), stderr.String())
	}
	assertFileRestored(t, libraryPath, libraryBefore)
	info, err := os.Stat(libraryPath)
	if err != nil {
		t.Fatalf("stat restored library: %v", err)
	}
	if got := info.Mode().Perm(); got != 0o600 {
		t.Fatalf("restored library mode = %o, want 600", got)
	}
}

func TestDuplicateSplitRestoresLibraryAfterProvenanceFailure(t *testing.T) {
	project := t.TempDir()
	libraryPath := filepath.Join(project, "data", "library.json")
	store, err := library.OpenStore(libraryPath)
	if err != nil {
		t.Fatalf("open library: %v", err)
	}
	left, _ := library.NewPaperRecord(library.PaperRecordInput{Title: "A catalyst review", Identifiers: library.Identifiers{DOI: "10.1000/left"}})
	right, _ := library.NewPaperRecord(library.PaperRecordInput{Title: "B catalyst review", Identifiers: library.Identifiers{ArXivID: "2401.00001"}})
	merged := library.MergeDuplicate(left, right)
	if err := store.Create(merged); err != nil {
		t.Fatalf("create merged record: %v", err)
	}
	splitPath := filepath.Join(t.TempDir(), "split.json")
	if err := library.ExportJSON(splitPath, []library.PaperRecord{left, right}); err != nil {
		t.Fatalf("write split records: %v", err)
	}
	if err := os.Chmod(libraryPath, 0o600); err != nil {
		t.Fatalf("chmod library: %v", err)
	}
	libraryBefore, err := os.ReadFile(libraryPath)
	if err != nil {
		t.Fatalf("read prior library: %v", err)
	}
	blockProvenance(t, project)
	var stdout, stderr bytes.Buffer

	code := Execute([]string{"--json", "--project", project, "duplicate", "split", "0", splitPath}, &stdout, &stderr)
	if code != 1 || !strings.Contains(stdout.String(), "duplicate_provenance_failed") {
		t.Fatalf("duplicate split code = %d, stdout = %s, stderr = %s, want provenance failure", code, stdout.String(), stderr.String())
	}
	assertFileRestored(t, libraryPath, libraryBefore)
	info, err := os.Stat(libraryPath)
	if err != nil {
		t.Fatalf("stat restored library: %v", err)
	}
	if got := info.Mode().Perm(); got != 0o600 {
		t.Fatalf("restored library mode = %o, want 600", got)
	}
}

func TestIdentityDecisionRecordRestoresLogAfterProvenanceFailure(t *testing.T) {
	project := t.TempDir()
	store, err := library.OpenStore(filepath.Join(project, "data", "library.json"))
	if err != nil {
		t.Fatalf("open library: %v", err)
	}
	left, _ := library.NewPaperRecord(library.PaperRecordInput{Title: "Catalyst A", Identifiers: library.Identifiers{DOI: "10.1000/same"}})
	right, _ := library.NewPaperRecord(library.PaperRecordInput{Title: "Catalyst B", Identifiers: library.Identifiers{CrossrefID: "10.1000/same"}})
	if err := store.ReplaceAll([]library.PaperRecord{left, right}); err != nil {
		t.Fatalf("seed library: %v", err)
	}
	logPath := filepath.Join(project, "data", "identity-decisions.jsonl")
	logBefore := []byte{}
	if err := os.WriteFile(logPath, logBefore, 0o600); err != nil {
		t.Fatalf("write prior identity log: %v", err)
	}
	blockProvenance(t, project)
	var stdout, stderr bytes.Buffer

	code := Execute([]string{"--json", "--project", project, "library", "identity-decision", "record", "--action", "merge", "--cluster", "cluster-1", "--reason", "reviewed", "--reviewer", "reviewer-a", "--before-indexes", "0,1", "--after-indexes", "0"}, &stdout, &stderr)
	if code != 1 || !strings.Contains(stdout.String(), "identity_decision_provenance_failed") {
		t.Fatalf("identity decision code = %d, stdout = %s, stderr = %s, want provenance failure", code, stdout.String(), stderr.String())
	}
	assertFileRestored(t, logPath, logBefore)
	info, err := os.Stat(logPath)
	if err != nil {
		t.Fatalf("stat restored identity log: %v", err)
	}
	if got := info.Mode().Perm(); got != 0o600 {
		t.Fatalf("restored identity log mode = %o, want 600", got)
	}
}

func blockProvenance(t *testing.T, project string) {
	t.Helper()
	if err := os.WriteFile(filepath.Join(project, "provenance"), []byte("not a directory"), 0o644); err != nil {
		t.Fatalf("create provenance blocker: %v", err)
	}
}

func assertFileRestored(t *testing.T, path string, previous []byte) {
	t.Helper()
	got, err := os.ReadFile(path)
	if previous == nil {
		if !os.IsNotExist(err) {
			t.Errorf("new output remains after provenance failure: %s: %v", path, err)
		}
		return
	}
	if err != nil {
		t.Fatalf("read restored output %s: %v", path, err)
	}
	if !bytes.Equal(got, previous) {
		t.Errorf("output after provenance failure = %q, want prior contents %q", got, previous)
	}
}

func TestCLIProductionCodeDoesNotIgnoreProvenanceAppendErrors(t *testing.T) {
	ignoredAppend := regexp.MustCompile(`(?m)^\s*(?:_\s*=\s*)?provenance\.Append\(`)
	paths, err := filepath.Glob("*.go")
	if err != nil {
		t.Fatalf("glob CLI sources: %v", err)
	}
	for _, path := range paths {
		if strings.HasSuffix(path, "_test.go") {
			continue
		}
		data, err := os.ReadFile(path)
		if err != nil {
			t.Fatalf("read %s: %v", path, err)
		}
		if ignoredAppend.Match(data) {
			t.Errorf("%s ignores a provenance.Append error", path)
		}
	}
}
