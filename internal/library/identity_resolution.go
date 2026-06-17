package library

import "strings"

type IdentityResolutionReport struct {
	SchemaVersion        string            `json:"schemaVersion"`
	SupportedIdentifiers map[string]bool   `json:"supportedIdentifiers"`
	Clusters             []IdentityCluster `json:"clusters"`
}

type IdentityCluster struct {
	ID            string            `json:"id"`
	RecordIndexes []int             `json:"recordIndexes"`
	Identifiers   map[string]string `json:"identifiers"`
	Matches       []IdentityMatch   `json:"matches"`
}

type IdentityMatch struct {
	LeftIndex   int     `json:"leftIndex"`
	RightIndex  int     `json:"rightIndex"`
	Rule        string  `json:"rule"`
	Identifier  string  `json:"identifier"`
	Value       string  `json:"value"`
	Confidence  float64 `json:"confidence"`
	Explanation string  `json:"explanation"`
}

func ResolveIdentityClusters(records []PaperRecord) IdentityResolutionReport {
	dsu := newIdentityDSU(len(records))
	matches := []IdentityMatch{}
	byIdentifier := map[string]int{}
	for i, record := range records {
		for _, candidate := range identityCandidates(record) {
			key := candidate.identifier + ":" + candidate.value
			if first, ok := byIdentifier[key]; ok {
				dsu.union(first, i)
				matches = append(matches, IdentityMatch{LeftIndex: first, RightIndex: i, Rule: candidate.rule, Identifier: candidate.identifier, Value: candidate.value, Confidence: candidate.confidence, Explanation: candidate.explanation})
			} else {
				byIdentifier[key] = i
			}
		}
	}
	clustersByRoot := map[int]*IdentityCluster{}
	for i, record := range records {
		root := dsu.find(i)
		cluster, ok := clustersByRoot[root]
		if !ok {
			cluster = &IdentityCluster{ID: "identity-cluster-" + intString(len(clustersByRoot)+1), Identifiers: map[string]string{}}
			clustersByRoot[root] = cluster
		}
		cluster.RecordIndexes = append(cluster.RecordIndexes, i)
		for _, candidate := range identityCandidates(record) {
			if cluster.Identifiers[candidate.identifier] == "" {
				cluster.Identifiers[candidate.identifier] = candidate.value
			}
		}
	}
	for _, match := range matches {
		root := dsu.find(match.LeftIndex)
		clustersByRoot[root].Matches = append(clustersByRoot[root].Matches, match)
	}
	clusters := []IdentityCluster{}
	for _, cluster := range clustersByRoot {
		if len(cluster.RecordIndexes) > 1 {
			clusters = append(clusters, *cluster)
		}
	}
	return IdentityResolutionReport{SchemaVersion: "1", SupportedIdentifiers: supportedIdentityIdentifiers(), Clusters: clusters}
}

func supportedIdentityIdentifiers() map[string]bool {
	return map[string]bool{"doi": true, "arxiv": true, "pmid": true, "pmcid": true, "openalex": true, "semantic_scholar": true, "crossref": true, "zotero": true, "ads_bibcode": true}
}

type identityCandidate struct {
	identifier, value, rule, explanation string
	confidence                           float64
}

func identityCandidates(record PaperRecord) []identityCandidate {
	ids := normalizeIdentifiers(record.Identifiers)
	out := []identityCandidate{}
	add := func(identifier, value, rule, explanation string, confidence float64) {
		value = strings.TrimSpace(value)
		if value != "" {
			out = append(out, identityCandidate{identifier: identifier, value: value, rule: rule, explanation: explanation, confidence: confidence})
		}
	}
	add("doi", ids.DOI, "exact_doi", "normalized DOI values are identical", 1)
	add("doi", ids.CrossrefID, "exact_doi_crossref", "Crossref work ID normalizes to the same DOI namespace", 1)
	add("crossref", ids.CrossrefID, "exact_crossref", "Crossref work IDs are identical", 1)
	add("arxiv", normalizeArXivDuplicateID(ids.ArXivID), "exact_arxiv", "arXiv IDs match after version suffix removal", .95)
	add("pmid", ids.PMID, "exact_pmid", "PMID values are identical", .98)
	add("pmcid", ids.PMCID, "exact_pmcid", "PMCID values are identical after PMC prefix normalization", .98)
	add("openalex", ids.OpenAlexID, "exact_openalex", "OpenAlex work IDs are identical", .98)
	add("semantic_scholar", ids.SemanticScholarID, "exact_semantic_scholar", "Semantic Scholar paper IDs are identical", .98)
	add("zotero", ids.ZoteroItemKey, "exact_zotero", "Zotero item keys are identical", .9)
	add("ads_bibcode", ids.ADSBibcode, "exact_ads_bibcode", "NASA ADS bibcodes are identical", .98)
	for _, ref := range record.SourceRefs {
		if ref.Metadata == nil {
			continue
		}
		add("zotero", firstNonEmpty(ref.Metadata["zotero_item_key"], ref.Metadata["zotero_rdf_id"], ref.Metadata["csl_id"]), "exact_zotero", "Zotero source metadata keys are identical", .9)
		add("ads_bibcode", firstNonEmpty(ref.Metadata["ads_bibcode"], ref.Metadata["bibcode"]), "exact_ads_bibcode", "ADS bibcode source metadata is identical", .98)
	}
	return out
}

func firstNonEmpty(values ...string) string {
	for _, value := range values {
		if strings.TrimSpace(value) != "" {
			return strings.TrimSpace(value)
		}
	}
	return ""
}

type identityDSU struct{ parent []int }

func newIdentityDSU(n int) *identityDSU {
	p := make([]int, n)
	for i := range p {
		p[i] = i
	}
	return &identityDSU{parent: p}
}
func (d *identityDSU) find(x int) int {
	if d.parent[x] != x {
		d.parent[x] = d.find(d.parent[x])
	}
	return d.parent[x]
}
func (d *identityDSU) union(a, b int) {
	ra, rb := d.find(a), d.find(b)
	if ra != rb {
		d.parent[rb] = ra
	}
}

func intString(value int) string {
	if value == 0 {
		return "0"
	}
	digits := []byte{}
	for value > 0 {
		digits = append([]byte{byte('0' + value%10)}, digits...)
		value /= 10
	}
	return string(digits)
}
