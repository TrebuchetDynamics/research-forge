package protocol

import "testing"

func TestCompileSourcePlanIncludesMetaAnalysisSpineSources(t *testing.T) {
	plan, err := CompileQuestion(QuestionInput{Framework: "pico", Question: "Do catalysts improve hydrogen evolution?", Population: "hydrogen evolution", Intervention: "catalysts", Outcome: "efficiency"})
	if err != nil {
		t.Fatalf("CompileQuestion: %v", err)
	}
	sourcePlan := CompileSourcePlan(plan)
	wantSources := []string{"openalex", "semantic-scholar", "crossref", "arxiv", "pubmed", "europepmc", "nasa-ads", "doaj", "core", "unpaywall", "zotero", "jabref", "local"}
	if len(sourcePlan.Sources) != len(wantSources) {
		t.Fatalf("source count = %d, want %d: %#v", len(sourcePlan.Sources), len(wantSources), sourcePlan.Sources)
	}
	for _, source := range wantSources {
		entry, ok := sourcePlan.BySource(source)
		if !ok {
			t.Fatalf("missing source %q in %#v", source, sourcePlan.Sources)
		}
		if entry.ReviewerApprovalRequired != true {
			t.Fatalf("%s should require approval: %#v", source, entry)
		}
		if entry.PrivacyWarning == "" {
			t.Fatalf("%s missing privacy warning: %#v", source, entry)
		}
	}
	if got := sourcePlan.MustSource("nasa-ads").AuthRequirement; got == "" {
		t.Fatalf("nasa-ads auth requirement missing")
	}
	if got := sourcePlan.MustSource("zotero").SourceKind; got != "reference-manager" {
		t.Fatalf("zotero source kind = %q", got)
	}
	if sourcePlan.MustSource("local").DryRunEstimate == "" {
		t.Fatalf("local import dry-run estimate missing")
	}
}

func TestCompileSourcePlanRejectsEmptyPlan(t *testing.T) {
	if _, err := CompileSourcePlanFromQuestion(QuestionInput{}); err == nil {
		t.Fatalf("expected missing question error")
	}
}
