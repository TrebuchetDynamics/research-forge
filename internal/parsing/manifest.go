package parsing

import (
	"crypto/sha256"
	"encoding/hex"
)

// ParserRunManifest records auditable parser-run provenance and layer counts.
type ParserRunManifest struct {
	SchemaVersion string   `json:"schemaVersion"`
	PaperID       string   `json:"paperId"`
	ParserName    string   `json:"parserName"`
	ParserVersion string   `json:"parserVersion"`
	InputChecksum string   `json:"inputChecksum"`
	ParsedPath    string   `json:"parsedPath"`
	Sections      int      `json:"sections"`
	Passages      int      `json:"passages"`
	References    int      `json:"references"`
	Warnings      []string `json:"warnings,omitempty"`
}

// NewParserRunManifest summarizes one parser output for reproducibility and comparison.
func NewParserRunManifest(doc ParsedDocument, input []byte, parsedPath string) ParserRunManifest {
	passages := 0
	for _, section := range doc.Sections {
		passages += len(section.Passages)
	}
	sum := sha256.Sum256(input)
	return ParserRunManifest{SchemaVersion: "1", PaperID: doc.PaperID, ParserName: doc.ParserName, ParserVersion: doc.ParserVersion, InputChecksum: hex.EncodeToString(sum[:]), ParsedPath: parsedPath, Sections: len(doc.Sections), Passages: passages, References: len(doc.References), Warnings: append([]string{}, doc.Warnings...)}
}
