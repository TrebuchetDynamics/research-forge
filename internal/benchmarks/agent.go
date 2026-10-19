package benchmarks

import (
	"os"
	"strconv"
	"strings"

	"github.com/TrebuchetDynamics/research-forge/internal/sources"
)

// ReadAgentArtifact applies the benchmark's strict JSON rules, including missing
// fields, duplicate keys, unknown fields and non-finite numbers, to agent files.
func ReadAgentArtifact(path string, dst any) error {
	b, err := os.ReadFile(path)
	if err != nil {
		return err
	}
	_, err = decodeStrict(b, dst)
	return err
}

// LoadAgentSet validates the full locked package before either scoring or
// preparing agent inputs. Callers must not expose this return value to agents.
func LoadAgentSet(dir string) (Set, error) {
	s, err := loadSet(dir, false)
	if err != nil {
		return s, err
	}
	_, _, err = loadIndex(dir, s)
	return s, err
}

// CheckAgentLeak rejects all reference reviews in the set, including other
// questions' reviews, in final identifiers and substantive answer text.
func CheckAgentLeak(s Set, bibliography [][]Identifier, text []string) error {
	for _, q := range s.Questions {
		forbidden := aliases(q.Review.Identifiers)
		for _, entry := range bibliography {
			for _, id := range entry {
				if forbidden[canonicalIdentifier(id)] {
					return invalid("reference_review_leak")
				}
			}
		}
		for _, value := range text {
			if strings.Contains(normalizeTitle(value), normalizeTitle(q.Review.Title)) {
				return invalid("reference_review_leak")
			}
			for _, id := range q.Review.Identifiers {
				if strings.Contains(strings.ToLower(value), strings.ToLower(id.Value)) {
					return invalid("reference_review_leak")
				}
			}
		}
	}
	return nil
}

// ScoreAgentBibliography reuses engine-tier identity/rank scoring. Each entry is
// one bibliography rank, not a hidden tool hit. Without raw membership, dedupe
// counts are unavailable rather than guessed zero. Citation facets require a
// separately pinned corpus/checker and are explicitly not implemented here.
func ScoreAgentBibliography(q Question, ks []int, bibliography [][]Identifier) (Run, error) {
	raw := map[string]sources.SourceRecord{}
	clusters := [][]string{}
	for i, entry := range bibliography {
		cluster := []string{}
		for j, id := range entry {
			if canonicalIdentifier(id) == "" {
				return Run{}, invalid("protocol_violation")
			}
			r := sources.SourceRecord{}
			switch id.Scheme {
			case "doi":
				r.Identifiers.DOI = id.Value
			case "pmid":
				r.Identifiers.PMID = id.Value
			case "pmcid":
				r.Identifiers.PMCID = id.Value
			case "arxiv":
				r.Identifiers.ArXivID = id.Value
			case "openalex":
				r.Identifiers.OpenAlexID = id.Value
			case "semantic_scholar":
				r.Identifiers.SemanticScholarID = id.Value
			case "ads_bibcode":
				r.Identifiers.ADSBibcode = id.Value
			case "url":
				r.URLs = []string{id.Value}
			}
			for _, reviewID := range q.Review.Identifiers {
				if canonicalIdentifier(reviewID) == canonicalIdentifier(id) {
					return Run{}, invalid("reference_review_leak")
				}
			}
			key := strconv.Itoa(i) + ":" + strconv.Itoa(j)
			raw[key] = r
			cluster = append(cluster, key)
		}
		clusters = append(clusters, cluster)
	}
	r := Run{QuestionID: q.QuestionID, Domain: q.Domain, GoldCount: len(q.Gold), Returned: count(len(bibliography)), AtK: []AtK{}, DedupeStatus: "unavailable", CitationStatus: "not_implemented"}
	score(&r, q, ks, clusters, raw, nil)
	r.TrueMerges = nil
	r.FalseMerges = nil
	r.UnjudgedMerges = nil
	r.MissedMerges = nil
	return r, nil
}
