package research_test

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/TrebuchetDynamics/research-forge/internal/research"
)

// Short optional timestamps must not panic or be normalized in the input.
func TestBriefTimestampBounds(t *testing.T) {
	for _, tc := range []struct{ name, createdAt, want string }{
		{"empty", "", ""},
		{"one_byte", "x", "- **Searched:** x\n"},
		{"nine_bytes", "123456789", "- **Searched:** 123456789\n"},
		{"ten_bytes", "2026-01-01", "- **Searched:** 2026-01-01\n"},
		{"rfc3339", "2026-01-01T12:34:56Z", "- **Searched:** 2026-01-01\n"},
	} {
		t.Run(tc.name, func(t *testing.T) {
			dir := session(t, t.TempDir(), "search", `{"title":"Paper","year":2026}`)
			manifest := fmt.Sprintf(`{"queries":["scaling"],"sources":["openalex"],"results":1,"deduped":1,"createdAt":%q}`, tc.createdAt)
			if err := os.WriteFile(filepath.Join(dir, "manifest.json"), []byte(manifest), 0o644); err != nil {
				t.Fatal(err)
			}
			inputs := map[string][]byte{}
			for _, name := range []string{"manifest.json", "results-deduped.jsonl"} {
				data, err := os.ReadFile(filepath.Join(dir, name))
				if err != nil {
					t.Fatal(err)
				}
				inputs[name] = data
			}
			out := filepath.Join(t.TempDir(), "brief.md")
			got, err := (research.Service{}).Brief(research.BriefOptions{SearchDir: dir, Out: out})
			if err != nil {
				t.Fatal(err)
			}
			var searched string
			for _, line := range strings.SplitAfter(got.Markdown, "\n") {
				if strings.HasPrefix(line, "- **Searched:**") {
					searched += line
				}
			}
			if searched != tc.want {
				t.Errorf("searched line = %q, want %q", searched, tc.want)
			}
			saved, err := os.ReadFile(out)
			if err != nil || string(saved) != got.Markdown || got.BriefSize != len(saved) {
				t.Fatalf("saved brief differs from result: %v", err)
			}
			for name, before := range inputs {
				after, err := os.ReadFile(filepath.Join(dir, name))
				if err != nil || string(after) != string(before) {
					t.Errorf("input %s changed: %v", name, err)
				}
			}
		})
	}
}
