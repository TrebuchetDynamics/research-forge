package webui

import (
	"encoding/json"
	"encoding/xml"
	"fmt"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/TrebuchetDynamics/research-forge/internal/ui"
)

func citationBoundsModel(count int) ui.CitationGraphViewModel {
	graph := ui.CitationGraphViewModel{}
	for i := 0; i < count; i++ {
		graph.Nodes = append(graph.Nodes, ui.GraphNode{ID: fmt.Sprintf("paper-%02d", i)})
	}
	if count > 0 {
		graph.Nodes[0].ID = `10.1000/<A&B>"'`
		first, last := graph.Nodes[0].ID, graph.Nodes[count-1].ID
		graph.Edges = []ui.GraphEdge{
			{Source: first, Target: last},
			{Source: last, Target: first},
			{Source: "absent", Target: first},
			{Source: last, Target: "absent"},
		}
	}
	return graph
}

func assertCitationBounds(t *testing.T, svg string, graph ui.CitationGraphViewModel) {
	t.Helper()
	var parsed struct {
		XMLName xml.Name `xml:"svg"`
		ViewBox string   `xml:"viewBox,attr"`
		Role    string   `xml:"role,attr"`
		Label   string   `xml:"aria-label,attr"`
		Title   string   `xml:"title"`
		Nodes   []struct {
			Href   string `xml:"href,attr"`
			Circle struct {
				X int `xml:"cx,attr"`
				Y int `xml:"cy,attr"`
				R int `xml:"r,attr"`
			} `xml:"circle"`
			Text struct {
				X     int    `xml:"x,attr"`
				Y     int    `xml:"y,attr"`
				Label string `xml:",chardata"`
			} `xml:"text"`
		} `xml:"a"`
		Edges []struct {
			X1 int `xml:"x1,attr"`
			Y1 int `xml:"y1,attr"`
			X2 int `xml:"x2,attr"`
			Y2 int `xml:"y2,attr"`
		} `xml:"line"`
	}
	if err := xml.Unmarshal([]byte(svg), &parsed); err != nil {
		t.Fatalf("parse SVG: %v", err)
	}
	var left, top, width, height int
	if n, err := fmt.Sscanf(parsed.ViewBox, "%d %d %d %d", &left, &top, &width, &height); err != nil || n != 4 {
		t.Fatalf("invalid viewBox %q: %v", parsed.ViewBox, err)
	}
	if left != 0 || top != 0 || width != 520 || height < 160 || (len(graph.Nodes) <= 10 && height != 160) {
		t.Errorf("unexpected viewBox %q", parsed.ViewBox)
	}
	if parsed.Role != "img" || parsed.Label != "Citation graph visualization" || parsed.Title != parsed.Label {
		t.Errorf("accessible name/title changed: role=%q label=%q title=%q", parsed.Role, parsed.Label, parsed.Title)
	}
	if len(parsed.Nodes) != len(graph.Nodes) || len(parsed.Edges) != 2 {
		t.Fatalf("rendered %d nodes/%d edges, want %d/2 (absent endpoints skipped)", len(parsed.Nodes), len(parsed.Edges), len(graph.Nodes))
	}
	for i, node := range parsed.Nodes {
		x, y := 80+(i%5)*100, 50+(i/5)*70
		if node.Circle.X != x || node.Circle.Y != y || node.Circle.R != 16 || node.Text.X != x || node.Text.Y != y+34 {
			t.Errorf("node %d layout changed: %+v", i, node)
		}
		if node.Circle.X-node.Circle.R < left || node.Circle.X+node.Circle.R > left+width || node.Circle.Y-node.Circle.R < top || node.Circle.Y+node.Circle.R > top+height {
			t.Errorf("node %d circle outside viewBox %q: %+v", i, parsed.ViewBox, node.Circle)
		}
		if node.Text.Y+6 > top+height {
			t.Errorf("node %d label baseline %d lacks 6 units bottom space in viewBox %q", i, node.Text.Y, parsed.ViewBox)
		}
		wantHref := "/papers/" + graph.Nodes[i].ID
		if i == 0 {
			wantHref = "/papers/10-1000-a-b"
		}
		if node.Text.Label != graph.Nodes[i].ID || node.Href != wantHref {
			t.Errorf("node %d label/link = %q/%q, want %q/%q", i, node.Text.Label, node.Href, graph.Nodes[i].ID, wantHref)
		}
	}
	first, last := parsed.Nodes[0].Circle, parsed.Nodes[len(parsed.Nodes)-1].Circle
	for i, edge := range parsed.Edges {
		source, target := first, last
		if i == 1 {
			source, target = last, first
		}
		if edge.X1 != source.X || edge.Y1 != source.Y || edge.X2 != target.X || edge.Y2 != target.Y {
			t.Errorf("edge %d endpoints changed: %+v", i, edge)
		}
	}
}

func TestCitationGraphSVGViewportBounds(t *testing.T) {
	for _, count := range []int{0, 1, 5, 10, 11, 16, 26} {
		t.Run(fmt.Sprint(count), func(t *testing.T) {
			graph := citationBoundsModel(count)
			state := ArtifactDashboardState{CitationGraph: graph}
			svg := string(state.CitationGraphSVG())
			if svg != string(state.CitationGraphSVG()) {
				t.Fatal("repeated rendering differs")
			}
			if count == 0 {
				if svg != "" {
					t.Fatalf("empty graph SVG = %q", svg)
				}
				return
			}
			assertCitationBounds(t, svg, graph)
		})
	}
}

func TestCitationGraphSVGViewportBoundsRouter(t *testing.T) {
	for _, count := range []int{0, 26} {
		t.Run(fmt.Sprint(count), func(t *testing.T) {
			graph := citationBoundsModel(count)
			data, err := json.Marshal(graph)
			if err != nil {
				t.Fatal(err)
			}
			dir := t.TempDir()
			writeCitationGraph(t, dir, string(data))
			recorder := httptest.NewRecorder()
			NewRouter(Config{ProjectPath: dir}).ServeHTTP(recorder, httptest.NewRequest(http.MethodGet, "/artifacts", nil))
			if recorder.Code != http.StatusOK {
				t.Fatalf("GET /artifacts: %d: %s", recorder.Code, recorder.Body.String())
			}
			body := recorder.Body.String()
			start := strings.Index(body, "<svg")
			if count == 0 {
				if start != -1 || !strings.Contains(body, "No citation graph exported yet") {
					t.Fatal("empty graph fallback changed")
				}
				return
			}
			end := strings.Index(body, "</svg>")
			if start == -1 || end < start {
				t.Fatal("server-rendered SVG missing")
			}
			assertCitationBounds(t, body[start:end+len("</svg>")], graph)
			for _, mount := range []string{`data-citation-graph data-src="/artifacts/graph.json"`, `<script src="/assets/citation-graph.js" defer></script>`, `<p>paper-25</p>`, ` → paper-25</p>`} {
				if !strings.Contains(body, mount) {
					t.Errorf("missing enhancement mount/text alternative %q", mount)
				}
			}
		})
	}
}
