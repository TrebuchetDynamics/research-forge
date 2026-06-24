package research

import (
	"encoding/csv"
	"encoding/json"
	"fmt"
	"html"
	"io"
	"os"
	"path/filepath"
	"regexp"
	"sort"
	"strings"

	"github.com/TrebuchetDynamics/research-forge/internal/library"
	"github.com/TrebuchetDynamics/research-forge/internal/parsing"
)

// ParsedTextDocument converts extracted paper text into RForge parsed-document JSON.
func ParsedTextDocument(paperID, title, text string, chunkSize int) parsing.ParsedDocument {
	if chunkSize <= 0 {
		chunkSize = 1400
	}
	passages := chunkText(paperID, text, chunkSize)
	return parsing.ParsedDocument{
		SchemaVersion: "1",
		PaperID:       strings.TrimSpace(paperID),
		ParserName:    "pdftotext-sectionizer",
		ParserVersion: "native-rforge",
		Title:         strings.TrimSpace(title),
		Sections: []parsing.Section{{
			ID:       "full-text",
			Title:    "Full text extracted by pdftotext",
			Passages: passages,
		}},
		Warnings: []string{
			"Lightweight text extraction only; not GROBID/PaperMage-quality structure.",
			"Evidence snippets are keyword-based triage and require human verification.",
		},
	}
}

func chunkText(paperID, text string, chunkSize int) []parsing.Passage {
	parts := regexp.MustCompile(`\n\s*\n`).Split(normalizeNewlines(text), -1)
	passages := []parsing.Passage{}
	var b strings.Builder
	flush := func() {
		body := strings.TrimSpace(b.String())
		if len(body) < 40 {
			b.Reset()
			return
		}
		id := fmt.Sprintf("p%04d", len(passages)+1)
		passages = append(passages, parsing.Passage{ID: id, PaperID: paperID, SectionID: "full-text", Text: body})
		b.Reset()
	}
	for _, part := range parts {
		part = strings.TrimSpace(part)
		if len(part) < 40 {
			continue
		}
		if b.Len() > 0 && b.Len()+len(part)+2 > chunkSize {
			flush()
		}
		if b.Len() > 0 {
			b.WriteString("\n\n")
		}
		b.WriteString(part)
	}
	flush()
	return passages
}

func normalizeNewlines(text string) string {
	text = strings.ReplaceAll(text, "\f", "\n")
	text = regexp.MustCompile(`[ \t]+`).ReplaceAllString(text, " ")
	text = regexp.MustCompile(`\n{3,}`).ReplaceAllString(text, "\n\n")
	return strings.TrimSpace(text)
}

// ScreeningRecord is one deterministic first-pass relevance decision.
type ScreeningRecord struct {
	Score             int      `json:"score"`
	Decision          string   `json:"decision"`
	MatchedGroups     []string `json:"matchedGroups"`
	NegativeTerms     []string `json:"negativeTerms"`
	Title             string   `json:"title"`
	Year              int      `json:"year,omitempty"`
	DOI               string   `json:"doi,omitempty"`
	ArXivID           string   `json:"arxivId,omitempty"`
	OpenAlexID        string   `json:"openAlexId,omitempty"`
	SemanticScholarID string   `json:"semanticScholarId,omitempty"`
	OpenAccess        bool     `json:"openAccess,omitempty"`
	Sources           []string `json:"sources"`
}

type sourceSearchPayload struct {
	Data struct {
		Papers []library.PaperRecord `json:"papers"`
	} `json:"data"`
}

var positiveScreeningPatterns = map[string][]*regexp.Regexp{
	"direct_lightgbm_crypto": compilePatterns(`lightgbm`, `xgboost`, `gradient boost`, `btc(?:usdt)?`, `bitcoin`, `cryptocurrency`),
	"hf_direction":           compilePatterns(`high.frequency`, `minute`, `intraday`, `mid.price`, `trend prediction`, `price movement`, `direction`),
	"microstructure":         compilePatterns(`order book`, `order flow`, `imbalance`, `spread`, `liquidity`, `market microstructure`),
	"cross_asset":            compilePatterns(`lead.lag`, `altcoin`, `cross cryptocurrency`, `interconnected`, `correlation network`, `spillover`),
	"validation_regime":      compilePatterns(`walk.forward`, `out.of.sample`, `changing market`, `regime`, `forecasting`),
}

var negativeScreeningPatterns = compilePatterns(`malware`, `diabetes`, `landslide`, `health`, `fake news`, `ransomware`, `legal`, `law enforcement`)

// BuildScreeningQueue scores library/search-result records for first-pass review.
func BuildScreeningQueue(libraryPath, searchResultsDir string) ([]ScreeningRecord, error) {
	records := map[string]library.PaperRecord{}
	sources := map[string]map[string]bool{}
	if strings.TrimSpace(libraryPath) != "" {
		papers, err := readPaperArray(libraryPath)
		if err != nil {
			return nil, err
		}
		mergeRecords(records, sources, papers, filepath.Base(libraryPath))
	}
	if strings.TrimSpace(searchResultsDir) != "" {
		entries, err := os.ReadDir(searchResultsDir)
		if err != nil {
			return nil, err
		}
		for _, entry := range entries {
			if entry.IsDir() || filepath.Ext(entry.Name()) != ".json" {
				continue
			}
			path := filepath.Join(searchResultsDir, entry.Name())
			papers, err := readSearchPayload(path)
			if err != nil {
				return nil, err
			}
			mergeRecords(records, sources, papers, entry.Name())
		}
	}
	queue := make([]ScreeningRecord, 0, len(records))
	for key, paper := range records {
		score, groups, negatives := scorePaper(paper)
		decision := "exclude-noise"
		if score >= 5 {
			decision = "include-review"
		} else if score >= 2 {
			decision = "maybe"
		}
		queue = append(queue, ScreeningRecord{
			Score:             score,
			Decision:          decision,
			MatchedGroups:     groups,
			NegativeTerms:     negatives,
			Title:             paper.Title,
			Year:              paper.Year,
			DOI:               paper.Identifiers.DOI,
			ArXivID:           paper.Identifiers.ArXivID,
			OpenAlexID:        paper.Identifiers.OpenAlexID,
			SemanticScholarID: paper.Identifiers.SemanticScholarID,
			OpenAccess:        paper.OpenAccess,
			Sources:           sortedSet(sources[key]),
		})
	}
	sort.Slice(queue, func(i, j int) bool {
		if queue[i].Score != queue[j].Score {
			return queue[i].Score > queue[j].Score
		}
		return strings.ToLower(queue[i].Title) < strings.ToLower(queue[j].Title)
	})
	return queue, nil
}

func readPaperArray(path string) ([]library.PaperRecord, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}
	var papers []library.PaperRecord
	if err := json.Unmarshal(data, &papers); err != nil {
		return nil, fmt.Errorf("read library records: %w", err)
	}
	return papers, nil
}

func readSearchPayload(path string) ([]library.PaperRecord, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}
	var payload sourceSearchPayload
	if err := json.Unmarshal(data, &payload); err != nil {
		return nil, fmt.Errorf("read search payload %s: %w", path, err)
	}
	return payload.Data.Papers, nil
}

func mergeRecords(records map[string]library.PaperRecord, sources map[string]map[string]bool, papers []library.PaperRecord, source string) {
	for _, paper := range papers {
		key := paperKey(paper)
		if key == "" {
			continue
		}
		existing, ok := records[key]
		if !ok || (existing.Abstract == "" && paper.Abstract != "") {
			records[key] = paper
		}
		if sources[key] == nil {
			sources[key] = map[string]bool{}
		}
		sources[key][source] = true
	}
}

func paperKey(paper library.PaperRecord) string {
	ids := paper.Identifiers
	for _, value := range []string{ids.DOI, ids.ArXivID, ids.OpenAlexID, ids.SemanticScholarID, paper.Title} {
		if strings.TrimSpace(value) != "" {
			return strings.ToLower(strings.TrimSpace(value))
		}
	}
	return ""
}

func scorePaper(paper library.PaperRecord) (int, []string, []string) {
	text := strings.ToLower(paper.Title + " " + paper.Abstract)
	score := 0
	groups := []string{}
	for group, patterns := range positiveScreeningPatterns {
		hits := countPatternHits(text, patterns)
		if hits > 0 {
			groups = append(groups, group)
			score += hits
		}
	}
	sort.Strings(groups)
	negatives := []string{}
	for _, pattern := range negativeScreeningPatterns {
		if pattern.MatchString(text) {
			negatives = append(negatives, pattern.String())
		}
	}
	score -= 4 * len(negatives)
	if paper.Identifiers.ArXivID != "" {
		score++
	}
	if paper.OpenAccess {
		score++
	}
	return score, groups, negatives
}

func countPatternHits(text string, patterns []*regexp.Regexp) int {
	hits := 0
	for _, pattern := range patterns {
		if pattern.MatchString(text) {
			hits++
		}
	}
	return hits
}

func compilePatterns(patterns ...string) []*regexp.Regexp {
	compiled := make([]*regexp.Regexp, 0, len(patterns))
	for _, pattern := range patterns {
		compiled = append(compiled, regexp.MustCompile(`(?i)`+pattern))
	}
	return compiled
}

func sortedSet(values map[string]bool) []string {
	out := make([]string, 0, len(values))
	for value := range values {
		out = append(out, value)
	}
	sort.Strings(out)
	return out
}

// WriteScreeningCSV writes queue rows in a review-friendly tabular form.
func WriteScreeningCSV(w io.Writer, queue []ScreeningRecord) error {
	writer := csv.NewWriter(w)
	defer writer.Flush()
	if err := writer.Write([]string{"score", "decision", "matched_groups", "negative_terms", "title", "year", "doi", "arxiv_id", "openalex_id", "semantic_scholar_id", "open_access", "sources"}); err != nil {
		return err
	}
	for _, row := range queue {
		if err := writer.Write([]string{fmt.Sprint(row.Score), row.Decision, strings.Join(row.MatchedGroups, ";"), strings.Join(row.NegativeTerms, ";"), row.Title, fmt.Sprint(row.Year), row.DOI, row.ArXivID, row.OpenAlexID, row.SemanticScholarID, fmt.Sprint(row.OpenAccess), strings.Join(row.Sources, ";")}); err != nil {
			return err
		}
	}
	return writer.Error()
}

// ScreeningMarkdown renders a deterministic top-N review queue.
func ScreeningMarkdown(queue []ScreeningRecord, limit int) string {
	if limit <= 0 || limit > len(queue) {
		limit = len(queue)
	}
	var b strings.Builder
	b.WriteString("# Structured screening queue\n\n")
	b.WriteString("Deterministic keyword triage over RForge library/search-result records. Manual review is still required.\n\n")
	b.WriteString("| Rank | Score | Decision | Paper | Matched groups | DOI/arXiv |\n")
	b.WriteString("|---:|---:|---|---|---|---|\n")
	for i := 0; i < limit; i++ {
		row := queue[i]
		identifier := firstNonEmpty(row.DOI, row.ArXivID, row.OpenAlexID, row.SemanticScholarID)
		fmt.Fprintf(&b, "| %d | %d | %s | %s (%d) | %s | `%s` |\n", i+1, row.Score, row.Decision, escapeMarkdown(row.Title), row.Year, strings.Join(row.MatchedGroups, ";"), identifier)
	}
	b.WriteString("\n## Queue counts\n\n")
	counts := map[string]int{}
	for _, row := range queue {
		counts[row.Decision]++
	}
	for _, decision := range []string{"include-review", "maybe", "exclude-noise"} {
		fmt.Fprintf(&b, "- %s: %d\n", decision, counts[decision])
	}
	return b.String()
}

// LeakageAuditRow captures keyword evidence for one parsed paper.
type LeakageAuditRow struct {
	PaperID             string          `json:"paperId"`
	Title               string          `json:"title"`
	Passages            int             `json:"passages"`
	ValidationEvidence  []string        `json:"validationEvidence"`
	FeatureEvidence     []string        `json:"featureEvidence"`
	HorizonDataEvidence []string        `json:"horizonDataEvidence"`
	LeakageRiskEvidence []string        `json:"leakageRiskEvidence"`
	LeakageFlags        map[string]bool `json:"leakageFlags"`
	TriageRisk          string          `json:"triageRisk"`
	RecommendedUse      string          `json:"recommendedUse"`
}

var evidencePatterns = map[string][]*regexp.Regexp{
	"validation":   compilePatterns(`walk[- ]?forward`, `out[- ]of[- ]sample`, `train(?:ing)?`, `test(?:ing)?`, `cross[- ]validation`, `validation`, `holdout`, `forecast`),
	"leakage_risk": compilePatterns(`random`, `shuffle`, `normalization`, `standardi[sz]`, `future`, `look[- ]?ahead`, `in[- ]sample`, `data snoop`, `overfit`),
	"features":     compilePatterns(`feature`, `technical indicator`, `return`, `volatility`, `volume`, `order book`, `order flow`, `imbalance`, `spread`, `liquidity`, `correlation`, `lead[- ]lag`, `GARCH`, `entropy`),
	"horizon_data": compilePatterns(`minute`, `5[- ]?min`, `five[- ]minute`, `high[- ]frequency`, `intraday`, `tick`, `hourly`, `daily`, `Bitcoin`, `cryptocurr`),
}

var leakageFlagPatterns = map[string][]*regexp.Regexp{
	"random_split_or_shuffle_mentions": compilePatterns(`random(?:ly)? split`, `random forest`, `shuffle`),
	"global_preprocessing_possible":    compilePatterns(`normalization`, `standardi[sz]ation`, `scal(?:e|ing)`),
	"non_temporal_cv_possible":         compilePatterns(`cross[- ]validation`, `k[- ]fold`),
	"lookahead_language":               compilePatterns(`future`, `look[- ]?ahead`),
	"overfit_data_snooping_language":   compilePatterns(`overfit`, `data snoop`, `in[- ]sample`),
}

// BuildLeakageAudit extracts conservative keyword evidence from parsed documents.
func BuildLeakageAudit(parsedDir string) ([]LeakageAuditRow, error) {
	entries, err := os.ReadDir(parsedDir)
	if err != nil {
		return nil, err
	}
	docs := []parsing.ParsedDocument{}
	for _, entry := range entries {
		if entry.IsDir() || filepath.Ext(entry.Name()) != ".json" {
			continue
		}
		data, err := os.ReadFile(filepath.Join(parsedDir, entry.Name()))
		if err != nil {
			return nil, err
		}
		var doc parsing.ParsedDocument
		if err := json.Unmarshal(data, &doc); err != nil {
			return nil, err
		}
		docs = append(docs, doc)
	}
	return buildLeakageAuditRows(docs), nil
}

// BuildLeakageAuditFromTextDir extracts conservative keyword evidence from plain text files.
func BuildLeakageAuditFromTextDir(textDir string, chunkSize int) ([]LeakageAuditRow, error) {
	entries, err := os.ReadDir(textDir)
	if err != nil {
		return nil, err
	}
	docs := []parsing.ParsedDocument{}
	for _, entry := range entries {
		if entry.IsDir() || filepath.Ext(entry.Name()) != ".txt" {
			continue
		}
		path := filepath.Join(textDir, entry.Name())
		data, err := os.ReadFile(path)
		if err != nil {
			return nil, err
		}
		text := string(data)
		paperID := strings.TrimSuffix(entry.Name(), filepath.Ext(entry.Name()))
		docs = append(docs, ParsedTextDocument(paperID, titleFromText(text, paperID), text, chunkSize))
	}
	return buildLeakageAuditRows(docs), nil
}

func buildLeakageAuditRows(docs []parsing.ParsedDocument) []LeakageAuditRow {
	rows := []LeakageAuditRow{}
	for _, doc := range docs {
		text, passages := documentText(doc)
		flags := map[string]bool{}
		for name, patterns := range leakageFlagPatterns {
			flags[name] = len(snippets(text, patterns, 1)) > 0
		}
		validation := snippets(text, evidencePatterns["validation"], 5)
		rows = append(rows, LeakageAuditRow{
			PaperID:             doc.PaperID,
			Title:               doc.Title,
			Passages:            passages,
			ValidationEvidence:  validation,
			FeatureEvidence:     snippets(text, evidencePatterns["features"], 5),
			HorizonDataEvidence: snippets(text, evidencePatterns["horizon_data"], 5),
			LeakageRiskEvidence: snippets(text, evidencePatterns["leakage_risk"], 5),
			LeakageFlags:        flags,
			TriageRisk:          triageRisk(flags, validation),
			RecommendedUse:      "Use as citation/idea source only until validation design is manually confirmed.",
		})
	}
	sort.Slice(rows, func(i, j int) bool { return rows[i].PaperID < rows[j].PaperID })
	return rows
}

func titleFromText(text, fallback string) string {
	for _, line := range strings.Split(normalizeNewlines(text), "\n") {
		line = strings.TrimSpace(line)
		if line == "" {
			continue
		}
		return shorten(line, 160)
	}
	return fallback
}

func documentText(doc parsing.ParsedDocument) (string, int) {
	var b strings.Builder
	count := 0
	for _, section := range doc.Sections {
		b.WriteString(section.Title)
		b.WriteString("\n")
		for _, passage := range section.Passages {
			count++
			b.WriteString(passage.Text)
			b.WriteString("\n\n")
		}
	}
	return b.String(), count
}

func snippets(text string, patterns []*regexp.Regexp, limit int) []string {
	out := []string{}
	for _, pattern := range patterns {
		matches := pattern.FindAllStringIndex(text, -1)
		for _, match := range matches {
			start := match[0] - 220
			if start < 0 {
				start = 0
			}
			end := match[1] + 220
			if end > len(text) {
				end = len(text)
			}
			snippet := normalizeWhitespace(text[start:end])
			if snippet != "" && !hasSimilarSnippet(out, snippet) {
				out = append(out, shorten(snippet, 520))
			}
			if len(out) >= limit {
				return out
			}
		}
	}
	return out
}

func normalizeWhitespace(value string) string {
	return strings.Join(strings.Fields(value), " ")
}

func hasSimilarSnippet(snippets []string, candidate string) bool {
	prefix := candidate
	if len(prefix) > 120 {
		prefix = prefix[:120]
	}
	for _, existing := range snippets {
		if strings.HasPrefix(existing, prefix) {
			return true
		}
	}
	return false
}

func shorten(value string, limit int) string {
	if len(value) <= limit {
		return value
	}
	if limit <= 4 {
		return value[:limit]
	}
	return value[:limit-4] + " ..."
}

func triageRisk(flags map[string]bool, validation []string) string {
	points := 0
	for _, value := range flags {
		if value {
			points++
		}
	}
	if len(validation) == 0 {
		points++
	}
	if points >= 3 {
		return "high/manual-review"
	}
	if points >= 1 {
		return "medium/manual-review"
	}
	return "low-from-keywords/manual-review-still-required"
}

// LeakageAuditMarkdown renders a paper-level leakage/evidence table and snippets.
func LeakageAuditMarkdown(rows []LeakageAuditRow) string {
	var b strings.Builder
	b.WriteString("# Leakage-risk and feature-evidence triage\n\n")
	b.WriteString("Generated from parsed RForge documents using conservative keyword extraction. This is a screening aid, not a substitute for manual paper review.\n\n")
	b.WriteString("| Paper | Passages | Triage risk | Validation evidence? | Feature evidence? | Leakage-risk terms? |\n")
	b.WriteString("|---|---:|---|---:|---:|---:|\n")
	for _, row := range rows {
		fmt.Fprintf(&b, "| `%s` %s | %d | %s | %d | %d | %d |\n", row.PaperID, escapeMarkdown(row.Title), row.Passages, row.TriageRisk, len(row.ValidationEvidence), len(row.FeatureEvidence), len(row.LeakageRiskEvidence))
	}
	b.WriteString("\n## Paper evidence snippets\n\n")
	for _, row := range rows {
		fmt.Fprintf(&b, "### %s — %s\n\n", row.PaperID, row.Title)
		fmt.Fprintf(&b, "- Triage risk: **%s**\n", row.TriageRisk)
		flagJSON, _ := json.Marshal(row.LeakageFlags)
		fmt.Fprintf(&b, "- Leakage flags: `%s`\n", flagJSON)
		writeSnippetList(&b, "Validation/design evidence", row.ValidationEvidence)
		writeSnippetList(&b, "Feature/microstructure evidence", row.FeatureEvidence)
		writeSnippetList(&b, "Horizon/data evidence", row.HorizonDataEvidence)
		writeSnippetList(&b, "Leakage-risk terms", row.LeakageRiskEvidence)
		b.WriteString("\n")
	}
	return b.String()
}

func writeSnippetList(b *strings.Builder, title string, snippets []string) {
	fmt.Fprintf(b, "- %s:\n", title)
	if len(snippets) == 0 {
		b.WriteString("  - Not found by keyword triage.\n")
		return
	}
	limit := len(snippets)
	if limit > 3 {
		limit = 3
	}
	for i := 0; i < limit; i++ {
		fmt.Fprintf(b, "  - %s\n", snippets[i])
	}
}

func firstNonEmpty(values ...string) string {
	for _, value := range values {
		if strings.TrimSpace(value) != "" {
			return value
		}
	}
	return ""
}

func escapeMarkdown(value string) string {
	return strings.ReplaceAll(html.EscapeString(value), "|", "\\|")
}
