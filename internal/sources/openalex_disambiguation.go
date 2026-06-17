package sources

import "strings"

type OpenAlexDisambiguationQueue struct {
	SchemaVersion string                       `json:"schemaVersion"`
	Query         string                       `json:"query"`
	Entity        string                       `json:"entity"`
	Items         []OpenAlexDisambiguationItem `json:"items"`
}

type OpenAlexDisambiguationItem struct {
	Key           string           `json:"key"`
	Reason        string           `json:"reason"`
	Candidates    []OpenAlexEntity `json:"candidates"`
	ProvenanceRef string           `json:"provenanceRef"`
}

func BuildOpenAlexDisambiguationQueue(query, entity string, candidates []OpenAlexEntity, rawRef string) OpenAlexDisambiguationQueue {
	queue := OpenAlexDisambiguationQueue{SchemaVersion: "1", Query: strings.TrimSpace(query), Entity: strings.TrimSpace(entity)}
	if len(candidates) > 1 {
		queue.Items = append(queue.Items, OpenAlexDisambiguationItem{Key: queue.Query, Reason: "multiple_candidates", Candidates: append([]OpenAlexEntity{}, candidates...), ProvenanceRef: rawRef})
	}
	byName := map[string][]OpenAlexEntity{}
	for _, candidate := range candidates {
		byName[strings.ToLower(strings.TrimSpace(candidate.DisplayName))] = append(byName[strings.ToLower(strings.TrimSpace(candidate.DisplayName))], candidate)
	}
	for name, grouped := range byName {
		if name != "" && len(grouped) > 1 && len(candidates) <= 1 {
			queue.Items = append(queue.Items, OpenAlexDisambiguationItem{Key: name, Reason: "duplicate_display_name", Candidates: grouped, ProvenanceRef: rawRef})
		}
	}
	return queue
}
