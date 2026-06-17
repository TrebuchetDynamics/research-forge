package cli

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"time"

	"github.com/TrebuchetDynamics/research-forge/internal/documents"
	"github.com/TrebuchetDynamics/research-forge/internal/parsing"
	"github.com/TrebuchetDynamics/research-forge/internal/retrieval"
)

const maxParsePDFBytes int64 = 100 << 20

func executeIndex(args []string, stdout, stderr io.Writer, opts globalOptions) int {
	if len(args) == 1 && args[0] == "embedding-providers" {
		registry := retrieval.DefaultEmbeddingProviderRegistry()
		if opts.JSON {
			return writeJSON(stdout, 0, map[string]any{"embeddingProviders": registry})
		}
		for _, provider := range registry.Providers {
			fmt.Fprintf(stdout, "%s\t%d\t%s\n", provider.Name, provider.Dimensions, provider.Compliance.TextEgress)
		}
		return 0
	}
	backend, ok := parseIndexArgs(args)
	if !ok {
		return writeError(stdout, stderr, opts, 2, "usage", "usage: rforge --project <path> index rebuild [--backend sqlite|opensearch|qdrant|hybrid]")
	}
	if opts.Project == "" {
		return writeError(stdout, stderr, opts, 2, "missing_project", "--project is required for index commands")
	}
	docs, err := readParsedDocuments(filepath.Join(opts.Project, "parsed"))
	if err != nil {
		return writeError(stdout, stderr, opts, 1, "parsed_read_failed", fmt.Sprintf("read parsed documents: %v", err))
	}
	index, err := openRetrievalBackend(opts.Project, backend)
	if err != nil {
		return writeError(stdout, stderr, opts, 1, "index_open_failed", fmt.Sprintf("open index: %v", err))
	}
	defer index.Close()
	var openSearchReport *retrieval.OpenSearchBulkReport
	var qdrantReport *retrieval.QdrantRebuildReport
	if osIndex, ok := index.(*retrieval.OpenSearchIndex); ok {
		report, err := osIndex.RebuildWithReport(docs)
		if err != nil {
			return writeError(stdout, stderr, opts, 1, "index_rebuild_failed", fmt.Sprintf("rebuild index: %v", err))
		}
		openSearchReport = &report
		if err := writeJSONFile(filepath.Join(opts.Project, "data", "opensearch.bulk-report.json"), report); err != nil {
			return writeError(stdout, stderr, opts, 1, "index_bulk_report_failed", fmt.Sprintf("write OpenSearch bulk report: %v", err))
		}
		if err := writeOpenSearchMappingLock(opts.Project, report); err != nil {
			return writeError(stdout, stderr, opts, 1, "index_mapping_lock_failed", fmt.Sprintf("write OpenSearch mapping lock: %v", err))
		}
	} else if qdrantIndex, ok := index.(*retrieval.QdrantIndex); ok {
		report, err := qdrantIndex.RebuildWithReport(docs)
		if err != nil {
			return writeError(stdout, stderr, opts, 1, "index_rebuild_failed", fmt.Sprintf("rebuild index: %v", err))
		}
		qdrantReport = &report
		if err := writeJSONFile(filepath.Join(opts.Project, "data", "qdrant.index-report.json"), report); err != nil {
			return writeError(stdout, stderr, opts, 1, "index_qdrant_report_failed", fmt.Sprintf("write Qdrant report: %v", err))
		}
		if err := writeQdrantVectorLock(opts.Project, report); err != nil {
			return writeError(stdout, stderr, opts, 1, "index_qdrant_lock_failed", fmt.Sprintf("write Qdrant vector lock: %v", err))
		}
	} else if err := index.Rebuild(docs); err != nil {
		return writeError(stdout, stderr, opts, 1, "index_rebuild_failed", fmt.Sprintf("rebuild index: %v", err))
	}
	if err := writeRetrievalLock(opts.Project, backend, len(docs)); err != nil {
		return writeError(stdout, stderr, opts, 1, "index_lock_failed", fmt.Sprintf("write retrieval lock: %v", err))
	}
	if opts.JSON {
		data := map[string]any{"indexedDocuments": len(docs), "backend": backend}
		if openSearchReport != nil {
			data["openSearchBulkReport"] = openSearchReport
		}
		if qdrantReport != nil {
			data["qdrantIndexReport"] = qdrantReport
		}
		return writeJSON(stdout, 0, data)
	}
	fmt.Fprintf(stdout, "indexed %d parsed documents with %s\n", len(docs), backend)
	return 0
}

type retrievalLock struct {
	SchemaVersion    string `json:"schemaVersion"`
	Backend          string `json:"backend"`
	IndexedDocuments int    `json:"indexedDocuments"`
	LexicalBackend   string `json:"lexicalBackend,omitempty"`
	VectorBackend    string `json:"vectorBackend,omitempty"`
	EmbeddingBackend string `json:"embeddingBackend,omitempty"`
	EmbeddingVersion string `json:"embeddingVersion,omitempty"`
	MappingVersion   string `json:"mappingVersion,omitempty"`
	VectorDimension  int    `json:"vectorDimension,omitempty"`
	PayloadPrivacy   string `json:"payloadPrivacy,omitempty"`
}

func writeQdrantVectorLock(project string, report retrieval.QdrantRebuildReport) error {
	return writeJSONFile(filepath.Join(project, "data", "qdrant.vector.lock.json"), map[string]any{
		"schemaVersion":           "1",
		"backend":                 "qdrant",
		"collection":              report.Collection,
		"embeddingProvider":       report.EmbeddingProvider,
		"dimension":               report.Dimension,
		"payloadPrivacy":          report.PayloadPrivacy,
		"textEgress":              report.TextEgress,
		"invalidatedBeforeUpsert": report.InvalidatedBeforeUpsert,
	})
}

func writeOpenSearchMappingLock(project string, report retrieval.OpenSearchBulkReport) error {
	return writeJSONFile(filepath.Join(project, "data", "opensearch.mapping.lock.json"), map[string]any{
		"schemaVersion":  "1",
		"backend":        "opensearch",
		"index":          report.Index,
		"mappingVersion": report.MappingVersion,
	})
}

func writeRetrievalLock(project, backend string, indexedDocuments int) error {
	lock := retrievalLock{SchemaVersion: "1", Backend: backend, IndexedDocuments: indexedDocuments}
	switch backend {
	case "qdrant":
		embedding := embeddingModelFromEnv()
		lock.VectorBackend = "qdrant"
		lock.EmbeddingBackend = embedding.EmbeddingBackendName()
		lock.EmbeddingVersion = embeddingVersionFromEnv()
		lock.VectorDimension = embeddingDimensionsFromEnv()
		lock.PayloadPrivacy = qdrantPayloadPrivacyFromEnv()
	case "hybrid":
		embedding := embeddingModelFromEnv()
		lock.LexicalBackend = "sqlite-fts5"
		lock.VectorBackend = "qdrant"
		lock.EmbeddingBackend = embedding.EmbeddingBackendName()
		lock.EmbeddingVersion = embeddingVersionFromEnv()
	case "opensearch":
		lock.LexicalBackend = "opensearch"
		lock.MappingVersion = retrieval.OpenSearchMappingVersion
	default:
		lock.LexicalBackend = "sqlite-fts5"
	}
	return writeJSONFile(filepath.Join(project, "data", "retrieval.lock.json"), lock)
}

func executeRetrieve(args []string, stdout, stderr io.Writer, opts globalOptions) int {
	query, backend, ok := parseRetrieveArgs(args)
	if !ok {
		return writeError(stdout, stderr, opts, 2, "usage", "usage: rforge --project <path> retrieve --query <query> [--backend sqlite|opensearch|qdrant|hybrid]")
	}
	if opts.Project == "" {
		return writeError(stdout, stderr, opts, 2, "missing_project", "--project is required for retrieve commands")
	}
	index, err := openRetrievalBackend(opts.Project, backend)
	if err != nil {
		return writeError(stdout, stderr, opts, 1, "index_open_failed", fmt.Sprintf("open index: %v", err))
	}
	defer index.Close()
	results, err := index.Retrieve(query)
	if err != nil {
		return writeError(stdout, stderr, opts, 1, "retrieve_failed", fmt.Sprintf("retrieve: %v", err))
	}
	if opts.JSON {
		return writeJSON(stdout, 0, map[string]any{"results": results, "backend": backend})
	}
	for _, result := range results {
		fmt.Fprintf(stdout, "%s\t%s\n", result.PassageID, result.Text)
	}
	return 0
}

func parseIndexArgs(args []string) (string, bool) {
	if len(args) == 1 && args[0] == "rebuild" {
		return "sqlite", true
	}
	if len(args) == 3 && args[0] == "rebuild" && args[1] == "--backend" && validRetrievalBackend(args[2]) {
		return args[2], true
	}
	return "", false
}

func parseRetrieveArgs(args []string) (string, string, bool) {
	backend := "sqlite"
	query := ""
	for i := 0; i < len(args); i++ {
		switch args[i] {
		case "--query":
			if i+1 >= len(args) {
				return "", "", false
			}
			query = args[i+1]
			i++
		case "--backend":
			if i+1 >= len(args) || !validRetrievalBackend(args[i+1]) {
				return "", "", false
			}
			backend = args[i+1]
			i++
		default:
			return "", "", false
		}
	}
	return query, backend, strings.TrimSpace(query) != ""
}

func validRetrievalBackend(backend string) bool {
	return backend == "sqlite" || backend == "opensearch" || backend == "qdrant" || backend == "hybrid"
}

func embeddingModelFromEnv() retrieval.EmbeddingModel {
	if endpoint := strings.TrimSpace(os.Getenv("RFORGE_EMBEDDING_URL")); endpoint != "" {
		return retrieval.HTTPEmbedding{Endpoint: endpoint, Model: os.Getenv("RFORGE_EMBEDDING_MODEL")}
	}
	return retrieval.DeterministicEmbedding{Dimensions: embeddingDimensionsFromEnv()}
}

func embeddingVersionFromEnv() string {
	if strings.TrimSpace(os.Getenv("RFORGE_EMBEDDING_URL")) != "" {
		model := strings.TrimSpace(os.Getenv("RFORGE_EMBEDDING_MODEL"))
		if model == "" {
			model = "default"
		}
		return "http-model=" + model
	}
	return fmt.Sprintf("dimensions=%d", embeddingDimensionsFromEnv())
}

func embeddingDimensionsFromEnv() int {
	dimensions := 8
	if raw := strings.TrimSpace(os.Getenv("RFORGE_EMBEDDING_DIMENSIONS")); raw != "" {
		if parsed, err := strconv.Atoi(raw); err == nil && parsed > 0 {
			dimensions = parsed
		}
	}
	return dimensions
}

func qdrantPayloadPrivacyFromEnv() string {
	privacy := strings.TrimSpace(os.Getenv("RFORGE_QDRANT_PAYLOAD_PRIVACY"))
	if privacy == retrieval.PayloadPrivacyRedacted {
		return retrieval.PayloadPrivacyRedacted
	}
	return retrieval.PayloadPrivacyFull
}

func qdrantInvalidateFromEnv() bool {
	value := strings.TrimSpace(strings.ToLower(os.Getenv("RFORGE_QDRANT_INVALIDATE")))
	return value == "1" || value == "true" || value == "yes"
}

func openRetrievalBackend(project, backend string) (retrieval.SearchAdapter, error) {
	switch backend {
	case "opensearch":
		baseURL := os.Getenv("RFORGE_OPENSEARCH_URL")
		if strings.TrimSpace(baseURL) == "" {
			return nil, fmt.Errorf("RFORGE_OPENSEARCH_URL is required for opensearch backend")
		}
		return retrieval.NewOpenSearchIndex(retrieval.OpenSearchOptions{BaseURL: baseURL, Index: os.Getenv("RFORGE_OPENSEARCH_INDEX")})
	case "qdrant":
		baseURL := os.Getenv("RFORGE_QDRANT_URL")
		if strings.TrimSpace(baseURL) == "" {
			return nil, fmt.Errorf("RFORGE_QDRANT_URL is required for qdrant backend")
		}
		return retrieval.NewQdrantIndex(retrieval.QdrantOptions{BaseURL: baseURL, Collection: os.Getenv("RFORGE_QDRANT_COLLECTION"), Embeddings: embeddingModelFromEnv(), PayloadPrivacy: qdrantPayloadPrivacyFromEnv(), InvalidateBeforeUpsert: qdrantInvalidateFromEnv()})
	case "hybrid":
		qdrantURL := os.Getenv("RFORGE_QDRANT_URL")
		if strings.TrimSpace(qdrantURL) == "" {
			return nil, fmt.Errorf("RFORGE_QDRANT_URL is required for hybrid backend")
		}
		lexical, err := retrieval.OpenSQLiteIndex(filepath.Join(project, "data", "retrieval.db"))
		if err != nil {
			return nil, err
		}
		vector, err := retrieval.NewQdrantIndex(retrieval.QdrantOptions{BaseURL: qdrantURL, Collection: os.Getenv("RFORGE_QDRANT_COLLECTION"), Embeddings: embeddingModelFromEnv(), PayloadPrivacy: qdrantPayloadPrivacyFromEnv(), InvalidateBeforeUpsert: qdrantInvalidateFromEnv()})
		if err != nil {
			_ = lexical.Close()
			return nil, err
		}
		return retrieval.HybridIndex{Lexical: lexical, Vector: vector}, nil
	default:
		return retrieval.OpenSQLiteIndex(filepath.Join(project, "data", "retrieval.db"))
	}
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
	if len(args) > 0 && args[0] == "manifest-policies" {
		policies := parsing.DefaultParserOutputPolicies()
		if opts.JSON {
			return writeJSON(stdout, 0, map[string]any{"policies": policies})
		}
		for _, policy := range policies.Policies {
			fmt.Fprintf(stdout, "%s\t%s\t%s\n", policy.ParserName, policy.ParserSource, policy.Shareability)
		}
		return 0
	}
	if len(args) > 0 && args[0] == "adjudicate-ref" {
		parsedPath, logPath, index, decision, reviewer, reason, correction, ok := parseAdjudicateRefArgs(args[1:], opts.Project)
		if !ok {
			return writeError(stdout, stderr, opts, 2, "usage", "usage: rforge --project <path> parse adjudicate-ref --parsed <parsed.json> --index <n> --decision accept|correct|reject|defer --reviewer <name> --reason <text> [--title <text> --doi <doi> --raw <text> --log <jsonl>]")
		}
		var doc parsing.ParsedDocument
		if err := readJSONFile(parsedPath, &doc); err != nil {
			return writeError(stdout, stderr, opts, 1, "parse_refs_read_failed", err.Error())
		}
		record, err := parsing.NewReferenceAdjudication(doc, index, decision, reviewer, reason, correction)
		if err != nil {
			return writeError(stdout, stderr, opts, 1, "parse_ref_adjudication_invalid", err.Error())
		}
		if err := parsing.AppendReferenceAdjudication(logPath, record); err != nil {
			return writeError(stdout, stderr, opts, 1, "parse_ref_adjudication_write_failed", err.Error())
		}
		if err := recordDuplicateEvent(opts.Project, "reference.adjudication.recorded", map[string]any{"parsed": parsedPath, "paperId": record.PaperID, "referenceIndex": record.ReferenceIndex, "decision": record.Decision}, map[string]any{"path": logPath, "reviewer": record.Reviewer, "reason": record.Reason}); err != nil {
			return writeError(stdout, stderr, opts, 1, "parse_ref_adjudication_provenance_failed", err.Error())
		}
		if opts.JSON {
			return writeJSON(stdout, 0, map[string]any{"referenceAdjudication": record, "path": logPath})
		}
		fmt.Fprintf(stdout, "recorded reference adjudication in %s\n", logPath)
		return 0
	}
	if len(args) > 0 && args[0] == "adjudicated-refs" {
		parsedPath, logPath, out, ok := parseAdjudicatedRefsArgs(args[1:], opts.Project)
		if !ok {
			return writeError(stdout, stderr, opts, 2, "usage", "usage: rforge --project <path> parse adjudicated-refs --parsed <parsed.json> [--log <jsonl> --out <report.json>]")
		}
		var doc parsing.ParsedDocument
		if err := readJSONFile(parsedPath, &doc); err != nil {
			return writeError(stdout, stderr, opts, 1, "parse_refs_read_failed", err.Error())
		}
		records, err := parsing.LoadReferenceAdjudications(logPath)
		if err != nil {
			return writeError(stdout, stderr, opts, 1, "parse_ref_adjudication_read_failed", err.Error())
		}
		report := parsing.ApplyReferenceAdjudications(doc, records)
		if out != "" {
			if err := writeJSONFile(out, report); err != nil {
				return writeError(stdout, stderr, opts, 1, "parse_ref_adjudication_report_write_failed", err.Error())
			}
		}
		if opts.JSON {
			return writeJSON(stdout, 0, map[string]any{"referenceAdjudication": report, "path": out, "logPath": logPath})
		}
		if out != "" {
			fmt.Fprintf(stdout, "wrote reference adjudication report to %s\n", out)
		} else {
			fmt.Fprintf(stdout, "reference adjudication: accepted=%d corrected=%d rejected=%d deferred=%d unreviewed=%d\n", report.Accepted, report.Corrected, report.Rejected, report.Deferred, report.Unreviewed)
		}
		return 0
	}
	if len(args) > 0 && args[0] == "review-refs" {
		parsedPath, out, threshold, ok := parseReviewRefsArgs(args[1:])
		if !ok {
			return writeError(stdout, stderr, opts, 2, "usage", "usage: rforge --project <path> parse review-refs --parsed <parsed.json> --out <report.json> [--threshold 0.75]")
		}
		var doc parsing.ParsedDocument
		if err := readJSONFile(parsedPath, &doc); err != nil {
			return writeError(stdout, stderr, opts, 1, "parse_refs_read_failed", err.Error())
		}
		report := parsing.AmbiguousReferences(doc, threshold)
		if err := writeJSONFile(out, report); err != nil {
			return writeError(stdout, stderr, opts, 1, "parse_refs_review_write_failed", err.Error())
		}
		if opts.JSON {
			return writeJSON(stdout, 0, map[string]any{"referenceReview": report, "path": out})
		}
		fmt.Fprintf(stdout, "wrote reference review queue to %s\n", out)
		return 0
	}
	if len(args) > 0 && args[0] == "references" {
		paperID, parserName, inputPath, out, ok := parseReferenceParseArgs(args[1:])
		if !ok || parserName != "anystyle" {
			return writeError(stdout, stderr, opts, 2, "usage", "usage: rforge --project <path> parse references --paper <id> --parser anystyle --file <refs.txt> --out <refs.json>")
		}
		input, err := os.ReadFile(inputPath)
		if err != nil {
			return writeError(stdout, stderr, opts, 1, "parse_references_read_failed", err.Error())
		}
		command := strings.Fields(os.Getenv("RFORGE_ANYSTYLE_CMD"))
		if len(command) == 0 {
			return writeError(stdout, stderr, opts, 2, "parse_references_command_missing", "RFORGE_ANYSTYLE_CMD is required for anystyle reference parsing")
		}
		parser := parsing.AnyStyleReferenceParser{Runner: parsing.ExecCommandRunner{Command: command}, Version: "external"}
		doc, err := parser.ParseReferences(context.Background(), paperID, input)
		if err != nil {
			return writeError(stdout, stderr, opts, 1, "parse_references_failed", err.Error())
		}
		parsedData, err := json.MarshalIndent(doc, "", "  ")
		if err != nil {
			return writeError(stdout, stderr, opts, 1, "parse_references_write_failed", err.Error())
		}
		parsedData = append(parsedData, '\n')
		if err := os.MkdirAll(filepath.Dir(out), 0o755); err != nil {
			return writeError(stdout, stderr, opts, 1, "parse_references_write_failed", err.Error())
		}
		if err := os.WriteFile(out, parsedData, 0o644); err != nil {
			return writeError(stdout, stderr, opts, 1, "parse_references_write_failed", err.Error())
		}
		manifestPath := strings.TrimSuffix(out, filepath.Ext(out)) + ".manifest.json"
		manifest := parsing.NewParserRunManifestWithOutput(doc, input, parsedData, out, command)
		if err := writeJSONFile(manifestPath, manifest); err != nil {
			return writeError(stdout, stderr, opts, 1, "parse_references_manifest_failed", err.Error())
		}
		if opts.JSON {
			return writeJSON(stdout, 0, map[string]any{"parsed": doc, "path": out, "manifestPath": manifestPath})
		}
		fmt.Fprintf(stdout, "wrote parsed references to %s\n", out)
		return 0
	}
	if len(args) > 0 && args[0] == "normalize-refs" {
		parsedPath, sourceName, out, ok := parseNormalizeRefsArgs(args[1:])
		if !ok || !validReferenceNormalizationSource(sourceName) {
			return writeError(stdout, stderr, opts, 2, "usage", "usage: rforge --project <path> parse normalize-refs --parsed <parsed.json> --source crossref|openalex|semantic-scholar|ads --out <report.json>")
		}
		var doc parsing.ParsedDocument
		if err := readJSONFile(parsedPath, &doc); err != nil {
			return writeError(stdout, stderr, opts, 1, "parse_refs_read_failed", err.Error())
		}
		connector, ok := searchConnector(sourceName)
		if !ok {
			return writeError(stdout, stderr, opts, 2, "unknown_source", fmt.Sprintf("unknown source %q", sourceName))
		}
		report, err := parsing.NormalizeParsedReferences(context.Background(), connector, doc)
		if err != nil {
			return writeError(stdout, stderr, opts, 1, "parse_refs_normalize_failed", err.Error())
		}
		if err := writeJSONFile(out, report); err != nil {
			return writeError(stdout, stderr, opts, 1, "parse_refs_write_failed", err.Error())
		}
		if opts.JSON {
			return writeJSON(stdout, 0, map[string]any{"referenceNormalization": report, "path": out})
		}
		fmt.Fprintf(stdout, "wrote reference normalization to %s\n", out)
		return 0
	}
	if len(args) > 0 && args[0] == "arbitrate" {
		left, right, out, accepted, reason, reviewer, ok := parseParseArbitrateArgs(args[1:])
		if !ok {
			return writeError(stdout, stderr, opts, 2, "usage", "usage: rforge --project <path> parse arbitrate --left <parsed.json> --right <parsed.json> --out <report.json> [--accept <parser> --reason <text> --reviewer <name>]")
		}
		docs, err := readParsedDocumentPair(left, right)
		if err != nil {
			return writeError(stdout, stderr, opts, 1, "parse_arbitrate_read_failed", err.Error())
		}
		report := parsing.ArbitrateParserOutputs(docs, parsing.ArbitrationDecisionInput{AcceptedParser: accepted, Reason: reason, Reviewer: reviewer})
		if err := writeJSONFile(out, report); err != nil {
			return writeError(stdout, stderr, opts, 1, "parse_arbitrate_write_failed", err.Error())
		}
		if err := recordDuplicateEvent(opts.Project, "parser.arbitration.decided", map[string]any{"left": left, "right": right, "accepted": report.Decision.AcceptedParser}, map[string]any{"path": out, "reason": report.Decision.Reason, "reviewer": report.Decision.Reviewer}); err != nil {
			return writeError(stdout, stderr, opts, 1, "parse_arbitrate_provenance_failed", err.Error())
		}
		if opts.JSON {
			return writeJSON(stdout, 0, map[string]any{"arbitration": report, "path": out})
		}
		fmt.Fprintf(stdout, "wrote parser arbitration to %s\n", out)
		return 0
	}
	if len(args) > 0 && args[0] == "compare" {
		left, right, out, ok := parseParseCompareArgs(args[1:])
		if !ok {
			return writeError(stdout, stderr, opts, 2, "usage", "usage: rforge --project <path> parse compare --left <parsed.json> --right <parsed.json> --out <report.json>")
		}
		report, err := compareParsedFiles(left, right)
		if err != nil {
			return writeError(stdout, stderr, opts, 1, "parse_compare_failed", err.Error())
		}
		if err := writeJSONFile(out, report); err != nil {
			return writeError(stdout, stderr, opts, 1, "parse_compare_write_failed", err.Error())
		}
		if opts.JSON {
			return writeJSON(stdout, 0, map[string]any{"comparison": report, "path": out})
		}
		fmt.Fprintf(stdout, "wrote parser comparison to %s\n", out)
		return 0
	}
	paperID, parserName, inputPath, ok := parseParseArgs(args)
	if !ok || (parserName != "grobid" && parserName != "tex" && parserName != "s2orc" && parserName != "papermage") {
		return writeError(stdout, stderr, opts, 2, "usage", "usage: rforge --project <path> parse --paper <id> --parser grobid --pdf <file> | --parser tex --tex <file> | --parser s2orc --s2orc <file> | --parser papermage --papermage <file>")
	}
	var doc parsing.ParsedDocument
	var inputData []byte
	if parserName == "grobid" {
		pdf, err := readParsePDF(inputPath)
		if err != nil {
			return writeError(stdout, stderr, opts, 1, "parse_pdf_read_failed", fmt.Sprintf("read PDF: %v", err))
		}
		inputData = pdf
		baseURL := os.Getenv("RFORGE_GROBID_URL")
		client := parsing.NewGROBIDClient(parsing.GROBIDClientOptions{BaseURL: baseURL, Timeout: 30 * time.Second, Version: "configured"})
		doc, err = client.Parse(context.Background(), pdf, parsing.ParseOptions{PaperID: paperID})
		if err != nil {
			return writeError(stdout, stderr, opts, 1, "parse_failed", fmt.Sprintf("parse: %v", err))
		}
	} else if parserName == "tex" {
		tex, err := os.ReadFile(inputPath)
		if err != nil {
			return writeError(stdout, stderr, opts, 1, "parse_tex_read_failed", fmt.Sprintf("read TeX: %v", err))
		}
		inputData = tex
		doc, err = (parsing.TeXParser{}).Parse(context.Background(), tex, parsing.ParseOptions{PaperID: paperID})
		if err != nil {
			return writeError(stdout, stderr, opts, 1, "parse_failed", fmt.Sprintf("parse: %v", err))
		}
	} else if parserName == "s2orc" {
		data, err := os.ReadFile(inputPath)
		if err != nil {
			return writeError(stdout, stderr, opts, 1, "parse_s2orc_read_failed", fmt.Sprintf("read S2ORC JSON: %v", err))
		}
		inputData = data
		doc, err = (parsing.S2ORCJSONParser{}).Parse(context.Background(), data, parsing.ParseOptions{PaperID: paperID})
		if err != nil {
			return writeError(stdout, stderr, opts, 1, "parse_failed", fmt.Sprintf("parse: %v", err))
		}
	} else {
		data, err := os.ReadFile(inputPath)
		if err != nil {
			return writeError(stdout, stderr, opts, 1, "parse_papermage_read_failed", fmt.Sprintf("read PaperMage JSON: %v", err))
		}
		inputData = data
		doc, err = (parsing.PaperMageJSONParser{}).Parse(context.Background(), data, parsing.ParseOptions{PaperID: paperID})
		if err != nil {
			return writeError(stdout, stderr, opts, 1, "parse_failed", fmt.Sprintf("parse: %v", err))
		}
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
	manifestPath := filepath.Join(parsedDir, safeFileStem(paperID)+".manifest.json")
	manifest := parsing.NewParserRunManifestWithOutput(doc, inputData, data, parsedPath, parserCommand(parserName, inputPath))
	if err := writeJSONFile(manifestPath, manifest); err != nil {
		return writeError(stdout, stderr, opts, 1, "parse_manifest_failed", fmt.Sprintf("write parser manifest: %v", err))
	}
	if err := recordDuplicateEvent(opts.Project, "parser.run", map[string]any{"paperID": paperID, "parser": parserName, "input": inputPath}, map[string]any{"parsedPath": parsedPath, "manifestPath": manifestPath, "parserVersion": doc.ParserVersion, "warnings": doc.Warnings}); err != nil {
		return writeError(stdout, stderr, opts, 1, "parse_provenance_failed", fmt.Sprintf("record parse provenance: %v", err))
	}
	if opts.JSON {
		return writeJSON(stdout, 0, map[string]any{"parsed": doc, "path": parsedPath, "manifestPath": manifestPath})
	}
	fmt.Fprintf(stdout, "parsed %s to %s\n", paperID, parsedPath)
	return 0
}

func parserCommand(parserName, inputPath string) []string {
	switch parserName {
	case "grobid":
		return []string{"grobid", "processFulltextDocument", inputPath}
	case "s2orc":
		return []string{"rforge", "parse", "--parser", "s2orc", "--s2orc", inputPath}
	case "papermage":
		return []string{"rforge", "parse", "--parser", "papermage", "--papermage", inputPath}
	case "tex":
		return []string{"rforge", "parse", "--parser", "tex", "--tex", inputPath}
	default:
		return []string{"rforge", "parse", "--parser", parserName, inputPath}
	}
}

func validReferenceNormalizationSource(sourceName string) bool {
	return sourceName == "crossref" || sourceName == "openalex" || sourceName == "semantic-scholar" || sourceName == "ads"
}

func compareParsedFiles(left, right string) (parsing.ComparisonReport, error) {
	docs, err := readParsedDocumentPair(left, right)
	if err != nil {
		return parsing.ComparisonReport{}, err
	}
	return parsing.CompareParsedDocuments(docs), nil
}

func readParsedDocumentPair(left, right string) ([]parsing.ParsedDocument, error) {
	docs := []parsing.ParsedDocument{}
	for _, path := range []string{left, right} {
		data, err := os.ReadFile(path)
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

func defaultReferenceAdjudicationLog(project string) string {
	return filepath.Join(project, "data", "reference-adjudications.jsonl")
}

func parseAdjudicateRefArgs(args []string, project string) (string, string, int, string, string, string, parsing.ReferenceCorrection, bool) {
	values := map[string]string{"--log": defaultReferenceAdjudicationLog(project)}
	for i := 0; i < len(args); i++ {
		switch args[i] {
		case "--parsed", "--log", "--index", "--decision", "--reviewer", "--reason", "--title", "--doi", "--raw":
			if i+1 >= len(args) {
				return "", "", 0, "", "", "", parsing.ReferenceCorrection{}, false
			}
			values[args[i]] = args[i+1]
			i++
		default:
			return "", "", 0, "", "", "", parsing.ReferenceCorrection{}, false
		}
	}
	index, err := strconv.Atoi(values["--index"])
	if err != nil {
		return "", "", 0, "", "", "", parsing.ReferenceCorrection{}, false
	}
	correction := parsing.ReferenceCorrection{Title: values["--title"], DOI: values["--doi"], Raw: values["--raw"]}
	ok := values["--parsed"] != "" && values["--log"] != "" && values["--decision"] != "" && values["--reviewer"] != "" && values["--reason"] != ""
	return values["--parsed"], values["--log"], index, values["--decision"], values["--reviewer"], values["--reason"], correction, ok
}

func parseAdjudicatedRefsArgs(args []string, project string) (string, string, string, bool) {
	values := map[string]string{"--log": defaultReferenceAdjudicationLog(project)}
	for i := 0; i < len(args); i++ {
		switch args[i] {
		case "--parsed", "--log", "--out":
			if i+1 >= len(args) {
				return "", "", "", false
			}
			values[args[i]] = args[i+1]
			i++
		default:
			return "", "", "", false
		}
	}
	return values["--parsed"], values["--log"], values["--out"], values["--parsed"] != "" && values["--log"] != ""
}

func parseReviewRefsArgs(args []string) (string, string, float64, bool) {
	values := map[string]string{"--threshold": "0.75"}
	for i := 0; i < len(args); i++ {
		switch args[i] {
		case "--parsed", "--out", "--threshold":
			if i+1 >= len(args) {
				return "", "", 0, false
			}
			values[args[i]] = args[i+1]
			i++
		default:
			return "", "", 0, false
		}
	}
	threshold, err := strconv.ParseFloat(values["--threshold"], 64)
	if err != nil || threshold <= 0 || threshold > 1 {
		return "", "", 0, false
	}
	return values["--parsed"], values["--out"], threshold, values["--parsed"] != "" && values["--out"] != ""
}

func parseReferenceParseArgs(args []string) (string, string, string, string, bool) {
	values := map[string]string{}
	for i := 0; i < len(args); i++ {
		switch args[i] {
		case "--paper", "--parser", "--file", "--out":
			if i+1 >= len(args) {
				return "", "", "", "", false
			}
			values[args[i]] = args[i+1]
			i++
		default:
			return "", "", "", "", false
		}
	}
	return values["--paper"], values["--parser"], values["--file"], values["--out"], values["--paper"] != "" && values["--parser"] != "" && values["--file"] != "" && values["--out"] != ""
}

func parseNormalizeRefsArgs(args []string) (string, string, string, bool) {
	values := map[string]string{}
	for i := 0; i < len(args); i++ {
		switch args[i] {
		case "--parsed", "--source", "--out":
			if i+1 >= len(args) {
				return "", "", "", false
			}
			values[args[i]] = args[i+1]
			i++
		default:
			return "", "", "", false
		}
	}
	return values["--parsed"], values["--source"], values["--out"], values["--parsed"] != "" && values["--source"] != "" && values["--out"] != ""
}

func parseParseArbitrateArgs(args []string) (string, string, string, string, string, string, bool) {
	values := map[string]string{}
	for i := 0; i < len(args); i++ {
		switch args[i] {
		case "--left", "--right", "--out", "--accept", "--reason", "--reviewer":
			if i+1 >= len(args) {
				return "", "", "", "", "", "", false
			}
			values[args[i]] = args[i+1]
			i++
		default:
			return "", "", "", "", "", "", false
		}
	}
	return values["--left"], values["--right"], values["--out"], values["--accept"], values["--reason"], values["--reviewer"], values["--left"] != "" && values["--right"] != "" && values["--out"] != ""
}

func parseParseCompareArgs(args []string) (string, string, string, bool) {
	values := map[string]string{}
	for i := 0; i < len(args); i++ {
		switch args[i] {
		case "--left", "--right", "--out":
			if i+1 >= len(args) {
				return "", "", "", false
			}
			values[args[i]] = args[i+1]
			i++
		default:
			return "", "", "", false
		}
	}
	return values["--left"], values["--right"], values["--out"], values["--left"] != "" && values["--right"] != "" && values["--out"] != ""
}

func parseParseArgs(args []string) (string, string, string, bool) {
	values := map[string]string{}
	for i := 0; i < len(args); i++ {
		switch args[i] {
		case "--paper", "--parser", "--pdf", "--tex", "--s2orc", "--papermage":
			if i+1 >= len(args) {
				return "", "", "", false
			}
			values[args[i]] = args[i+1]
			i++
		default:
			return "", "", "", false
		}
	}
	input := values["--pdf"]
	if values["--parser"] == "tex" {
		input = values["--tex"]
	}
	if values["--parser"] == "s2orc" {
		input = values["--s2orc"]
	}
	if values["--parser"] == "papermage" {
		input = values["--papermage"]
	}
	return values["--paper"], values["--parser"], input, values["--paper"] != "" && values["--parser"] != "" && input != ""
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
	if len(args) == 0 || (args[0] != "fetch" && args[0] != "fetch-arxiv" && args[0] != "import-biomedical" && args[0] != "biomedical-drift-smoke-plan") {
		return writeError(stdout, stderr, opts, 2, "usage", "usage: rforge --project <path> pdf fetch --doi <doi> --pdf-url <url> --license <license> --oa-status <status> | pdf fetch-arxiv --paper <arxiv-id> --kind pdf|source --url <url> | pdf import-biomedical --xml <file> --out <json>")
	}
	if opts.Project == "" {
		return writeError(stdout, stderr, opts, 2, "missing_project", "--project is required for pdf commands")
	}
	if args[0] == "biomedical-drift-smoke-plan" {
		plan := documents.NewBiomedicalLiveDriftSmokeSnapshot()
		if opts.JSON {
			return writeJSON(stdout, 0, map[string]any{"biomedicalDriftSmoke": plan})
		}
		for _, connector := range plan.Connectors {
			fmt.Fprintf(stdout, "%s\t%s\t%s\n", connector.Source, connector.OptInEnv, strings.Join(connector.ExpectedFields, ","))
		}
		return 0
	}
	if args[0] == "import-biomedical" {
		xmlPath, outPath, ok := parseBiomedicalImport(args[1:])
		if !ok {
			return writeError(stdout, stderr, opts, 2, "usage", "usage: rforge --project <path> pdf import-biomedical --xml <file> --out <json>")
		}
		fullText, err := documents.ImportStructuredBiomedicalFullText(xmlPath)
		if err != nil {
			return writeError(stdout, stderr, opts, 1, "biomedical_import_failed", err.Error())
		}
		if err := fullText.Validate(); err != nil {
			return writeError(stdout, stderr, opts, 1, "biomedical_import_invalid", err.Error())
		}
		if err := writeJSONFile(outPath, fullText); err != nil {
			return writeError(stdout, stderr, opts, 1, "biomedical_import_write_failed", err.Error())
		}
		if opts.JSON {
			return writeJSON(stdout, 0, map[string]any{"fullText": fullText, "path": outPath})
		}
		fmt.Fprintf(stdout, "imported biomedical full text to %s\n", outPath)
		return 0
	}
	if args[0] == "fetch-arxiv" {
		paperID, kind, assetURL, ok := parseArXivFetch(args[1:])
		if !ok {
			return writeError(stdout, stderr, opts, 2, "usage", "usage: rforge --project <path> pdf fetch-arxiv --paper <arxiv-id> --kind pdf|source --url <url>")
		}
		asset, err := documents.FetchArXivAsset(context.Background(), opts.Project, paperID, assetURL, kind)
		if err != nil {
			return writeError(stdout, stderr, opts, 1, "arxiv_fetch_failed", fmt.Sprintf("fetch arXiv asset: %v", err))
		}
		if opts.JSON {
			return writeJSON(stdout, 0, map[string]any{"asset": asset})
		}
		fmt.Fprintf(stdout, "fetched arXiv asset %s\n", asset.LocalPath)
		return 0
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

func parseBiomedicalImport(args []string) (string, string, bool) {
	values := map[string]string{}
	for i := 0; i < len(args); i++ {
		switch args[i] {
		case "--xml", "--out":
			if i+1 >= len(args) {
				return "", "", false
			}
			values[args[i]] = args[i+1]
			i++
		default:
			return "", "", false
		}
	}
	return values["--xml"], values["--out"], values["--xml"] != "" && values["--out"] != ""
}

func parseArXivFetch(args []string) (string, string, string, bool) {
	values := map[string]string{}
	for i := 0; i < len(args); i++ {
		switch args[i] {
		case "--paper", "--kind", "--url":
			if i+1 >= len(args) {
				return "", "", "", false
			}
			values[args[i]] = args[i+1]
			i++
		default:
			return "", "", "", false
		}
	}
	kind := values["--kind"]
	return values["--paper"], kind, values["--url"], values["--paper"] != "" && (kind == "pdf" || kind == "source") && values["--url"] != ""
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
