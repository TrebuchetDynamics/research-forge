package protocol

import "testing"

func TestConnectorCapabilityRegistryCoversMetaAnalysisSpineSources(t *testing.T) {
	registry := DefaultConnectorCapabilityRegistry()
	want := []string{"openalex", "semantic-scholar", "crossref", "arxiv", "pubmed", "europepmc", "nasa-ads", "doaj", "core", "biorxiv", "zenodo", "inspire-hep", "dblp", "clinicaltrials", "osf", "opencitations", "base", "zbmath", "figshare", "datacite", "lens", "eric", "hal", "dimensions", "pubchem", "chemrxiv", "ntrs", "doab", "openaire", "plos", "osti", "dryad", "researchsquare", "cinii", "biostudies", "unpaywall", "zotero", "jabref", "local"}
	if len(registry.Connectors) != len(want) {
		t.Fatalf("connector count = %d, want %d", len(registry.Connectors), len(want))
	}
	for _, id := range want {
		capability, ok := registry.ByID(id)
		if !ok {
			t.Fatalf("missing connector capability %q", id)
		}
		if len(capability.SupportedEntities) == 0 {
			t.Fatalf("%s missing supported entities", id)
		}
		if capability.RateLimitPolicy == "" {
			t.Fatalf("%s missing rate limit policy", id)
		}
		if capability.AuthNeeds == "" {
			t.Fatalf("%s missing auth needs", id)
		}
		if capability.LiveSmokeStatus == "" {
			t.Fatalf("%s missing live smoke status", id)
		}
		if capability.LicenseShareabilityPolicy == "" {
			t.Fatalf("%s missing license/shareability policy", id)
		}
		if capability.Cacheability == "" {
			t.Fatalf("%s missing cacheability", id)
		}
		if len(capability.ProvenanceFields) == 0 {
			t.Fatalf("%s missing provenance fields", id)
		}
	}
}

func TestSourcePlanUsesCapabilityRegistryMetadata(t *testing.T) {
	plan, err := CompileSourcePlanFromQuestion(QuestionInput{Question: "Does catalyst choice change efficiency?"})
	if err != nil {
		t.Fatalf("CompileSourcePlanFromQuestion: %v", err)
	}
	openalex := plan.MustSource("openalex")
	if openalex.RateLimitPolicy == "" || openalex.Cacheability == "" || len(openalex.SupportedEntities) == 0 || len(openalex.ProvenanceFields) == 0 {
		t.Fatalf("source plan did not include capability metadata: %#v", openalex)
	}
	ads := plan.MustSource("nasa-ads")
	if ads.AuthRequirement != "NASA ADS token required for live API" {
		t.Fatalf("nasa ads auth = %q", ads.AuthRequirement)
	}
}
