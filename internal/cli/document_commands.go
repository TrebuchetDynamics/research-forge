package cli

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/TrebuchetDynamics/research-forge/internal/documents"
	"github.com/TrebuchetDynamics/research-forge/internal/parsing"
	"github.com/TrebuchetDynamics/research-forge/internal/retrieval"
)

const maxParsePDFBytes int64 = 100 << 20

func executeIndex(args []string, stdout, stderr io.Writer, opts globalOptions) int {
	if len(args) != 1 || args[0] != "rebuild" {
		return writeError(stdout, stderr, opts, 2, "usage", "usage: rforge --project <path> index rebuild")
	}
	if opts.Project == "" {
		return writeError(stdout, stderr, opts, 2, "missing_project", "--project is required for index commands")
	}
	docs, err := readParsedDocuments(filepath.Join(opts.Project, "parsed"))
	if err != nil {
		return writeError(stdout, stderr, opts, 1, "parsed_read_failed", fmt.Sprintf("read parsed documents: %v", err))
	}
	index, err := retrieval.OpenSQLiteIndex(filepath.Join(opts.Project, "data", "retrieval.db"))
	if err != nil {
		return writeError(stdout, stderr, opts, 1, "index_open_failed", fmt.Sprintf("open index: %v", err))
	}
	defer index.Close()
	if err := index.Rebuild(docs); err != nil {
		return writeError(stdout, stderr, opts, 1, "index_rebuild_failed", fmt.Sprintf("rebuild index: %v", err))
	}
	if opts.JSON {
		return writeJSON(stdout, 0, map[string]any{"indexedDocuments": len(docs)})
	}
	fmt.Fprintf(stdout, "indexed %d parsed documents\n", len(docs))
	return 0
}

func executeRetrieve(args []string, stdout, stderr io.Writer, opts globalOptions) int {
	query, ok := parseRetrieveArgs(args)
	if !ok {
		return writeError(stdout, stderr, opts, 2, "usage", "usage: rforge --project <path> retrieve --query <query>")
	}
	if opts.Project == "" {
		return writeError(stdout, stderr, opts, 2, "missing_project", "--project is required for retrieve commands")
	}
	index, err := retrieval.OpenSQLiteIndex(filepath.Join(opts.Project, "data", "retrieval.db"))
	if err != nil {
		return writeError(stdout, stderr, opts, 1, "index_open_failed", fmt.Sprintf("open index: %v", err))
	}
	defer index.Close()
	results, err := index.Retrieve(query)
	if err != nil {
		return writeError(stdout, stderr, opts, 1, "retrieve_failed", fmt.Sprintf("retrieve: %v", err))
	}
	if opts.JSON {
		return writeJSON(stdout, 0, map[string]any{"results": results})
	}
	for _, result := range results {
		fmt.Fprintf(stdout, "%s\t%s\n", result.PassageID, result.Text)
	}
	return 0
}

func parseRetrieveArgs(args []string) (string, bool) {
	if len(args) != 2 || args[0] != "--query" || strings.TrimSpace(args[1]) == "" {
		return "", false
	}
	return args[1], true
}

func readParsedDocuments(dir string) ([]parsing.ParsedDocument, error) {
	entries, err := os.ReadDir(dir)
	if err != nil {
		return nil, err
	}
	docs := []parsing.ParsedDocument{}
	for _, entry := range entries {
		if entry.IsDir() || filepath.Ext(entry.Name()) != ".json" {
			continue
		}
		data, err := os.ReadFile(filepath.Join(dir, entry.Name()))
		if err != nil {
			return nil, err
		}
		var doc parsing.ParsedDocument
		if err := json.Unmarshal(data, &doc); err != nil {
			return nil, err
		}
		docs = append(docs, doc)
	}
	return docs, nil
}

func executeParse(args []string, stdout, stderr io.Writer, opts globalOptions) int {
	if opts.Project == "" {
		return writeError(stdout, stderr, opts, 2, "missing_project", "--project is required for parse commands")
	}
	paperID, parserName, pdfPath, ok := parseParseArgs(args)
	if !ok || parserName != "grobid" {
		return writeError(stdout, stderr, opts, 2, "usage", "usage: rforge --project <path> parse --paper <id> --parser grobid --pdf <file>")
	}
	pdf, err := readParsePDF(pdfPath)
	if err != nil {
		return writeError(stdout, stderr, opts, 1, "parse_pdf_read_failed", fmt.Sprintf("read PDF: %v", err))
	}
	baseURL := os.Getenv("RFORGE_GROBID_URL")
	client := parsing.NewGROBIDClient(parsing.GROBIDClientOptions{BaseURL: baseURL, Timeout: 30 * time.Second, Version: "configured"})
	doc, err := client.Parse(context.Background(), pdf, parsing.ParseOptions{PaperID: paperID})
	if err != nil {
		return writeError(stdout, stderr, opts, 1, "parse_failed", fmt.Sprintf("parse: %v", err))
	}
	parsedDir := filepath.Join(opts.Project, "parsed")
	if err := os.MkdirAll(parsedDir, 0o755); err != nil {
		return writeError(stdout, stderr, opts, 1, "parse_store_failed", fmt.Sprintf("create parsed dir: %v", err))
	}
	parsedPath := filepath.Join(parsedDir, safeFileStem(paperID)+".json")
	data, err := json.MarshalIndent(doc, "", "  ")
	if err != nil {
		return writeError(stdout, stderr, opts, 1, "parse_store_failed", fmt.Sprintf("marshal parsed doc: %v", err))
	}
	data = append(data, '\n')
	if err := os.WriteFile(parsedPath, data, 0o644); err != nil {
		return writeError(stdout, stderr, opts, 1, "parse_store_failed", fmt.Sprintf("write parsed doc: %v", err))
	}
	if err := recordDuplicateEvent(opts.Project, "parser.run", map[string]any{"paperID": paperID, "parser": parserName, "pdf": pdfPath}, map[string]any{"parsedPath": parsedPath, "parserVersion": doc.ParserVersion, "warnings": doc.Warnings}); err != nil {
		return writeError(stdout, stderr, opts, 1, "parse_provenance_failed", fmt.Sprintf("record parse provenance: %v", err))
	}
	if opts.JSON {
		return writeJSON(stdout, 0, map[string]any{"parsed": doc, "path": parsedPath})
	}
	fmt.Fprintf(stdout, "parsed %s to %s\n", paperID, parsedPath)
	return 0
}

func readParsePDF(path string) ([]byte, error) {
	file, err := os.Open(path)
	if err != nil {
		return nil, err
	}
	defer file.Close()
	info, err := file.Stat()
	if err != nil {
		return nil, err
	}
	if info.Size() > maxParsePDFBytes {
		return nil, fmt.Errorf("pdf input too large: %d bytes exceeds %d", info.Size(), maxParsePDFBytes)
	}
	data, err := io.ReadAll(io.LimitReader(file, maxParsePDFBytes+1))
	if err != nil {
		return nil, err
	}
	if int64(len(data)) > maxParsePDFBytes {
		return nil, fmt.Errorf("pdf input too large: exceeds %d", maxParsePDFBytes)
	}
	return data, nil
}

func parseParseArgs(args []string) (string, string, string, bool) {
	values := map[string]string{}
	for i := 0; i < len(args); i++ {
		switch args[i] {
		case "--paper", "--parser", "--pdf":
			if i+1 >= len(args) {
				return "", "", "", false
			}
			values[args[i]] = args[i+1]
			i++
		default:
			return "", "", "", false
		}
	}
	return values["--paper"], values["--parser"], values["--pdf"], values["--paper"] != "" && values["--parser"] != "" && values["--pdf"] != ""
}

func safeFileStem(value string) string {
	parts := strings.FieldsFunc(strings.ToLower(strings.TrimSpace(value)), func(r rune) bool {
		return !(r >= 'a' && r <= 'z') && !(r >= '0' && r <= '9')
	})
	if len(parts) == 0 {
		return "item"
	}
	return strings.Join(parts, "-")
}

func executePDF(args []string, stdout, stderr io.Writer, opts globalOptions) int {
	if len(args) == 0 || args[0] != "fetch" {
		return writeError(stdout, stderr, opts, 2, "usage", "usage: rforge --project <path> pdf fetch --doi <doi> --pdf-url <url> --license <license> --oa-status <status>")
	}
	if opts.Project == "" {
		return writeError(stdout, stderr, opts, 2, "missing_project", "--project is required for pdf commands")
	}
	doi, pdfURL, license, oaStatus, ok := parsePDFFetch(args[1:])
	if !ok {
		return writeError(stdout, stderr, opts, 2, "usage", "usage: rforge --project <path> pdf fetch --doi <doi> --pdf-url <url> --license <license> --oa-status <status>")
	}
	asset, err := documents.FetchPDFByDOI(context.Background(), opts.Project, doi, documents.OpenAccessMetadata{OpenAccess: true, OAStatus: oaStatus, License: license, PDFURL: pdfURL})
	if err != nil {
		return writeError(stdout, stderr, opts, 1, "pdf_fetch_failed", fmt.Sprintf("fetch PDF: %v", err))
	}
	if opts.JSON {
		return writeJSON(stdout, 0, map[string]any{"asset": asset})
	}
	fmt.Fprintf(stdout, "fetched PDF %s\n", asset.LocalPath)
	return 0
}

func parsePDFFetch(args []string) (string, string, string, string, bool) {
	values := map[string]string{}
	for i := 0; i < len(args); i++ {
		switch args[i] {
		case "--doi", "--pdf-url", "--license", "--oa-status":
			if i+1 >= len(args) {
				return "", "", "", "", false
			}
			values[args[i]] = args[i+1]
			i++
		default:
			return "", "", "", "", false
		}
	}
	return values["--doi"], values["--pdf-url"], values["--license"], values["--oa-status"], values["--doi"] != "" && values["--pdf-url"] != "" && values["--license"] != "" && values["--oa-status"] != ""
}
