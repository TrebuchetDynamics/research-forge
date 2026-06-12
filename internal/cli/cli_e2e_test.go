package cli

import (
	"bytes"
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/TrebuchetDynamics/research-forge/internal/provenance"
)

func TestExternalE2EArtificialPhotosynthesisWorkspace(t *testing.T) {
	target := os.Getenv("RFORGE_EXTERNAL_E2E_DIR")
	if target == "" {
		t.Skip("set RFORGE_EXTERNAL_E2E_DIR to run external artificial photosynthesis e2e")
	}
	oldwd, err := os.Getwd()
	if err != nil {
		t.Fatalf("get cwd: %v", err)
	}
	if err := os.Chdir(target); err != nil {
		t.Fatalf("chdir external e2e target: %v", err)
	}
	t.Cleanup(func() { _ = os.Chdir(oldwd) })

	configBytes, err := os.ReadFile(filepath.Join(target, ".researchforge"))
	if err != nil {
		t.Fatalf("external target missing .researchforge: %v", err)
	}
	if !strings.Contains(string(configBytes), `e2e_topic = "artificial photosynthesis"`) {
		t.Fatalf("external target .researchforge missing artificial photosynthesis topic:\n%s", string(configBytes))
	}

	stdout := new(bytes.Buffer)
	stderr := new(bytes.Buffer)
	code := Execute([]string{"--json", "project", "discover-assets"}, stdout, stderr)
	if code != 0 {
		t.Fatalf("exit code = %d, stderr = %s", code, stderr.String())
	}
	var envelope map[string]any
	if err := json.Unmarshal(stdout.Bytes(), &envelope); err != nil {
		t.Fatalf("stdout is not JSON: %v\n%s", err, stdout.String())
	}
	data := envelope["data"].(map[string]any)
	assets := data["assets"].([]any)
	var sawReadme, sawReferences bool
	for _, raw := range assets {
		asset := raw.(map[string]any)
		switch asset["path"] {
		case "README.md":
			sawReadme = asset["kind"] == "note" && asset["imported"] == false
		case "references.bib":
			sawReferences = asset["kind"] == "bibliography" && asset["imported"] == false
		}
	}
	if !sawReadme || !sawReferences {
		t.Fatalf("external e2e assets missing README/references: %#v", assets)
	}
}

func TestE2EDiscoverAssetsDoesNotAppendDuplicateProvenanceWhenAssetsUnchanged(t *testing.T) {
	repo := t.TempDir()
	if err := os.Mkdir(filepath.Join(repo, ".git"), 0o755); err != nil {
		t.Fatalf("create fake git repo: %v", err)
	}
	if err := os.WriteFile(filepath.Join(repo, "references.bib"), []byte("@article{fixture,title={Artificial photosynthesis}}\n"), 0o644); err != nil {
		t.Fatalf("write bibliography: %v", err)
	}
	oldwd, err := os.Getwd()
	if err != nil {
		t.Fatalf("get cwd: %v", err)
	}
	if err := os.Chdir(repo); err != nil {
		t.Fatalf("chdir repo: %v", err)
	}
	t.Cleanup(func() { _ = os.Chdir(oldwd) })
	if code := Execute([]string{"project", "create", "--title", "Artificial Photosynthesis Review"}, new(bytes.Buffer), new(bytes.Buffer)); code != 0 {
		t.Fatalf("project create exit code = %d", code)
	}
	if code := Execute([]string{"project", "discover-assets"}, new(bytes.Buffer), new(bytes.Buffer)); code != 0 {
		t.Fatalf("first discover-assets exit code = %d", code)
	}
	firstEvents, err := provenance.Read(filepath.Join(repo, "research-forge"))
	if err != nil {
		t.Fatalf("read first events: %v", err)
	}
	if code := Execute([]string{"project", "discover-assets"}, new(bytes.Buffer), new(bytes.Buffer)); code != 0 {
		t.Fatalf("second discover-assets exit code = %d", code)
	}
	secondEvents, err := provenance.Read(filepath.Join(repo, "research-forge"))
	if err != nil {
		t.Fatalf("read second events: %v", err)
	}
	if countEvents(firstEvents, "project.assets.discover") != countEvents(secondEvents, "project.assets.discover") {
		t.Fatalf("duplicate unchanged discovery appended domain provenance:\nfirst=%#v\nsecond=%#v", firstEvents, secondEvents)
	}
}

func countEvents(events []provenance.Event, action string) int {
	count := 0
	for _, event := range events {
		if event.Action == action {
			count++
		}
	}
	return count
}

func TestE2EDiscoverAssetsRejectsUnsafeResearchForgeConfigPath(t *testing.T) {
	repo := t.TempDir()
	if err := os.Mkdir(filepath.Join(repo, ".git"), 0o755); err != nil {
		t.Fatalf("create fake git repo: %v", err)
	}
	if err := os.WriteFile(filepath.Join(repo, ".researchforge"), []byte("default_project_path = \"../escape\"\n"), 0o644); err != nil {
		t.Fatalf("write unsafe config: %v", err)
	}
	oldwd, err := os.Getwd()
	if err != nil {
		t.Fatalf("get cwd: %v", err)
	}
	if err := os.Chdir(repo); err != nil {
		t.Fatalf("chdir repo: %v", err)
	}
	t.Cleanup(func() { _ = os.Chdir(oldwd) })

	stdout := new(bytes.Buffer)
	stderr := new(bytes.Buffer)
	code := Execute([]string{"--json", "project", "discover-assets"}, stdout, stderr)
	if code == 0 {
		t.Fatalf("exit code = 0, want failure for unsafe .researchforge path")
	}
	if stderr.Len() != 0 {
		t.Fatalf("stderr = %q, want empty for JSON error", stderr.String())
	}
	var envelope map[string]any
	if err := json.Unmarshal(stdout.Bytes(), &envelope); err != nil {
		t.Fatalf("stdout is not JSON: %v\n%s", err, stdout.String())
	}
	if envelope["ok"] != false {
		t.Fatalf("ok = %#v, want false", envelope["ok"])
	}
	errorBody := envelope["error"].(map[string]any)
	if errorBody["code"] != "repo_project_defaults_failed" {
		t.Fatalf("error.code = %#v, want repo_project_defaults_failed", errorBody["code"])
	}
	if _, err := os.Stat(filepath.Join(repo, "escape")); !os.IsNotExist(err) {
		t.Fatalf("unsafe configured path was created, err=%v", err)
	}
}

func TestE2EDiscoverAssetsFromRepoSubdirectoryUsesResearchForgeConfig(t *testing.T) {
	repo := t.TempDir()
	if err := os.Mkdir(filepath.Join(repo, ".git"), 0o755); err != nil {
		t.Fatalf("create fake git repo: %v", err)
	}
	if err := os.WriteFile(filepath.Join(repo, "paper.pdf"), []byte("%PDF-1.4 artificial photosynthesis fixture"), 0o644); err != nil {
		t.Fatalf("write existing PDF: %v", err)
	}
	oldwd, err := os.Getwd()
	if err != nil {
		t.Fatalf("get cwd: %v", err)
	}
	if err := os.Chdir(repo); err != nil {
		t.Fatalf("chdir repo: %v", err)
	}
	t.Cleanup(func() { _ = os.Chdir(oldwd) })
	configuredProject := filepath.Join(repo, "custom-research")
	if code := Execute([]string{"project", "create", configuredProject, "--title", "Artificial Photosynthesis Review"}, new(bytes.Buffer), new(bytes.Buffer)); code != 0 {
		t.Fatalf("project create exit code = %d", code)
	}
	subdir := filepath.Join(repo, "src", "nested")
	if err := os.MkdirAll(subdir, 0o755); err != nil {
		t.Fatalf("create subdir: %v", err)
	}
	if err := os.Chdir(subdir); err != nil {
		t.Fatalf("chdir subdir: %v", err)
	}

	stdout := new(bytes.Buffer)
	stderr := new(bytes.Buffer)
	code := Execute([]string{"--json", "project", "discover-assets"}, stdout, stderr)
	if code != 0 {
		t.Fatalf("exit code = %d, stderr = %s", code, stderr.String())
	}
	eventsBytes, err := os.ReadFile(filepath.Join(configuredProject, "provenance", "events.jsonl"))
	if err != nil {
		t.Fatalf("read configured project provenance: %v", err)
	}
	if !strings.Contains(string(eventsBytes), `"action":"project.assets.discover"`) {
		t.Fatalf("discovery provenance event missing from configured project:\n%s", string(eventsBytes))
	}
	if _, err := os.Stat(filepath.Join(repo, "research-forge")); !os.IsNotExist(err) {
		t.Fatalf("discover-assets ignored .researchforge and used default workspace, err=%v", err)
	}
}

func TestE2EDiscoverExistingAcademicAssetsRecordsCLICommandProvenance(t *testing.T) {
	repo := t.TempDir()
	if err := os.Mkdir(filepath.Join(repo, ".git"), 0o755); err != nil {
		t.Fatalf("create fake git repo: %v", err)
	}
	if err := os.WriteFile(filepath.Join(repo, "paper.pdf"), []byte("%PDF-1.4 artificial photosynthesis fixture"), 0o644); err != nil {
		t.Fatalf("write PDF: %v", err)
	}
	oldwd, err := os.Getwd()
	if err != nil {
		t.Fatalf("get cwd: %v", err)
	}
	if err := os.Chdir(repo); err != nil {
		t.Fatalf("chdir repo: %v", err)
	}
	t.Cleanup(func() { _ = os.Chdir(oldwd) })
	if code := Execute([]string{"project", "create", "--title", "Artificial Photosynthesis Review"}, new(bytes.Buffer), new(bytes.Buffer)); code != 0 {
		t.Fatalf("project create exit code = %d", code)
	}

	stdout := new(bytes.Buffer)
	stderr := new(bytes.Buffer)
	code := Execute([]string{"--json", "project", "discover-assets"}, stdout, stderr)
	if code != 0 {
		t.Fatalf("exit code = %d, stderr = %s", code, stderr.String())
	}
	events, err := provenance.Read(filepath.Join(repo, "research-forge"))
	if err != nil {
		t.Fatalf("read provenance: %v", err)
	}
	var sawCommand bool
	for _, event := range events {
		if event.Action != "cli.command" || event.Inputs["command"] != "project discover-assets" {
			continue
		}
		if event.Outputs["assetCount"] != float64(1) {
			t.Fatalf("cli.command assetCount = %#v", event.Outputs["assetCount"])
		}
		sawCommand = true
	}
	if !sawCommand {
		t.Fatalf("missing discover-assets cli.command provenance event: %#v", events)
	}
}

func TestE2EDiscoverExistingAcademicAssetsRecordsProvenanceBeforeImport(t *testing.T) {
	repo := t.TempDir()
	if err := os.Mkdir(filepath.Join(repo, ".git"), 0o755); err != nil {
		t.Fatalf("create fake git repo: %v", err)
	}
	for path, content := range map[string]string{
		"paper.pdf":      "%PDF-1.4 artificial photosynthesis fixture",
		"references.bib": "@article{fixture,title={Artificial photosynthesis}}\n",
		"notes.md":       "# artificial photosynthesis notes\n",
	} {
		if err := os.WriteFile(filepath.Join(repo, path), []byte(content), 0o644); err != nil {
			t.Fatalf("write existing asset %s: %v", path, err)
		}
	}
	oldwd, err := os.Getwd()
	if err != nil {
		t.Fatalf("get cwd: %v", err)
	}
	if err := os.Chdir(repo); err != nil {
		t.Fatalf("chdir repo: %v", err)
	}
	t.Cleanup(func() { _ = os.Chdir(oldwd) })
	if code := Execute([]string{"project", "create", "--title", "Artificial Photosynthesis Review"}, new(bytes.Buffer), new(bytes.Buffer)); code != 0 {
		t.Fatalf("project create exit code = %d", code)
	}

	stdout := new(bytes.Buffer)
	stderr := new(bytes.Buffer)
	code := Execute([]string{"--json", "project", "discover-assets"}, stdout, stderr)
	if code != 0 {
		t.Fatalf("exit code = %d, stderr = %s", code, stderr.String())
	}
	var envelope map[string]any
	if err := json.Unmarshal(stdout.Bytes(), &envelope); err != nil {
		t.Fatalf("stdout is not JSON: %v\n%s", err, stdout.String())
	}
	data := envelope["data"].(map[string]any)
	assets := data["assets"].([]any)
	if len(assets) != 3 {
		t.Fatalf("len(assets) = %d, want 3: %#v", len(assets), assets)
	}
	for _, raw := range assets {
		asset := raw.(map[string]any)
		if asset["imported"] != false {
			t.Fatalf("asset was imported during discovery: %#v", asset)
		}
	}
	eventsBytes, err := os.ReadFile(filepath.Join(repo, "research-forge", "provenance", "events.jsonl"))
	if err != nil {
		t.Fatalf("read provenance: %v", err)
	}
	if !strings.Contains(string(eventsBytes), `"action":"project.assets.discover"`) {
		t.Fatalf("discovery provenance event missing:\n%s", string(eventsBytes))
	}
}

func TestE2EProjectCreateInsideRepoLeavesExistingAcademicAssetsInPlace(t *testing.T) {
	repo := t.TempDir()
	if err := os.Mkdir(filepath.Join(repo, ".git"), 0o755); err != nil {
		t.Fatalf("create fake git repo: %v", err)
	}
	pdfPath := filepath.Join(repo, "paper.pdf")
	bibPath := filepath.Join(repo, "references.bib")
	if err := os.WriteFile(pdfPath, []byte("%PDF-1.4 fake fixture"), 0o644); err != nil {
		t.Fatalf("write existing PDF: %v", err)
	}
	if err := os.WriteFile(bibPath, []byte("@article{fixture,title={Artificial photosynthesis}}\n"), 0o644); err != nil {
		t.Fatalf("write existing bibliography: %v", err)
	}
	oldwd, err := os.Getwd()
	if err != nil {
		t.Fatalf("get cwd: %v", err)
	}
	if err := os.Chdir(repo); err != nil {
		t.Fatalf("chdir repo: %v", err)
	}
	t.Cleanup(func() { _ = os.Chdir(oldwd) })

	stdout := new(bytes.Buffer)
	stderr := new(bytes.Buffer)
	code := Execute([]string{"project", "create", "--title", "Artificial Photosynthesis Review"}, stdout, stderr)
	if code != 0 {
		t.Fatalf("exit code = %d, stderr = %s", code, stderr.String())
	}

	for _, path := range []string{pdfPath, bibPath} {
		if _, err := os.Stat(path); err != nil {
			t.Fatalf("existing academic asset was moved or removed: %s: %v", path, err)
		}
	}
	for _, path := range []string{
		filepath.Join(repo, "research-forge", "paper.pdf"),
		filepath.Join(repo, "research-forge", "references.bib"),
	} {
		if _, err := os.Stat(path); !os.IsNotExist(err) {
			t.Fatalf("existing academic asset was silently imported to %s, err=%v", path, err)
		}
	}
}

func TestE2EProjectCreateInsideRepoUsesArtificialPhotosynthesisDefaults(t *testing.T) {
	repo := t.TempDir()
	if err := os.Mkdir(filepath.Join(repo, ".git"), 0o755); err != nil {
		t.Fatalf("create fake git repo: %v", err)
	}
	oldwd, err := os.Getwd()
	if err != nil {
		t.Fatalf("get cwd: %v", err)
	}
	if err := os.Chdir(repo); err != nil {
		t.Fatalf("chdir repo: %v", err)
	}
	t.Cleanup(func() { _ = os.Chdir(oldwd) })

	stdout := new(bytes.Buffer)
	stderr := new(bytes.Buffer)
	code := Execute([]string{"project", "create", "--title", "Artificial Photosynthesis Review"}, stdout, stderr)
	if code != 0 {
		t.Fatalf("exit code = %d, stderr = %s", code, stderr.String())
	}

	configBytes, err := os.ReadFile(filepath.Join(repo, ".researchforge"))
	if err != nil {
		t.Fatalf("repo config not created: %v", err)
	}
	config := string(configBytes)
	for _, want := range []string{
		`default_project_path = "research-forge"`,
		`e2e_topic = "artificial photosynthesis"`,
	} {
		if !strings.Contains(config, want) {
			t.Fatalf(".researchforge missing %q:\n%s", want, config)
		}
	}

	projectDir := filepath.Join(repo, "research-forge")
	manifestBytes, err := os.ReadFile(filepath.Join(projectDir, "rforge.project.toml"))
	if err != nil {
		t.Fatalf("default Research project manifest not created: %v", err)
	}
	if !strings.Contains(string(manifestBytes), `title = "Artificial Photosynthesis Review"`) {
		t.Fatalf("manifest missing artificial photosynthesis title:\n%s", string(manifestBytes))
	}
	if _, err := os.Stat(filepath.Join(projectDir, "data", "rforge.sqlite")); err != nil {
		t.Fatalf("default Research project storage missing: %v", err)
	}
}
