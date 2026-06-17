package cli

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"path/filepath"
	"testing"

	"github.com/TrebuchetDynamics/research-forge/internal/parsing"
)

func TestExecuteParseNormalizeRefsADS(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/v1/search/query" || r.URL.Query().Get("q") != "10.1000/ads" {
			t.Fatalf("unexpected request: %s", r.URL.String())
		}
		_, _ = w.Write([]byte(`{"response":{"docs":[{"bibcode":"2024ApJ...123A...1B","title":["ADS fixture title"],"doi":["10.1000/ADS"],"year":"2024"}]}}`))
	}))
	defer server.Close()
	t.Setenv("RFORGE_ADS_URL", server.URL)
	project := t.TempDir()
	parsed := filepath.Join(project, "parsed.json")
	writeParsedFixture(t, parsed, parsing.ParsedDocument{SchemaVersion: "1", PaperID: "paper-1", References: []parsing.Reference{{Title: "ADS fixture title", DOI: "10.1000/ads", Raw: "raw ref", Confidence: 0.9}}})
	out := filepath.Join(project, "refs.json")
	var stdout, stderr bytes.Buffer
	code := Execute([]string{"--json", "--project", project, "parse", "normalize-refs", "--parsed", parsed, "--source", "ads", "--out", out}, &stdout, &stderr)
	if code != 0 {
		t.Fatalf("code=%d stderr=%s stdout=%s", code, stderr.String(), stdout.String())
	}
	var env struct {
		Data struct {
			Report parsing.ReferenceNormalizationReport `json:"referenceNormalization"`
		} `json:"data"`
	}
	if err := json.Unmarshal(stdout.Bytes(), &env); err != nil {
		t.Fatalf("decode: %v\n%s", err, stdout.String())
	}
	match := env.Data.Report.Matches[0]
	if env.Data.Report.Connector != "ads" || match.SourceID != "2024ApJ...123A...1B" || match.Raw != "raw ref" || match.ParserConfidence != 0.9 || match.ResponseRawRef == "" {
		t.Fatalf("report = %#v", env.Data.Report)
	}
}
