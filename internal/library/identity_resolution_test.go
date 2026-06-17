package library

import "testing"

func TestResolveIdentityClustersMatchesAcrossAllSupportedIdentifiers(t *testing.T) {
	records := []PaperRecord{
		{Title: "Seed DOI", Identifiers: Identifiers{DOI: "10.1000/Shared", ZoteroItemKey: "ZOT-1"}},
		{Title: "Crossref same DOI", Identifiers: Identifiers{CrossrefID: "https://doi.org/10.1000/shared"}},
		{Title: "arXiv seed", Identifiers: Identifiers{ArXivID: "arXiv:2401.12345v2", SemanticScholarID: "S2-1", OpenAlexID: "W1"}},
		{Title: "Semantic Scholar same work", Identifiers: Identifiers{SemanticScholarID: "S2-1", OpenAlexID: "W1"}},
		{Title: "PubMed seed", Identifiers: Identifiers{PMID: "123", PMCID: "456"}},
		{Title: "PMC same work", Identifiers: Identifiers{PMCID: "PMC456"}},
		{Title: "ADS seed", Identifiers: Identifiers{ADSBibcode: "2024ApJ...123A..45S"}},
		{Title: "ADS same bibcode", Identifiers: Identifiers{ADSBibcode: "2024ApJ...123A..45S"}},
	}
	report := ResolveIdentityClusters(records)
	if report.SchemaVersion != "1" {
		t.Fatalf("schema = %q", report.SchemaVersion)
	}
	for _, want := range []string{"doi", "arxiv", "pmid", "pmcid", "openalex", "semantic_scholar", "crossref", "zotero", "ads_bibcode"} {
		if !report.SupportedIdentifiers[want] {
			t.Fatalf("supported identifiers missing %s: %#v", want, report.SupportedIdentifiers)
		}
	}
	if len(report.Clusters) != 4 {
		t.Fatalf("clusters = %#v, want 4", report.Clusters)
	}
	if !hasIdentityRule(report, "exact_doi_crossref") {
		t.Fatalf("missing DOI/Crossref explainable rule: %#v", report.Clusters)
	}
	if !hasIdentityRule(report, "exact_semantic_scholar") || !hasIdentityRule(report, "exact_openalex") {
		t.Fatalf("missing graph identifier rules: %#v", report.Clusters)
	}
	if !hasIdentityRule(report, "exact_pmcid") {
		t.Fatalf("missing PMCID rule: %#v", report.Clusters)
	}
	if !hasIdentityRule(report, "exact_ads_bibcode") {
		t.Fatalf("missing ADS bibcode rule: %#v", report.Clusters)
	}
}

func TestMergeDuplicatePreservesZoteroAndADSIdentifiers(t *testing.T) {
	left := PaperRecord{Title: "Left", Identifiers: Identifiers{DOI: "10.1000/a", ZoteroItemKey: "ZOT-1"}}
	right := PaperRecord{Title: "Right", Identifiers: Identifiers{ADSBibcode: "2024ApJ...123A..45S"}}
	merged := MergeDuplicate(left, right)
	if merged.Identifiers.ZoteroItemKey != "ZOT-1" || merged.Identifiers.ADSBibcode != "2024ApJ...123A..45S" {
		t.Fatalf("merged identifiers = %#v", merged.Identifiers)
	}
}

func hasIdentityRule(report IdentityResolutionReport, rule string) bool {
	for _, cluster := range report.Clusters {
		for _, match := range cluster.Matches {
			if match.Rule == rule {
				return true
			}
		}
	}
	return false
}
