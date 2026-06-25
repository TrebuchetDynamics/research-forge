# Interactive paper knowledge graphs in the HTMX dashboard

## Method and limits

This research pass targeted one product question: how should ResearchForge add a fancy interactive JavaScript graph for paper knowledge graphs without making the user workflow complex? I searched OpenAlex, Crossref, Semantic Scholar, and arXiv for five query families around force-directed graph visualization, SVG/Canvas/WebGL performance, Three.js network graphs, accessibility, and HTMX/progressive enhancement. I also ran the local `modern-web-guidance` search/retrieve flow for browser accessibility and canvas/HTML interaction guidance.

Source coverage from `rforge search stats --dir .`: 100 arXiv records, 80 OpenAlex records, 61 Crossref records, 24 Semantic Scholar records, 210 unique DOI-bearing records. This is implementation-design research, not a formal systematic review.

## Bottom line

Do **not** start with Three.js for ResearchForge paper graphs. Start with progressive enhancement: server writes `knowledge-graph.json` plus a browser-openable HTML page; the page renders an inline SVG force-directed graph with search, drag, click-to-details, and an accessible node list fallback. Add Canvas/WebGL or Three.js only if real projects exceed SVG performance limits or need 3D spatial navigation.

## What the evidence suggests

### SVG force-directed graph is the right first fancy layer

The search surfaced many web graph / visualization systems that use web-native graph views, including D3/Cytoscape-style web graph tooling and biological/network visualizers. For ResearchForge, the graph is semantic and explanatory: papers, concepts, citations, claims, snippets. SVG keeps nodes/edges inspectable, easy to style, keyboard/ARIA-adaptable, and simple to embed in an HTMX/no-build Go template.

Product fit:

- good for small-to-medium research graphs;
- no frontend build chain;
- works as a static artifact (`data/knowledge-graph.html`);
- easy click/hover/drag interactions;
- graceful fallback to tables/lists.

### Canvas/WebGL help performance, but hurt accessibility and inspectability

Canvas/WebGL are better when graphs get very large or when edge/node counts cause SVG layout and DOM updates to lag. The modern web guidance emphasizes that canvas-rendered content is not naturally exposed to browser features and assistive technology unless mirrored with DOM/accessibility structures. That means Canvas/WebGL should come with a parallel list/table and a details pane. It is a performance tier, not the first UX tier.

### Three.js is only justified for 3D or very large graph exploration

Three.js is useful if ResearchForge later needs 3D spatial clusters, GPU-accelerated points/edges, camera controls, or immersive exploration. It is overkill for the first paper graph because it adds dependency weight, 3D interaction complexity, and more accessibility burden. For most literature review use, users need semantic filtering and supporting snippets more than 3D movement.

### HTMX should own workflow, not graph physics

HTMX remains useful for the surrounding dashboard: project selection, filters submitted to the server, source/paper panels, route refreshes, and loading graph snapshots. The force simulation itself can be plain inline JavaScript in the graph artifact. That keeps the user flow one command:

```bash
rforge --project . graph papers
rforge --project . ui
```

or even just open `data/knowledge-graph.html`.

## Recommended implementation ladder

1. **Now: inline SVG force-directed graph**
   - Source: `data/knowledge-graph.json`.
   - Output: `data/knowledge-graph.html`.
   - Features: drag nodes, search/filter, click node details, neighbor list, static fallback list.
   - No dependencies.

2. **Next: dashboard route integration**
   - Make `/map` or a new `/graph` route load the same graph JSON.
   - Keep no-JS tables available.
   - Add HTMX filters for node kind, concept, paper, and year.

3. **Later: Canvas renderer threshold**
   - If nodes > ~500 or edges > ~1500, switch edges or all marks to Canvas.
   - Keep DOM/SVG overlay for selected node labels and details.
   - Keep list/table fallback for accessibility.

4. **Only if needed: Three.js/WebGL tier**
   - Use for 3D cluster exploration or very large graphs.
   - Feature-detect WebGL.
   - Keep server-generated JSON and accessible fallback.
   - Avoid making Three.js mandatory for local review workflows.

## Accessibility and interaction rules

- Preserve a semantic node list below or beside the graph.
- Search/filter must work through an `<input type="search">` and not only pointer gestures.
- Clicked node details should be plain HTML text with snippets, not canvas-only labels.
- Respect `prefers-reduced-motion`: allow stopping or damping force animation in a later polish slice.
- Do not rely on color alone: labels and details must expose node kind.
- Use the graph as an exploration aid; source snippets remain the evidence.

## Implementation decision for current code

The current ResearchForge code path should stay dependency-free and generate a single HTML artifact. Inline SVG plus a small force simulation is enough to prove value. It also matches the product direction: automatic knowledge graphs from papers, visualized with as few user steps as possible.

## Evidence gaps

- I did not benchmark ResearchForge graph sizes yet.
- Search results were broad and often domain-specific; they guide implementation tradeoffs rather than proving a single best library.
- Three.js-specific scholarly evidence is thin for paper knowledge graphs; most relevant guidance is practical web-platform and visualization engineering.
