package parsing

import "testing"

func TestNewParserRunManifestCountsLayersAndChecksumsInput(t *testing.T) {
	doc := ParsedDocument{PaperID: "paper-1", ParserName: "tex", ParserVersion: "builtin", Sections: []Section{{Passages: []Passage{{ID: "p1"}, {ID: "p2"}}}}, References: []Reference{{Title: "Ref"}}, Warnings: []string{"missing title"}}

	manifest := NewParserRunManifest(doc, []byte("input"), "parsed/paper-1.json")

	if manifest.SchemaVersion != "1" || manifest.PaperID != "paper-1" || manifest.ParserName != "tex" || manifest.Sections != 1 || manifest.Passages != 2 || manifest.References != 1 {
		t.Fatalf("manifest = %#v", manifest)
	}
	if manifest.InputChecksum == "" || manifest.ParsedPath != "parsed/paper-1.json" || len(manifest.Warnings) != 1 {
		t.Fatalf("manifest details = %#v", manifest)
	}
}
