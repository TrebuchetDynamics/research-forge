package cli

import (
	"fmt"
	"io"
	"path/filepath"
	"strconv"
	"time"

	"github.com/TrebuchetDynamics/research-forge/internal/library"
	"github.com/TrebuchetDynamics/research-forge/internal/provenance"
)

func executeImport(args []string, stdout, stderr io.Writer, opts globalOptions) int {
	if len(args) != 2 || !supportedImportExportFormat(args[0]) {
		return writeError(stdout, stderr, opts, 2, "usage", "usage: rforge --project <path> import <json|csv|bibtex|ris> <file>")
	}
	if opts.Project == "" {
		return writeError(stdout, stderr, opts, 2, "missing_project", "--project is required for import commands")
	}
	var records []library.PaperRecord
	var err error
	switch args[0] {
	case "csv":
		records, err = library.ImportCSV(args[1])
	case "bibtex":
		records, err = library.ImportBibTeX(args[1])
	case "ris":
		records, err = library.ImportRIS(args[1])
	default:
		records, err = library.ImportJSON(args[1])
	}
	if err != nil {
		return writeError(stdout, stderr, opts, 1, "import_failed", fmt.Sprintf("import %s: %v", args[0], err))
	}
	store, err := library.OpenStore(filepath.Join(opts.Project, "data", "library.json"))
	if err != nil {
		return writeError(stdout, stderr, opts, 1, "library_open_failed", fmt.Sprintf("open library: %v", err))
	}
	for _, record := range records {
		if err := store.Create(record); err != nil {
			return writeError(stdout, stderr, opts, 1, "import_store_failed", fmt.Sprintf("store imported record: %v", err))
		}
	}
	if opts.JSON {
		return writeJSON(stdout, 0, map[string]any{"imported": len(records)})
	}
	fmt.Fprintf(stdout, "imported %d records\n", len(records))
	return 0
}

func supportedImportExportFormat(format string) bool {
	return format == "json" || format == "csv" || format == "bibtex" || format == "ris"
}

func executeExport(args []string, stdout, stderr io.Writer, opts globalOptions) int {
	if len(args) != 2 || !supportedImportExportFormat(args[0]) {
		return writeError(stdout, stderr, opts, 2, "usage", "usage: rforge --project <path> export <json|csv|bibtex|ris> <file>")
	}
	if opts.Project == "" {
		return writeError(stdout, stderr, opts, 2, "missing_project", "--project is required for export commands")
	}
	store, err := library.OpenStore(filepath.Join(opts.Project, "data", "library.json"))
	if err != nil {
		return writeError(stdout, stderr, opts, 1, "library_open_failed", fmt.Sprintf("open library: %v", err))
	}
	records, err := store.List()
	if err != nil {
		return writeError(stdout, stderr, opts, 1, "library_list_failed", fmt.Sprintf("list library: %v", err))
	}
	switch args[0] {
	case "csv":
		err = library.ExportCSV(args[1], records)
	case "bibtex":
		err = library.ExportBibTeX(args[1], records)
	case "ris":
		err = library.ExportRIS(args[1], records)
	default:
		err = library.ExportJSON(args[1], records)
	}
	if err != nil {
		return writeError(stdout, stderr, opts, 1, "export_failed", fmt.Sprintf("export %s: %v", args[0], err))
	}
	if opts.JSON {
		return writeJSON(stdout, 0, map[string]any{"exported": len(records), "path": args[1]})
	}
	fmt.Fprintf(stdout, "exported %d records to %s\n", len(records), args[1])
	return 0
}

func executeDuplicate(args []string, stdout, stderr io.Writer, opts globalOptions) int {
	if len(args) == 0 {
		return writeError(stdout, stderr, opts, 2, "usage", "usage: rforge --project <path> duplicate <report|merge|split>")
	}
	if opts.Project == "" {
		return writeError(stdout, stderr, opts, 2, "missing_project", "--project is required for duplicate commands")
	}
	store, err := library.OpenStore(filepath.Join(opts.Project, "data", "library.json"))
	if err != nil {
		return writeError(stdout, stderr, opts, 1, "library_open_failed", fmt.Sprintf("open library: %v", err))
	}
	papers, err := store.List()
	if err != nil {
		return writeError(stdout, stderr, opts, 1, "library_list_failed", fmt.Sprintf("list library: %v", err))
	}
	switch args[0] {
	case "report":
		if len(args) != 1 {
			return writeError(stdout, stderr, opts, 2, "usage", "usage: rforge --project <path> duplicate report")
		}
		matches := duplicateMatches(papers)
		if opts.JSON {
			return writeJSON(stdout, 0, map[string]any{"matches": matches})
		}
		for _, match := range matches {
			fmt.Fprintf(stdout, "%s\t%.2f\t%d\t%d\n", match.Reason, match.Score, match.LeftIndex, match.RightIndex)
		}
		return 0
	case "merge":
		if len(args) != 3 {
			return writeError(stdout, stderr, opts, 2, "usage", "usage: rforge --project <path> duplicate merge <left-index> <right-index>")
		}
		leftIndex, rightIndex, ok := parseTwoIndexes(args[1], args[2], len(papers))
		if !ok || leftIndex == rightIndex {
			return writeError(stdout, stderr, opts, 2, "invalid_duplicate_indexes", "duplicate merge indexes must reference two existing records")
		}
		merged := library.MergeDuplicate(papers[leftIndex], papers[rightIndex])
		updated := removePaperIndexes(papers, leftIndex, rightIndex)
		updated = append(updated, merged)
		if err := store.ReplaceAll(updated); err != nil {
			return writeError(stdout, stderr, opts, 1, "duplicate_merge_failed", fmt.Sprintf("merge duplicate: %v", err))
		}
		if err := recordDuplicateEvent(opts.Project, "duplicate.merge", map[string]any{"leftIndex": leftIndex, "rightIndex": rightIndex}, map[string]any{"recordCount": len(updated)}); err != nil {
			return writeError(stdout, stderr, opts, 1, "duplicate_provenance_failed", fmt.Sprintf("record duplicate provenance: %v", err))
		}
		if opts.JSON {
			return writeJSON(stdout, 0, map[string]any{"merged": true, "recordCount": len(updated)})
		}
		fmt.Fprintln(stdout, "merged duplicate records")
		return 0
	case "split":
		if len(args) != 3 {
			return writeError(stdout, stderr, opts, 2, "usage", "usage: rforge --project <path> duplicate split <record-index> <json-file>")
		}
		index, err := strconv.Atoi(args[1])
		if err != nil || index < 0 || index >= len(papers) {
			return writeError(stdout, stderr, opts, 2, "invalid_duplicate_index", "duplicate split index must reference an existing record")
		}
		replacements, err := library.ImportJSON(args[2])
		if err != nil || len(replacements) < 2 {
			return writeError(stdout, stderr, opts, 2, "invalid_split_file", "duplicate split requires a JSON file with at least two replacement records")
		}
		updated := append([]library.PaperRecord{}, papers[:index]...)
		updated = append(updated, replacements...)
		updated = append(updated, papers[index+1:]...)
		if err := store.ReplaceAll(updated); err != nil {
			return writeError(stdout, stderr, opts, 1, "duplicate_split_failed", fmt.Sprintf("split duplicate: %v", err))
		}
		if err := recordDuplicateEvent(opts.Project, "duplicate.split", map[string]any{"index": index, "replacementFile": args[2]}, map[string]any{"recordCount": len(updated), "replacementCount": len(replacements)}); err != nil {
			return writeError(stdout, stderr, opts, 1, "duplicate_provenance_failed", fmt.Sprintf("record duplicate provenance: %v", err))
		}
		if opts.JSON {
			return writeJSON(stdout, 0, map[string]any{"split": true, "recordCount": len(updated)})
		}
		fmt.Fprintln(stdout, "split duplicate record")
		return 0
	default:
		return writeError(stdout, stderr, opts, 2, "usage", "usage: rforge --project <path> duplicate <report|merge|split>")
	}
}

func parseTwoIndexes(left, right string, max int) (int, int, bool) {
	leftIndex, leftErr := strconv.Atoi(left)
	rightIndex, rightErr := strconv.Atoi(right)
	if leftErr != nil || rightErr != nil || leftIndex < 0 || rightIndex < 0 || leftIndex >= max || rightIndex >= max {
		return 0, 0, false
	}
	return leftIndex, rightIndex, true
}

func removePaperIndexes(papers []library.PaperRecord, leftIndex, rightIndex int) []library.PaperRecord {
	out := make([]library.PaperRecord, 0, len(papers)-2)
	for i, paper := range papers {
		if i != leftIndex && i != rightIndex {
			out = append(out, paper)
		}
	}
	return out
}

func recordDuplicateEvent(projectPath, action string, inputs, outputs map[string]any) error {
	now := time.Now().UTC()
	return provenance.Append(projectPath, provenance.Event{
		SchemaVersion: "1",
		ID:            "evt_" + now.Format("20060102T150405Z") + "_duplicate",
		Timestamp:     now.Format(time.RFC3339),
		Actor:         "rforge",
		Action:        action,
		Target:        projectPath,
		Inputs:        inputs,
		Outputs:       outputs,
		Warnings:      []string{},
	})
}

type duplicateReportMatch struct {
	LeftIndex  int
	RightIndex int
	Duplicate  bool
	Score      float64
	Reason     string
}

func duplicateMatches(papers []library.PaperRecord) []duplicateReportMatch {
	matches := []duplicateReportMatch{}
	for i := 0; i < len(papers); i++ {
		for j := i + 1; j < len(papers); j++ {
			match := library.ScoreDuplicate(papers[i], papers[j])
			if match.Duplicate {
				matches = append(matches, duplicateReportMatch{LeftIndex: i, RightIndex: j, Duplicate: true, Score: match.Score, Reason: match.Reason})
			}
		}
	}
	return matches
}

func executeLibrary(args []string, stdout, stderr io.Writer, opts globalOptions) int {
	if len(args) != 1 || args[0] != "list" {
		return writeError(stdout, stderr, opts, 2, "usage", "usage: rforge --project <path> library list")
	}
	if opts.Project == "" {
		return writeError(stdout, stderr, opts, 2, "missing_project", "--project is required for library commands")
	}
	store, err := library.OpenStore(filepath.Join(opts.Project, "data", "library.json"))
	if err != nil {
		return writeError(stdout, stderr, opts, 1, "library_open_failed", fmt.Sprintf("open library: %v", err))
	}
	papers, err := store.List()
	if err != nil {
		return writeError(stdout, stderr, opts, 1, "library_list_failed", fmt.Sprintf("list library: %v", err))
	}
	if opts.JSON {
		return writeJSON(stdout, 0, map[string]any{"papers": papers})
	}
	for _, paper := range papers {
		fmt.Fprintf(stdout, "%s\t%s\n", paper.Identifiers.DOI, paper.Title)
	}
	return 0
}
