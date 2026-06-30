package cli

import (
	"encoding/json"
	"encoding/xml"
	"fmt"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"regexp"
	"sort"
	"strings"
	"time"
)

// ── types ─────────────────────────────────────────────────────────────────────

type citationMeta struct {
	Index   int      `json:"index"`
	Title   string   `json:"title"`
	Authors []string `json:"authors"`
	Year    int      `json:"year,omitempty"`
	Journal string   `json:"journal,omitempty"`
	Volume  string   `json:"volume,omitempty"`
	Issue   string   `json:"issue,omitempty"`
	Pages   string   `json:"pages,omitempty"`
	DOI     string   `json:"doi,omitempty"`
	ArXivID string   `json:"arxivId,omitempty"`
	IsArXiv bool     `json:"isArxiv"`
	Type    string   `json:"type,omitempty"`
	Event   string   `json:"event,omitempty"`
}

// ── build ─────────────────────────────────────────────────────────────────────

func executeCitationsBuild(args []string, stdout, stderr io.Writer, opts globalOptions) int {
	researchDir := ""
	outFile := ""
	for i := 0; i < len(args); i++ {
		switch args[i] {
		case "--research-dir":
			if i+1 < len(args) {
				researchDir = args[i+1]
				i++
			}
		case "--out":
			if i+1 < len(args) {
				outFile = args[i+1]
				i++
			}
		}
	}
	if researchDir == "" {
		return writeError(stdout, stderr, opts, 2, "usage", "usage: rforge citations build --research-dir <dir> [--out <file>]")
	}
	if outFile == "" {
		outFile = filepath.Join(researchDir, "CITATIONS.md")
	}

	papers := collectDownloadedPapers(researchDir)

	crossrefBase := citEnvStr("RFORGE_CROSSREF_API_URL", "https://api.crossref.org")
	arxivBase := citEnvStr("RFORGE_ARXIV_ATOM_URL", "https://export.arxiv.org")
	delay := envDuration("RFORGE_CITATIONS_FETCH_DELAY", 200*time.Millisecond)

	arxivRe := regexp.MustCompile(`(?i)arxiv[./](.+)`)

	metas := make([]citationMeta, 0, len(papers))
	for _, p := range papers {
		var meta citationMeta
		if p.arxivID != "" || arxivRe.MatchString(p.doi) {
			arxivID := p.arxivID
			if arxivID == "" {
				if m := arxivRe.FindStringSubmatch(p.doi); m != nil {
					arxivID = m[1]
				}
			}
			meta, _ = fetchArXivCitMeta(arxivID, arxivBase)
			meta.IsArXiv = true
			meta.ArXivID = arxivID
		} else if p.doi != "" {
			meta, _ = fetchCrossrefCitMeta(p.doi, crossrefBase)
			meta.IsArXiv = false
		}
		if meta.Title == "" {
			meta.Title = p.title
		}
		meta.DOI = p.doi
		metas = append(metas, meta)
		if delay > 0 {
			time.Sleep(delay)
		}
	}

	sort.Slice(metas, func(i, j int) bool {
		ai := citFirstAuthorKey(metas[i].Authors)
		aj := citFirstAuthorKey(metas[j].Authors)
		if ai != aj {
			return ai < aj
		}
		return metas[i].Year < metas[j].Year
	})
	for i := range metas {
		metas[i].Index = i + 1
	}

	if opts.JSON {
		type jsonEntry struct {
			Index   int    `json:"index"`
			Authors string `json:"authors"`
			Title   string `json:"title"`
			Venue   string `json:"venue"`
			Year    int    `json:"year,omitempty"`
			DOI     string `json:"doi,omitempty"`
			ArXivID string `json:"arxivId,omitempty"`
		}
		entries := make([]jsonEntry, len(metas))
		for i, m := range metas {
			entries[i] = jsonEntry{
				Index:   m.Index,
				Authors: citFormatAuthors(m.Authors),
				Title:   m.Title,
				Venue:   citFormatVenue(m),
				Year:    m.Year,
				DOI:     m.DOI,
				ArXivID: m.ArXivID,
			}
		}
		return writeJSON(stdout, 0, map[string]any{"count": len(metas), "citations": entries})
	}

	var sb strings.Builder
	sb.WriteString("# Research Citations\n\n")
	sb.WriteString("All papers whose full text (PDF) was downloaded as part of this research program.\n")
	sb.WriteString("Cite these works when using any insight derived from this collection.\n")
	sb.WriteString("Papers are numbered [1]–[N] for inline reference.\n\n")
	fmt.Fprintf(&sb, "**Total:** %d papers\n\n", len(metas))
	sb.WriteString("## References\n\n")
	for _, m := range metas {
		sb.WriteString(citFormatCitation(m))
		sb.WriteString("\n\n")
	}

	if err := os.WriteFile(outFile, []byte(sb.String()), 0o644); err != nil {
		return writeError(stdout, stderr, opts, 1, "write_failed", err.Error())
	}
	fmt.Fprintf(stdout, "wrote %s (%d citations)\n", outFile, len(metas))
	return 0
}

// collectDownloadedPapers scans all topic subdirs of researchDir and returns
// papers whose PDF slug appears in <topic>/pdfs/.
type downloadedPaper struct {
	doi     string
	arxivID string
	title   string
}

func collectDownloadedPapers(researchDir string) []downloadedPaper {
	seenKeys := map[string]struct{}{}
	var papers []downloadedPaper

	entries, err := os.ReadDir(researchDir)
	if err != nil {
		return nil
	}
	for _, entry := range entries {
		if !entry.IsDir() {
			continue
		}
		topic := entry.Name()
		records, err := readResultsJSONL(filepath.Join(researchDir, topic, "results.jsonl"))
		if err != nil || len(records) == 0 {
			continue
		}
		pdfEntries, err := os.ReadDir(filepath.Join(researchDir, topic, "pdfs"))
		if err != nil {
			continue
		}
		downloaded := map[string]struct{}{}
		for _, f := range pdfEntries {
			downloaded[strings.TrimSuffix(f.Name(), ".pdf")] = struct{}{}
		}
		for _, rec := range records {
			doi := strings.TrimSpace(rec.Identifiers.DOI)
			arxivID := strings.TrimSpace(rec.Identifiers.ArXivID)
			key := doi
			if key == "" {
				key = arxivID
			}
			if key == "" {
				continue
			}
			slug := oaFetchSlug(rec)
			if _, ok := downloaded[slug]; !ok {
				continue
			}
			if _, already := seenKeys[key]; already {
				continue
			}
			seenKeys[key] = struct{}{}
			papers = append(papers, downloadedPaper{doi: doi, arxivID: arxivID, title: rec.Title})
		}
	}
	return papers
}

// ── Crossref ──────────────────────────────────────────────────────────────────

type crossrefDateField struct {
	DateParts [][]int `json:"date-parts"`
}

type crossrefCitWork struct {
	Message struct {
		Type   string   `json:"type"`
		Title  []string `json:"title"`
		Author []struct {
			Given  string `json:"given"`
			Family string `json:"family"`
		} `json:"author"`
		PublishedPrint  crossrefDateField `json:"published-print"`
		PublishedOnline crossrefDateField `json:"published-online"`
		Issued          crossrefDateField `json:"issued"`
		ContainerTitle  []string          `json:"container-title"`
		Volume          string            `json:"volume"`
		Issue           string            `json:"issue"`
		Page            string            `json:"page"`
		Event           struct {
			Name string `json:"name"`
		} `json:"event"`
	} `json:"message"`
}

func fetchCrossrefCitMeta(doi, baseURL string) (citationMeta, error) {
	req, err := http.NewRequest("GET", baseURL+"/works/"+doi, nil)
	if err != nil {
		return citationMeta{}, err
	}
	req.Header.Set("User-Agent", "ResearchForge/rforge")
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return citationMeta{}, err
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		return citationMeta{}, fmt.Errorf("crossref: status %d", resp.StatusCode)
	}
	var cr crossrefCitWork
	if err := json.NewDecoder(resp.Body).Decode(&cr); err != nil {
		return citationMeta{}, err
	}
	m := cr.Message
	var authors []string
	for _, a := range m.Author {
		if a.Family == "" {
			continue
		}
		init := ""
		if len([]rune(a.Given)) > 0 {
			init = string([]rune(a.Given)[0]) + ". "
		}
		authors = append(authors, init+a.Family)
	}
	title := ""
	if len(m.Title) > 0 {
		title = m.Title[0]
	}
	journal := ""
	if len(m.ContainerTitle) > 0 {
		journal = m.ContainerTitle[0]
	}
	year := citExtractYear(m.PublishedPrint.DateParts, m.PublishedOnline.DateParts, m.Issued.DateParts)
	return citationMeta{
		Title:   title,
		Authors: authors,
		Year:    year,
		Journal: journal,
		Volume:  m.Volume,
		Issue:   m.Issue,
		Pages:   m.Page,
		Type:    m.Type,
		Event:   m.Event.Name,
	}, nil
}

func citExtractYear(sets ...[][]int) int {
	for _, set := range sets {
		if len(set) > 0 && len(set[0]) > 0 {
			return set[0][0]
		}
	}
	return 0
}

// ── arXiv Atom API ────────────────────────────────────────────────────────────

type arxivAtomFeed struct {
	XMLName xml.Name         `xml:"feed"`
	Entries []arxivAtomEntry `xml:"entry"`
}

type arxivAtomEntry struct {
	Title     string            `xml:"title"`
	Published string            `xml:"published"`
	Authors   []arxivAtomAuthor `xml:"author"`
}

type arxivAtomAuthor struct {
	Name string `xml:"name"`
}

func fetchArXivCitMeta(arxivID, baseURL string) (citationMeta, error) {
	req, err := http.NewRequest("GET", baseURL+"/api/query?id_list="+arxivID, nil)
	if err != nil {
		return citationMeta{}, err
	}
	req.Header.Set("User-Agent", "ResearchForge/rforge")
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return citationMeta{}, err
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		return citationMeta{}, fmt.Errorf("arxiv: status %d", resp.StatusCode)
	}
	var feed arxivAtomFeed
	if err := xml.NewDecoder(resp.Body).Decode(&feed); err != nil {
		return citationMeta{}, err
	}
	if len(feed.Entries) == 0 {
		return citationMeta{}, fmt.Errorf("arxiv: no entries for %s", arxivID)
	}
	entry := feed.Entries[0]
	var authors []string
	for _, a := range entry.Authors {
		parts := strings.Fields(a.Name)
		if len(parts) >= 2 {
			family := parts[len(parts)-1]
			given := parts[0]
			init := string([]rune(given)[0]) + ". "
			authors = append(authors, init+family)
		} else if len(parts) == 1 {
			authors = append(authors, parts[0])
		}
	}
	year := 0
	if len(entry.Published) >= 4 {
		fmt.Sscanf(entry.Published[:4], "%d", &year)
	}
	return citationMeta{
		Title:   strings.TrimSpace(entry.Title),
		Authors: authors,
		Year:    year,
		Journal: "arXiv preprint",
		IsArXiv: true,
	}, nil
}

// ── formatting ────────────────────────────────────────────────────────────────

func citFormatAuthors(authors []string) string {
	if len(authors) == 0 {
		return "Author(s) unknown"
	}
	const maxAuthors = 6
	if len(authors) > maxAuthors {
		return strings.Join(authors[:maxAuthors], ", ") + " et al."
	}
	return strings.Join(authors, ", ")
}

func citFormatVenue(m citationMeta) string {
	if m.IsArXiv {
		if m.ArXivID != "" {
			return "*arXiv preprint* arXiv:" + m.ArXivID
		}
		return "*arXiv preprint*"
	}
	ctype := strings.ToLower(m.Type)
	if strings.Contains(ctype, "proceedings") || m.Event != "" {
		name := m.Event
		if name == "" {
			name = m.Journal
		}
		var parts []string
		if name != "" {
			parts = append(parts, "*"+name+"*")
		}
		if m.Pages != "" {
			parts = append(parts, "pp. "+m.Pages)
		}
		if len(parts) > 0 {
			return strings.Join(parts, ", ")
		}
		return "*conference proceedings*"
	}
	var parts []string
	if m.Journal != "" {
		parts = append(parts, "*"+m.Journal+"*")
	}
	if m.Volume != "" {
		parts = append(parts, "vol. "+m.Volume)
	}
	if m.Issue != "" {
		parts = append(parts, "no. "+m.Issue)
	}
	if m.Pages != "" {
		parts = append(parts, "pp. "+m.Pages)
	}
	if len(parts) > 0 {
		return strings.Join(parts, ", ")
	}
	return "*unpublished*"
}

func citFormatCitation(m citationMeta) string {
	authors := citFormatAuthors(m.Authors)
	year := "n.d."
	if m.Year > 0 {
		year = fmt.Sprintf("%d", m.Year)
	}
	venue := citFormatVenue(m)
	var link string
	if m.IsArXiv && m.ArXivID != "" {
		link = fmt.Sprintf(" [arXiv:%s](https://arxiv.org/abs/%s)", m.ArXivID, m.ArXivID)
	} else if m.DOI != "" {
		link = fmt.Sprintf(" [doi:%s](https://doi.org/%s)", m.DOI, m.DOI)
	}
	return fmt.Sprintf("[%d] %s, \"%s,\" %s, %s.%s", m.Index, authors, m.Title, venue, year, link)
}

func citFirstAuthorKey(authors []string) string {
	if len(authors) == 0 {
		return "zzz"
	}
	parts := strings.Fields(authors[0])
	if len(parts) == 0 {
		return "zzz"
	}
	return strings.ToLower(parts[len(parts)-1])
}

func citEnvStr(key, def string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return def
}
