package cli

import (
	"bytes"
	"encoding/csv"
	"encoding/json"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"time"

	"github.com/TrebuchetDynamics/research-forge/internal/filetxn"
	"github.com/TrebuchetDynamics/research-forge/internal/library"
)

// ── types ─────────────────────────────────────────────────────────────────────

type screenDirEntry struct {
	DOI       string `json:"doi"`
	ArXivID   string `json:"arxiv_id"`
	Decision  string `json:"decision"`
	Stage     string `json:"stage"`
	Reviewer  string `json:"reviewer"`
	Reason    string `json:"reason"`
	Timestamp string `json:"timestamp"`
}

// ── queue ─────────────────────────────────────────────────────────────────────

func executeScreenDirQueue(args []string, stdout, stderr io.Writer, opts globalOptions) int {
	dir, outFile := "", ""
	for i := 0; i < len(args)-1; i++ {
		switch args[i] {
		case "--dir":
			dir = args[i+1]
			i++
		case "--out":
			outFile = args[i+1]
			i++
		}
	}
	if dir == "" || outFile == "" {
		return writeError(stdout, stderr, opts, 2, "usage", "usage: rforge screen queue --dir <dir> --out <queue.csv>")
	}

	records, err := readResultsJSONL(filepath.Join(dir, "results.jsonl"))
	if err != nil {
		return writeError(stdout, stderr, opts, 1, "results_read_failed", fmt.Sprintf("read results.jsonl: %v", err))
	}

	decided, err := screenDirLoadDecided(dir)
	if err != nil {
		return writeError(stdout, stderr, opts, 1, "screening_read_failed", err.Error())
	}

	mode := os.FileMode(0o644)
	if info, err := os.Lstat(outFile); err == nil {
		if !info.Mode().IsRegular() {
			return writeError(stdout, stderr, opts, 1, "csv_create_failed", fmt.Sprintf("queue output is not a regular file: %s", outFile))
		}
		mode = info.Mode().Perm()
	} else if !os.IsNotExist(err) {
		return writeError(stdout, stderr, opts, 1, "csv_create_failed", err.Error())
	}
	var payload bytes.Buffer
	w := csv.NewWriter(&payload)

	if err := w.Write([]string{"doi", "arxiv_id", "title", "authors", "year", "abstract", "source", "decision", "reason"}); err != nil {
		return writeError(stdout, stderr, opts, 1, "csv_write_failed", err.Error())
	}

	pending := 0
	for _, rec := range records {
		doi := strings.TrimSpace(rec.Identifiers.DOI)
		arxivID := strings.TrimSpace(rec.Identifiers.ArXivID)
		key := screenDirRecordKey(rec)
		if key == "" || decided[key] {
			continue
		}
		year := ""
		if rec.Year > 0 {
			year = fmt.Sprintf("%d", rec.Year)
		}
		source := ""
		if len(rec.SourceRefs) > 0 {
			source = rec.SourceRefs[0].Source
		}
		if err := w.Write([]string{doi, arxivID, rec.Title, screenDirFormatAuthors(rec), year, rec.Abstract, source, "", ""}); err != nil {
			return writeError(stdout, stderr, opts, 1, "csv_write_failed", err.Error())
		}
		pending++
	}
	w.Flush()
	if err := w.Error(); err != nil {
		return writeError(stdout, stderr, opts, 1, "csv_flush_failed", err.Error())
	}
	if err := filetxn.Replace(outFile, payload.Bytes(), mode); err != nil {
		return writeError(stdout, stderr, opts, 1, "csv_replace_failed", err.Error())
	}

	fmt.Fprintf(stdout, "wrote %s (%d pending records)\n", outFile, pending)
	return 0
}

// ── import ────────────────────────────────────────────────────────────────────

func executeScreenDirImport(args []string, stdout, stderr io.Writer, opts globalOptions) int {
	dir, csvPath, reviewer := "", "", ""
	for i := 0; i < len(args)-1; i++ {
		switch args[i] {
		case "--dir":
			dir = args[i+1]
			i++
		case "--csv":
			csvPath = args[i+1]
			i++
		case "--reviewer":
			reviewer = args[i+1]
			i++
		}
	}
	if dir == "" || csvPath == "" {
		return writeError(stdout, stderr, opts, 2, "usage", "usage: rforge screen import --dir <dir> --csv <queue.csv> [--reviewer <name>]")
	}

	body, err := os.ReadFile(csvPath)
	if err != nil {
		return writeError(stdout, stderr, opts, 1, "csv_open_failed", err.Error())
	}
	// Spreadsheet exports (Excel/Numbers, CSV UTF-8) prefix the file with a BOM.
	// Left in place it becomes part of the first header cell, so the doi column is
	// never found and DOI decisions are silently dropped on import.
	body = bytes.TrimPrefix(body, []byte("\xef\xbb\xbf"))
	rows, err := csv.NewReader(bytes.NewReader(body)).ReadAll()
	if err != nil {
		return writeError(stdout, stderr, opts, 1, "csv_parse_failed", err.Error())
	}
	if len(rows) < 2 {
		fmt.Fprintln(stdout, "imported 0 decisions")
		return 0
	}

	header := rows[0]
	colIdx := func(col string) int {
		for i, h := range header {
			// Tolerate stray whitespace/BOM on header cells from spreadsheet edits.
			if strings.TrimSpace(strings.TrimPrefix(h, "\ufeff")) == col {
				return i
			}
		}
		return -1
	}
	doiIdx := colIdx("doi")
	arxivIdx := colIdx("arxiv_id")
	decIdx := colIdx("decision")
	reasonIdx := colIdx("reason")

	// Load existing entries then overwrite per-identifier (last-write-wins).
	existing, err := screenDirLoadEntries(dir)
	if err != nil {
		return writeError(stdout, stderr, opts, 1, "screening_read_failed", err.Error())
	}
	byKey := map[string]screenDirEntry{}
	for _, e := range existing {
		k := e.DOI
		if k == "" {
			k = e.ArXivID
		}
		if k != "" {
			byKey[k] = e
		}
	}

	imported := 0
	now := time.Now().UTC().Format(time.RFC3339)
	for rowIndex, row := range rows[1:] {
		if decIdx < 0 || decIdx >= len(row) {
			continue
		}
		decision := strings.TrimSpace(row[decIdx])
		if decision == "" {
			continue
		}
		// Reject unknown decisions instead of persisting them: progress would
		// never count them while queue would treat the paper as decided, so the
		// paper would silently leave the review queue for good.
		canonical, ok := canonicalScreenDirDecision(decision)
		if !ok {
			return writeError(stdout, stderr, opts, 1, "csv_invalid_decision",
				fmt.Sprintf("csv row %d: unknown decision %q (want include, exclude, or uncertain)", rowIndex+2, decision))
		}
		decision = canonical
		doi, arxivID := "", ""
		if doiIdx >= 0 && doiIdx < len(row) {
			doi = strings.TrimSpace(row[doiIdx])
		}
		if arxivIdx >= 0 && arxivIdx < len(row) {
			arxivID = strings.TrimSpace(row[arxivIdx])
		}
		key := doi
		if key == "" {
			key = arxivID
		}
		if key == "" {
			continue
		}
		reason := ""
		if reasonIdx >= 0 && reasonIdx < len(row) {
			reason = strings.TrimSpace(row[reasonIdx])
		}
		byKey[key] = screenDirEntry{
			DOI:       doi,
			ArXivID:   arxivID,
			Decision:  decision,
			Stage:     "title_abstract",
			Reviewer:  reviewer,
			Reason:    reason,
			Timestamp: now,
		}
		imported++
	}

	if err := screenDirWriteEntries(dir, byKey); err != nil {
		return writeError(stdout, stderr, opts, 1, "screening_write_failed", err.Error())
	}
	fmt.Fprintf(stdout, "imported %d decisions into %s\n", imported, filepath.Join(dir, "screening.jsonl"))
	return 0
}

// ── progress ──────────────────────────────────────────────────────────────────

func executeScreenDirProgress(args []string, stdout, stderr io.Writer, opts globalOptions) int {
	dir := ""
	for i := 0; i < len(args)-1; i++ {
		if args[i] == "--dir" {
			dir = args[i+1]
			i++
		}
	}
	if dir == "" {
		return writeError(stdout, stderr, opts, 2, "usage", "usage: rforge screen progress --dir <dir>")
	}

	records, err := readResultsJSONL(filepath.Join(dir, "results.jsonl"))
	if err != nil {
		return writeError(stdout, stderr, opts, 1, "results_read_failed", fmt.Sprintf("read results.jsonl: %v", err))
	}
	totalRecords := len(records)

	entries, err := screenDirLoadEntries(dir)
	if err != nil {
		return writeError(stdout, stderr, opts, 1, "screening_read_failed", err.Error())
	}
	included, excluded, uncertain := 0, 0, 0
	for _, e := range entries {
		switch strings.ToLower(e.Decision) {
		case "include":
			included++
		case "exclude":
			excluded++
		case "uncertain":
			uncertain++
		}
	}
	decided := included + excluded + uncertain
	pending := totalRecords - decided
	if pending < 0 {
		pending = 0
	}

	decidedSet := screenDirDecidedSet(entries)
	queuePending, missingIdentifier := 0, 0
	for _, rec := range records {
		key := screenDirRecordKey(rec)
		if key == "" {
			missingIdentifier++
		} else if !decidedSet[key] {
			queuePending++
		}
	}

	if opts.JSON {
		return writeJSON(stdout, 0, map[string]any{
			"total":              totalRecords,
			"decided":            decided,
			"included":           included,
			"excluded":           excluded,
			"uncertain":          uncertain,
			"pending":            pending,
			"queue_pending":      queuePending,
			"missing_identifier": missingIdentifier,
		})
	}
	fmt.Fprintf(stdout, "total: %d  decided: %d  pending: %d\n", totalRecords, decided, pending)
	fmt.Fprintf(stdout, "  include: %d  exclude: %d  uncertain: %d\n", included, excluded, uncertain)
	fmt.Fprintf(stdout, "  queue_pending: %d  missing_identifier: %d\n", queuePending, missingIdentifier)
	return 0
}

// ── helpers ───────────────────────────────────────────────────────────────────

func screenDirLoadEntries(dir string) ([]screenDirEntry, error) {
	path := filepath.Join(dir, "screening.jsonl")
	info, err := os.Lstat(path)
	if os.IsNotExist(err) {
		return []screenDirEntry{}, nil
	}
	if err != nil {
		return nil, err
	}
	if !info.Mode().IsRegular() {
		return nil, fmt.Errorf("screening log is not a regular file: %s", path)
	}
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}
	var entries []screenDirEntry
	for index, line := range strings.Split(string(data), "\n") {
		line = strings.TrimSpace(line)
		if line == "" {
			continue
		}
		var e screenDirEntry
		if err := json.Unmarshal([]byte(line), &e); err != nil {
			return nil, fmt.Errorf("decode screening log line %d: %w", index+1, err)
		}
		if strings.TrimSpace(e.DOI) == "" && strings.TrimSpace(e.ArXivID) == "" {
			return nil, fmt.Errorf("decode screening log line %d: DOI or arXiv ID is required", index+1)
		}
		entries = append(entries, e)
	}
	return entries, nil
}

// canonicalScreenDirDecision reports whether value is a decision this workflow
// understands and returns its canonical lowercase form.
func canonicalScreenDirDecision(value string) (string, bool) {
	switch lowered := strings.ToLower(strings.TrimSpace(value)); lowered {
	case "include", "exclude", "uncertain":
		return lowered, true
	}
	return "", false
}

// screenDirRecordKey matches queue eligibility: trimmed DOI first, then arXiv.
func screenDirRecordKey(rec library.PaperRecord) string {
	if doi := strings.TrimSpace(rec.Identifiers.DOI); doi != "" {
		return doi
	}
	return strings.TrimSpace(rec.Identifiers.ArXivID)
}

func screenDirLoadDecided(dir string) (map[string]bool, error) {
	entries, err := screenDirLoadEntries(dir)
	if err != nil {
		return nil, err
	}
	return screenDirDecidedSet(entries), nil
}

// screenDirDecidedSet preserves the queue's historical key and decision rules.
func screenDirDecidedSet(entries []screenDirEntry) map[string]bool {
	decided := map[string]bool{}
	for _, e := range entries {
		k := e.DOI
		if k == "" {
			k = e.ArXivID
		}
		if k != "" && e.Decision != "" {
			decided[k] = true
		}
	}
	return decided
}

func screenDirWriteEntries(dir string, byKey map[string]screenDirEntry) error {
	path := filepath.Join(dir, "screening.jsonl")
	mode := os.FileMode(0o644)
	if info, err := os.Lstat(path); err == nil {
		if !info.Mode().IsRegular() {
			return fmt.Errorf("screening log is not a regular file: %s", path)
		}
		mode = info.Mode().Perm()
	} else if !os.IsNotExist(err) {
		return err
	}
	var payload bytes.Buffer
	enc := json.NewEncoder(&payload)
	keys := make([]string, 0, len(byKey))
	for key := range byKey {
		keys = append(keys, key)
	}
	sort.Strings(keys)
	for _, key := range keys {
		if err := enc.Encode(byKey[key]); err != nil {
			return err
		}
	}
	return filetxn.Replace(path, payload.Bytes(), mode)
}

func screenDirFormatAuthors(rec library.PaperRecord) string {
	names := make([]string, 0, len(rec.Authors))
	for _, a := range rec.Authors {
		if a.Family != "" {
			init := ""
			if len([]rune(a.Given)) > 0 {
				init = string([]rune(a.Given)[0]) + ". "
			}
			names = append(names, init+a.Family)
		} else if a.Given != "" {
			names = append(names, a.Given)
		}
	}
	return strings.Join(names, ", ")
}

// screenDirHasFlag reports whether flag appears in args.
func screenDirHasFlag(args []string, flag string) bool {
	for _, a := range args {
		if a == flag {
			return true
		}
	}
	return false
}
