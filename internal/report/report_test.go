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

func TestReportAuditDetectsMissingProvenance(t *testing.T) {
	issues := Audit(Data{Title: "No provenance"})
	if len(issues) == 0 || issues[0] != "missing provenance" {
		t.Fatalf("issues = %#v", issues)
	}
}
