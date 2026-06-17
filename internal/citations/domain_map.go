package citations

import (
	"crypto/sha256"
	"encoding/json"
	"fmt"
	"sort"
	"strings"

	"github.com/TrebuchetDynamics/research-forge/internal/parsing"
)

type DomainMapArtifact struct {
	SchemaVersion     string                 `json:"schemaVersion"`
	QuerySetChecksum  string                 `json:"querySetChecksum"`
	ModelSettings     DomainMapModelSettings `json:"modelSettings"`
	Topics            []DomainTopic          `json:"topics"`
	MergeSplitHistory []TopicHistoryEvent    `json:"mergeSplitHistory,omitempty"`
}

type DomainMapModelSettings struct {
	Model             string            `json:"model"`
	EmbeddingProvider string            `json:"embeddingProvider"`
	MinTopicSize      int               `json:"minTopicSize"`
	Parameters        map[string]string `json:"parameters,omitempty"`
}

type DomainTopic struct {
	TopicID                string                  `json:"topicId"`
	Label                  string                  `json:"label"`
	ReviewerEditedLabel    bool                    `json:"reviewerEditedLabel"`
	RepresentativePapers   []RepresentativePaper   `json:"representativePapers"`
	RepresentativePassages []RepresentativePassage `json:"representativePassages"`
	CitationGraphLinks     []CitationGraphLink     `json:"citationGraphLinks,omitempty"`
}

type RepresentativePaper struct {
	PaperID string `json:"paperId"`
	Title   string `json:"title,omitempty"`
}

type RepresentativePassage struct {
	PaperID   string `json:"paperId"`
	PassageID string `json:"passageId"`
	Text      string `json:"text"`
}

type CitationGraphLink struct {
	SourceID string `json:"sourceId"`
	TargetID string `json:"targetId"`
	Relation string `json:"relation"`
}

type TopicHistoryEvent struct {
	Action        string   `json:"action"`
	TopicIDs      []string `json:"topicIds"`
	ResultTopicID string   `json:"resultTopicId"`
	Reviewer      string   `json:"reviewer,omitempty"`
	Reason        string   `json:"reason,omitempty"`
}

type DomainMapOptions struct {
	ReviewerLabels    map[string]string
	MergeSplitHistory []TopicHistoryEvent
	ModelSettings     DomainMapModelSettings
}

func BuildDomainMapArtifact(docs []parsing.ParsedDocument, graphData []byte, opts DomainMapOptions) (DomainMapArtifact, error) {
	if len(docs) == 0 {
		return DomainMapArtifact{}, fmt.Errorf("domain map requires parsed documents")
	}
	settings := opts.ModelSettings
	if strings.TrimSpace(settings.Model) == "" {
		settings.Model = "bertopic-style-deterministic-keyword-fixture"
	}
	if strings.TrimSpace(settings.EmbeddingProvider) == "" {
		settings.EmbeddingProvider = "deterministic-keyword"
	}
	if settings.MinTopicSize <= 0 {
		settings.MinTopicSize = 1
	}
	linksByPaper := citationLinksByPaper(graphData)
	topics := map[string]*DomainTopic{}
	for _, doc := range docs {
		topicID := domainTopicID(doc)
		if topicID == "" {
			topicID = "unlabeled"
		}
		topic := topics[topicID]
		if topic == nil {
			label := titleCaseTopic(topicID)
			reviewerEdited := false
			if reviewerLabel := strings.TrimSpace(opts.ReviewerLabels[topicID]); reviewerLabel != "" {
				label = reviewerLabel
				reviewerEdited = true
			}
			topic = &DomainTopic{TopicID: topicID, Label: label, ReviewerEditedLabel: reviewerEdited}
			topics[topicID] = topic
		}
		topic.RepresentativePapers = appendUniquePaper(topic.RepresentativePapers, RepresentativePaper{PaperID: doc.PaperID, Title: doc.Title})
		if passage, ok := firstRepresentativePassage(doc); ok {
			topic.RepresentativePassages = append(topic.RepresentativePassages, passage)
		}
		topic.CitationGraphLinks = append(topic.CitationGraphLinks, linksByPaper[doc.PaperID]...)
	}
	artifact := DomainMapArtifact{SchemaVersion: "1", QuerySetChecksum: checksumDomainDocs(docs), ModelSettings: settings, MergeSplitHistory: append([]TopicHistoryEvent{}, opts.MergeSplitHistory...)}
	for _, topicID := range sortedKeys(topics) {
		topic := *topics[topicID]
		sort.Slice(topic.RepresentativePapers, func(i, j int) bool {
			return topic.RepresentativePapers[i].PaperID < topic.RepresentativePapers[j].PaperID
		})
		sort.Slice(topic.RepresentativePassages, func(i, j int) bool {
			return topic.RepresentativePassages[i].PassageID < topic.RepresentativePassages[j].PassageID
		})
		sort.Slice(topic.CitationGraphLinks, func(i, j int) bool {
			if topic.CitationGraphLinks[i].SourceID == topic.CitationGraphLinks[j].SourceID {
				return topic.CitationGraphLinks[i].TargetID < topic.CitationGraphLinks[j].TargetID
			}
			return topic.CitationGraphLinks[i].SourceID < topic.CitationGraphLinks[j].SourceID
		})
		artifact.Topics = append(artifact.Topics, topic)
	}
	return artifact, nil
}

func domainTopicID(doc parsing.ParsedDocument) string {
	text := doc.Title
	if strings.TrimSpace(text) == "" {
		for _, section := range doc.Sections {
			for _, passage := range section.Passages {
				text = passage.Text
				break
			}
			if text != "" {
				break
			}
		}
	}
	for _, token := range strings.Fields(strings.ToLower(text)) {
		token = strings.Trim(token, " .,;:()[]{}!?\"'")
		if len(token) >= 4 && !domainStopwords[token] {
			return token
		}
	}
	return ""
}

var domainStopwords = map[string]bool{"study": true, "paper": true, "review": true, "with": true, "from": true, "this": true, "that": true, "using": true}

func titleCaseTopic(topicID string) string {
	if topicID == "" {
		return "Unlabeled"
	}
	return strings.ToUpper(topicID[:1]) + topicID[1:]
}

func firstRepresentativePassage(doc parsing.ParsedDocument) (RepresentativePassage, bool) {
	for _, section := range doc.Sections {
		for _, passage := range section.Passages {
			if strings.TrimSpace(passage.Text) != "" {
				return RepresentativePassage{PaperID: doc.PaperID, PassageID: passage.ID, Text: passage.Text}, true
			}
		}
	}
	return RepresentativePassage{}, false
}

func appendUniquePaper(papers []RepresentativePaper, paper RepresentativePaper) []RepresentativePaper {
	for _, existing := range papers {
		if existing.PaperID == paper.PaperID {
			return papers
		}
	}
	return append(papers, paper)
}

func citationLinksByPaper(data []byte) map[string][]CitationGraphLink {
	links := map[string][]CitationGraphLink{}
	if len(data) == 0 {
		return links
	}
	var exported exportGraph
	if err := json.Unmarshal(data, &exported); err != nil {
		return links
	}
	for _, edge := range exported.Edges {
		links[edge.Source] = append(links[edge.Source], CitationGraphLink{SourceID: edge.Source, TargetID: edge.Target, Relation: "cites"})
		links[edge.Target] = append(links[edge.Target], CitationGraphLink{SourceID: edge.Source, TargetID: edge.Target, Relation: "cited-by"})
	}
	return links
}

func checksumDomainDocs(docs []parsing.ParsedDocument) string {
	projection := []map[string]string{}
	for _, doc := range docs {
		projection = append(projection, map[string]string{"paperId": doc.PaperID, "title": doc.Title, "topic": domainTopicID(doc)})
	}
	data, _ := json.Marshal(projection)
	sum := sha256.Sum256(data)
	return fmt.Sprintf("sha256:%x", sum[:])
}
