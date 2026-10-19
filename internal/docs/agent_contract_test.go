package docs

import (
	"bytes"
	"os"
	"path/filepath"
	"regexp"
	"strings"
	"testing"

	"github.com/TrebuchetDynamics/research-forge/internal/cli"
)

// Catch install guidance that drifts from the module's minimum, not the
// separately pinned toolchain version.
func TestDocumentedGoMinimumMatchesModule(t *testing.T) {
	root := filepath.Join("..", "..")
	mod := readContractFile(t, root, "go.mod")
	directive := regexp.MustCompile(`(?m)^go[ \t]+([0-9]+\.[0-9]+(?:\.[0-9]+)?)[ \t]*$`).FindStringSubmatch(mod)
	if directive == nil {
		t.Fatal("go.mod has no go directive")
	}
	version := regexp.MustCompile("(?i)\\bgo[ \\t`]*v?([0-9]+\\.[0-9]+(?:\\.[0-9]+)?)\\b")
	for _, path := range []string{
		"skills/research-forge/SKILL.md", "README.md", "install.sh",
		"docs/install.md", "docs/developer-setup.md",
	} {
		t.Run(path, func(t *testing.T) {
			for i, line := range strings.Split(readContractFile(t, root, path), "\n") {
				for _, match := range version.FindAllStringSubmatch(line, -1) {
					if match[1] != directive[1] {
						t.Errorf("%s:%d states Go %s; go.mod requires Go %s", path, i+1, match[1], directive[1])
					}
				}
			}
		})
	}
}

func TestSkillCommandsExistInTopLevelCatalog(t *testing.T) {
	var stdout, stderr bytes.Buffer
	// Bash completion exposes topLevelCommands without running any research
	// commands (which could use the network or change a project).
	if code := cli.Execute([]string{"completion", "bash"}, &stdout, &stderr); code != 0 {
		t.Fatalf("completion exit %d: %s", code, stderr.String())
	}
	catalog := regexp.MustCompile(`local commands="([^"]+)"`).FindStringSubmatch(stdout.String())
	if catalog == nil {
		t.Fatal("completion output has no command catalog")
	}
	known := make(map[string]bool)
	for _, name := range strings.Fields(catalog[1]) {
		known[name] = true
	}

	// Inspect code examples, not prose such as "rforge is a standalone tool".
	// Global options can precede the top-level command.
	command := regexp.MustCompile(`\brforge[ \t]+(?:(?:--json[ \t]+)|(?:--(?:project|config|log-level)[ \t]+(?:<[^>\n]+>|"[^"\n]*"|'[^'\n]*'|[^ \t\n]+)[ \t]+))*([a-z][a-z0-9-]*)`)
	inline := regexp.MustCompile("`([^`]+)`")
	inFence, checked := false, 0
	for i, line := range strings.Split(readContractFile(t, filepath.Join("..", ".."), "skills/research-forge/SKILL.md"), "\n") {
		if strings.HasPrefix(strings.TrimSpace(line), "```") {
			inFence = !inFence
			continue
		}
		var examples []string
		if inFence {
			examples = []string{line}
		} else {
			for _, span := range inline.FindAllStringSubmatch(line, -1) {
				examples = append(examples, span[1])
			}
		}
		for _, example := range examples {
			for _, match := range command.FindAllStringSubmatch(example, -1) {
				checked++
				if !known[match[1]] {
					t.Errorf("skills/research-forge/SKILL.md:%d: rforge %s is absent from topLevelCommands", i+1, match[1])
				}
			}
		}
	}
	if checked == 0 {
		t.Fatal("skill contains no rforge command examples")
	}
}

func readContractFile(t *testing.T, root, path string) string {
	t.Helper()
	data, err := os.ReadFile(filepath.Join(root, path))
	if err != nil {
		t.Fatalf("read %s: %v", path, err)
	}
	return string(data)
}
