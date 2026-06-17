package documents

import (
	"encoding/xml"
	"fmt"
	"os"
	"strings"

	"github.com/TrebuchetDynamics/research-forge/internal/library"
)

type PMCIDPMIDLink struct {
	PMID  string `json:"pmid,omitempty"`
	PMCID string `json:"pmcid,omitempty"`
	DOI   string `json:"doi,omitempty"`
	Title string `json:"title,omitempty"`
}

type BiomedicalFullText struct {
	SchemaVersion      string                  `json:"schemaVersion"`
	PMID               string                  `json:"pmid,omitempty"`
	PMCID              string                  `json:"pmcid,omitempty"`
	DOI                string                  `json:"doi,omitempty"`
	Title              string                  `json:"title"`
	Sections           []BiomedicalSection     `json:"sections"`
	SupplementaryFiles []SupplementaryFileInfo `json:"supplementaryFiles"`
}

type BiomedicalSection struct {
	Title string `json:"title"`
	Text  string `json:"text"`
}

type SupplementaryFileInfo struct {
	ID    string `json:"id,omitempty"`
	Label string `json:"label,omitempty"`
	Href  string `json:"href"`
}

type BiomedicalLiveDriftSmokeSnapshot struct {
	SchemaVersion string                           `json:"schemaVersion"`
	Connectors    []BiomedicalLiveDriftSmokeSource `json:"connectors"`
}

type BiomedicalLiveDriftSmokeSource struct {
	Source         string   `json:"source"`
	OptInEnv       string   `json:"optInEnv"`
	ExpectedFields []string `json:"expectedFields"`
}

func LinkPMCIDPMID(records []library.PaperRecord) []PMCIDPMIDLink {
	links := make([]PMCIDPMIDLink, 0, len(records))
	for _, record := range records {
		ids := record.Identifiers
		if strings.TrimSpace(ids.PMID) == "" && strings.TrimSpace(ids.PMCID) == "" {
			continue
		}
		links = append(links, PMCIDPMIDLink{PMID: strings.TrimSpace(ids.PMID), PMCID: normalizeBiomedicalPMCID(ids.PMCID), DOI: strings.TrimSpace(ids.DOI), Title: record.Title})
	}
	return links
}

func ImportStructuredBiomedicalFullText(path string) (BiomedicalFullText, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return BiomedicalFullText{}, err
	}
	var article jatsArticle
	if err := xml.Unmarshal(data, &article); err != nil {
		return BiomedicalFullText{}, err
	}
	fullText := BiomedicalFullText{SchemaVersion: "1", Title: compactBiomedicalText(article.Front.ArticleMeta.TitleGroup.ArticleTitle)}
	for _, id := range article.Front.ArticleMeta.ArticleIDs {
		switch strings.ToLower(strings.TrimSpace(id.Type)) {
		case "pmid":
			fullText.PMID = strings.TrimSpace(id.Value)
		case "pmc", "pmcid":
			fullText.PMCID = normalizeBiomedicalPMCID(id.Value)
		case "doi":
			fullText.DOI = strings.ToLower(strings.TrimSpace(id.Value))
		}
	}
	if abstract := compactBiomedicalText(article.Front.ArticleMeta.Abstract.Text()); abstract != "" {
		fullText.Sections = append(fullText.Sections, BiomedicalSection{Title: "Abstract", Text: abstract})
	}
	for _, sec := range article.Body.Sections {
		fullText.Sections = append(fullText.Sections, BiomedicalSection{Title: compactBiomedicalText(sec.Title), Text: compactBiomedicalText(sec.Text())})
	}
	fullText.SupplementaryFiles = DiscoverSupplementaryFiles(article)
	return fullText, nil
}

func DiscoverSupplementaryFiles(article jatsArticle) []SupplementaryFileInfo {
	out := []SupplementaryFileInfo{}
	var walk func([]jatsSection)
	walk = func(sections []jatsSection) {
		for _, section := range sections {
			for _, supplement := range section.Supplements {
				href := strings.TrimSpace(supplement.Href)
				if href != "" {
					out = append(out, SupplementaryFileInfo{ID: strings.TrimSpace(supplement.ID), Label: compactBiomedicalText(supplement.Label), Href: href})
				}
			}
			walk(section.Sections)
		}
	}
	walk(article.Body.Sections)
	walk(article.Back.Sections)
	return out
}

func NewBiomedicalLiveDriftSmokeSnapshot() BiomedicalLiveDriftSmokeSnapshot {
	return BiomedicalLiveDriftSmokeSnapshot{SchemaVersion: "1", Connectors: []BiomedicalLiveDriftSmokeSource{
		{Source: "pubmed", OptInEnv: "RFORGE_RUN_LIVE_SOURCE_SMOKE=1", ExpectedFields: []string{"pmid", "pmcid", "doi", "title", "mesh_terms"}},
		{Source: "europepmc", OptInEnv: "RFORGE_RUN_LIVE_SOURCE_SMOKE=1", ExpectedFields: []string{"pmid", "pmcid", "doi", "title", "full_text_url", "license"}},
	}}
}

func normalizeBiomedicalPMCID(value string) string {
	value = strings.ToUpper(strings.TrimSpace(value))
	if value == "" || strings.HasPrefix(value, "PMC") {
		return value
	}
	return "PMC" + value
}

func compactBiomedicalText(value string) string { return strings.Join(strings.Fields(value), " ") }

type jatsArticle struct {
	Front jatsFront `xml:"front"`
	Body  jatsBody  `xml:"body"`
	Back  jatsBody  `xml:"back"`
}

type jatsFront struct {
	ArticleMeta jatsArticleMeta `xml:"article-meta"`
}
type jatsArticleMeta struct {
	ArticleIDs []jatsArticleID `xml:"article-id"`
	TitleGroup struct {
		ArticleTitle string `xml:"article-title"`
	} `xml:"title-group"`
	Abstract jatsTextBlock `xml:"abstract"`
}
type jatsArticleID struct {
	Type  string `xml:"pub-id-type,attr"`
	Value string `xml:",chardata"`
}
type jatsBody struct {
	Sections []jatsSection `xml:"sec"`
}
type jatsSection struct {
	ID          string           `xml:"id,attr"`
	SecType     string           `xml:"sec-type,attr"`
	Title       string           `xml:"title"`
	Paragraphs  []string         `xml:"p"`
	Sections    []jatsSection    `xml:"sec"`
	Supplements []jatsSupplement `xml:"supplementary-material"`
}
type jatsSupplement struct {
	ID    string `xml:"id,attr"`
	Href  string `xml:"http://www.w3.org/1999/xlink href,attr"`
	Label string `xml:"label"`
}
type jatsTextBlock struct {
	Paragraphs []string `xml:"p"`
}

func (b jatsTextBlock) Text() string { return strings.Join(b.Paragraphs, " ") }
func (s jatsSection) Text() string   { return strings.Join(s.Paragraphs, " ") }

func (f BiomedicalFullText) Validate() error {
	if strings.TrimSpace(f.PMID) == "" && strings.TrimSpace(f.PMCID) == "" {
		return fmt.Errorf("biomedical full text requires PMID or PMCID")
	}
	return nil
}
