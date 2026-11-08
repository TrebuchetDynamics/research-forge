package cli

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"os"
	"path/filepath"
	"reflect"
	"strconv"
	"strings"
	"testing"
)

func TestSearchRefreshManifestPlan(t *testing.T) {
	fixture := func(t *testing.T, manifest string, sparse bool) string {
		t.Helper()
		dir := t.TempDir()
		files := []struct {
			name, data string
			mode       os.FileMode
		}{
			{"manifest.json", manifest, 0o640},
			{"results-deduped.jsonl", "{\"Identifiers\":{\"DOI\":\"10.0000/a\"}}\n{\"Identifiers\":{\"DOI\":\"10.0000/b\"}}\n", 0o600},
			{"batch-state.json", "prior checkpoint\n", 0o602},
			{"notes.txt", "unrelated notes\n", 0o606},
			{"results.jsonl", "prior source results\n", 0o604},
			{"failures.jsonl", "prior failures\n", 0o660},
			{"results.md", "# Prior markdown\n", 0o644},
			{"raw/search-openalex-001-first.txt", "prior raw response\n", 0o620},
		}
		for i, file := range files {
			if sparse && i >= 4 {
				continue
			}
			path := filepath.Join(dir, file.name)
			if err := os.MkdirAll(filepath.Dir(path), 0o750); err != nil {
				t.Fatal(err)
			}
			if err := os.WriteFile(path, []byte(file.data), file.mode); err != nil {
				t.Fatal(err)
			}
			if err := os.Chmod(path, file.mode); err != nil {
				t.Fatal(err)
			}
		}
		return dir
	}
	var requests []url.Values
	t.Setenv("RFORGE_OPENALEX_URL", "https://refresh.invalid")
	t.Setenv("RFORGE_SOURCE_REQUEST_DELAY", "0s")
	transport := http.DefaultTransport
	t.Cleanup(func() { http.DefaultTransport = transport })
	http.DefaultTransport = rankFixtureTransport(func(r *http.Request) (*http.Response, error) {
		requests = append(requests, r.URL.Query())
		if r.URL.Host != "refresh.invalid" {
			t.Errorf("unexpected connector request: %s", r.URL)
		}
		body := `{"results":[{"id":"https://openalex.org/Wb","doi":"https://doi.org/10.0000/b","title":"B"},{"id":"https://openalex.org/Wc","doi":"https://doi.org/10.0000/c","title":"C"}]}`
		return &http.Response{StatusCode: http.StatusOK, Header: make(http.Header), Body: io.NopCloser(strings.NewReader(body)), Request: r}, nil
	})
	type rejection struct {
		name, input, field, code, cause, prior string
		index                                  int
	}
	cases := []rejection{
		{name: "top-null", input: "null", field: "queries"},
		{name: "empty-object", input: "{}", field: "queries"},
		{name: "missing-manifest", code: "search_refresh_no_manifest", cause: "no manifest.json"},
		{name: "directory-manifest", code: "search_refresh_read_manifest", cause: "is a directory"},
		{name: "malformed-json", input: "{broken}", code: "search_refresh_parse_manifest", cause: "invalid character"},
		{name: "limit-type", input: `{"queries":["first"],"sources":["openalex"],"limit":"bad"}`, code: "search_refresh_parse_manifest", cause: "limit"},
	}
	for _, field := range []string{"queries", "sources"} {
		valid, other := "first", `"sources":["openalex"]`
		if field == "sources" {
			valid, other = "openalex", `"queries":["first"]`
		}
		for _, value := range []struct {
			name, json string
			index      int
		}{
			{"omitted", "", 0},
			{"null", "null", 0},
			{"empty", "[]", 0},
			{"blank", `[""]`, 1},
			{"ascii-whitespace", `[" \t\r\n"]`, 1},
			{"unicode-whitespace", `["\u00a0\u2003\u2028\u3000"]`, 1},
			{"null-element", `[null]`, 1},
			{"blank-after-prefix", fmt.Sprintf(`[%q," \t\u2003"]`, valid), 2},
			{"null-after-prefix", fmt.Sprintf(`[%q,null]`, valid), 2},
		} {
			input := "{" + other
			if value.json != "" {
				input += fmt.Sprintf(",%q:%s", field, value.json)
			}
			cases = append(cases, rejection{name: field + "/" + value.name, input: input + "}", field: field, index: value.index})
		}
		for _, value := range []string{`"not-an-array"`, `{}`, `42`, `true`, `[42]`} {
			cases = append(cases, rejection{name: field + "/type-" + value, input: fmt.Sprintf(`{%s,%q:%s}`, other, field, value), code: "search_refresh_parse_manifest", cause: field})
		}
		for _, prior := range []string{"missing", "malformed"} {
			cases = append(cases, rejection{name: field + "/before-" + prior + "-results", input: fmt.Sprintf(`{%s,%q:[]}`, other, field), field: field, prior: prior})
		}
	}
	for _, tc := range cases {
		for _, format := range []string{"text", "json"} {
			for _, mode := range []string{"dry-run", "normal"} {
				for _, layout := range []string{"full", "sparse"} {
					t.Run(strings.Join([]string{tc.name, format, mode, layout}, "/"), func(t *testing.T) {
						dir := fixture(t, tc.input, layout == "sparse")
						manifestPath := filepath.Join(dir, "manifest.json")
						if tc.name == "missing-manifest" || tc.name == "directory-manifest" {
							if err := os.Remove(manifestPath); err != nil {
								t.Fatal(err)
							}
							if tc.name == "directory-manifest" {
								if err := os.Mkdir(manifestPath, 0o750); err != nil {
									t.Fatal(err)
								}
							}
						}
						priorPath := filepath.Join(dir, "results-deduped.jsonl")
						if tc.prior == "missing" {
							if err := os.Remove(priorPath); err != nil {
								t.Fatal(err)
							}
						} else if tc.prior == "malformed" {
							if err := os.WriteFile(priorPath, []byte("{broken}\n"), 0o600); err != nil {
								t.Fatal(err)
							}
						}
						before := snapshotRefreshArtifacts(t, dir)
						requests = nil
						args := []string{"search", "refresh", "--dir", dir}
						if format == "json" {
							args = append([]string{"--json"}, args...)
						}
						if mode == "dry-run" {
							args = append(args, "--dry-run")
						}
						stdout, stderr := new(bytes.Buffer), new(bytes.Buffer)
						code := Execute(args, stdout, stderr)
						if code != 1 || len(requests) != 0 {
							t.Errorf("exit/requests = %d/%d, want 1/0; stdout=%s stderr=%s", code, len(requests), stdout, stderr)
						}
						message := stderr.String()
						if format == "json" {
							var envelope struct {
								OK    bool                           `json:"ok"`
								Data  json.RawMessage                `json:"data"`
								Error struct{ Code, Message string } `json:"error"`
							}
							if err := json.Unmarshal(stdout.Bytes(), &envelope); err != nil {
								t.Fatalf("invalid JSON envelope: %s (%v)", stdout, err)
							}
							wantCode := tc.code
							if wantCode == "" {
								wantCode = "search_refresh_invalid_manifest"
							}
							if envelope.OK || len(envelope.Data) != 0 || envelope.Error.Code != wantCode || stderr.Len() != 0 {
								t.Errorf("lost manifest error or published summary: stdout=%s stderr=%s", stdout, stderr)
							}
							message = envelope.Error.Message
						} else if stdout.Len() != 0 {
							t.Errorf("published refresh summary: %s", stdout)
						}
						if tc.code == "" {
							field := tc.field
							if tc.index != 0 {
								field += fmt.Sprintf("[%d]", tc.index)
							}
							if !strings.Contains(message, manifestPath) || !strings.Contains(message, field) {
								t.Errorf("missing manifest path/field/index %q: %q", field, message)
							}
						} else if !strings.Contains(message, tc.cause) {
							t.Errorf("lost manifest read/parse cause %q: %q", tc.cause, message)
						}
						if after := snapshotRefreshArtifacts(t, dir); !reflect.DeepEqual(after, before) {
							t.Error("changed prior bytes/modes, checkpoint or optional/staged artifacts")
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
	for _, tc := range []struct {
		name, input string
		queries     []string
		sources     []string
		limit       int
	}{
		{"minimal", `{"queries":["first"],"sources":["openalex"]}`, []string{"first"}, []string{"openalex"}, 0},
		{"optional-and-unknown", `{"queries":["first"],"sources":["openalex"],"schemaVersion":"legacy","createdAt":"not-a-date","results":-3,"deduped":99,"failures":8,"limit":-1,"future":{"nested":[null,42]}}`, []string{"first"}, []string{"openalex"}, -1},
		{"unchanged-values-order-duplicates", `{"queries":[" first \u2003","second"," first \u2003"],"sources":["openalex"," openalex ","openalex"],"limit":7}`, []string{" first \u2003", "second", " first \u2003"}, []string{"openalex", " openalex ", "openalex"}, 7},
	} {
		for _, format := range []string{"text", "json"} {
			for _, mode := range []string{"dry-run", "normal"} {
				t.Run(strings.Join([]string{"valid", tc.name, format, mode}, "/"), func(t *testing.T) {
					dir := fixture(t, tc.input, true)
					before := snapshotRefreshArtifacts(t, dir)
					requests = nil
					args := []string{"search", "refresh", "--dir", dir}
					if format == "json" {
						args = append([]string{"--json"}, args...)
					}
					if mode == "dry-run" {
						args = append(args, "--dry-run")
					}
					stdout, stderr := new(bytes.Buffer), new(bytes.Buffer)
					if code := Execute(args, stdout, stderr); code != 0 || stderr.Len() != 0 {
						t.Fatalf("valid plan exit=%d stdout=%s stderr=%s", code, stdout, stderr)
					}
					var envelope struct {
						OK   bool            `json:"ok"`
						Data json.RawMessage `json:"data"`
					}
					if format == "json" {
						if err := json.Unmarshal(stdout.Bytes(), &envelope); err != nil || !envelope.OK {
							t.Fatalf("invalid success envelope: %s (%v)", stdout, err)
						}
					}
					var manifest searchBatchManifest
					if mode == "dry-run" {
						if len(requests) != 0 || !reflect.DeepEqual(snapshotRefreshArtifacts(t, dir), before) {
							t.Error("preview dispatched requests or changed artifacts")
						}
						if _, err := os.Stat(filepath.Join(dir, "jobs")); !os.IsNotExist(err) {
							t.Errorf("dry-run created lease state: %v", err)
						}
						if format == "json" {
							if err := json.Unmarshal(envelope.Data, &manifest); err != nil {
								t.Fatal(err)
							}
						} else {
							for _, value := range append(append([]string{}, tc.queries...), tc.sources...) {
								if !strings.Contains(stdout.String(), value) {
									t.Errorf("preview lost original value %q: %s", value, stdout)
								}
							}
							return
						}
					} else {
						var wantQueries []string
						for _, query := range tc.queries {
							for _, source := range tc.sources {
								if source == "openalex" {
									wantQueries = append(wantQueries, query)
								}
							}
						}
						var gotQueries []string
						limit := tc.limit
						if limit <= 0 {
							limit = 25
						}
						for _, request := range requests {
							gotQueries = append(gotQueries, request.Get("search"))
							if request.Get("per-page") != strconv.Itoa(limit) {
								t.Errorf("changed connector limit/default: %v", request)
							}
						}
						if !reflect.DeepEqual(gotQueries, wantQueries) {
							t.Errorf("changed query values/order/duplicates: %q, want %q", gotQueries, wantQueries)
						}
						if err := readJSONFile(filepath.Join(dir, "manifest.json"), &manifest); err != nil {
							t.Fatal(err)
						}
						if format == "json" {
							var delta struct {
								New, Gone []string
								Unchanged int
							}
							if err := json.Unmarshal(envelope.Data, &delta); err != nil {
								t.Fatal(err)
							}
							if !reflect.DeepEqual(delta.New, []string{"10.0000/c"}) || !reflect.DeepEqual(delta.Gone, []string{"10.0000/a"}) || delta.Unchanged != 1 {
								t.Errorf("changed DOI delta: %+v", delta)
							}
						} else if !strings.Contains(stdout.String(), "1 new, 1 unchanged, 1 gone") {
							t.Errorf("changed DOI delta: %s", stdout)
						}
					}
					if !reflect.DeepEqual(manifest.Queries, tc.queries) || !reflect.DeepEqual(manifest.Sources, tc.sources) || manifest.Limit != tc.limit {
						t.Errorf("changed original plan values: %+v", manifest)
					}
				})
			}
		}
	}
}
