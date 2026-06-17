package parsing

import "testing"

func TestParserOutputManifestIncludesLicenseProvenanceAndShareability(t *testing.T) {
	doc := ParsedDocument{PaperID: "paper-1", ParserName: "grobid", ParserVersion: "0.8", Sections: []Section{{Passages: []Passage{{ID: "p1"}}}}}
	manifest := NewParserRunManifestWithOutput(doc, []byte("input pdf"), []byte(`{"parsed":true}`), "parsed/paper-1.json", []string{"grobid", "processFulltextDocument"})
	if manifest.ParserSource != "external-service" || manifest.Command == nil || manifest.OutputChecksum == "" || manifest.OutputKind != "tei-json" {
		t.Fatalf("manifest missing provenance fields: %#v", manifest)
	}
	if manifest.LicenseConstraints == "" || manifest.Shareability == "" || !manifest.ReviewerApprovalRequired {
		t.Fatalf("manifest missing license/shareability gates: %#v", manifest)
	}
}

func TestDefaultParserOutputPoliciesCoverRequiredParsers(t *testing.T) {
	policies := DefaultParserOutputPolicies()
	for _, parser := range []string{"grobid", "s2orc", "s2orc-doc2json", "papermage", "cermine", "science-parse", "anystyle"} {
		policy, ok := policies.Policy(parser)
		if !ok {
			t.Fatalf("missing parser policy %s", parser)
		}
		if policy.ParserSource == "" || policy.LicenseConstraints == "" || policy.Shareability == "" || len(policy.ProvenanceFields) == 0 {
			t.Fatalf("policy %s incomplete: %#v", parser, policy)
		}
	}
}
