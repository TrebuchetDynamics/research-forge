package docs

import (
	"bytes"
	"encoding/json"
	"os"
	"path/filepath"
	"reflect"
	"regexp"
	"strings"
	"testing"

	"github.com/TrebuchetDynamics/research-forge/internal/cli"
)

// The packaging boundary is passive and exposes the methodology skill, never
// the repository's developer skills. Keep this offline and dependency-free.
func TestPassivePluginConformance(t *testing.T) {
	root := filepath.Join("..", "..")
	read := func(path string) string { return readContractFile(t, root, path) }
	parse := func(path string) map[string]any {
		t.Helper()
		var value map[string]any
		// plugin.yaml deliberately uses JSON, a YAML subset, so the same
		// manifest is readable by Hermes and Go's standard library.
		if err := json.Unmarshal([]byte(read(path)), &value); err != nil {
			t.Fatalf("parse %s: %v", path, err)
		}
		forbidden := regexp.MustCompile(`(?i)hook|agent|setting|permission|approval|schedul|autostart|background|scripts?$|^install$|^preinstall$|^postinstall$|requires_plugins|command|tool|mcp|cron|default`)
		var check func(any)
		check = func(v any) {
			switch v := v.(type) {
			case map[string]any:
				for key, child := range v {
					if forbidden.MatchString(key) {
						t.Errorf("%s contains forbidden key %q", path, key)
					}
					check(child)
				}
			case []any:
				for _, child := range v {
					check(child)
				}
			}
		}
		check(value)
		if value["name"] != "research-forge" {
			t.Errorf("%s: unexpected name %v", path, value["name"])
		}
		return value
	}
	claude := parse("adapters/.claude-plugin/plugin.json")
	codex := parse("adapters/.codex-plugin/plugin.json")
	pi := parse("package.json")
	hermes := parse("plugin.yaml")
	version, _ := pi["version"].(string)
	if !regexp.MustCompile(`^[0-9]+\.[0-9]+\.[0-9]+(?:-[0-9A-Za-z.-]+)?$`).MatchString(version) {
		t.Fatalf("invalid package version %q", version)
	}
	for host, manifest := range map[string]map[string]any{"Claude": claude, "Codex": codex, "Hermes": hermes} {
		if manifest["version"] != version {
			t.Errorf("%s version %v differs from Pi %s", host, manifest["version"], version)
		}
	}
	if tag := os.Getenv("RFORGE_RELEASE_VERSION"); tag != "" && strings.TrimPrefix(tag, "v") != version {
		t.Errorf("manifest version %s differs from release %s", version, tag)
	}
	wantSkills := []any{"./skills/research-forge"}
	if !reflect.DeepEqual(claude["skills"], wantSkills) || codex["skills"] != "./skills/research-forge" {
		t.Fatal("Claude and Codex must expose only the public skill")
	}
	for _, host := range []string{"claude", "codex"} {
		path := ".claude-plugin/marketplace.json"
		if host == "codex" {
			path = ".agents/plugins/marketplace.json"
		}
		market := parse(path)
		plugins, _ := market["plugins"].([]any)
		if len(plugins) != 1 {
			t.Fatalf("%s: want exactly one plugin", path)
		}
		entry, _ := plugins[0].(map[string]any)
		if entry["name"] != "research-forge" {
			t.Fatalf("%s: wrong plugin name", path)
		}
		if host == "claude" {
			if entry["source"] != "./adapters" || !reflect.DeepEqual(entry["skills"], wantSkills) {
				t.Fatal("Claude marketplace must use isolated adapters root and explicit skill path")
			}
		} else if !reflect.DeepEqual(entry["source"], map[string]any{"source": "local", "path": "./adapters"}) {
			t.Fatal("Codex marketplace must use isolated adapters root")
		}
	}
	resources, _ := pi["pi"].(map[string]any)
	if !reflect.DeepEqual(resources, map[string]any{
		"skills": []any{"./skills/research-forge"}, "extensions": []any{"./adapters/pi.mjs"},
	}) {
		t.Fatal("Pi resources must expose only the public skill and passive extension")
	}
	if !reflect.DeepEqual(pi["files"], []any{"adapters/pi.mjs", "skills/research-forge/", "LICENSE", "docs/agent-install.md"}) {
		t.Fatal("npm package must not ship developer skills or unrelated adapters")
	}
	if hermes["kind"] != "standalone" {
		t.Fatal("Hermes plugin must be standalone")
	}
	// Check the entire isolated skill root, including implicitly discovered
	// folders, not merely the declared manifest paths.
	entries, err := os.ReadDir(filepath.Join(root, "adapters", "skills"))
	if err != nil {
		t.Fatal(err)
	}
	if len(entries) != 1 || entries[0].Name() != "research-forge" || !entries[0].IsDir() {
		t.Fatal("isolated plugin root must contain exactly the public skill")
	}
	skill := read("skills/research-forge/SKILL.md")
	if read("adapters/skills/research-forge/SKILL.md") != skill {
		t.Fatal("generated adapter skill differs; refresh it from skills/research-forge/SKILL.md")
	}
	frontmatter := strings.SplitN(skill, "---", 3)
	if len(frontmatter) != 3 || frontmatter[0] != "" {
		t.Fatal("skill has no frontmatter")
	}
	metadata := func(key string) string {
		t.Helper()
		match := regexp.MustCompile(`(?m)^  ` + key + `: "([^"]+)"$`).FindStringSubmatch(frontmatter[1])
		if match == nil {
			t.Fatalf("skill metadata missing %s", key)
		}
		return match[1]
	}
	args := []string{"capabilities", "--json", "--min-rforge-version", metadata("min_rforge_version")}
	for _, requirement := range strings.Split(metadata("required_capabilities"), ",") {
		args = append(args, "--require", requirement)
	}
	var out, errOut bytes.Buffer
	if code := cli.Execute(args, &out, &errOut); code != 0 {
		t.Fatalf("skill requires unavailable engine contracts: %s%s", out.String(), errOut.String())
	}
	for _, path := range []string{"hooks", "agents", "settings.json", "defaults.json", "plugins/defaults.json", ".mcp.json", ".claude-plugin/hooks.json"} {
		for _, base := range []string{"", "adapters"} {
			if _, err := os.Stat(filepath.Join(root, base, path)); !os.IsNotExist(err) {
				t.Errorf("unexpected runtime surface %s", filepath.Join(base, path))
			}
		}
	}
	if strings.Contains(read("adapters/pi.mjs"), "child_process") {
		t.Fatal("Pi must not spawn subprocesses before typed-tool integration")
	}
}
