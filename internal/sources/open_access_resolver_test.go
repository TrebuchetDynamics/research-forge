package sources

import (
	"strings"
	"testing"
)

func TestBuildOpenAccessResolvePlanCoversLegalSourcesAndGates(t *testing.T) {
	plan, err := BuildOpenAccessResolvePlan("https://doi.org/10.1000/Example")
	if err != nil {
		t.Fatalf("BuildOpenAccessResolvePlan: %v", err)
	}
	if plan.SchemaVersion != "1" || plan.DOI != "10.1000/example" {
		t.Fatalf("plan header = %#v", plan)
	}
	for _, want := range []string{"unpaywall", "openalex", "europepmc", "pmc", "arxiv", "biorxiv-medrxiv", "chemrxiv", "doaj", "core", "semantic-scholar", "crossref", "openlibrary", "software-heritage"} {
		if !hasResolveSource(plan.Sources, want) {
			t.Fatalf("missing source %q in %#v", want, plan.Sources)
		}
	}
	if len(plan.HumanGates) == 0 || !strings.Contains(strings.Join(plan.HumanGates, " "), "acquisition approval") {
		t.Fatalf("missing human gates: %#v", plan.HumanGates)
	}
	if len(plan.UnsupportedSources) != 0 {
		t.Fatalf("unsupported sources = %#v", plan.UnsupportedSources)
	}
	for _, source := range plan.Sources {
		if source.Lookup == "" || source.LicensePolicy == "" || len(source.ProvenanceRequired) == 0 {
			t.Fatalf("source lacks policy/provenance: %#v", source)
		}
	}
}

func TestBuildOpenAccessResolvePlanRejectsBlankDOI(t *testing.T) {
	if _, err := BuildOpenAccessResolvePlan(" "); err == nil {
		t.Fatal("expected blank DOI to fail")
	}
}

func hasResolveSource(sources []OpenAccessResolveSource, id string) bool {
	for _, source := range sources {
		if source.ID == id {
			return true
		}
	}
	return false
}
