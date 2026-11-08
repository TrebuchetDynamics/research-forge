package cli

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"reflect"
	"strings"
	"testing"
)

func TestSearchRefreshPriorInput(t *testing.T) {
	cases := []struct {
		name, input, cause string
		line               int
		prior              []string
	}{
		{name: "missing", cause: "no such file"},
		{name: "directory", cause: "is a directory"},
		{name: "malformed-tail", input: "{\"Identifiers\":{\"DOI\":\"10.0000/a\"}}\n\n \t\n{broken}\n", cause: "invalid character", line: 4},
		{name: "null", input: "null\n", cause: "expected a JSON object", line: 1},
		{name: "array", input: "[]\n", cause: "cannot unmarshal array", line: 1},
		{name: "string", input: "\"not a record\"\n", cause: "cannot unmarshal string", line: 1},
		{name: "number", input: "42\n", cause: "cannot unmarshal number", line: 1},
		{name: "boolean", input: "true\n", cause: "cannot unmarshal bool", line: 1},
		{name: "identifiers-type", input: `{"Identifiers":[]}`, cause: "cannot unmarshal array", line: 1},
		{name: "doi-type", input: `{"Identifiers":{"DOI":123}}`, cause: "Identifiers.DOI", line: 1},
		{name: "title-type", input: `{"Title":false}`, cause: "PaperRecord.Title", line: 1},
		{name: "year-type", input: `{"Year":"2026"}`, cause: "PaperRecord.Year", line: 1},
		{name: "authors-type", input: `{"Authors":{}}`, cause: "PaperRecord.Authors", line: 1},
		{name: "urls-type", input: `{"URLs":"https://example.invalid"}`, cause: "PaperRecord.URLs", line: 1},
		{name: "source-metadata-type", input: `{"SourceRefs":[{"Metadata":{"key":42}}]}`, cause: "SourceRefs.Metadata", line: 1},
		{name: "empty"},
		{name: "whitespace", input: "\n \t\r\n\n"},
		{name: "no-doi", input: "{}\n{\"Title\":\"No identifiers\"}\n{\"Identifiers\":{\"ArXivID\":\"2601.00001\"}}\n"},
		{name: "legacy-and-unknown", input: `{"Identifiers":{"DOI":" Legacy-ID "},"unknown":{"nested":[null,42]}}`, prior: []string{"legacy-id"}},
		{name: "normalized-set", input: "{\"Identifiers\":{\"DOI\":\" 10.0000/A \"}}\n{\"Identifiers\":{\"DOI\":\"10.0000/a\"}}\n{\"Identifiers\":{\"DOI\":\"10.0000/B\"}}\n", prior: []string{"10.0000/a", "10.0000/b"}},
	}
	for _, tc := range cases {
		for _, format := range []string{"text", "json"} {
			for _, mode := range []string{"dry-run", "normal"} {
				fixtures := []string{"full"}
				if tc.cause != "" {
					fixtures = append(fixtures, "sparse")
				}
				for _, fixture := range fixtures {
					t.Run(strings.Join([]string{tc.name, format, mode, fixture}, "/"), func(t *testing.T) {
						dir := t.TempDir()
						writePriorTopic(t, dir, []string{"first"}, []string{"openalex"}, nil)
						path := filepath.Join(dir, "results-deduped.jsonl")
						if tc.name == "missing" || tc.name == "directory" {
							if err := os.Remove(path); err != nil {
								t.Fatal(err)
							}
						} else if err := os.WriteFile(path, []byte(tc.input), 0o600); err != nil {
							t.Fatal(err)
						}
						if tc.name == "directory" {
							if err := os.Mkdir(path, 0o750); err != nil {
								t.Fatal(err)
							}
							if err := os.WriteFile(filepath.Join(path, "keep.txt"), []byte("directory contents\n"), 0o640); err != nil {
								t.Fatal(err)
							}
						}
						files := []struct {
							name, data string
							mode       os.FileMode
						}{
							{"batch-state.json", "prior checkpoint\n", 0o600},
							{"notes.txt", "unrelated notes\n", 0o640},
							{"results.jsonl", "prior source results\n", 0o604},
							{"failures.jsonl", "prior failures\n", 0o660},
							{"results.md", "# Prior markdown\n", 0o644},
							{"raw/search-openalex-001-first.txt", "prior raw response\n", 0o600},
						}
						for i, file := range files {
							if fixture == "sparse" && i >= 2 {
								continue
							}
							if err := os.MkdirAll(filepath.Dir(filepath.Join(dir, file.name)), 0o750); err != nil {
								t.Fatal(err)
							}
							if err := os.WriteFile(filepath.Join(dir, file.name), []byte(file.data), file.mode); err != nil {
								t.Fatal(err)
							}
						}
						before := snapshotRefreshArtifacts(t, dir)
						t.Setenv("RFORGE_OPENALEX_URL", "https://refresh.invalid")
						t.Setenv("RFORGE_SOURCE_REQUEST_DELAY", "0s")
						calls := 0
						transport := http.DefaultTransport
						t.Cleanup(func() { http.DefaultTransport = transport })
						http.DefaultTransport = rankFixtureTransport(func(r *http.Request) (*http.Response, error) {
							calls++
							if r.URL.Host != "refresh.invalid" {
								t.Fatalf("unexpected connector request: %s", r.URL)
							}
							body := `{"results":[{"id":"https://openalex.org/Wb","doi":"https://doi.org/10.0000/b","title":"B"},{"id":"https://openalex.org/Wc","doi":"https://doi.org/10.0000/c","title":"C"}]}`
							return &http.Response{StatusCode: http.StatusOK, Header: make(http.Header), Body: io.NopCloser(strings.NewReader(body)), Request: r}, nil
						})
						args := []string{"search", "refresh", "--dir", dir}
						if format == "json" {
							args = append([]string{"--json"}, args...)
						}
						if mode == "dry-run" {
							args = append(args, "--dry-run")
						}
						stdout, stderr := new(bytes.Buffer), new(bytes.Buffer)
						code := Execute(args, stdout, stderr)
						var envelope struct {
							OK    bool                           `json:"ok"`
							Data  json.RawMessage                `json:"data"`
							Error struct{ Code, Message string } `json:"error"`
						}
						if format == "json" {
							if err := json.Unmarshal(stdout.Bytes(), &envelope); err != nil {
								t.Fatalf("invalid JSON envelope: %s (%v)", stdout, err)
							}
							if stderr.Len() != 0 {
								t.Errorf("JSON wrote stderr: %s", stderr)
							}
						}
						if tc.cause != "" {
							if code != 1 || calls != 0 {
								t.Errorf("exit/calls = %d/%d, want 1/0; stdout=%s stderr=%s", code, calls, stdout, stderr)
							}
							message := stderr.String()
							if format == "json" {
								message = envelope.Error.Message
								if envelope.OK || len(envelope.Data) != 0 || envelope.Error.Code != "search_refresh_read_results" {
									t.Errorf("lost input error or published delta: %s", stdout)
								}
							} else if stdout.Len() != 0 {
								t.Errorf("published refresh summary: %s", stdout)
							}
							if !strings.Contains(message, path) || !strings.Contains(message, tc.cause) || (tc.line != 0 && !strings.Contains(message, fmt.Sprintf("line %d:", tc.line))) {
								t.Errorf("missing filename/cause/physical line: %q", message)
							}
						} else {
							wantCalls := 1
							if mode == "dry-run" {
								wantCalls = 0
							}
							if code != 0 || calls != wantCalls || stderr.Len() != 0 || (format == "json" && !envelope.OK) {
								t.Fatalf("valid input exit/calls = %d/%d; stdout=%s stderr=%s", code, calls, stdout, stderr)
							}
							if mode == "dry-run" {
								if format == "json" {
									var data struct {
										PriorRecords int
										PriorDOIs    map[string]struct{}
									}
									if err := json.Unmarshal(envelope.Data, &data); err != nil {
										t.Fatal(err)
									}
									want := map[string]struct{}{}
									for _, doi := range tc.prior {
										want[doi] = struct{}{}
									}
									if data.PriorRecords != len(want) || !reflect.DeepEqual(data.PriorDOIs, want) {
										t.Errorf("prior DOI set = %+v, want %v", data, want)
									}
								} else if !strings.Contains(stdout.String(), fmt.Sprintf("prior records: %d", len(tc.prior))) {
									t.Errorf("incorrect preview count: %s", stdout)
								}
							} else {
								wantNew, wantGone, wantUnchanged := []string{"10.0000/b", "10.0000/c"}, tc.prior, 0
								if tc.name == "normalized-set" {
									wantNew, wantGone, wantUnchanged = []string{"10.0000/c"}, []string{"10.0000/a"}, 1
								}
								if format == "json" {
									var data struct {
										New, Gone []string
										Unchanged int
									}
									if err := json.Unmarshal(envelope.Data, &data); err != nil {
										t.Fatal(err)
									}
									if strings.Join(data.New, ",") != strings.Join(wantNew, ",") || strings.Join(data.Gone, ",") != strings.Join(wantGone, ",") || data.Unchanged != wantUnchanged {
										t.Errorf("delta = %+v, want %v/%v/%d", data, wantNew, wantGone, wantUnchanged)
									}
								} else if !strings.Contains(stdout.String(), fmt.Sprintf("%d new, %d unchanged, %d gone", len(wantNew), wantUnchanged, len(wantGone))) {
									t.Errorf("incorrect delta: %s", stdout)
								}
								if _, err := os.Stat(filepath.Join(dir, searchBatchStateFile)); !os.IsNotExist(err) {
									t.Errorf("successful refresh retained checkpoint: %v", err)
								}
							}
						}
						if tc.cause != "" || mode == "dry-run" {
							if after := snapshotRefreshArtifacts(t, dir); !reflect.DeepEqual(after, before) {
								t.Error("refresh changed prior bytes/modes or published staged artifacts")
							}
						}
						if mode == "dry-run" {
							if _, err := os.Stat(filepath.Join(dir, "jobs")); !os.IsNotExist(err) {
								t.Errorf("dry-run created lease state: %v", err)
							}
						}
					})
				}
			}
		}
	}
}
