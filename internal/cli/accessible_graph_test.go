package cli

import (
	"bytes"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/TrebuchetDynamics/research-forge/internal/citations"
)

func TestExecuteCitationsAccessibleViewExportsMarkdown(t *testing.T) {
	project := t.TempDir()
	graph := citations.NewGraph()
	graph.AddCitation("paper-1", "paper-2")
	graphData, _ := graph.ExportJSON()
	graphPath := filepath.Join(project, "graph.json")
	if err := os.WriteFile(graphPath, graphData, 0o644); err != nil {
		t.Fatalf("write graph: %v", err)
	}
	domain := citations.DomainMapArtifact{Topics: []citations.DomainTopic{{TopicID: "solar", Label: "Solar fuels", RepresentativePapers: []citations.RepresentativePaper{{PaperID: "paper-1", Title: "Solar catalyst"}}}}}
	domainPath := filepath.Join(project, "domain.json")
	if err := writeJSONFile(domainPath, domain); err != nil {
		t.Fatalf("write domain: %v", err)
	}
	out := filepath.Join(project, "accessible.md")
	var stdout, stderr bytes.Buffer
	code := Execute([]string{"--json", "--project", project, "citations", "accessible-view", "--graph", graphPath, "--domain-map", domainPath, "--out", out, "--filter", "paper-1", "--format", "markdown"}, &stdout, &stderr)
	if code != 0 {
		t.Fatalf("code=%d stderr=%s stdout=%s", code, stderr.String(), stdout.String())
	}
	data, err := os.ReadFile(out)
	if err != nil {
		t.Fatalf("read out: %v", err)
	}
	for _, want := range []string{"# Accessible graph view", "Filtered node table", "Edge list", "Keyboard navigation", "Solar fuels"} {
		if !strings.Contains(string(data), want) {
			t.Fatalf("output missing %q:\n%s", want, string(data))
		}
	}
}

func TestExecuteCitationsAccessibleViewDoesNotWriteMarkdownThroughSymlink(t *testing.T) {
	project := t.TempDir()
	graph := citations.NewGraph()
	graph.AddCitation("paper-1", "paper-2")
	graphData, _ := graph.ExportJSON()
	graphPath := filepath.Join(project, "graph.json")
	if err := os.WriteFile(graphPath, graphData, 0o644); err != nil {
		t.Fatalf("write graph: %v", err)
	}
	out := filepath.Join(project, "accessible.md")
	outsidePath := filepath.Join(t.TempDir(), "outside.md")
	outsideBefore := []byte("outside accessible view\n")
	if err := os.WriteFile(outsidePath, outsideBefore, 0o600); err != nil {
		t.Fatalf("write outside view: %v", err)
	}
	if err := os.Symlink(outsidePath, out); err != nil {
		t.Skipf("symlink unsupported: %v", err)
	}
	var stdout, stderr bytes.Buffer

	code := Execute([]string{
		"--json", "--project", project, "citations", "accessible-view",
		"--graph", graphPath, "--out", out, "--format", "markdown",
	}, &stdout, &stderr)
	if code != 1 || !strings.Contains(stdout.String(), "accessible_graph_write_failed") {
		t.Fatalf("code=%d stderr=%s stdout=%s, want write failure", code, stderr.String(), stdout.String())
	}
	outsideAfter, err := os.ReadFile(outsidePath)
	if err != nil {
		t.Fatalf("read outside view: %v", err)
	}
	if !bytes.Equal(outsideAfter, outsideBefore) {
		t.Fatalf("outside view changed through symlink:\n got: %s\nwant: %s", outsideAfter, outsideBefore)
	}
	info, err := os.Lstat(out)
	if err != nil {
		t.Fatalf("lstat output symlink: %v", err)
	}
	if info.Mode()&os.ModeSymlink == 0 {
		t.Fatalf("output target is no longer a symlink: %v", info.Mode())
	}
}
