package knowledge

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"

	"github.com/TrebuchetDynamics/research-forge/internal/analysis"
	"github.com/TrebuchetDynamics/research-forge/internal/evidence"
	"github.com/TrebuchetDynamics/research-forge/internal/library"
	"github.com/TrebuchetDynamics/research-forge/internal/parsing"
	"github.com/TrebuchetDynamics/research-forge/internal/provenance"
	"github.com/TrebuchetDynamics/research-forge/internal/report"
	"github.com/TrebuchetDynamics/research-forge/internal/screening"
)

type ProjectKnowledgeGraph struct {
	SchemaVersion string          `json:"schemaVersion"`
	Nodes         []KnowledgeNode `json:"nodes"`
	Edges         []KnowledgeEdge `json:"edges"`
}

type KnowledgeNode struct {
	ID         string            `json:"id"`
	Kind       string            `json:"kind"`
	Label      string            `json:"label"`
	Properties map[string]string `json:"properties,omitempty"`
}

type KnowledgeEdge struct {
	ID     string `json:"id"`
	Source string `json:"source"`
	Target string `json:"target"`
	Kind   string `json:"kind"`
}

type CitationEdge struct{ Source, Target string }

type ProjectGraphInput struct {
	LibraryRecords   []library.PaperRecord
	CitationEdges    []CitationEdge
	ParsedDocuments  []parsing.ParsedDocument
	EvidenceItems    []evidence.EvidenceItem
	ScreeningEvents  []screening.DecisionEvent
	AnalysisRuns     []analysis.AnalysisRun
	ReportTrace      report.CitationEvidenceTraceView
	ProvenanceEvents []provenance.Event
}

func BuildProjectKnowledgeGraph(input ProjectGraphInput) ProjectKnowledgeGraph {
	b := &builder{nodes: map[string]KnowledgeNode{}, edges: map[string]KnowledgeEdge{}}
	for _, record := range input.LibraryRecords {
		b.addLibraryRecord(record)
	}
	for _, edge := range input.CitationEdges {
		if edge.Source == "" || edge.Target == "" {
			continue
		}
		b.addNode("paper:"+edge.Source, "paper", edge.Source, nil)
		b.addNode("paper:"+edge.Target, "paper", edge.Target, nil)
		b.addNode("citation:"+edge.Source+"->"+edge.Target, "citation", edge.Source+" cites "+edge.Target, nil)
		b.addEdge("paper:"+edge.Source, "paper:"+edge.Target, "cites")
	}
	for _, doc := range input.ParsedDocuments {
		b.addParsedDocument(doc)
	}
	for i, item := range input.EvidenceItems {
		b.addEvidenceItem(i, item)
	}
	for i, event := range input.ScreeningEvents {
		b.addScreeningEvent(i, event)
	}
	for _, run := range input.AnalysisRuns {
		b.addAnalysisRun(run)
	}
	for _, claim := range input.ReportTrace.Claims {
		b.addClaim(claim, input.EvidenceItems)
	}
	for _, event := range input.ProvenanceEvents {
		b.addProvenance(event)
	}
	return b.graph()
}

func QueryProjectKnowledgeGraph(graph ProjectKnowledgeGraph, term string) ProjectKnowledgeGraph {
	term = strings.ToLower(strings.TrimSpace(term))
	if term == "" {
		return graph
	}
	keep := map[string]bool{}
	for _, node := range graph.Nodes {
		if strings.Contains(strings.ToLower(node.Label), term) || strings.Contains(strings.ToLower(node.Kind), term) || propertiesContain(node.Properties, term) {
			keep[node.ID] = true
		}
	}
	for _, edge := range graph.Edges {
		if keep[edge.Source] || keep[edge.Target] {
			keep[edge.Source] = true
			keep[edge.Target] = true
		}
	}
	out := ProjectKnowledgeGraph{SchemaVersion: graph.SchemaVersion}
	for _, node := range graph.Nodes {
		if keep[node.ID] {
			out.Nodes = append(out.Nodes, node)
		}
	}
	for _, edge := range graph.Edges {
		if keep[edge.Source] && keep[edge.Target] {
			out.Edges = append(out.Edges, edge)
		}
	}
	return out
}

func BuildProjectKnowledgeGraphFromProject(projectPath string) (ProjectKnowledgeGraph, error) {
	input := ProjectGraphInput{}
	_ = readJSON(filepath.Join(projectPath, "data", "library.json"), &input.LibraryRecords)
	input.CitationEdges = readCitationEdges(filepath.Join(projectPath, "data", "citation-graph.json"))
	input.ParsedDocuments = readParsedDocuments(filepath.Join(projectPath, "parsed"))
	_ = readJSON(filepath.Join(projectPath, "data", "evidence.json"), &input.EvidenceItems)
	_ = readJSON(filepath.Join(projectPath, "data", "screening.events.json"), &input.ScreeningEvents)
	input.AnalysisRuns = readAnalysisRuns(filepath.Join(projectPath, "analysis"))
	_ = readJSON(filepath.Join(projectPath, "data", "report-trace.json"), &input.ReportTrace)
	if events, err := provenance.Read(projectPath); err == nil {
		input.ProvenanceEvents = events
	}
	return BuildProjectKnowledgeGraph(input), nil
}

func (g ProjectKnowledgeGraph) HasNode(id string) bool {
	for _, n := range g.Nodes {
		if n.ID == id {
			return true
		}
	}
	return false
}
func (g ProjectKnowledgeGraph) HasEdge(source, target, kind string) bool {
	for _, e := range g.Edges {
		if e.Source == source && e.Target == target && e.Kind == kind {
			return true
		}
	}
	return false
}

type builder struct {
	nodes map[string]KnowledgeNode
	edges map[string]KnowledgeEdge
}

func (b *builder) addNode(id, kind, label string, props map[string]string) {
	if id == "" {
		return
	}
	if existing, ok := b.nodes[id]; ok {
		if existing.Label == "" {
			existing.Label = label
		}
		if existing.Properties == nil {
			existing.Properties = map[string]string{}
		}
		for k, v := range props {
			existing.Properties[k] = v
		}
		b.nodes[id] = existing
		return
	}
	b.nodes[id] = KnowledgeNode{ID: id, Kind: kind, Label: label, Properties: props}
}
func (b *builder) addEdge(source, target, kind string) {
	if source == "" || target == "" || kind == "" {
		return
	}
	id := source + " -" + kind + "-> " + target
	b.edges[id] = KnowledgeEdge{ID: id, Source: source, Target: target, Kind: kind}
}
func (b *builder) graph() ProjectKnowledgeGraph {
	g := ProjectKnowledgeGraph{SchemaVersion: "1"}
	for _, n := range b.nodes {
		g.Nodes = append(g.Nodes, n)
	}
	for _, e := range b.edges {
		g.Edges = append(g.Edges, e)
	}
	sort.Slice(g.Nodes, func(i, j int) bool { return g.Nodes[i].ID < g.Nodes[j].ID })
	sort.Slice(g.Edges, func(i, j int) bool { return g.Edges[i].ID < g.Edges[j].ID })
	return g
}

func (b *builder) addLibraryRecord(record library.PaperRecord) {
	id := paperID(record)
	if id == "" {
		return
	}
	paperNode := "paper:" + id
	b.addNode(paperNode, "paper", record.Title, map[string]string{"year": fmt.Sprint(record.Year), "venue": record.Venue})
	for _, ref := range record.SourceRefs {
		for key, kind := range map[string]string{"collections": "collection", "groups": "collection", "tags": "tag", "keywords": "tag", "concepts": "concept"} {
			for _, value := range splitValues(ref.Metadata[key]) {
				nodeID := kind + ":" + value
				b.addNode(nodeID, kind, value, nil)
				b.addEdge(paperNode, nodeID, relationForKind(kind))
			}
		}
	}
}
func (b *builder) addParsedDocument(doc parsing.ParsedDocument) {
	if doc.PaperID == "" {
		return
	}
	p := "paper:" + doc.PaperID
	b.addNode(p, "paper", doc.PaperID, nil)
	for i, ref := range doc.References {
		id := fmt.Sprintf("reference:%s:%d", doc.PaperID, i)
		label := ref.Title
		if label == "" {
			label = ref.Raw
		}
		b.addNode(id, "parsed_reference", label, map[string]string{"doi": ref.DOI})
		b.addEdge(p, id, "has_parsed_reference")
		if ref.DOI != "" {
			b.addNode("paper:"+ref.DOI, "paper", ref.DOI, nil)
			b.addEdge(id, "paper:"+ref.DOI, "resolves_to")
		}
	}
}
func (b *builder) addEvidenceItem(i int, item evidence.EvidenceItem) {
	if item.PaperID == "" {
		return
	}
	id := fmt.Sprintf("evidence:%s:%d", item.PaperID, i)
	b.addNode(id, "evidence", item.SchemaName, map[string]string{"status": string(item.Status), "support": string(item.Support.Kind) + ":" + item.Support.Ref, "values": joinMap(item.Values)})
	b.addNode("paper:"+item.PaperID, "paper", item.PaperID, nil)
	b.addEdge("paper:"+item.PaperID, id, "has_evidence")
}
func (b *builder) addScreeningEvent(i int, event screening.DecisionEvent) {
	if event.PaperID == "" {
		return
	}
	id := fmt.Sprintf("screening:%s:%s:%d", event.PaperID, event.Stage, i)
	b.addNode(id, "screening_decision", string(event.Decision), map[string]string{"stage": string(event.Stage), "reviewer": event.Reviewer, "reason": event.Reason})
	b.addNode("paper:"+event.PaperID, "paper", event.PaperID, nil)
	b.addEdge("paper:"+event.PaperID, id, "has_screening_decision")
}
func (b *builder) addAnalysisRun(run analysis.AnalysisRun) {
	if run.ID == "" {
		return
	}
	id := "analysis:" + run.ID
	b.addNode(id, "analysis_run", run.ID, nil)
	for _, row := range run.InputRows {
		if row.PaperID != "" {
			b.addNode("paper:"+row.PaperID, "paper", row.PaperID, nil)
			b.addEdge(id, "paper:"+row.PaperID, "analyzes")
		}
	}
}
func (b *builder) addClaim(claim report.ClaimTraceView, items []evidence.EvidenceItem) {
	id := "claim:" + claim.ClaimID
	b.addNode(id, "report_claim", claim.ClaimText, map[string]string{"status": string(claim.ClaimStatus)})
	if claim.PaperID != "" {
		b.addNode("paper:"+claim.PaperID, "paper", claim.PaperID, nil)
		b.addEdge(id, "paper:"+claim.PaperID, "claims_about")
	}
	for i, item := range items {
		if item.PaperID == claim.PaperID && item.Status == evidence.StatusAccepted {
			b.addEdge(id, fmt.Sprintf("evidence:%s:%d", item.PaperID, i), "supported_by")
		}
	}
}
func (b *builder) addProvenance(event provenance.Event) {
	if event.ID == "" {
		return
	}
	id := "provenance:" + event.ID
	b.addNode(id, "provenance_event", event.Action, map[string]string{"target": event.Target, "actor": event.Actor, "timestamp": event.Timestamp})
	if event.Target != "" {
		b.addEdge(id, "paper:"+event.Target, "targets")
	}
}

func paperID(r library.PaperRecord) string {
	ids := r.Identifiers
	for _, v := range []string{ids.DOI, ids.PMID, ids.PMCID, ids.ArXivID, ids.OpenAlexID, ids.SemanticScholarID, ids.ZoteroItemKey, r.Title} {
		if strings.TrimSpace(v) != "" {
			return strings.TrimSpace(v)
		}
	}
	return ""
}
func relationForKind(kind string) string {
	if kind == "collection" {
		return "in_collection"
	}
	if kind == "tag" {
		return "tagged_with"
	}
	return "has_concept"
}
func splitValues(raw string) []string {
	fields := strings.FieldsFunc(raw, func(r rune) bool { return r == ',' || r == ';' || r == '|' })
	out := []string{}
	seen := map[string]bool{}
	for _, f := range fields {
		f = strings.TrimSpace(f)
		if f != "" && !seen[f] {
			seen[f] = true
			out = append(out, f)
		}
	}
	return out
}
func joinMap(m map[string]string) string {
	keys := make([]string, 0, len(m))
	for k := range m {
		keys = append(keys, k)
	}
	sort.Strings(keys)
	parts := []string{}
	for _, k := range keys {
		parts = append(parts, k+"="+m[k])
	}
	return strings.Join(parts, ";")
}
func propertiesContain(m map[string]string, term string) bool {
	for k, v := range m {
		if strings.Contains(strings.ToLower(k), term) || strings.Contains(strings.ToLower(v), term) {
			return true
		}
	}
	return false
}

func readJSON(path string, out any) error {
	data, err := os.ReadFile(path)
	if err != nil {
		return err
	}
	return json.Unmarshal(data, out)
}
func readCitationEdges(path string) []CitationEdge {
	var raw struct {
		Edges []CitationEdge `json:"edges"`
	}
	_ = readJSON(path, &raw)
	return raw.Edges
}
func readParsedDocuments(dir string) []parsing.ParsedDocument {
	docs := []parsing.ParsedDocument{}
	entries, err := os.ReadDir(dir)
	if err != nil {
		return docs
	}
	for _, entry := range entries {
		if entry.IsDir() || !strings.HasSuffix(entry.Name(), ".json") {
			continue
		}
		var doc parsing.ParsedDocument
		if readJSON(filepath.Join(dir, entry.Name()), &doc) == nil {
			docs = append(docs, doc)
		}
	}
	return docs
}
func readAnalysisRuns(dir string) []analysis.AnalysisRun {
	runs := []analysis.AnalysisRun{}
	entries, err := os.ReadDir(dir)
	if err != nil {
		return runs
	}
	for _, entry := range entries {
		if entry.IsDir() || !strings.HasSuffix(entry.Name(), "-run.json") {
			continue
		}
		var run analysis.AnalysisRun
		if readJSON(filepath.Join(dir, entry.Name()), &run) == nil {
			runs = append(runs, run)
		}
	}
	return runs
}
