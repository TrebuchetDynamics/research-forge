package webui

import (
	"fmt"
	"html/template"
	"net/http"
	"path/filepath"
	"strings"
	"time"

	"github.com/TrebuchetDynamics/research-forge/internal/project"
	"github.com/TrebuchetDynamics/research-forge/internal/protocol"
	"github.com/TrebuchetDynamics/research-forge/internal/provenance"
	"github.com/TrebuchetDynamics/research-forge/internal/screening"
	"github.com/TrebuchetDynamics/research-forge/internal/ui"
)

var shellTemplate = template.Must(template.New("shell").Parse(`<!doctype html>
<html lang="en" hx-boost="true">
<head>
  <meta charset="utf-8">
  <meta name="viewport" content="width=device-width, initial-scale=1">
  <title>ResearchForge</title>
  <script src="/assets/htmx.min.js" defer></script>
  <link rel="stylesheet" href="/assets/researchforge.css">
</head>
<body>
  <main class="rf-shell">
    <header>
      <p class="eyebrow">Local Go + HTMX workspace</p>
      <h1>ResearchForge</h1>
      <p>Read papers, browse CLI-generated artifacts, and explore the knowledge graph for the open research project.</p>
      <nav aria-label="Dashboard sections">
        <ul class="rf-nav">
          <li><a href="/forge">Forge</a></li>
          <li><a href="/workbenches">Workbenches</a></li>
          <li><a href="/notebook">Notebook</a></li>
          <li><a href="/papers">Papers</a></li>
          <li><a href="/library">Library</a></li>
          <li><a href="/dedupe">Dedupe</a></li>
          <li><a href="/screening">Screening</a></li>
          <li><a href="/artifacts">Artifacts</a></li>
          <li><a href="/oss">OSS studies</a></li>
          <li><a href="/search">Search</a></li>
          <li><a href="/sources">Source plan</a></li>
          <li><a href="/connectors">Connector health</a></li>
          <li><a href="/projects">Projects</a></li>
        </ul>
      </nav>
    </header>
    <div hx-get="/projects/active" hx-trigger="load"></div>
    <section aria-labelledby="dashboard-title">
      <h2 id="dashboard-title">Project dashboard</h2>
      <p>Open a research folder, then read every parsed paper in the browser and review its analysis, PRISMA flow, and citation graph.</p>
    </section>
    <section aria-labelledby="artifacts-title">
      <h2 id="artifacts-title">CLI-generated artifacts</h2>
      <p>Papers, PRISMA diagrams, citation graphs, analyses, and reports remain reproducible from CLI output.</p>
    </section>
  </main>
</body>
</html>`))

var searchTemplate = template.Must(template.New("search").Parse(`<section aria-labelledby="search-title" class="rf-card">
  <h2 id="search-title">Search papers</h2>
  <form hx-get="/search/results" hx-target="#search-results" hx-indicator="#search-loading">
    <label for="query">Query</label>
    <input id="query" name="query" type="search" value="{{.Query}}">
    <fieldset>
      <legend>Sources</legend>
      {{range .Sources}}
      <label><input type="checkbox" name="source" value="{{.}}" checked> {{.}}</label>
      {{end}}
    </fieldset>
    <button type="submit">Search</button>
  </form>
  <p id="search-loading" class="htmx-indicator">Loading</p>
  <div id="search-results" role="status">No results yet</div>
</section>`))

var sourcePlanningTemplate = template.Must(template.New("source-planning").Parse(`<section aria-labelledby="source-planning-title" class="rf-card">
  <h2 id="source-planning-title">Source planning cockpit</h2>
  <form method="get" action="/sources">
    <label for="source-question">Research question</label>
    <input id="source-question" name="question" value="{{.Question}}" required>
    <label for="source-type">Framework</label>
    <select id="source-type" name="type">
      <option value="freeform">freeform</option>
      <option value="pico">pico</option>
      <option value="peco">peco</option>
      <option value="spider">spider</option>
    </select>
    <button type="submit">Preview sources</button>
  </form>
  <p>CLI equivalent: <code>rforge protocol plan-sources --question '{{.Question}}'</code></p>
  <p><strong>Reviewer approval required</strong> before network calls, imports, downloads, or package inclusion.</p>
  <div role="table" aria-label="Source plan preview">
    <div role="row"><strong role="columnheader">Source</strong> <strong role="columnheader">Kind</strong> <strong role="columnheader">Dry run</strong> <strong role="columnheader">Privacy/Auth</strong></div>
    {{range .Sources}}
    <div role="row">
      <span role="cell">{{.Label}}</span>
      <span role="cell">{{.SourceKind}}</span>
      <span role="cell">{{.DryRunEstimate}}</span>
      <span role="cell">{{.PrivacyWarning}} Auth: {{.AuthRequirement}}</span>
      <code role="cell">{{.CLICommand}}</code>
    </div>
    {{end}}
  </div>
</section>`))

var parserConflictTemplate = template.Must(template.New("parser-conflict").Parse(`<section aria-labelledby="parser-conflict-title" class="rf-card">
  <h2 id="parser-conflict-title">Parser conflict review</h2>
  <p>Parser arbitration screen comparing {{range .Parsers}}{{index $.ParserLabels .}} {{end}} field-by-field.</p>
  <p>{{if .ReviewerRequired}}reviewer required{{else}}reviewer optional{{end}}. {{if .AutoAcceptConflicts}}Conflicts may be auto-accepted.{{else}}No conflicting fields are auto-accepted.{{end}}</p>
  <div role="table" aria-label="Parser arbitration fields">
    <div role="row"><strong role="columnheader">Field</strong><strong role="columnheader">Parser</strong><strong role="columnheader">confidence</strong><strong role="columnheader">raw text</strong><strong role="columnheader">offsets</strong><strong role="columnheader">warnings</strong><strong role="columnheader">Controls</strong></div>
    {{range .Fields}}<div role="row"><span role="cell">{{.Field}}</span><span role="cell">{{.ParserName}}</span><span role="cell">{{.Confidence}}</span><span role="cell">{{.RawText}}</span><span role="cell">{{.Offsets}}</span><span role="cell">{{.Warnings}}</span><span role="cell">{{range $.Controls}}<button type="button">{{.}}</button> {{end}}</span></div>{{end}}
  </div>
  <p>CLI equivalent: <code>{{.CLIEquivalent}}</code></p>
</section>`))

var labNotebookTemplate = template.Must(template.New("lab-notebook").Parse(`<section aria-labelledby="notebook-title" class="rf-card" hx-get="/notebook" hx-trigger="refresh-notebook from:body">
  <h2 id="notebook-title">Lab notebook timeline</h2>
  <p>Total workflow events: {{.TotalEvents}}. Human workflow events: {{.HumanEvents}}. Automated workflow events: {{.AutomatedEvents}}.</p>
  <p>Snapshot export: <a href="{{.SnapshotPath}}">{{.SnapshotPath}}</a></p>
  <ol>{{range .Events}}<li><strong>{{.Timestamp}}</strong> <span>{{.ActorKind}}</span> <span>{{.Category}}</span> <span>{{.Actor}}</span> <code>{{.Action}}</code> {{.Target}}</li>{{else}}<li>No workflow events recorded.</li>{{end}}</ol>
</section>`))

var researchMapTemplate = template.Must(template.New("research-map").Parse(`<section aria-labelledby="research-map-title" class="rf-card" hx-get="/map" hx-trigger="refresh-map from:body">
  <h2 id="research-map-title">Research map cockpit</h2>
  <p>Unifies citation graph, OpenAlex concepts, Zotero collections/tags, screening status, retrieval clusters, and evidence coverage with filters, keyboard navigation, no-JS tables, and snapshot export.</p>
  <p>Snapshot export: <a href="{{.SnapshotExportPath}}">{{.SnapshotExportPath}}</a></p>
  {{if .Filter}}<p>Filter: {{.Filter}}</p>{{else}}<p>filters: use <code>?filter=&lt;term&gt;</code></p>{{end}}
  {{if .Neighborhood}}<p>Neighborhood: {{.Neighborhood}}</p>{{end}}
  <section><h3>Concept maps</h3>{{range .ConceptMap}}<p>{{.Label}} — {{.Detail}}</p>{{else}}<p>No concepts available.</p>{{end}}</section>
  <section><h3>Citation neighborhoods</h3>{{range .CitationNeighborhoods}}<p>{{.Label}} — {{.Detail}}</p>{{else}}<p>No citation neighborhoods available.</p>{{end}}</section>
  <section><h3>Retrieval clusters</h3>{{range .RetrievalClusters}}<p>{{.Label}} — {{.Detail}}</p>{{else}}<p>No retrieval clusters available.</p>{{end}}</section>
  <section><h3>Retrieval hits</h3>{{range .RetrievalHits}}<p>{{.Label}} — {{.Detail}}</p>{{else}}<p>No retrieval hits available.</p>{{end}}</section>
  <section><h3>Screening priority</h3>{{range .ScreeningPriority}}<p>{{.Label}} — {{.Detail}}</p>{{else}}<p>No screening priority available.</p>{{end}}</section>
  <section><h3>screening status</h3>{{range .ScreeningStatus}}<p>{{.Label}} — {{.Detail}}</p>{{else}}<p>No screening status available.</p>{{end}}</section>
  <section><h3>Parser quality</h3>{{range .ParserQuality}}<p>{{.Label}} — {{.Detail}}</p>{{else}}<p>No parser quality report available.</p>{{end}}</section>
  <section><h3>Evidence coverage</h3><p>Accepted: {{.EvidenceCoverage.Accepted}} Suggested: {{.EvidenceCoverage.Suggested}} Other: {{.EvidenceCoverage.Other}}</p></section>
  <section><h3>Provenance overlays</h3>{{range .ProvenanceOverlays}}<p>{{.Label}} — {{.Detail}}</p>{{else}}<p>No provenance overlay selected.</p>{{end}}</section>
  <section><h3>Keyboard-accessible alternatives</h3><p>keyboard navigation</p>{{range .KeyboardAlternatives}}<p>{{.}}</p>{{end}}</section>
  <p>CLI equivalent: <code>rforge knowledge query --project &lt;path&gt;</code></p>
</section>`))

var privacyModelTemplate = template.Must(template.New("privacy-model").Parse(`<section aria-labelledby="privacy-title" class="rf-card">
  <h2 id="privacy-title">Dashboard permissions/privacy model</h2>
  {{range .Assets}}<article><h3>{{.Name}}</h3><p>Default permission: {{.DefaultPermission}}</p><p>Export rule: {{.ExportRule}}</p><p>Review gate: {{.ReviewGate}}</p><p>UI behavior: {{.UIBehavior}}</p></article>{{end}}
</section>`))

var informationArchitectureTemplate = template.Must(template.New("dashboard-ia").Parse(`<section aria-labelledby="ia-title" class="rf-card">
  <h2 id="ia-title">Dashboard information architecture</h2>
  <h3>Diagram</h3>
  <pre aria-label="Dashboard information architecture diagram">{{range .Diagram}}{{.}}
{{end}}</pre>
  <h3>routes</h3>
  {{range .Routes}}<article><h4>{{.Path}}</h4><p>Partial endpoints: <code>{{.Partial}}</code></p><p>View models: {{.ViewModel}}</p><p>No-JS fallbacks: {{.NoJSFallback}}</p><p>Owner: {{.Owner}}</p></article>{{end}}
  <h3>Background jobs</h3>
  {{range .BackgroundJobs}}<p>{{.Name}} — {{.Trigger}} — {{.StatusArtifact}}</p>{{end}}
  <h3>Ownership boundaries</h3>
  {{range .OwnershipBoundaries}}<p>{{.Area}} — {{.Boundary}}</p>{{end}}
</section>`))

var workbenchIndexTemplate = template.Must(template.New("workbench-index").Parse(`<section aria-labelledby="workbenches-title" class="rf-card">
  <h2 id="workbenches-title">HTMX workbenches</h2>
  <p>No-JS fallback: each workbench is server-rendered with CLI-equivalent commands.</p>
  {{range .Workbenches}}<article><h3><a href="{{.Route}}">{{.Label}}</a></h3><p>{{.Purpose}}</p><p>CLI equivalent: <code>{{.CLI}}</code></p><p>No-JS fallback: {{.Fallback}}</p></article>{{end}}
</section>`))

var packageExportCenterTemplate = template.Must(template.New("package-export-center").Parse(`<section aria-labelledby="package-title" class="rf-card">
  <h2 id="package-title">reproducibility/export Workbench</h2>
  <p>Preview Reproducible review package contents, redaction results, checksums, lockfiles, external-tool versions, parser manifests, analysis artifacts, report outputs, and reviewer decision logs before package creation.</p>
  <p>CLI equivalent: <code>rforge package create --project {{.ProjectPath}} --out packages/latest</code></p>
  <section><h3>Reproducible review package contents</h3>{{range .PackageContents}}<p>{{.}}</p>{{else}}<p>No core package contents found yet.</p>{{end}}</section>
  <section><h3>redaction results</h3>{{range .RedactionResults}}<p>{{.}}</p>{{end}}</section>
  <section><h3>checksums</h3>{{range .Checksums}}<p>{{.}}</p>{{end}}</section>
  <section><h3>lockfiles</h3>{{range .Lockfiles}}<p>{{.}}</p>{{else}}<p>No lockfiles found.</p>{{end}}</section>
  <section><h3>external-tool versions</h3>{{range .ExternalToolVersions}}<p>{{.}}</p>{{end}}</section>
  <section><h3>parser manifests</h3>{{range .ParserManifests}}<p>{{.}}</p>{{else}}<p>No parser manifests found.</p>{{end}}</section>
  <section><h3>analysis artifacts</h3>{{range .AnalysisArtifacts}}<p>{{.}}</p>{{else}}<p>No analysis artifacts found.</p>{{end}}</section>
  <section><h3>report outputs</h3>{{range .ReportOutputs}}<p>{{.}}</p>{{else}}<p>No report outputs found.</p>{{end}}</section>
  <section><h3>reviewer decision logs</h3>{{range .ReviewerDecisionLogs}}<p>{{.}}</p>{{else}}<p>No reviewer decision logs found.</p>{{end}}</section>
</section>`))

var reportClaimPanelTemplate = template.Must(template.New("report-claim-panel").Parse(`<section aria-labelledby="report-title" class="rf-card">
  <h2 id="report-title">Claim traceability panel</h2>
  <p>Every generated paragraph/table/figure must reference accepted evidence; unresolved or weakly supported claims block final export.</p>
  <p>block final export: {{.BlockFinalExport}}</p>
  <p>CLI equivalent: <code>rforge report claim-panel --trace data/trace.json --out {{.PanelPath}} && rforge report final-export --panel {{.PanelPath}}</code></p>
  <div role="table" aria-label="Claim traceability panel">
    <div role="row"><strong role="columnheader">Claim</strong><strong role="columnheader">Kind</strong><strong role="columnheader">Status</strong><strong role="columnheader">Reason</strong><strong role="columnheader">accepted evidence</strong></div>
    {{range .Rows}}<div role="row"><span role="cell">{{.ClaimID}}</span><span role="cell">{{.Kind}}</span><span role="cell">{{.Status}}</span><span role="cell">{{.Reason}}</span><span role="cell">{{len .Trace.AcceptedEvidence}}</span></div>{{else}}<div role="row"><span role="cell">No claim panel rows. Run the CLI command to build {{$.PanelPath}}.</span></div>{{end}}
  </div>
  {{if .Blockers}}<h3>Export blockers</h3>{{range .Blockers}}<p>{{.}}</p>{{end}}{{end}}
</section>`))

var analysisWorkbenchTemplate = template.Must(template.New("analysis-workbench").Parse(`<section aria-labelledby="analysis-title" class="rf-card">
  <h2 id="analysis-title">meta-analysis Workbench</h2>
  <p>Shows prepared effect-size inputs, model choices, metafor script, warnings, heterogeneity, sensitivity/influence diagnostics, forest/funnel artifacts, and publication-ready artifact manifests.</p>
  <p>CLI equivalent: <code>rforge analysis prepare &lt;run-id&gt; && rforge analysis run &lt;run-id&gt;</code></p>
  <section><h3>prepared effect-size inputs</h3>{{range .Runs}}{{range .InputRows}}<p>{{.PaperID}} effect={{.EffectSize}} variance={{.Variance}}</p>{{end}}{{else}}<p>No prepared analysis input rows.</p>{{end}}</section>
  <section><h3>model choices</h3><p>Effect-size models and random/fixed analysis choices are locked by analysis run artifacts.</p></section>
  <section><h3>metafor script and warnings</h3>{{range .Manifests}}<p>metafor script: {{.Script.Path}} engine={{.Script.Engine}}</p>{{range .Warnings}}<p>warnings: {{.}}</p>{{end}}{{else}}<p>No metafor artifact manifest available.</p>{{end}}</section>
  <section><h3>heterogeneity</h3><p>Inspect <code>analysis/*-result.json</code> for I², τ², and Q.</p></section>
  <section><h3>sensitivity/influence diagnostics</h3><p>CLI equivalent: <code>rforge analysis sensitivity &lt;run-id&gt; --method influence</code></p></section>
  <section><h3>forest/funnel artifacts</h3>{{range .Manifests}}{{range .Plots}}<p>{{.Kind}} — {{.Path}}</p>{{end}}{{else}}<p>No forest/funnel plots registered.</p>{{end}}</section>
  <section><h3>publication-ready artifact manifests</h3>{{range .Manifests}}<p>{{.RunID}} manifest with {{len .Plots}} plot(s)</p>{{else}}<p>No artifact manifests found.</p>{{end}}</section>
</section>`))

var evidenceGridTemplate = template.Must(template.New("evidence-grid").Parse(`<section aria-labelledby="evidence-title" class="rf-card">
  <h2 id="evidence-title">Evidence extraction grid</h2>
  <p>Links each field to source passage/table/figure/equation support, parser offset, PDF view, reviewer status, correction history, and downstream analysis inclusion.</p>
  <p>CLI equivalent: <code>rforge evidence grid --out {{.GridPath}} [--parsed &lt;parsed.json&gt;] [--analysis &lt;run.json&gt;]</code></p>
  <div role="table" aria-label="Evidence extraction grid">
    <div role="row"><strong role="columnheader">Paper</strong><strong role="columnheader">Field</strong><strong role="columnheader">Support</strong><strong role="columnheader">parser offset</strong><strong role="columnheader">PDF view</strong><strong role="columnheader">reviewer status</strong><strong role="columnheader">correction history</strong><strong role="columnheader">downstream analysis inclusion</strong></div>
    {{range .Rows}}<div role="row"><span role="cell">{{.PaperID}}</span><span role="cell">{{.FieldName}}={{.FieldValue}}</span><span role="cell">{{.SupportKind}} {{.SupportRef}}</span><span role="cell">{{.ParserName}} {{.ParserOffset.Start}}-{{.ParserOffset.End}}</span><span role="cell"><a href="{{.PDFViewURL}}">PDF view</a></span><span role="cell">{{.ReviewerStatus}}</span><span role="cell">{{range .CorrectionHistory}}{{.Reviewer}} {{.Status}} {{.Note}}; {{end}}</span><span role="cell">{{.DownstreamAnalysisIncluded}}</span></div>{{else}}<div role="row"><span role="cell">No evidence grid rows. Run the CLI command to build {{$.GridPath}}.</span></div>{{end}}
  </div>
</section>`))

var retrievalTuningTemplate = template.Must(template.New("retrieval-tuning").Parse(`<section aria-labelledby="retrieval-title" class="rf-card">
  <h2 id="retrieval-title">Retrieval tuning</h2>
  <p>Compare SQLite FTS, OpenSearch, Qdrant vector, and hybrid results for the same query set with passage provenance, ranking explanations, embedding privacy status, and benchmark scores.</p>
  <p>CLI equivalent: <code>{{.CLIEquivalent}}</code></p>
  <h3>Benchmark scores</h3>
  <div role="table" aria-label="Retrieval backend benchmark scores">
    <div role="row"><strong role="columnheader">Backend</strong><strong role="columnheader">Recall@K</strong><strong role="columnheader">MRR</strong><strong role="columnheader">Score</strong><strong role="columnheader">embedding privacy status</strong><strong role="columnheader">ranking explanations</strong></div>
    {{range .Backends}}<div role="row"><span role="cell">{{if eq .Backend "sqlite-fts"}}SQLite FTS{{else if eq .Backend "qdrant"}}Qdrant vector{{else if eq .Backend "opensearch"}}OpenSearch{{else}}{{.Backend}}{{end}}</span><span role="cell">{{printf "%.3f" .RecallAtK}}</span><span role="cell">{{printf "%.3f" .MRR}}</span><span role="cell">{{printf "%.3f" .Score}}</span><span role="cell">{{.PrivacyNote}}</span><span role="cell">{{.ReproducibilityNote}}</span></div>{{end}}
  </div>
  <h3>Passage provenance</h3>
  {{range .QueryResults}}<p>{{.QueryID}} {{.Backend}} top passages: {{.TopK}}</p>{{end}}
  <h3>Privacy/reproducibility notes</h3>
  {{range .PrivacyNotes}}<p>{{.}}</p>{{end}}{{range .ReproducibilityNotes}}<p>{{.}}</p>{{end}}
</section>`))

var acquisitionQueueTemplate = template.Must(template.New("acquisition-queue").Parse(`<section aria-labelledby="acquisition-title" class="rf-card">
  <h2 id="acquisition-title">Legal full-text acquisition queue</h2>
  <p>OA/license status, source URL, expected stored path, restricted/shareable flags, and explicit reviewer approval before downloading or archiving documents.</p>
  <p>CLI equivalent: <code>rforge --project {{.ProjectPath}} oa acquisition-queue</code>; approve with <code>rforge --project {{.ProjectPath}} oa acquisition-approve &lt;id&gt; --reviewer &lt;name&gt; --reason &lt;text&gt;</code></p>
  <div role="table" aria-label="Legal acquisition queue">
    <div role="row"><strong role="columnheader">Paper</strong><strong role="columnheader">OA/license status</strong><strong role="columnheader">Source URL</strong><strong role="columnheader">Expected stored path</strong><strong role="columnheader">Policy</strong></div>
    {{range .Items}}<div role="row"><span role="cell">{{.PaperTitle}}</span><span role="cell">{{.OAStatus}} {{.License}}</span><span role="cell">{{.SourceURL}}</span><span role="cell">{{.ExpectedLocalPath}}</span><span role="cell">restricted={{.Restricted}} shareable={{.Shareable}} approved={{.ReviewerApproved}}</span></div>{{else}}<div role="row"><span role="cell">No acquisition queue items. Run the CLI command to build {{$.QueuePath}}.</span></div>{{end}}
  </div>
</section>`))

var connectorHealthTemplate = template.Must(template.New("connector-health").Parse(`<section aria-labelledby="connector-health-title" class="rf-card">
  <h2 id="connector-health-title">Connector health/control center</h2>
  <p>Shows live-smoke history, API drift alerts, cache status, credential redaction checks, rate-limit budgets, and adapter capability registry coverage. Live service checks are opt-in. This view reads stored API drift/live-smoke snapshots from <code>data/source-live-smoke-snapshots/latest.json</code> and alerts before connector use.</p>
  <p>Snapshot: {{if .Snapshot.CapturedAt}}{{.Snapshot.CapturedAt}}{{else}}not recorded{{end}}</p>
  <h3>API drift alerts</h3>
  {{if .Alerts}}
  <ul>
    {{range .Alerts}}<li><strong>{{.Label}}</strong>: {{.Kind}} — {{.Message}}</li>{{end}}
  </ul>
  {{else}}<p>No connector alerts.</p>{{end}}
  <h3>live-smoke history</h3>
  <div role="table" aria-label="Connector live-smoke snapshots">
    <div role="row"><strong role="columnheader">Connector</strong> <strong role="columnheader">Status</strong> <strong role="columnheader">Checked</strong> <strong role="columnheader">Message</strong></div>
    {{range .Snapshot.Results}}
    <div role="row"><span role="cell">{{.Label}}</span> <span role="cell">{{.Status}}</span> <span role="cell">{{.CheckedAt}}</span> <span role="cell">{{.Message}}</span></div>
    {{end}}
  </div>
  <h3>adapter capability registry coverage</h3>
  <div role="table" aria-label="Connector capability registry">
    <div role="row"><strong role="columnheader">Connector</strong> <strong role="columnheader">rate-limit budgets</strong> <strong role="columnheader">cache status</strong> <strong role="columnheader">credential redaction checks</strong></div>
    {{range .Registry.Connectors}}<div role="row"><span role="cell">{{.Label}}</span> <span role="cell">{{.RateLimitPolicy}}</span> <span role="cell">{{.Cacheability}}</span> <span role="cell">{{.AuthNeeds}}; secrets redacted in snapshots/logs</span></div>{{end}}
  </div>
</section>`))

var dedupeReviewTemplate = template.Must(template.New("dedupe-review").Parse(`<section aria-labelledby="dedupe-title" class="rf-card">
  <h2 id="dedupe-title">Dedupe/cluster review</h2>
  <p>A revtools-inspired visual clustering screen for duplicate review and screening triage, covering identity matches, conflicts, reversible decisions, exportable cluster decisions, and PRISMA/audit provenance.</p>
  <p>Export decision history: <code>rforge --json --project {{.ProjectPath}} library identity-decision log</code>; screening audit bundle: <code>rforge --project {{.ProjectPath}} screen audit-bundle --stage title-abstract --out data/screening-audit-bundle.json</code></p>
  <h3>Visual clustering for duplicate review</h3>
  <p>Cluster decisions are exportable through the reversible identity decision log and can be cross-checked against PRISMA counts.</p>
  <h3>Identity clusters</h3>
  {{if .Clusters}}
  {{range .Clusters}}
  <article class="identity-cluster">
    <h4>{{.ID}}</h4>
    <p>Records: {{.RecordIndexes}} Identifiers: {{.Identifiers}}</p>
    <ul>{{range .Matches}}<li>{{.Rule}} — {{.Explanation}}</li>{{end}}</ul>
  </article>
  {{end}}
  {{else}}<p>No duplicate identity clusters detected.</p>{{end}}
  <h3>Conflicting source fields</h3>
  {{if .Conflicts}}<ul>{{range .Conflicts}}<li>{{.ClusterID}} — {{.Severity}} — {{.Reason}}</li>{{end}}</ul>{{else}}<p>No unresolved identity conflicts.</p>{{end}}
  <h3>Zotero collection/tag context and citation-key preservation</h3>
  {{range .Records}}{{range .SourceRefs}}<p>{{.Source}} — collections={{index .Metadata "collections"}} tags={{index .Metadata "tags"}} citation-key preservation={{index .Metadata "citation-key"}}</p>{{end}}{{end}}
  <h3>Decision history</h3>
  {{if .DecisionLog.Decisions}}<ul>{{range .DecisionLog.Decisions}}<li>{{.ID}} — {{.Action}} — reversible={{.Reversible}} — {{.Reason}}</li>{{end}}</ul>{{else}}<p>No identity decisions recorded.</p>{{end}}
  <h3>screening triage</h3>
  <p>Use the linked screening audit bundle and PRISMA counts to route uncertain records from duplicate review into screening triage.</p>
  <h3>PRISMA/audit provenance</h3>
  <p>Records: {{.PRISMA.Records}} Screened: {{.PRISMA.Screened}} Included: {{.PRISMA.Included}}</p>
  {{if .AuditEvents}}<ul>{{range .AuditEvents}}<li>{{.Action}} — {{.Target}}</li>{{end}}</ul>{{else}}<p>No identity audit events recorded.</p>{{end}}
  <div role="table" aria-label="Cluster record titles">
    <div role="row"><strong role="columnheader">Record</strong><strong role="columnheader">DOI</strong></div>
    {{range .Records}}<div role="row"><span role="cell">{{.Title}}</span><span role="cell">{{.Identifiers.DOI}}</span></div>{{end}}
  </div>
</section>`))

var libraryTemplate = template.Must(template.New("library").Parse(`<section aria-labelledby="library-title" class="rf-card" hx-get="/library/rows" hx-trigger="refresh-library from:body">
  <h2 id="library-title">Library</h2>
  {{if .Empty}}
  <div role="status" class="empty-state">
    <p>No papers yet</p>
    <p>Import or search for papers to populate the library.</p>
  </div>
  {{else}}
  <div role="table" aria-label="Paper library">
    <div role="row">
      <strong role="columnheader">Paper title</strong>
    </div>
    {{range .Rows}}
    <div role="row">
      <span role="cell">{{.Title}}</span>
    </div>
    {{end}}
  </div>
  {{end}}
</section>`))

var projectTemplate = template.Must(template.New("project").Parse(`<section aria-labelledby="projects-title" class="rf-card">
  <h2 id="projects-title">Projects</h2>
  <form hx-post="/projects/create" hx-target="#project-status">
    <h3>Create project</h3>
    <label for="project-title">Title</label>
    <input id="project-title" name="title" required>
    <label for="project-create-path">Path</label>
    <input id="project-create-path" name="path" required>
    <button type="submit">Create project</button>
  </form>
  <form hx-post="/projects/open" hx-target="#project-status">
    <h3>Open project</h3>
    <label for="project-open-path">Path</label>
    <input id="project-open-path" name="path" required>
    <button type="submit">Open project</button>
  </form>
  <div id="project-status" role="status">No project open</div>
</section>`))

var projectStatusTemplate = template.Must(template.New("project-status").Parse(`<article class="rf-status">
  <h3>{{.Action}} project</h3>
  <dl>
    <dt>Title</dt><dd>{{.Project.Title}}</dd>
    <dt>Path</dt><dd>{{.Project.Path}}</dd>
    <dt>Storage</dt><dd>{{.Project.StorageMode}}</dd>
    <dt>Manifest</dt><dd>{{.Project.ManifestPath}}</dd>
  </dl>
</article>`))

var ossTemplate = template.Must(template.New("oss").Parse(`<section aria-labelledby="oss-title" class="rf-card" hx-get="/oss/repositories" hx-trigger="refresh-oss from:body">
  <h2 id="oss-title">OSS repository studies</h2>
  {{if .Repositories}}
  <div role="table" aria-label="OSS repository studies">
    <div role="row">
      <strong role="columnheader">Repository</strong>
    </div>
    {{range .Repositories}}
    <div role="row">
      <span role="cell">{{.Name}}</span>
    </div>
    {{end}}
  </div>
  {{else}}
  <div role="status" class="empty-state">
    <p>No repository studies yet</p>
    <p>Run an OSS repository study to compare repository evidence.</p>
  </div>
  {{end}}
</section>`))

var forgeHomeTemplate = template.Must(template.New("forge-home").Parse(`<section aria-labelledby="forge-title" class="rf-card" hx-get="/forge/refresh" hx-trigger="refresh-forge from:body">
  <h2 id="forge-title">Forge home</h2>
  <p>Active project: {{if .ProjectTitle}}{{.ProjectTitle}}{{else}}{{.ActiveProject}}{{end}}</p>
  <p>Current state: <strong>{{.CurrentState}}</strong></p>
  <section aria-labelledby="forge-provenance-title">
    <h3 id="forge-provenance-title">Provenance timeline</h3>
    {{range .ProvenanceEvents}}<p>{{.Timestamp}} {{.Action}} {{.Target}}</p>{{else}}<p>No provenance events recorded.</p>{{end}}
  </section>
  <section aria-labelledby="forge-gates-title">
    <h3 id="forge-gates-title">Blocked review gates</h3>
    {{range .BlockedReviewGates}}<p>{{.Gate}} — {{.Reason}}</p>{{else}}<p>No blocked review gates detected for this state.</p>{{end}}
  </section>
  <section aria-labelledby="forge-jobs-title">
    <h3 id="forge-jobs-title">Background jobs</h3>
    {{range .BackgroundJobs}}<p>{{.ID}} — {{.Status}} — <code>{{.Command}}</code></p>{{else}}<p>No background jobs recorded.</p>{{end}}
  </section>
  <section aria-labelledby="forge-actions-title">
    <h3 id="forge-actions-title">Next safe actions</h3>
    {{range .NextSafeActions}}<p>{{.Label}} — CLI equivalent: <code>{{.CLI}}</code></p>{{else}}<p>No next actions available.</p>{{end}}
  </section>
</section>`))

type ForgeHomeState struct {
	ActiveProject      string
	ProjectTitle       string
	CurrentState       string
	ProvenanceEvents   []provenance.Event
	BlockedReviewGates []ForgeGate
	BackgroundJobs     []ForgeBackgroundJob
	NextSafeActions    []ForgeNextAction
}

type ForgeGate struct{ Gate, Reason string }
type ForgeBackgroundJob struct{ ID, Status, Command string }
type ForgeNextAction struct{ Label, CLI string }

var screeningCockpitTemplate = template.Must(template.New("screening-cockpit").Parse(`<section aria-labelledby="screening-title" class="rf-card" hx-get="/screening/refresh" hx-trigger="refresh-screening from:body">
  <h2 id="screening-title">Screening cockpit</h2>
  <p>ASReview-inspired screening cockpit for active-learning review and auditable human decisions.</p>
  <p>Stage: {{.Stage}} Records: {{.TotalRecords}} Active run: {{if .ActiveRunID}}{{.ActiveRunID}}{{else}}not generated{{end}}</p>
  <section aria-labelledby="screening-active-title">
    <h3 id="screening-active-title">Active-learning queue</h3>
    <div role="table" aria-label="Active-learning queue with uncertainty and exploration flags">
      <div role="row"><strong role="columnheader">Paper</strong><strong role="columnheader">Score</strong><strong role="columnheader">uncertainty</strong><strong role="columnheader">exploration</strong><strong role="columnheader">policy</strong></div>
      {{range .ActiveLearningQueue}}<div role="row"><span role="cell">{{.ID}}</span><span role="cell">{{printf "%.3f" .Score}}</span><span role="cell">{{printf "%.3f" .Uncertainty}}</span><span role="cell">{{printf "%.3f" .ExplorationScore}}</span><span role="cell">{{.Policy}}</span></div>{{else}}<p>No active-learning records queued.</p>{{end}}
    </div>
  </section>
  <section aria-labelledby="screening-uncertainty-title">
    <h3 id="screening-uncertainty-title">Uncertainty/exploration flags</h3>
    {{range .UncertaintyQueue}}<p>{{.ID}} uncertainty={{printf "%.3f" .Uncertainty}} exploration={{printf "%.3f" .ExplorationScore}}</p>{{else}}<p>No uncertainty queue records.</p>{{end}}
    <h4>Uncertain reviewer decisions</h4>
    {{range .UncertainQueue}}<p>{{.PaperID}}</p>{{else}}<p>No unresolved uncertain decisions.</p>{{end}}
  </section>
  <section aria-labelledby="screening-assignment-title">
    <h3 id="screening-assignment-title">Reviewer assignment</h3>
    <p>CLI equivalent: <code>rforge screen assign --stage {{.Stage}} --reviewer &lt;name&gt; --out data/screening-assignments.json</code></p>
  </section>
  <section aria-labelledby="screening-conflict-title">
    <h3 id="screening-conflict-title">Conflict/adjudication panels</h3>
    <p>{{.Progress.Conflicts}} conflicts. CLI equivalent: <code>rforge screen panel --stage {{.Stage}} --out data/screening-panel.json</code></p>
  </section>
  <section aria-labelledby="screening-progress-title">
    <h3 id="screening-progress-title">Progress metrics</h3>
    <p>{{.Progress.ScreenedRecords}} screened, {{.Progress.Remaining}} remaining, {{.Progress.Conflicts}} conflicts</p>
    {{range .Progress.Reviewers}}<p>{{.Reviewer}}: {{.Decisions}} decisions</p>{{end}}
  </section>
  <section aria-labelledby="screening-recall-title">
    <h3 id="screening-recall-title">Recall/effort curves</h3>
    <p>CLI equivalent: <code>rforge screen recall --stage {{.Stage}}</code></p>
  </section>
  <section aria-labelledby="screening-stopping-title">
    <h3 id="screening-stopping-title">Stopping diagnostics</h3>
    <p>Can stop: {{.Stopping.CanStop}} Current recall: {{printf "%.3f" .Stopping.CurrentRecall}} Target: {{printf "%.3f" .Stopping.TargetRecall}}</p>
    <p>{{.Stopping.Reason}}</p>
  </section>
  <section aria-labelledby="screening-audit-title">
    <h3 id="screening-audit-title">Exportable audit bundle links</h3>
    {{if .HasAuditBundle}}<p><a href="/{{.AuditBundlePath}}">screening-audit-bundle.json</a></p>{{else}}<p>No screening audit bundle exported yet. Run <code>rforge screen audit-bundle --stage {{.Stage}} --out {{.AuditBundlePath}}</code>.</p>{{end}}
  </section>
</section>`))

// ScreeningCockpitState combines active-learning, uncertainty, progress,
// stopping, and audit-bundle state for the HTMX screening cockpit.
type ScreeningCockpitState struct {
	ProjectPath         string
	Stage               screening.Stage
	TotalRecords        int
	ActiveRunID         string
	ActiveLearningQueue []screening.PrioritizedRecord
	UncertaintyQueue    []screening.PrioritizedRecord
	UncertainQueue      []screening.UncertainQueueItem
	Progress            screening.ProgressReport
	Stopping            screening.StoppingRecommendation
	HasAuditBundle      bool
	AuditBundlePath     string
}

// PRISMAFlowState summarizes CLI-generated screening counts for the web artifacts view.
type PRISMAFlowState struct {
	Records  int
	Screened int
	Included int
}

// AnalysisDetail surfaces a stored meta-analysis result for the cockpit:
// heterogeneity metrics, plot availability, and runner warnings.
type AnalysisDetail struct {
	Ready         bool
	RunID         string
	I2            float64
	Tau2          float64
	Q             float64
	HasForestPlot bool
	HasFunnelPlot bool
	Warnings      []string
}

// ArtifactDashboardState combines CLI-generated outputs for the local web GUI.
type ArtifactDashboardState struct {
	Papers         ui.LibraryViewModel
	Analysis       ui.AnalysisViewModel
	AnalysisDetail AnalysisDetail
	CitationGraph  ui.CitationGraphViewModel
	PRISMA         PRISMAFlowState
	Reports        ui.ReportViewModel
}

// CitationGraphSVG renders a small accessible SVG preview for exported citation graphs.
func (s ArtifactDashboardState) CitationGraphSVG() template.HTML {
	if len(s.CitationGraph.Nodes) == 0 {
		return ""
	}
	positions := map[string][2]int{}
	var b strings.Builder
	b.WriteString(`<svg role="img" aria-label="Citation graph visualization" viewBox="0 0 520 160" class="citation-graph-svg">`)
	b.WriteString(`<title>Citation graph visualization</title>`)
	for i, node := range s.CitationGraph.Nodes {
		x := 80 + (i%5)*100
		y := 50 + (i/5)*70
		positions[node.ID] = [2]int{x, y}
	}
	for _, edge := range s.CitationGraph.Edges {
		source, sourceOK := positions[edge.Source]
		target, targetOK := positions[edge.Target]
		if !sourceOK || !targetOK {
			continue
		}
		fmt.Fprintf(&b, `<line x1="%d" y1="%d" x2="%d" y2="%d" stroke="currentColor" stroke-width="1" />`, source[0], source[1], target[0], target[1])
	}
	for _, node := range s.CitationGraph.Nodes {
		pos := positions[node.ID]
		label := template.HTMLEscapeString(node.ID)
		stem := template.HTMLEscapeString(graphNodeStem(node.ID))
		fmt.Fprintf(&b, `<a href="/papers/%s">`, stem)
		fmt.Fprintf(&b, `<circle cx="%d" cy="%d" r="16" fill="none" stroke="currentColor" stroke-width="2" />`, pos[0], pos[1])
		fmt.Fprintf(&b, `<text x="%d" y="%d" text-anchor="middle">%s</text>`, pos[0], pos[1]+34, label)
		b.WriteString(`</a>`)
	}
	b.WriteString(`</svg>`)
	return template.HTML(b.String())
}

var artifactsTemplate = template.Must(template.New("artifacts").Parse(`<section aria-labelledby="artifacts-title" class="rf-card" hx-get="/artifacts/refresh" hx-trigger="refresh-artifacts from:body">
  <h2 id="artifacts-title">CLI-generated artifacts</h2>
  <section aria-labelledby="artifact-papers-title">
    <h3 id="artifact-papers-title">Papers</h3>
    {{if .Papers.Rows}}{{range .Papers.Rows}}<p>{{.Title}}</p>{{end}}{{else}}<p>No papers exported yet</p>{{end}}
  </section>
  <section aria-labelledby="artifact-analysis-title">
    <h3 id="artifact-analysis-title">Meta-analysis outputs</h3>
    {{if .AnalysisDetail.Ready}}
    <p>Run: {{.AnalysisDetail.RunID}}</p>
    <dl>
      <dt>Heterogeneity I²</dt><dd>{{.AnalysisDetail.I2}}</dd>
      <dt>τ²</dt><dd>{{.AnalysisDetail.Tau2}}</dd>
      <dt>Q</dt><dd>{{.AnalysisDetail.Q}}</dd>
    </dl>
    <ul>
      {{if .AnalysisDetail.HasForestPlot}}<li>Forest plot registered</li>{{end}}
      {{if .AnalysisDetail.HasFunnelPlot}}<li>Funnel plot registered</li>{{end}}
    </ul>
    {{if .AnalysisDetail.Warnings}}<p class="analysis-warnings">Warnings: {{range .AnalysisDetail.Warnings}}{{.}}; {{end}}</p>{{end}}
    {{else}}<p>No analysis run ready</p>{{end}}
  </section>
  <section aria-labelledby="artifact-prisma-title">
    <h3 id="artifact-prisma-title">PRISMA diagram</h3>
    <p>Records: {{.PRISMA.Records}}</p>
    <p>Screened: {{.PRISMA.Screened}}</p>
    <p>Included: {{.PRISMA.Included}}</p>
  </section>
  <section aria-labelledby="artifact-citations-title">
    <h3 id="artifact-citations-title">Citation graph</h3>
    {{if .CitationGraph.Nodes}}
    <div id="citation-graph" data-citation-graph data-src="/artifacts/graph.json" role="application" aria-label="Citation graph (drag to pan, scroll to zoom, click a node to open the paper)">
      {{.CitationGraphSVG}}
    </div>
    <script src="/assets/citation-graph.js" defer></script>
    {{range .CitationGraph.Nodes}}<p>{{.ID}}</p>{{end}}
    {{range .CitationGraph.Edges}}<p>{{.Source}} → {{.Target}}</p>{{end}}
    {{else}}<p>No citation graph exported yet</p>{{end}}
  </section>
  <section aria-labelledby="artifact-reports-title">
    <h3 id="artifact-reports-title">Report artifacts</h3>
    {{if .Reports.Formats}}{{range .Reports.Formats}}<p>{{.}}</p>{{end}}{{else}}<p>No report formats exported yet</p>{{end}}
  </section>
</section>`))

// NewShellHandler returns the dependency-light local web GUI shell handler.
func NewShellHandler() http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "text/html; charset=utf-8")
		_ = shellTemplate.Execute(w, nil)
	})
}

// NewSearchHandler renders the HTMX search screen from the shared search view model.
func NewSearchHandler(state ui.SearchFormState) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "text/html; charset=utf-8")
		_ = searchTemplate.Execute(w, state)
	})
}

// NewSourcePlanningHandler renders the source-planning cockpit over protocol compiler output.
func NewSourcePlanningHandler() http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		question := strings.TrimSpace(r.URL.Query().Get("question"))
		if question == "" {
			question = "Does the intervention improve the outcome compared with control?"
		}
		plan, err := protocol.CompileSourcePlanFromQuestion(protocol.QuestionInput{
			Framework:    r.URL.Query().Get("type"),
			Question:     question,
			Population:   r.URL.Query().Get("population"),
			Intervention: r.URL.Query().Get("intervention"),
			Comparator:   r.URL.Query().Get("comparator"),
			Outcome:      r.URL.Query().Get("outcome"),
			Exposure:     r.URL.Query().Get("exposure"),
			Sample:       r.URL.Query().Get("sample"),
			Phenomenon:   r.URL.Query().Get("phenomenon"),
			Design:       r.URL.Query().Get("design"),
			Evaluation:   r.URL.Query().Get("evaluation"),
			ResearchType: r.URL.Query().Get("research-type"),
		})
		if err != nil {
			http.Error(w, err.Error(), http.StatusBadRequest)
			return
		}
		w.Header().Set("Content-Type", "text/html; charset=utf-8")
		_ = sourcePlanningTemplate.Execute(w, plan)
	})
}

type connectorHealthView struct {
	Snapshot protocol.ConnectorLiveSmokeSnapshot
	Alerts   []protocol.ConnectorLiveSmokeAlert
	Registry protocol.ConnectorCapabilityRegistry
}

func newConnectorHealthHandler(projectPath func() string) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		registry := protocol.DefaultConnectorCapabilityRegistry()
		now := time.Now().UTC()
		snapshot := protocol.NewLiveSmokeSnapshot(registry, now)
		if project := strings.TrimSpace(projectPath()); project != "" {
			path := filepath.Join(project, "data", "source-live-smoke-snapshots", "latest.json")
			if loaded, err := protocol.LoadLiveSmokeSnapshot(path); err == nil {
				snapshot = loaded
			}
		}
		view := connectorHealthView{Snapshot: snapshot, Alerts: protocol.ConnectorLiveSmokeAlerts(registry, snapshot, now), Registry: registry}
		w.Header().Set("Content-Type", "text/html; charset=utf-8")
		_ = connectorHealthTemplate.Execute(w, view)
	})
}

func newDedupeReviewHandler(projectPath func() string) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		state, err := BuildDedupeReviewState(projectPath())
		if err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}
		w.Header().Set("Content-Type", "text/html; charset=utf-8")
		_ = dedupeReviewTemplate.Execute(w, state)
	})
}

func NewPrivacyModelHandler(state DashboardPrivacyModel) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "text/html; charset=utf-8")
		_ = privacyModelTemplate.Execute(w, state)
	})
}

func NewInformationArchitectureHandler(state DashboardInformationArchitecture) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "text/html; charset=utf-8")
		_ = informationArchitectureTemplate.Execute(w, state)
	})
}

func NewWorkbenchIndexHandler(state WorkbenchIndexState) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "text/html; charset=utf-8")
		_ = workbenchIndexTemplate.Execute(w, state)
	})
}

func newPackageExportCenterHandler(projectPath func() string) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "text/html; charset=utf-8")
		_ = packageExportCenterTemplate.Execute(w, BuildPackageExportCenterState(projectPath()))
	})
}

func newReportClaimPanelHandler(projectPath func() string) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "text/html; charset=utf-8")
		_ = reportClaimPanelTemplate.Execute(w, BuildReportClaimPanelState(projectPath()))
	})
}

func newAnalysisWorkbenchHandler(projectPath func() string) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "text/html; charset=utf-8")
		_ = analysisWorkbenchTemplate.Execute(w, BuildAnalysisWorkbenchState(projectPath()))
	})
}

func newEvidenceGridHandler(projectPath func() string) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "text/html; charset=utf-8")
		_ = evidenceGridTemplate.Execute(w, BuildEvidenceGridState(projectPath()))
	})
}

func newRetrievalTuningHandler(projectPath func() string) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "text/html; charset=utf-8")
		_ = retrievalTuningTemplate.Execute(w, BuildRetrievalTuningState(projectPath()))
	})
}

func newAcquisitionQueueHandler(projectPath func() string) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "text/html; charset=utf-8")
		_ = acquisitionQueueTemplate.Execute(w, BuildAcquisitionQueueState(projectPath()))
	})
}

func newForgeHomeHandler(projectPath func() string) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		state, err := BuildForgeHomeState(projectPath())
		if err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}
		NewForgeHomeHandler(state).ServeHTTP(w, r)
	})
}

// NewForgeHomeHandler renders the Forge home timeline.
func NewForgeHomeHandler(state ForgeHomeState) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "text/html; charset=utf-8")
		_ = forgeHomeTemplate.Execute(w, state)
	})
}

func newScreeningCockpitHandler(projectPath func() string) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		state, err := BuildScreeningCockpitState(projectPath())
		if err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}
		NewScreeningCockpitHandler(state).ServeHTTP(w, r)
	})
}

// NewScreeningCockpitHandler renders the HTMX screening cockpit.
func NewScreeningCockpitHandler(state ScreeningCockpitState) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "text/html; charset=utf-8")
		_ = screeningCockpitTemplate.Execute(w, state)
	})
}

// NewLibraryHandler renders the HTMX library screen from the shared library view model.
func NewLibraryHandler(state ui.LibraryViewModel) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "text/html; charset=utf-8")
		_ = libraryTemplate.Execute(w, state)
	})
}

// NewProjectHandler renders the HTMX create/open project screen.
func NewProjectHandler() http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "text/html; charset=utf-8")
		_ = projectTemplate.Execute(w, nil)
	})
}

// NewOSSHandler renders OSS repository studies from the shared OSS dashboard view model.
func NewOSSHandler(state ui.OSSDashboardViewModel) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "text/html; charset=utf-8")
		_ = ossTemplate.Execute(w, state)
	})
}

// NewArtifactsHandler renders CLI-generated papers, analyses, diagrams, and reports.
func NewArtifactsHandler(state ArtifactDashboardState) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "text/html; charset=utf-8")
		_ = artifactsTemplate.Execute(w, state)
	})
}

// NewCreateProjectHandler creates a local project workspace from an HTMX form post.
func NewCreateProjectHandler() http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if err := r.ParseForm(); err != nil {
			http.Error(w, err.Error(), http.StatusBadRequest)
			return
		}
		proj, err := project.Create(r.FormValue("path"), project.CreateOptions{Title: r.FormValue("title")})
		if err != nil {
			http.Error(w, err.Error(), http.StatusBadRequest)
			return
		}
		renderProjectStatus(w, "Created", proj)
	})
}

// NewOpenProjectHandler opens an existing local project workspace from an HTMX form post.
func NewOpenProjectHandler() http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if err := r.ParseForm(); err != nil {
			http.Error(w, err.Error(), http.StatusBadRequest)
			return
		}
		proj, err := project.Inspect(r.FormValue("path"))
		if err != nil {
			http.Error(w, err.Error(), http.StatusBadRequest)
			return
		}
		renderProjectStatus(w, "Opened", proj)
	})
}

func renderProjectStatus(w http.ResponseWriter, action string, proj project.Project) {
	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	w.Header().Set("HX-Trigger", "project-opened")
	_ = projectStatusTemplate.Execute(w, struct {
		Action  string
		Project project.Project
	}{Action: action, Project: proj})
}
