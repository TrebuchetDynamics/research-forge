package cli

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"strconv"

	"github.com/TrebuchetDynamics/research-forge/internal/documents"
	"github.com/TrebuchetDynamics/research-forge/internal/research"
)

func executeResearch(args []string, stdout, stderr io.Writer, opts globalOptions) int {
	if opts.Project == "" {
		return writeError(stdout, stderr, opts, 2, "missing_project", "--project is required for research commands")
	}
	if len(args) == 0 {
		return writeError(stdout, stderr, opts, 2, "usage", "usage: rforge --project <path> research screen-queue|leakage-audit ...")
	}
	switch args[0] {
	case "parse-pdftotext":
		return executeResearchParsePDFText(args[1:], stdout, stderr, opts)
	case "acquire-pdftotext":
		return executeResearchAcquirePDFText(args[1:], stdout, stderr, opts)
	case "screen-queue":
		return executeResearchScreenQueue(args[1:], stdout, stderr, opts)
	case "leakage-audit":
		return executeResearchLeakageAudit(args[1:], stdout, stderr, opts)
	default:
		return writeError(stdout, stderr, opts, 2, "unknown_research_command", fmt.Sprintf("unknown research command %q", args[0]))
	}
}

func executeResearchParsePDFText(args []string, stdout, stderr io.Writer, opts globalOptions) int {
	values, ok := parseFlagValues(args, map[string]bool{"--paper": true, "--title": true, "--pdf": true, "--out": true, "--chunk-size": true})
	if !ok || values["--paper"] == "" || values["--pdf"] == "" || values["--out"] == "" {
		return writeError(stdout, stderr, opts, 2, "usage", "usage: rforge --project <path> research parse-pdftotext --paper <id> --pdf <file> --out <parsed.json> [--title <title>] [--chunk-size N]")
	}
	return writePDFText(values, stdout, stderr, opts, "")
}

func executeResearchAcquirePDFText(args []string, stdout, stderr io.Writer, opts globalOptions) int {
	values, ok := parseFlagValues(args, map[string]bool{"--doi": true, "--paper": true, "--title": true, "--pdf-url": true, "--license": true, "--oa-status": true, "--out": true, "--chunk-size": true})
	if !ok || values["--doi"] == "" || values["--pdf-url"] == "" || values["--license"] == "" || values["--oa-status"] == "" || values["--out"] == "" {
		return writeError(stdout, stderr, opts, 2, "usage", "usage: rforge --project <path> research acquire-pdftotext --doi <doi> --pdf-url <url> --license <license> --oa-status <status> --out <parsed.json> [--paper <id>] [--title <title>] [--chunk-size N]")
	}
	asset, err := documents.FetchPDFByDOI(context.Background(), opts.Project, values["--doi"], documents.OpenAccessMetadata{OpenAccess: true, OAStatus: values["--oa-status"], License: values["--license"], PDFURL: values["--pdf-url"]})
	if err != nil {
		return writeError(stdout, stderr, opts, 1, "pdf_fetch_failed", fmt.Sprintf("fetch PDF: %v", err))
	}
	if values["--paper"] == "" {
		values["--paper"] = values["--doi"]
	}
	values["--pdf"] = asset.LocalPath
	return writePDFText(values, stdout, stderr, opts, asset.LocalPath)
}

func writePDFText(values map[string]string, stdout, stderr io.Writer, opts globalOptions, pdfPath string) int {
	chunkSize := 1400
	if values["--chunk-size"] != "" {
		parsed, err := strconv.Atoi(values["--chunk-size"])
		if err != nil || parsed <= 0 {
			return writeError(stdout, stderr, opts, 2, "invalid_chunk_size", "--chunk-size must be a positive integer")
		}
		chunkSize = parsed
	}
	text, err := exec.CommandContext(context.Background(), pdftotextCommand(), "-layout", values["--pdf"], "-").Output()
	if err != nil {
		return writeError(stdout, stderr, opts, 1, "pdftotext_failed", fmt.Sprintf("run pdftotext: %v", err))
	}
	doc := research.ParsedTextDocument(values["--paper"], values["--title"], string(text), chunkSize)
	data, err := json.MarshalIndent(doc, "", "  ")
	if err != nil {
		return writeError(stdout, stderr, opts, 1, "parse_pdftotext_marshal_failed", err.Error())
	}
	if dir := filepath.Dir(values["--out"]); dir != "." {
		if err := os.MkdirAll(dir, 0o755); err != nil {
			return writeError(stdout, stderr, opts, 1, "parse_pdftotext_mkdir_failed", err.Error())
		}
	}
	if err := os.WriteFile(values["--out"], data, 0o644); err != nil {
		return writeError(stdout, stderr, opts, 1, "parse_pdftotext_write_failed", err.Error())
	}
	if opts.JSON {
		return writeJSON(stdout, 0, map[string]any{"path": values["--out"], "paper": values["--paper"], "parser": doc.ParserName, "passages": len(doc.Sections[0].Passages), "pdf": pdfPath})
	}
	fmt.Fprintf(stdout, "wrote pdftotext parsed document to %s\n", values["--out"])
	return 0
}

func pdftotextCommand() string {
	if cmd := os.Getenv("RFORGE_PDFTOTEXT_CMD"); cmd != "" {
		return cmd
	}
	return "pdftotext"
}

func executeResearchScreenQueue(args []string, stdout, stderr io.Writer, opts globalOptions) int {
	values, ok := parseFlagValues(args, map[string]bool{"--library": true, "--search-results": true, "--out": true, "--markdown": true})
	if !ok || values["--out"] == "" {
		return writeError(stdout, stderr, opts, 2, "usage", "usage: rforge --project <path> research screen-queue --out <queue.csv> [--markdown <queue.md>] [--library <library.json>] [--search-results <dir>]")
	}
	queue, err := research.BuildScreeningQueue(values["--library"], values["--search-results"])
	if err != nil {
		return writeError(stdout, stderr, opts, 1, "screen_queue_failed", err.Error())
	}
	if err := writeScreeningCSV(values["--out"], queue); err != nil {
		return writeError(stdout, stderr, opts, 1, "screen_queue_write_failed", err.Error())
	}
	if values["--markdown"] != "" {
		if err := os.WriteFile(values["--markdown"], []byte(research.ScreeningMarkdown(queue, 40)), 0o644); err != nil {
			return writeError(stdout, stderr, opts, 1, "screen_queue_markdown_failed", err.Error())
		}
	}
	if opts.JSON {
		return writeJSON(stdout, 0, map[string]any{"records": len(queue), "path": values["--out"], "markdown": values["--markdown"]})
	}
	fmt.Fprintf(stdout, "wrote screening queue with %d records to %s\n", len(queue), values["--out"])
	return 0
}

func writeScreeningCSV(path string, queue []research.ScreeningRecord) error {
	file, err := os.Create(path)
	if err != nil {
		return err
	}
	defer file.Close()
	return research.WriteScreeningCSV(file, queue)
}

func executeResearchLeakageAudit(args []string, stdout, stderr io.Writer, opts globalOptions) int {
	values, ok := parseFlagValues(args, map[string]bool{"--parsed": true, "--out": true, "--markdown": true})
	if !ok || values["--parsed"] == "" || values["--out"] == "" {
		return writeError(stdout, stderr, opts, 2, "usage", "usage: rforge --project <path> research leakage-audit --parsed <parsed-dir> --out <audit.json> [--markdown <audit.md>]")
	}
	rows, err := research.BuildLeakageAudit(values["--parsed"])
	if err != nil {
		return writeError(stdout, stderr, opts, 1, "leakage_audit_failed", err.Error())
	}
	data, err := json.MarshalIndent(rows, "", "  ")
	if err != nil {
		return writeError(stdout, stderr, opts, 1, "leakage_audit_marshal_failed", err.Error())
	}
	if err := os.WriteFile(values["--out"], data, 0o644); err != nil {
		return writeError(stdout, stderr, opts, 1, "leakage_audit_write_failed", err.Error())
	}
	if values["--markdown"] != "" {
		if err := os.WriteFile(values["--markdown"], []byte(research.LeakageAuditMarkdown(rows)), 0o644); err != nil {
			return writeError(stdout, stderr, opts, 1, "leakage_audit_markdown_failed", err.Error())
		}
	}
	if opts.JSON {
		return writeJSON(stdout, 0, map[string]any{"records": len(rows), "path": values["--out"], "markdown": values["--markdown"]})
	}
	fmt.Fprintf(stdout, "wrote leakage audit with %d records to %s\n", len(rows), values["--out"])
	return 0
}

func parseFlagValues(args []string, allowed map[string]bool) (map[string]string, bool) {
	values := map[string]string{}
	for i := 0; i < len(args); i++ {
		flag := args[i]
		if !allowed[flag] || i+1 >= len(args) {
			return nil, false
		}
		values[flag] = args[i+1]
		i++
	}
	return values, true
}
