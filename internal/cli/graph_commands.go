package cli

import (
	"context"
	"encoding/json"
	"fmt"
	"html"
	"io"
	"os"
	"path/filepath"
	"strings"

	"github.com/TrebuchetDynamics/research-forge/internal/knowledge"
	"github.com/TrebuchetDynamics/research-forge/internal/library"
)

func executeGraph(args []string, stdout, stderr io.Writer, opts globalOptions) int {
	if len(args) != 1 || args[0] != "papers" {
		return writeError(stdout, stderr, opts, 2, "usage", "usage: rforge --project <path> graph papers")
	}
	if opts.Project == "" {
		return writeError(stdout, stderr, opts, 2, "missing_project", "--project is required for graph papers")
	}
	fetches := fetchPDFsResult{}
	if _, err := os.Stat(filepath.Join(opts.Project, "data", "library.json")); err == nil {
		store, err := library.OpenStore(filepath.Join(opts.Project, "data", "library.json"))
		if err != nil {
			return writeError(stdout, stderr, opts, 1, "library_open_failed", err.Error())
		}
		records, err := store.List()
		if err != nil {
			return writeError(stdout, stderr, opts, 1, "library_read_failed", err.Error())
		}
		fetches = fetchProjectPDFs(context.Background(), opts.Project, records)
	}
	graph, err := knowledge.BuildProjectKnowledgeGraphFromProject(opts.Project)
	if err != nil {
		return writeError(stdout, stderr, opts, 1, "graph_build_failed", err.Error())
	}
	jsonPath := filepath.Join(opts.Project, "data", "knowledge-graph.json")
	htmlPath := filepath.Join(opts.Project, "data", "knowledge-graph.html")
	reportPath := filepath.Join(opts.Project, "data", "knowledge-graph-report.md")
	if err := writeJSONFile(jsonPath, graph); err != nil {
		return writeError(stdout, stderr, opts, 1, "graph_write_failed", err.Error())
	}
	if err := os.WriteFile(htmlPath, []byte(knowledgeGraphHTML(graph)), 0o644); err != nil {
		return writeError(stdout, stderr, opts, 1, "graph_html_failed", err.Error())
	}
	if err := os.WriteFile(reportPath, []byte(knowledge.BuildKnowledgeGraphReport(graph)), 0o644); err != nil {
		return writeError(stdout, stderr, opts, 1, "graph_report_failed", err.Error())
	}
	if opts.JSON {
		return writeJSON(stdout, 0, map[string]any{"graph": jsonPath, "html": htmlPath, "report": reportPath, "nodes": len(graph.Nodes), "edges": len(graph.Edges), "fetched": len(fetches.assets), "fetchFailures": fetches.failures})
	}
	fmt.Fprintf(stdout, "wrote paper graph with %d nodes and %d edges (fetched %d PDFs)\n%s\n%s\n%s\n", len(graph.Nodes), len(graph.Edges), len(fetches.assets), jsonPath, htmlPath, reportPath)
	return 0
}

func knowledgeGraphHTML(graph knowledge.ProjectKnowledgeGraph) string {
	data, _ := json.Marshal(graph)
	var b strings.Builder
	b.WriteString(`<!doctype html><meta charset="utf-8"><title>Paper knowledge graph</title>
<style>
:root{color-scheme:light dark;--paper:#2563eb;--concept:#16a34a;--other:#64748b;--edge:#94a3b8;--panel:#f8fafc}body{font-family:Inter,ui-sans-serif,system-ui;margin:0;background:#0f172a;color:#e2e8f0}.shell{display:grid;grid-template-columns:minmax(0,1fr) 320px;min-height:100vh}.stage{padding:1rem}.panel{background:#111827;border-left:1px solid #334155;padding:1rem;overflow:auto}h1{font-size:1.35rem;margin:.25rem 0}.toolbar{display:flex;gap:.75rem;align-items:center;flex-wrap:wrap;margin:.75rem 0}input{background:#020617;color:#e2e8f0;border:1px solid #475569;border-radius:.5rem;padding:.55rem .7rem;min-width:18rem}button{background:#1e293b;color:#e2e8f0;border:1px solid #475569;border-radius:.5rem;padding:.5rem .7rem}svg{width:100%;height:72vh;min-height:520px;border:1px solid #334155;border-radius:1rem;background:radial-gradient(circle at 50% 40%,#1e293b,#020617)}line{stroke:var(--edge);stroke-opacity:.45}.node circle{stroke:#e2e8f0;stroke-width:1.2}.node text{fill:#e2e8f0;font-size:11px;paint-order:stroke;stroke:#020617;stroke-width:3px}.paper circle{fill:var(--paper)}.concept circle{fill:var(--concept)}.other circle{fill:var(--other)}.muted{opacity:.12}.selected circle{stroke:#fbbf24;stroke-width:4}.legend{display:flex;gap:.75rem;font-size:.9rem}.dot{display:inline-block;width:.8rem;height:.8rem;border-radius:999px;margin-right:.25rem}.fallback{padding:1rem 2rem}li{margin:.35rem 0}small{color:#94a3b8}</style>
<div class="shell"><main class="stage"><h1>Paper knowledge graph</h1>`)
	fmt.Fprintf(&b, "<p>Interactive force-directed UI. Nodes: %d · Edges: %d. Drag nodes, search concepts/papers, click for supporting snippets. Colors indicate detected communities.</p>", len(graph.Nodes), len(graph.Edges))
	b.WriteString(`<div class="toolbar"><input id="filter" type="search" placeholder="Filter papers, concepts, snippets…" aria-label="Filter graph"><button id="reset" type="button">Reset layout</button><span class="legend"><span><i class="dot" style="background:var(--paper)"></i>paper</span><span><i class="dot" style="background:var(--concept)"></i>concept</span><span><i class="dot" style="background:var(--other)"></i>other</span></span></div><svg id="graph" viewBox="0 0 1200 760" role="img" aria-label="Interactive paper knowledge graph"><g id="edges"></g><g id="nodes"></g></svg></main><aside class="panel"><h2>Selection</h2><div id="details">Click a node to see paper/concept details and source snippet.</div><h2>Neighbors</h2><ul id="neighbors"></ul></aside></div>`)
	b.WriteString(`<script type="application/json" id="graph-data">`)
	b.Write(data)
	b.WriteString(`</script><script>
const graph=JSON.parse(document.getElementById('graph-data').textContent);const W=1200,H=760;const svg=document.getElementById('graph'),edgeLayer=document.getElementById('edges'),nodeLayer=document.getElementById('nodes'),details=document.getElementById('details'),neighbors=document.getElementById('neighbors');
let nodes=graph.nodes.map((n,i)=>({...n,x:W/2+Math.cos(i)*260*Math.random(),y:H/2+Math.sin(i)*220*Math.random(),vx:0,vy:0,hidden:false,community:-1}));let byId=new Map(nodes.map(n=>[n.id,n]));let edges=graph.edges.map(e=>({...e,sourceNode:byId.get(e.source),targetNode:byId.get(e.target)})).filter(e=>e.sourceNode&&e.targetNode);const palette=['#60a5fa','#34d399','#fbbf24','#f472b6','#a78bfa','#22d3ee','#fb7185','#84cc16'];assignCommunities();let selected=null,drag=null;
function assignCommunities(){let c=0;for(const n of nodes){if(n.community>=0)continue;const q=[n];n.community=c;n.hidden=false;while(q.length){const cur=q.shift();for(const e of edges){const other=e.sourceNode===cur?e.targetNode:e.targetNode===cur?e.sourceNode:null;if(other&&other.community<0){other.community=c;q.push(other)}}}c++}}
function kindClass(n){return n.kind==='paper'?'paper':n.kind==='concept'?'concept':'other'}function radius(n){return n.kind==='paper'?10:n.kind==='concept'?8:6}function short(s){s=(s||'').replace(/\s+/g,' ');return s.length>28?s.slice(0,27)+'…':s}
function draw(){edgeLayer.textContent='';nodeLayer.textContent='';for(const e of edges){const line=document.createElementNS('http://www.w3.org/2000/svg','line');line.setAttribute('x1',e.sourceNode.x);line.setAttribute('y1',e.sourceNode.y);line.setAttribute('x2',e.targetNode.x);line.setAttribute('y2',e.targetNode.y);line.classList.toggle('muted',e.sourceNode.hidden||e.targetNode.hidden);edgeLayer.appendChild(line)}for(const n of nodes){const g=document.createElementNS('http://www.w3.org/2000/svg','g');g.classList.add('node',kindClass(n));g.classList.toggle('muted',n.hidden);if(selected&&selected.id===n.id)g.classList.add('selected');g.setAttribute('transform','translate('+n.x+','+n.y+')');const c=document.createElementNS('http://www.w3.org/2000/svg','circle');c.setAttribute('r',radius(n));c.style.fill=palette[n.community%palette.length];const t=document.createElementNS('http://www.w3.org/2000/svg','text');t.setAttribute('x',radius(n)+4);t.setAttribute('y',4);t.textContent=short(n.label||n.id);g.append(c,t);g.addEventListener('pointerdown',ev=>{drag=n;select(n);g.setPointerCapture(ev.pointerId)});g.addEventListener('pointermove',ev=>{if(drag===n){const p=point(ev);n.x=p.x;n.y=p.y;n.vx=n.vy=0;draw()}});g.addEventListener('pointerup',()=>drag=null);g.addEventListener('click',()=>select(n));nodeLayer.appendChild(g)}}
function point(ev){const p=svg.createSVGPoint();p.x=ev.clientX;p.y=ev.clientY;return p.matrixTransform(svg.getScreenCTM().inverse())}
function tick(){for(let k=0;k<2;k++){for(const e of edges){if(e.sourceNode.hidden||e.targetNode.hidden)continue;const dx=e.targetNode.x-e.sourceNode.x,dy=e.targetNode.y-e.sourceNode.y,d=Math.hypot(dx,dy)||1,force=(d-120)*0.004,fx=dx/d*force,fy=dy/d*force;e.sourceNode.vx+=fx;e.sourceNode.vy+=fy;e.targetNode.vx-=fx;e.targetNode.vy-=fy}for(let i=0;i<nodes.length;i++)for(let j=i+1;j<nodes.length;j++){const a=nodes[i],b=nodes[j];if(a.hidden||b.hidden)continue;const dx=b.x-a.x,dy=b.y-a.y,d2=Math.max(80,dx*dx+dy*dy),f=1800/d2,ix=dx/Math.sqrt(d2)*f,iy=dy/Math.sqrt(d2)*f;a.vx-=ix;a.vy-=iy;b.vx+=ix;b.vy+=iy}for(const n of nodes){if(n.hidden||drag===n)continue;n.vx+=(W/2-n.x)*0.002;n.vy+=(H/2-n.y)*0.002;n.vx*=0.88;n.vy*=0.88;n.x=Math.max(20,Math.min(W-20,n.x+n.vx));n.y=Math.max(20,Math.min(H-20,n.y+n.vy))}}draw();requestAnimationFrame(tick)}
function select(n){selected=n;const props=n.properties||{};details.innerHTML='<h3>'+escapeHtml(n.label||n.id)+'</h3><p><b>'+escapeHtml(n.kind)+'</b><br><code>'+escapeHtml(n.id)+'</code></p><p>'+escapeHtml(props.snippet||props.values||'No snippet recorded.')+'</p>';neighbors.textContent='';for(const e of edges.filter(e=>e.source===n.id||e.target===n.id)){const other=e.source===n.id?e.targetNode:e.sourceNode;const li=document.createElement('li');li.textContent=e.kind+' → '+(other.label||other.id);neighbors.appendChild(li)}draw()}
function escapeHtml(s){return String(s).replace(/[&<>"']/g,c=>({'&':'&amp;','<':'&lt;','>':'&gt;','"':'&quot;',"'":'&#39;'}[c]))}
document.getElementById('filter').addEventListener('input',ev=>{const q=ev.target.value.toLowerCase().trim();for(const n of nodes){const hay=(n.id+' '+n.kind+' '+(n.label||'')+' '+JSON.stringify(n.properties||{})).toLowerCase();n.hidden=q&&!hay.includes(q)}draw()});document.getElementById('reset').addEventListener('click',()=>{nodes.forEach((n,i)=>{n.x=W/2+Math.cos(i*2*Math.PI/nodes.length)*330;n.y=H/2+Math.sin(i*2*Math.PI/nodes.length)*260;n.vx=n.vy=0});draw()});draw();requestAnimationFrame(tick);
</script><div class="fallback"><h2>Static node list</h2><ul>`)
	for _, node := range graph.Nodes {
		fmt.Fprintf(&b, "<li><strong>%s</strong> %s <small>%s</small></li>", html.EscapeString(node.Kind), html.EscapeString(node.Label), html.EscapeString(node.Properties["snippet"]))
	}
	b.WriteString("</ul></div>")
	return b.String()
}

func shortLabel(value string) string {
	value = strings.Join(strings.Fields(value), " ")
	if len(value) > 28 {
		return value[:27] + "…"
	}
	return value
}
