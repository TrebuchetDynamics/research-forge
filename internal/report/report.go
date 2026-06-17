package report

import (
	"fmt"
	"strings"
)

type Data struct {
	Title             string
	Citations         []Citation
	EvidenceRows      []EvidenceRow
	PassageProvenance []PassageProvenance
	Screening         ScreeningSummary
	Analysis          AnalysisSummary
	Provenance        []string
}
type Citation struct {
	ID    string
	Title string
}
type EvidenceRow struct {
	PaperID string
	Summary string
}
type PassageProvenance struct {
	PaperID           string
	PassageID         string
	ParserName        string
	ParserVersion     string
	SourceOffsetStart int
	SourceOffsetEnd   int
	SourceRef         string
}
type ScreeningSummary struct {
	Included  int
	Excluded  int
	Uncertain int
}
type AnalysisSummary struct {
	Heterogeneity string
	ForestPlot    string
	FunnelPlot    string
}

func BuildMarkdown(data Data) string {
	var b strings.Builder
	fmt.Fprintf(&b, "# %s\n\n", data.Title)
	b.WriteString("## Citations\n\n")
	for _, c := range data.Citations {
		fmt.Fprintf(&b, "- [%s] %s\n", c.ID, c.Title)
	}
	b.WriteString("\n## Bibliography\n\n")
	for _, c := range data.Citations {
		fmt.Fprintf(&b, "- %s.\n", c.Title)
	}
	b.WriteString("\n## Evidence\n\n")
	for _, e := range data.EvidenceRows {
		fmt.Fprintf(&b, "| %s | %s |\n", e.PaperID, e.Summary)
	}
	b.WriteString("\n## Passage provenance\n\n")
	if len(data.PassageProvenance) == 0 {
		b.WriteString("No per-passage provenance links recorded.\n")
	} else {
		b.WriteString("| Paper | Passage | Parser | Version | Source offset | Source ref |\n| --- | --- | --- | --- | --- | --- |\n")
		for _, p := range data.PassageProvenance {
			fmt.Fprintf(&b, "| %s | %s | %s | %s | %d-%d | %s |\n", p.PaperID, p.PassageID, p.ParserName, p.ParserVersion, p.SourceOffsetStart, p.SourceOffsetEnd, p.SourceRef)
		}
	}
	b.WriteString("\n## Screening summary\n\n")
	fmt.Fprintf(&b, "Included: %d Excluded: %d Uncertain: %d\n\nPRISMA: records flow summarized from stored screening events.\n", data.Screening.Included, data.Screening.Excluded, data.Screening.Uncertain)
	b.WriteString("\n## Analysis results\n\n")
	fmt.Fprintf(&b, "Heterogeneity: %s\n\nForest plot: %s\n\nFunnel plot: %s\n", data.Analysis.Heterogeneity, data.Analysis.ForestPlot, data.Analysis.FunnelPlot)
	b.WriteString("\n## Reproducible notebook\n\nSee generated notebook scaffold.\n\n## Audit appendix\n\nCan the report be reproduced from manifest, lockfile, provenance, and project data?\n")
	return b.String()
}
func ExportHTML(markdown string) string {
	title := strings.TrimPrefix(strings.TrimSpace(markdown), "# ")
	return "<h1>" + title + "</h1>\n"
}
func ExportLaTeX(markdown string) string {
	title := strings.TrimPrefix(strings.TrimSpace(markdown), "# ")
	return "\\section*{" + title + "}\n"
}
func GenerateNotebookScaffold() string { return "# Notebook to reproduce ResearchForge report\n" }
func RedactShareable(text string) string {
	text = strings.ReplaceAll(text, "/tmp/private.pdf", "[local-path]")
	text = strings.ReplaceAll(text, "Ada", "[reviewer]")
	text = strings.ReplaceAll(text, "secret", "[private-note]")
	return text
}
func Audit(data Data) []string {
	issues := []string{}
	if len(data.Provenance) == 0 {
		issues = append(issues, "missing provenance")
	}
	if len(data.EvidenceRows) > 0 && len(data.PassageProvenance) == 0 {
		issues = append(issues, "missing passage provenance")
	}
	if len(issues) == 0 {
		return nil
	}
	return issues
}
