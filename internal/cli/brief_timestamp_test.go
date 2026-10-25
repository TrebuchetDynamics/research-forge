package cli

import (
	"bytes"
	"encoding/json"
	"os"
	"path/filepath"
	"reflect"
	"strings"
	"testing"

	"github.com/TrebuchetDynamics/research-forge/internal/research"
)

// Both CLI formats must accept short dates and persist the service's Markdown.
func TestBriefShortTimestampCLI(t *testing.T) {
	for _, format := range []string{"text", "json"} {
		t.Run(format, func(t *testing.T) {
			dir := t.TempDir()
			writeFixtureSearchDir(t, dir, "scaling", []map[string]any{{"title": "Paper", "year": 2026}})
			path := filepath.Join(dir, "manifest.json")
			manifest, err := os.ReadFile(path)
			if err != nil {
				t.Fatal(err)
			}
			manifest = bytes.ReplaceAll(manifest, []byte("2026-01-01T00:00:00Z"), []byte("x"))
			if err := os.WriteFile(path, manifest, 0o644); err != nil {
				t.Fatal(err)
			}
			out := filepath.Join(t.TempDir(), "brief.md")
			args := []string{"brief", dir, "--out", out}
			if format == "json" {
				args = append(args, "--json")
			}
			var stdout, stderr bytes.Buffer
			if code := Execute(args, &stdout, &stderr); code != 0 || stderr.Len() != 0 {
				t.Fatalf("exit %d: stdout=%s stderr=%s", code, &stdout, &stderr)
			}
			want, err := (research.Service{}).Brief(research.BriefOptions{SearchDir: dir})
			if err != nil {
				t.Fatal(err)
			}
			saved, err := os.ReadFile(out)
			if err != nil || string(saved) != want.Markdown || !strings.Contains(string(saved), "- **Searched:** x\n") {
				t.Fatalf("saved brief differs from service or lost short date: %v", err)
			}
			if format == "text" {
				if stdout.String() != want.Markdown+"\nwrote brief to "+out+"\n" {
					t.Fatalf("text output differs from service: %s", &stdout)
				}
			} else {
				var got struct {
					OK   bool            `json:"ok"`
					Data json.RawMessage `json:"data"`
				}
				if err := json.Unmarshal(stdout.Bytes(), &got); err != nil || !got.OK {
					t.Fatalf("JSON success output = %s, %v", &stdout, err)
				}
				want.Out = out
				expected, err := json.Marshal(want)
				if err != nil {
					t.Fatal(err)
				}
				var actualData, expectedData map[string]any
				if err := json.Unmarshal(got.Data, &actualData); err != nil {
					t.Fatal(err)
				}
				if err := json.Unmarshal(expected, &expectedData); err != nil {
					t.Fatal(err)
				}
				if !reflect.DeepEqual(actualData, expectedData) {
					t.Fatalf("JSON result = %s, want %s", got.Data, expected)
				}
			}
		})
	}
}
