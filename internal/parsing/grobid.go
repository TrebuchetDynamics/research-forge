package parsing

import (
	"bytes"
	"context"
	"encoding/xml"
	"fmt"
	"io"
	"mime/multipart"
	"net/http"
	"strings"
	"time"
)

const maxGROBIDTEIBytes int64 = 50 << 20

// GROBIDClientOptions configures the GROBID parser adapter.
type GROBIDClientOptions struct {
	BaseURL     string
	Timeout     time.Duration
	Version     string
	MaxTEIBytes int64
}

// GROBIDClient parses PDFs through a GROBID endpoint.
type GROBIDClient struct {
	baseURL     string
	client      *http.Client
	version     string
	maxTEIBytes int64
}

// NewGROBIDClient creates a GROBID parser adapter.
func NewGROBIDClient(options GROBIDClientOptions) GROBIDClient {
	timeout := options.Timeout
	if timeout == 0 {
		timeout = 30 * time.Second
	}
	maxBytes := options.MaxTEIBytes
	if maxBytes == 0 {
		maxBytes = maxGROBIDTEIBytes
	}
	return GROBIDClient{baseURL: strings.TrimRight(options.BaseURL, "/"), client: &http.Client{Timeout: timeout}, version: strings.TrimSpace(options.Version), maxTEIBytes: maxBytes}
}

// Parse sends a PDF to GROBID and normalizes the returned TEI.
func (c GROBIDClient) Parse(ctx context.Context, pdf []byte, options ParseOptions) (ParsedDocument, error) {
	if c.baseURL == "" {
		return ParsedDocument{}, fmt.Errorf("grobid base URL is required")
	}
	var body bytes.Buffer
	writer := multipart.NewWriter(&body)
	part, err := writer.CreateFormFile("input", "document.pdf")
	if err != nil {
		return ParsedDocument{}, err
	}
	if _, err := part.Write(pdf); err != nil {
		return ParsedDocument{}, err
	}
	if err := writer.Close(); err != nil {
		return ParsedDocument{}, err
	}
	request, err := http.NewRequestWithContext(ctx, http.MethodPost, c.baseURL+"/api/processFulltextDocument", &body)
	if err != nil {
		return ParsedDocument{}, err
	}
	request.Header.Set("Content-Type", writer.FormDataContentType())
	response, err := c.client.Do(request)
	if err != nil {
		return ParsedDocument{}, err
	}
	defer response.Body.Close()
	if response.StatusCode < 200 || response.StatusCode >= 300 {
		return ParsedDocument{}, fmt.Errorf("grobid status %d", response.StatusCode)
	}
	tei, err := readBoundedTEI(response, c.maxTEIBytes)
	if err != nil {
		return ParsedDocument{}, err
	}
	doc, err := parseTEI(tei, options.PaperID)
	if err != nil {
		return ParsedDocument{}, err
	}
	doc.ParserName = "grobid"
	doc.ParserVersion = c.version
	return doc, nil
}

func readBoundedTEI(response *http.Response, maxBytes int64) ([]byte, error) {
	if response.ContentLength > maxBytes {
		return nil, fmt.Errorf("grobid TEI response too large: %d bytes exceeds %d", response.ContentLength, maxBytes)
	}
	data, err := io.ReadAll(io.LimitReader(response.Body, maxBytes+1))
	if err != nil {
		return nil, err
	}
	if int64(len(data)) > maxBytes {
		return nil, fmt.Errorf("grobid TEI response too large: exceeds %d", maxBytes)
	}
	return data, nil
}

type teiDocument struct {
	Header teiHeader `xml:"teiHeader"`
	Text   teiText   `xml:"text"`
}

type teiHeader struct {
	FileDesc    teiFileDesc    `xml:"fileDesc"`
	ProfileDesc teiProfileDesc `xml:"profileDesc"`
}

type teiFileDesc struct {
	TitleStmt   teiTitleStmt   `xml:"titleStmt"`
	ProfileDesc teiProfileDesc `xml:"profileDesc"`
}

type teiTitleStmt struct {
	Title   string      `xml:"title"`
	Authors []teiAuthor `xml:"author"`
}

type teiAuthor struct {
	PersName teiPersName `xml:"persName"`
}

type teiPersName struct {
	Forename string `xml:"forename"`
	Surname  string `xml:"surname"`
}

type teiProfileDesc struct {
	Abstract teiAbstract `xml:"abstract"`
}

type teiAbstract struct {
	Paragraphs []string `xml:"p"`
}

type teiText struct {
	Body teiBody `xml:"body"`
	Back teiBack `xml:"back"`
}

type teiBody struct {
	Divs []teiDiv `xml:"div"`
}

type teiDiv struct {
	Head       string   `xml:"head"`
	Paragraphs []string `xml:"p"`
}

type teiBack struct {
	ListBibl teiListBibl `xml:"listBibl"`
}

type teiListBibl struct {
	Items []teiBiblStruct `xml:"biblStruct"`
}

type teiBiblStruct struct {
	Analytic teiAnalytic `xml:"analytic"`
}

type teiAnalytic struct {
	Title string `xml:"title"`
}

func parseTEI(data []byte, paperID string) (ParsedDocument, error) {
	var tei teiDocument
	if err := xml.Unmarshal(data, &tei); err != nil {
		return ParsedDocument{}, err
	}
	abstract := compactText(strings.Join(tei.Header.ProfileDesc.Abstract.Paragraphs, " "))
	if abstract == "" {
		abstract = compactText(strings.Join(tei.Header.FileDesc.ProfileDesc.Abstract.Paragraphs, " "))
	}
	doc := ParsedDocument{SchemaVersion: "1", PaperID: strings.TrimSpace(paperID), Title: compactText(tei.Header.FileDesc.TitleStmt.Title), Abstract: abstract}
	for _, author := range tei.Header.FileDesc.TitleStmt.Authors {
		doc.Authors = append(doc.Authors, ParsedAuthor{Given: compactText(author.PersName.Forename), Family: compactText(author.PersName.Surname)})
	}
	for i, div := range tei.Text.Body.Divs {
		sectionID := fmt.Sprintf("%s-sec-%d", doc.PaperID, i+1)
		section := Section{ID: sectionID, Title: compactText(div.Head)}
		for j, paragraph := range div.Paragraphs {
			section.Passages = append(section.Passages, Passage{ID: fmt.Sprintf("%s-p-%d", sectionID, j+1), PaperID: doc.PaperID, SectionID: sectionID, Text: compactText(paragraph)})
		}
		doc.Sections = append(doc.Sections, section)
	}
	for _, item := range tei.Text.Back.ListBibl.Items {
		if title := compactText(item.Analytic.Title); title != "" {
			doc.References = append(doc.References, Reference{Title: title})
		}
	}
	if doc.Title == "" {
		doc.Warnings = append(doc.Warnings, "missing title")
	}
	return EnrichParsedDocumentModel(doc), nil
}

func compactText(value string) string {
	return strings.Join(strings.Fields(value), " ")
}
