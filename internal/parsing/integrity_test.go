package parsing

import (
	"context"
	"crypto/sha256"
	"encoding/json"
	"fmt"
	"strings"
	"testing"
)

// Read hashes through the persisted JSON seam so legacy documents without the
// additive field decode as missing, rather than looking integrity-checked.
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

func TestEnrichParsedDocumentHashesTextStream(t *testing.T) {
	doc := ParsedDocument{
		PaperID: "paper-1", Title: "Title", Abstract: "Abstract",
		Sections: []Section{{ID: "s1", Title: "Results", Passages: []Passage{
			{ID: "p1", Text: "Café [1]."}, {ID: "p2", Text: "Control."},
		}}},
		References: []Reference{{Title: "Reference"}},
	}
	for _, tc := range []struct {
		name string
		edit func()
		text string
	}{
		{"original", func() {}, "Title\nAbstract\nResults\nCafé [1].\nControl.\nReference"},
		{"metadata only", func() { doc.ParserVersion = "v2"; doc.Warnings = []string{"warning"} }, "Title\nAbstract\nResults\nCafé [1].\nControl.\nReference"},
		{"title edit", func() { doc.Title = "Other" }, "Other\nAbstract\nResults\nCafé [1].\nControl.\nReference"},
		{"reference edit", func() { doc.References[0].Title = "Revised" }, "Other\nAbstract\nResults\nCafé [1].\nControl.\nRevised"},
	} {
		t.Run(tc.name, func(t *testing.T) {
			tc.edit()
			doc = EnrichParsedDocumentModel(doc)
			want := fmt.Sprintf("sha256:%x", sha256.Sum256([]byte(tc.text)))
			if got := storedContentHash(t, doc); got != want {
				t.Fatalf("document hash = %q, want %q", got, want)
			}
		})
	}
}

func TestS2ORCReparsePreservesPassageIntegrity(t *testing.T) {
	fixture := `{"title":"Title","body_text":[{"section":"Results","text":"Café [1]."},{"section":"Results","text":"Control."}]}`
	parse := func(input string, version string) ParsedDocument {
		t.Helper()
		doc, err := (S2ORCJSONParser{Version: version}).Parse(context.Background(), []byte(input), ParseOptions{PaperID: "paper-1"})
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
