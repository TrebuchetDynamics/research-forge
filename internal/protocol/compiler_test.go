package protocol

import "testing"

func TestCompilePICOQuestionDraftsAuditableProtocol(t *testing.T) {
	plan, err := CompileQuestion(QuestionInput{
		Framework:    "pico",
		Question:     "Do high entropy alloy catalysts improve hydrogen evolution efficiency compared with platinum catalysts?",
		Population:   "hydrogen evolution studies",
		Intervention: "high entropy alloy catalysts",
		Comparator:   "platinum catalysts",
		Outcome:      "efficiency",
	})
	if err != nil {
		t.Fatalf("CompileQuestion returned error: %v", err)
	}
	if plan.SchemaVersion != "1" {
		t.Fatalf("SchemaVersion = %q", plan.SchemaVersion)
	}
	if plan.Framework != FrameworkPICO {
		t.Fatalf("Framework = %q", plan.Framework)
	}
	if plan.ReviewerApprovalRequired != true || plan.AutoAcceptedClaims != false {
		t.Fatalf("review gates wrong: approval=%t autoAccepted=%t", plan.ReviewerApprovalRequired, plan.AutoAcceptedClaims)
	}
	for _, source := range []string{"openalex", "semantic-scholar", "crossref", "arxiv", "pubmed", "europepmc"} {
		if _, ok := plan.SourceQueries[source]; !ok {
			t.Fatalf("missing source query %q in %#v", source, plan.SourceQueries)
		}
	}
	openalex := plan.SourceQueries["openalex"].Query
	for _, want := range []string{"high entropy alloy catalysts", "hydrogen evolution studies", "efficiency"} {
		if !contains(openalex, want) {
			t.Fatalf("openalex query missing %q: %s", want, openalex)
		}
	}
	if len(plan.InclusionCriteria) == 0 || len(plan.ExclusionCriteria) == 0 {
		t.Fatalf("criteria were not drafted: %#v %#v", plan.InclusionCriteria, plan.ExclusionCriteria)
	}
	for _, field := range []string{"population", "intervention", "comparator", "outcome", "effect_size", "support_ref"} {
		if !schemaHasField(plan.ExtractionSchema, field) {
			t.Fatalf("schema missing %q: %#v", field, plan.ExtractionSchema)
		}
	}
	if len(plan.ReviewerPrompts) < 3 {
		t.Fatalf("reviewer prompts too short: %#v", plan.ReviewerPrompts)
	}
	for _, prompt := range plan.ReviewerPrompts {
		if prompt.ProvenanceTag != "protocol.plan.created" || prompt.Accepted {
			t.Fatalf("prompt must be provenance-tagged and unaccepted: %#v", prompt)
		}
	}
}

func TestCompileFreeformQuestionStillRequiresReview(t *testing.T) {
	plan, err := CompileQuestion(QuestionInput{Framework: "freeform", Question: "Which parser produces the most reliable citations for scientific PDFs?"})
	if err != nil {
		t.Fatalf("CompileQuestion returned error: %v", err)
	}
	if plan.Framework != FrameworkFreeform {
		t.Fatalf("Framework = %q", plan.Framework)
	}
	if !plan.ReviewerApprovalRequired || plan.AutoAcceptedClaims {
		t.Fatalf("freeform plan must be reviewer gated: %#v", plan)
	}
	if plan.SourceQueries["openalex"].Query == "" {
		t.Fatalf("freeform source query empty: %#v", plan.SourceQueries)
	}
	if !schemaHasField(plan.ExtractionSchema, "finding") {
		t.Fatalf("freeform schema missing finding: %#v", plan.ExtractionSchema)
	}
}

func TestCompileQuestionRejectsMissingQuestion(t *testing.T) {
	if _, err := CompileQuestion(QuestionInput{Framework: "pico"}); err == nil {
		t.Fatalf("expected missing question error")
	}
}

func contains(s, sub string) bool {
	return len(sub) == 0 || (len(s) >= len(sub) && (s == sub || contains(s[1:], sub) || s[:len(sub)] == sub))
}

func schemaHasField(schema ExtractionSchemaSeed, name string) bool {
	for _, field := range schema.Fields {
		if field.Name == name {
			return true
		}
	}
	return false
}
