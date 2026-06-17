package webui

import "strings"

type WorkbenchIndexState struct{ Workbenches []WorkbenchCard }
type WorkbenchCard struct{ Label, Route, Purpose, CLI, Fallback string }

func BuildWorkbenchIndexState() WorkbenchIndexState {
	return WorkbenchIndexState{Workbenches: []WorkbenchCard{
		{"source planning", "/sources", "Compile protocol source plans before network use", "rforge protocol plan-sources --question <text>", "GET /sources with plain form submit"},
		{"import/dedupe", "/dedupe", "Review imported records, identity clusters, and reversible dedupe decisions", "rforge library identity-resolve && rforge duplicate report", "Server-rendered cluster tables"},
		{"legal acquisition", "/acquisition", "Review OA candidates, licensing, privacy, and approval queues", "rforge oa acquisition-queue", "Static approval checklist"},
		{"parser arbitration", "/parsing", "Compare parser outputs, manifests, references, and arbitration decisions", "rforge parse compare --left <a> --right <b>", "Server-rendered parser artifact list"},
		{"retrieval tuning", "/retrieve", "Review retrieval locks, benchmarks, and hybrid tuning", "rforge retrieve benchmark --out <report.json>", "Server-rendered benchmark checklist"},
		{"screening", "/screening", "Active-learning queues, uncertainty, conflicts, and progress", "rforge screen active-run --stage title_abstract --out <run.json>", "No-JS screening tables"},
		{"evidence extraction", "/evidence", "Evidence grid, support links, corrections, and gaps", "rforge evidence grid --out <grid.json>", "Static extraction grid"},
		{"meta-analysis", "/analysis", "Prepared inputs, effect models, diagnostics, and artifacts", "rforge analysis prepare run1 && rforge analysis run run1", "Static analysis artifact list"},
		{"report traceability", "/report", "Claim traces, weak support blockers, and final export", "rforge report trace --claims <queue.json> --analysis <run.json> --out <trace.json>", "Static claim trace panel"},
		{"research map", "/map", "Citation graph, domain map, topics, and accessible exports", "rforge citations accessible-view --graph <graph.json> --out <view.md>", "No-JS graph tables"},
		{"connector health", "/connectors", "Capability registry and live-smoke drift alerts", "rforge protocol capabilities", "Static connector table"},
		{"reproducibility/export", "/package", "Package manifest, redaction, checksums, audit, and replay", "rforge package create --out review.rforgepkg", "Static package checklist"},
	}}
}

func (s WorkbenchIndexState) Has(label string) bool {
	for _, card := range s.Workbenches {
		if strings.EqualFold(card.Label, label) {
			return true
		}
	}
	return false
}

func (s WorkbenchIndexState) CardByRoute(route string) (WorkbenchCard, bool) {
	for _, card := range s.Workbenches {
		if card.Route == route {
			return card, true
		}
	}
	return WorkbenchCard{}, false
}
