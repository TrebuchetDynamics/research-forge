package report

import (
	"strings"

	"github.com/TrebuchetDynamics/research-forge/internal/parsing"
)

func BuildPassageProvenanceFromParsedDocuments(docs []parsing.ParsedDocument, sourcePath string) []PassageProvenance {
	links := []PassageProvenance{}
	for _, doc := range docs {
		for _, section := range doc.Sections {
			for _, passage := range section.Passages {
				if strings.TrimSpace(passage.ID) == "" {
					continue
				}
				ref := strings.TrimSpace(sourcePath)
				if ref != "" {
					ref += "#" + passage.ID
				} else {
					ref = passage.ID
				}
				links = append(links, PassageProvenance{PaperID: firstNonEmptyReport(passage.PaperID, doc.PaperID), PassageID: passage.ID, ParserName: doc.ParserName, ParserVersion: doc.ParserVersion, SourceOffsetStart: passage.Offset.Start, SourceOffsetEnd: passage.Offset.End, SourceRef: ref})
			}
		}
	}
	return links
}

func firstNonEmptyReport(values ...string) string {
	for _, value := range values {
		if strings.TrimSpace(value) != "" {
			return value
		}
	}
	return ""
}
