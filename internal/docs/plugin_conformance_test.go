package docs

import (
	"bytes"
	"encoding/json"
	"maps"
	"os"
	"os/exec"
	"path/filepath"
	"reflect"
	"regexp"
	"strings"
	"testing"

	"github.com/TrebuchetDynamics/research-forge/internal/cli"
)

// The packaging boundary is passive: public skills, explicit workflows and one
// compatibility-checked engine MCP server. No developer skills or runtime policy.
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
		scan := maps.Clone(value)
		// Exempt only the exact host-owned registration, never arbitrary MCP,
		// command or tool keys elsewhere in the manifests.
		switch path {
		case "adapters/.claude-plugin/plugin.json", "adapters/.codex-plugin/plugin.json":
			server := map[string]any{"command": "node", "args": []any{"${CLAUDE_PLUGIN_ROOT}/mcp.mjs"}}
			if strings.Contains(path, ".codex-plugin") {
				server = map[string]any{"command": "node", "args": []any{"mcp.mjs"}, "cwd": "."}
			}
			if !reflect.DeepEqual(value["mcpServers"], map[string]any{"rforge": server}) {
				t.Fatalf("%s must register only the checked rforge MCP launcher", path)
			}
			delete(scan, "mcpServers")
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
		check(scan)
		if value["name"] != "research-forge" {
			t.Errorf("%s: unexpected name %v", path, value["name"])
		}
		return value
	}
	claude := parse("adapters/.claude-plugin/plugin.json")
	codex := parse("adapters/.codex-plugin/plugin.json")
	pi := parse("package.json")
	hermes := parse("plugin.yaml")
	hermesPackage := parse("hermes-plugin/plugin.yaml")
	version, _ := pi["version"].(string)
	if !regexp.MustCompile(`^[0-9]+\.[0-9]+\.[0-9]+(?:-[0-9A-Za-z.-]+)?$`).MatchString(version) {
		t.Fatalf("invalid package version %q", version)
	}
	for host, manifest := range map[string]map[string]any{"Claude": claude, "Codex": codex, "Hermes root": hermes, "Hermes package": hermesPackage} {
		if manifest["version"] != version {
			t.Errorf("%s version %v differs from Pi %s", host, manifest["version"], version)
		}
	}
	if tag := os.Getenv("RFORGE_RELEASE_VERSION"); tag != "" && strings.TrimPrefix(tag, "v") != version {
		t.Errorf("manifest version %s differs from release %s", version, tag)
	}
	workflows := []string{"brief", "compare", "lit", "paper", "replicate"}
	wantSkills := []any{"./skills/research-forge"}
	piSkills := []any{"./skills/research-forge"}
	packageFiles := []any{"adapters/pi.mjs", "adapters/mcp.mjs", "skills/research-forge/", "LICENSE", "docs/agent-install.md"}
	for _, name := range workflows {
		wantSkills = append(wantSkills, "./skills/"+name)
		piSkills = append(piSkills, "./adapters/skills/"+name)
		packageFiles = append(packageFiles, "adapters/skills/"+name+"/")
	}
	if !reflect.DeepEqual(claude["skills"], wantSkills) || codex["skills"] != "./skills/" {
		t.Fatal("Claude and Codex must expose only methodology and explicit workflows")
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
		"skills": piSkills, "extensions": []any{"./adapters/pi.mjs"},
	}) {
		t.Fatal("Pi resources must expose only public skills and the passive command extension")
	}
	if !reflect.DeepEqual(pi["files"], packageFiles) {
		t.Fatal("npm package must not ship developer skills or unrelated adapters")
	}
	if hermes["kind"] != "standalone" || hermesPackage["kind"] != "standalone" {
		t.Fatal("Hermes plugin must be standalone")
	}
	// Check the entire isolated skill root, including implicitly discovered
	// folders, not merely the declared manifest paths.
	entries, err := os.ReadDir(filepath.Join(root, "adapters", "skills"))
	if err != nil {
		t.Fatal(err)
	}
	allowed := map[string]bool{"research-forge": true}
	for _, name := range workflows {
		allowed[name] = true
	}
	if len(entries) != len(allowed) {
		t.Fatal("isolated plugin root must contain exactly methodology and five explicit workflows")
	}
	for _, entry := range entries {
		if !entry.IsDir() || !allowed[entry.Name()] {
			t.Errorf("unexpected skill surface %s", entry.Name())
		}
	}
	for _, name := range workflows {
		path := "adapters/skills/" + name + "/"
		header := strings.SplitN(read(path+"SKILL.md"), "---", 3)
		if len(header) != 3 || header[0] != "" ||
			!regexp.MustCompile(`(?m)^disable-model-invocation: true$`).MatchString(header[1]) {
			t.Fatalf("%s must disable model invocation in frontmatter", name)
		}
		var policy map[string]any
		if err := json.Unmarshal([]byte(read(path+"agents/openai.yaml")), &policy); err != nil {
			t.Fatalf("%s: parse Codex policy (JSON subset of YAML): %v", name, err)
		}
		if !reflect.DeepEqual(policy, map[string]any{"policy": map[string]any{"allow_implicit_invocation": false}}) {
			t.Fatalf("%s must disallow Codex implicit invocation", name)
		}
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
		for _, base := range []string{"", "adapters", "hermes-plugin"} {
			if _, err := os.Stat(filepath.Join(root, base, path)); !os.IsNotExist(err) {
				t.Errorf("unexpected runtime surface %s", filepath.Join(base, path))
			}
		}
	}
	hermesCmd := exec.Command("python3", "-B", "-m", "unittest", "adapters/test_hermes.py")
	hermesCmd.Dir = root
	if out, err := hermesCmd.CombinedOutput(); err != nil {
		t.Fatalf("Hermes registration and isolated package contract: %v\n%s", err, out)
	}
	cmd := exec.Command("node", "--test", "adapters/test")
	cmd.Dir = root
	if out, err := cmd.CombinedOutput(); err != nil {
		t.Fatalf("Pi workflow and engine MCP contract: %v\n%s", err, out)
	}
}
