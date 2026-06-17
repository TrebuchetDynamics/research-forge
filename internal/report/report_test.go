package report

import (
	"strings"
	"testing"
)

func TestBuildMarkdownReportIncludesRequiredSectionsAndAuditQuestions(t *testing.T) {
	data := Data{Title: "Artificial Photosynthesis Review", Citations: []Citation{{ID: "paper-1", Title: "Catalyst study"}}, EvidenceRows: []EvidenceRow{{PaperID: "paper-1", Summary: "TiO2 improves yield"}}, Screening: ScreeningSummary{Included: 1, Excluded: 2, Uncertain: 0}, Analysis: AnalysisSummary{Heterogeneity: "I2=0", ForestPlot: "forest.png", FunnelPlot: "funnel.png"}}
	report := BuildMarkdown(data)
	for _, want := range []string{"# Artificial Photosynthesis Review", "## Citations", "Catalyst study", "## Evidence", "TiO2", "## Screening summary", "PRISMA", "## Analysis results", "forest.png", "funnel.png", "## Audit appendix", "Can the report be reproduced"} {
		if !strings.Contains(report, want) {
			t.Fatalf("report missing %q:\n%s", want, report)
		}
	}
}

func TestReportExportsHTMLLaTeXNotebookAndRedactsShareableData(t *testing.T) {
	markdown := "# Report\n\nLocal path /tmp/private.pdf reviewer Ada private note secret\n"
	redacted := RedactShareable(markdown)
	if strings.Contains(redacted, "/tmp/private.pdf") || strings.Contains(redacted, "Ada") || strings.Contains(redacted, "secret") {
		t.Fatalf("redacted = %s", redacted)
	}
	if !strings.Contains(ExportHTML("# Report"), "<h1>Report</h1>") {
		t.Fatalf("html export failed")
	}
	if !strings.Contains(ExportLaTeX("# Report"), "\\section*{Report}") {
		t.Fatalf("latex export failed")
	}
	if !strings.Contains(GenerateNotebookScaffold(), "reproduce ResearchForge report") {
		t.Fatalf("notebook scaffold failed")
	}
}

func TestReportIncludesPerPassageParserProvenanceAndAuditLinks(t *testing.T) {
	data := Data{Title: "Traceable report", Provenance: []string{"provenance"}, EvidenceRows: []EvidenceRow{{PaperID: "paper-1", Summary: "claim"}}, PassageProvenance: []PassageProvenance{{PaperID: "paper-1", PassageID: "p1", ParserName: "grobid", ParserVersion: "0.8", SourceOffsetStart: 42, SourceOffsetEnd: 88, SourceRef: "parsed/paper-1.json#p1"}}}
	markdown := BuildMarkdown(data)
	for _, want := range []string{"## Passage provenance", "paper-1", "p1", "grobid", "0.8", "42-88", "parsed/paper-1.json#p1"} {
		if !strings.Contains(markdown, want) {
			t.Fatalf("report missing %q:\n%s", want, markdown)
		}
	}
	if issues := Audit(data); len(issues) != 0 {
		t.Fatalf("expected provenance audit clean, got %#v", issues)
	}
}

func TestReportAuditDetectsMissingProvenance(t *testing.T) {
	issues := Audit(Data{Title: "No provenance", EvidenceRows: []EvidenceRow{{PaperID: "paper-1", Summary: "claim"}}})
	if len(issues) < 2 || issues[0] != "missing provenance" || issues[1] != "missing passage provenance" {
		t.Fatalf("issues = %#v", issues)
	}
}
