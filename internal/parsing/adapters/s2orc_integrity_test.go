package adapters

import (
	"context"
	"crypto/sha256"
	"encoding/json"
	"fmt"
	"strings"
	"testing"

	"github.com/TrebuchetDynamics/research-forge/internal/parsing/model"
)

// storedContentHash reads the hash through the persisted JSON seam so legacy
// documents without the additive field decode as missing.
func storedContentHash(t *testing.T, value any) string {
	t.Helper()
	data, err := json.Marshal(value)
	if err != nil {
		t.Fatal(err)
	}
	var stored struct{ ContentHash string }
	if err := json.Unmarshal(data, &stored); err != nil {
		t.Fatal(err)
	}
	return stored.ContentHash
}

func TestS2ORCReparsePreservesPassageIntegrity(t *testing.T) {
	fixture := `{"title":"Title","body_text":[{"section":"Results","text":"Café [1]."},{"section":"Results","text":"Control."}]}`
	parse := func(input string, version string) model.ParsedDocument {
		t.Helper()
		doc, err := (S2ORCJSONParser{Version: version}).Parse(context.Background(), []byte(input), model.ParseOptions{PaperID: "paper-1"})
		if err != nil {
			t.Fatal(err)
		}
		return doc
	}
	original := parse(fixture, "v1")
	for _, tc := range []struct {
		name, input string
		changed     bool
	}{
		{"unchanged", fixture, false},
		{"normalized whitespace", strings.Replace(fixture, "Café [1].", "  Café   [1].  ", 1), false},
		{"one character edit", strings.Replace(fixture, "Café", "Cafe", 1), true},
	} {
		t.Run(tc.name, func(t *testing.T) {
			reparsed := parse(tc.input, "v2")
			if same := storedContentHash(t, original) == storedContentHash(t, reparsed); same == tc.changed {
				t.Fatalf("document hash unchanged = %v, text changed = %v", same, tc.changed)
			}
			for i, section := range original.Sections {
				before, after := section.Passages[0], reparsed.Sections[i].Passages[0]
				if before.ID == "" || before.ID != after.ID {
					t.Fatalf("passage identity changed: %q -> %q", before.ID, after.ID)
				}
				want := fmt.Sprintf("sha256:%x", sha256.Sum256([]byte(after.Text)))
				if got := storedContentHash(t, after); got != want {
					t.Fatalf("passage hash = %q, want %q", got, want)
				}
				if same := storedContentHash(t, before) == storedContentHash(t, after); same == (tc.changed && i == 0) {
					t.Fatalf("passage %d hash locality violated", i)
				}
			}
		})
	}
}
