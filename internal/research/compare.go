package research

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"
)

type CompareOptions struct {
	Dirs, Labels []string
	Out          string
}
type ComparisonPaper struct {
	Title string   `json:"title"`
	ID    string   `json:"id"`
	In    []string `json:"in"`
}
type CompareResult struct {
	Labels   []string          `json:"labels"`
	Papers   []ComparisonPaper `json:"papers"`
	Total    int               `json:"total"`
	Markdown string            `json:"-"`
}

// Compare builds the identifier-indexed union and overlap of search sessions.
func (s Service) Compare(opts CompareOptions) (CompareResult, error) {
	dirs, labels, outFile := opts.Dirs, opts.Labels, opts.Out
	if len(dirs) < 2 {
		return CompareResult{}, invalid("usage", "at least two search directories are required")
	}
	if len(labels) == 0 {
		labels = make([]string, len(dirs))
		for i, d := range dirs {
			labels[i] = filepath.Base(d)
		}
	}
	if len(labels) != len(dirs) {
		return CompareResult{}, invalid("label_count_mismatch", fmt.Sprintf("got %d labels for %d dirs", len(labels), len(dirs)))
	}

	// Read deduped results from each dir
	type dirPapers struct {
		label  string
		papers []Paper
	}
	sets := make([]dirPapers, len(dirs))
	for i, d := range dirs {
		papers, err := ReadPaperResults(d)
		if err != nil {
			return CompareResult{}, failure(fmt.Sprintf("compare_read_%d_failed", i), err)
		}
		sets[i] = dirPapers{label: labels[i], papers: papers}
	}

	// Build DOI-indexed union
	type paperInfo struct {
		Title           string
		DOI             string
		ArXivID         string
		PresentIn       []string // label indices
		AbstractPresent bool
	}
	union := map[string]*paperInfo{} // DOI -> info
	for _, s := range sets {
		for _, p := range s.papers {
			doi := p.Identifiers.DOI
			if doi == "" {
				doi = p.Identifiers.ArXivID
			}
			if doi == "" {
				continue
			}
			if _, ok := union[doi]; !ok {
				union[doi] = &paperInfo{
					Title:   p.Title,
					DOI:     p.Identifiers.DOI,
					ArXivID: p.Identifiers.ArXivID,
				}
			}
			union[doi].PresentIn = append(union[doi].PresentIn, s.label)
			if p.Abstract != "" {
				union[doi].AbstractPresent = true
			}
		}
	}

	// Build comparison table
	type compareRow struct {
		Title    string
		DOI      string
		ArXivID  string
		Matches  []bool // one per dir, in order
		UniqueIn string // which label if only in one
		Abstract bool
	}
	rows := make([]compareRow, 0, len(union))
	for _, info := range union {
		r := compareRow{
			Title:    info.Title,
			DOI:      info.DOI,
			ArXivID:  info.ArXivID,
			Matches:  make([]bool, len(dirs)),
			Abstract: info.AbstractPresent,
		}
		unique := ""
		seenCount := 0
		for i, s := range sets {
			for _, label := range info.PresentIn {
				if label == s.label {
					r.Matches[i] = true
					unique = s.label
					seenCount++
					break
				}
			}
		}
		if seenCount == 1 {
			r.UniqueIn = unique
		}
		rows = append(rows, r)
	}

	// Sort rows: overlap first (more dirs = higher), then title
	sort.Slice(rows, func(i, j int) bool {
		ci, cj := countTrue(rows[i].Matches), countTrue(rows[j].Matches)
		if ci != cj {
			return ci > cj
		}
		return rows[i].Title < rows[j].Title
	})

	// Build markdown output
	var sb strings.Builder
	sb.WriteString("# Comparison\n\n")
	for i, d := range dirs {
		fmt.Fprintf(&sb, "- **%s:** `%s` (%d papers)\n", labels[i], d, len(sets[i].papers))
	}
	sb.WriteString("\n")

	// Overlap table
	sb.WriteString("## Paper Overlap\n\n| # | Title |")
	for _, l := range labels {
		fmt.Fprintf(&sb, " %s |", l)
	}
	sb.WriteString(" ID | Abs |\n")
	sb.WriteString("|---|------|")
	for range labels {
		sb.WriteString("-----|")
	}
	sb.WriteString("-----|-----|\n")

	if len(rows) == 0 {
		sb.WriteString("| | _No overlapping identifiers_ |")
		for range labels {
			sb.WriteString(" |")
		}
		sb.WriteString(" | |\n")
	} else {
		for i, r := range rows {
			id := r.DOI
			if id == "" {
				id = r.ArXivID
			}
			absSym := ""
			if r.Abstract {
				absSym = "✓"
			}
			fmt.Fprintf(&sb, "| %d | %s |", i+1, r.Title)
			for _, m := range r.Matches {
				sym := " "
				if m {
					sym = "✓"
				}
				fmt.Fprintf(&sb, " %s |", sym)
			}
			fmt.Fprintf(&sb, " %s | %s |\n", id, absSym)
		}
	}
	sb.WriteString("\n")

	// Summary
	sb.WriteString("## Summary\n\n")
	totalOverlap := 0
	for _, r := range rows {
		if countTrue(r.Matches) == len(dirs) {
			totalOverlap++
		}
	}
	uniqueOnly := 0
	for _, r := range rows {
		if countTrue(r.Matches) == 1 {
			uniqueOnly++
		}
	}
	fmt.Fprintf(&sb, "- **Total unique papers across all dirs:** %d\n", len(rows))
	fmt.Fprintf(&sb, "- **In all directories:** %d\n", totalOverlap)
	fmt.Fprintf(&sb, "- **Unique to one directory:** %d\n", uniqueOnly)
	for i, s := range sets {
		inThis := 0
		for _, r := range rows {
			if r.Matches[i] {
				inThis++
			}
		}
		onlyIn := 0
		for _, r := range rows {
			if countTrue(r.Matches) == 1 && r.Matches[i] {
				onlyIn++
			}
		}
		fmt.Fprintf(&sb, "- **%s:** %d papers (%d shared, %d unique)\n", labels[i], len(s.papers), inThis, onlyIn)
	}

	// Gap analysis
	sb.WriteString("\n## Gaps\n\n")
	for i := range sets {
		var missingTitle []string
		for _, r := range rows {
			if !r.Matches[i] {
				missingTitle = append(missingTitle, r.Title)
				if len(missingTitle) >= 5 {
					break
				}
			}
		}
		if len(missingTitle) > 0 {
			fmt.Fprintf(&sb, "- **%s missing** from %s: ", missingTitle[0], labels[i])
			if len(missingTitle) > 1 {
				fmt.Fprintf(&sb, " (+ %d more)", len(missingTitle)-1)
			}
			sb.WriteString("\n")
		}
	}
	sb.WriteString("\n")

	result := sb.String()

	if outFile != "" {
		if err := os.MkdirAll(filepath.Dir(outFile), 0o755); err != nil {
			return CompareResult{}, failure("compare_write_failed", err)
		}
		if err := os.WriteFile(outFile, []byte(result), 0o644); err != nil {
			return CompareResult{}, failure("compare_write_failed", err)
		}
	}

	items := make([]ComparisonPaper, len(rows))
	for i, r := range rows {
		in := []string{}
		for j, m := range r.Matches {
			if m {
				in = append(in, labels[j])
			}
		}
		id := r.DOI
		if id == "" {
			id = r.ArXivID
		}
		items[i] = ComparisonPaper{Title: r.Title, ID: id, In: in}
	}
	return CompareResult{Labels: labels, Papers: items, Total: len(items), Markdown: result}, nil
}
func countTrue(vals []bool) int {
	n := 0
	for _, v := range vals {
		if v {
			n++
		}
	}
	return n
}

// ReadPaperResults reads deduped paper results from a search dir.
func ReadPaperResults(searchDir string) ([]Paper, error) {
	var papers []Paper
	data, err := os.ReadFile(filepath.Join(searchDir, "results-deduped.jsonl"))
	if err != nil {
		return nil, err
	}
	for _, line := range strings.Split(string(data), "\n") {
		line = strings.TrimSpace(line)
		if line == "" {
			continue
		}
		var p Paper
		if err := json.Unmarshal([]byte(line), &p); err == nil && p.Title != "" {
			papers = append(papers, p)
		}
	}
	return papers, nil
}
