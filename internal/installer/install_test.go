// Package installer_test exercises install.sh end to end in an isolated HOME
// with mocked host CLIs, so the unified installer contract stays deterministic
// and offline. No real host, network, or user configuration is touched.
package installer_test

import (
	"crypto/sha256"
	"encoding/hex"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"sort"
	"strings"
	"testing"
)

const fakeHostScript = `#!/usr/bin/env bash
set -u
host=$(basename "$0")
[ -n "${SANDBOX_LOG:-}" ] && printf '%s %s\n' "$host" "$*" >> "$SANDBOX_LOG"
state="${FAKE_STATE_DIR:-}/$host.state"
fail="${FAKE_STATE_DIR:-}/$host.fail"
if [ -f "$fail" ]; then
  while IFS= read -r pattern; do
    case "$*" in
      *"$pattern"*) echo "$host: simulated failure for: $*" >&2; exit 1 ;;
    esac
  done < "$fail"
fi
has() { grep -Fxq -- "$1" "$state" 2>/dev/null; }
record() { printf '%s\n' "$1" >> "$state"; }
case "$host" in
  claude)
    case "${1:-} ${2:-}" in
      "plugin list") if has research-forge; then printf '[{"id":"research-forge@research-forge","name":"research-forge"}]\n'; else printf '[]\n'; fi ;;
      "plugin install"|"plugin update") record research-forge ;;
      "plugin marketplace") if [ "${3:-}" = list ] && has research-forge; then printf 'research-forge\n'; fi ;;
    esac ;;
  codex)
    case "${1:-} ${2:-}" in
      "plugin list") if has research-forge; then printf '{"plugins":[{"name":"research-forge"}]}\n'; else printf '{"plugins":[]}\n'; fi ;;
      "plugin add") record research-forge ;;
      "plugin marketplace") if [ "${3:-}" = list ] && has research-forge; then printf 'research-forge\n'; fi ;;
    esac ;;
  pi)
    case "${1:-}" in
      list) if has research-forge; then printf 'User packages:\n  git:github.com/TrebuchetDynamics/research-forge\n    /tmp/fake-pi-package\n'; else printf 'User packages:\n'; fi ;;
      install|update) record research-forge ;;
    esac ;;
  hermes)
    profile=default
    if [ "${1:-}" = "--profile" ]; then profile="$2"; shift 2; fi
    case "${1:-} ${2:-}" in
      "profile list") printf ' Profile   Model\n ─────────  ─────\n ◆default   gpt\n  research-forge  gpt\n' ;;
      "plugins list") if has "research-forge@$profile"; then printf 'research-forge  enabled\n'; fi ;;
      "plugins install"|"plugins enable"|"plugins update") record "research-forge@$profile" ;;
    esac ;;
esac
exit 0
`

const fakeCurlScript = `#!/usr/bin/env bash
out=""; url=""
while [ $# -gt 0 ]; do
  case "$1" in
    -o) out="$2"; shift 2 ;;
    -*) shift ;;
    *) url="$1"; shift ;;
  esac
done
[ -n "${SANDBOX_LOG:-}" ] && printf 'curl %s\n' "$url" >> "$SANDBOX_LOG"
if [ "${FAKE_CURL_ECHO:-0}" = 1 ]; then printf '%s\n' "$url" > "$out"; exit 0; fi
path="${url#*://}"; path="${path#*/}"
key=$(printf '%s' "$path" | tr '/' '_')
for candidate in "$FIXTURES/$key" "$FIXTURES/$(basename "$url")"; do
  if [ -f "$candidate" ]; then cp "$candidate" "$out"; exit 0; fi
done
echo "curl: (22) 404 $url" >&2
exit 22
`

const fakeRforgeScript = `#!/usr/bin/env bash
case "${1:-}" in
  version) printf 'rforge 0.1.22 (test, test)\n' ;;
  capabilities) exit 0 ;;
  doctor) printf 'ok\n' ;;
esac
exit 0
`

type sandbox struct {
	t        *testing.T
	root     string
	home     string
	bin      string
	repo     string
	state    string
	log      string
	fixtures string
	env      []string
}

var allHosts = []string{"claude", "codex", "pi", "hermes"}

// newSandbox builds an isolated HOME, a fake host-CLI bin directory and a
// minimal checkout holding install.sh plus the curated public skills.
func newSandbox(t *testing.T, hosts ...string) *sandbox {
	t.Helper()
	root := t.TempDir()
	s := &sandbox{
		t:        t,
		root:     root,
		home:     filepath.Join(root, "home"),
		bin:      filepath.Join(root, "bin"),
		repo:     filepath.Join(root, "repo"),
		state:    filepath.Join(root, "state"),
		log:      filepath.Join(root, "invocations.log"),
		fixtures: filepath.Join(root, "fixtures"),
	}
	for _, dir := range []string{s.home, s.bin, s.repo, s.state, s.fixtures, filepath.Join(s.home, ".local", "bin")} {
		if err := os.MkdirAll(dir, 0o755); err != nil {
			t.Fatalf("mkdir %s: %v", dir, err)
		}
	}
	s.copyFile(filepath.Join(repoRoot(t), "install.sh"), filepath.Join(s.repo, "install.sh"), 0o755)
	copyTree(t, filepath.Join(repoRoot(t), "adapters", "skills"), filepath.Join(s.repo, "adapters", "skills"))
	for _, host := range hosts {
		s.addHost(host)
	}
	s.writeScript(filepath.Join(s.bin, "curl"), fakeCurlScript)
	s.addReleaseFixtures()
	s.env = []string{
		"HOME=" + s.home,
		"PATH=" + s.bin + ":/usr/bin:/bin:/usr/sbin:/sbin",
		"SANDBOX_LOG=" + s.log,
		"FAKE_STATE_DIR=" + s.state,
		"FIXTURES=" + s.fixtures,
		"RFORGE_INSTALL_DIR=" + filepath.Join(s.home, ".local", "bin"),
	}
	return s
}

func repoRoot(t *testing.T) string {
	t.Helper()
	abs, err := filepath.Abs(filepath.Join("..", ".."))
	if err != nil {
		t.Fatalf("resolve repo root: %v", err)
	}
	return abs
}

func (s *sandbox) addHost(host string) {
	s.writeScript(filepath.Join(s.bin, host), fakeHostScript)
}

func (s *sandbox) removeHost(host string) {
	if err := os.Remove(filepath.Join(s.bin, host)); err != nil && !os.IsNotExist(err) {
		s.t.Fatalf("remove fake %s: %v", host, err)
	}
}

// seed marks a host component as already installed for detection.
func (s *sandbox) seed(host, id string) {
	path := filepath.Join(s.state, host+".state")
	f, err := os.OpenFile(path, os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0o644)
	if err != nil {
		s.t.Fatalf("seed %s: %v", host, err)
	}
	defer f.Close()
	if _, err := f.WriteString(id + "\n"); err != nil {
		s.t.Fatalf("seed %s: %v", host, err)
	}
}

// failOn makes a host CLI exit nonzero for commands containing substr.
func (s *sandbox) failOn(host, substr string) {
	path := filepath.Join(s.state, host+".fail")
	f, err := os.OpenFile(path, os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0o644)
	if err != nil {
		s.t.Fatalf("failOn %s: %v", host, err)
	}
	defer f.Close()
	if _, err := f.WriteString(substr + "\n"); err != nil {
		s.t.Fatalf("failOn %s: %v", host, err)
	}
}

func (s *sandbox) setEnv(extra ...string) { s.env = append(s.env, extra...) }

// run executes install.sh and returns combined output, exit code, and the log
// of host/curl invocations.
func (s *sandbox) run(args ...string) (string, int, string) {
	s.t.Helper()
	cmd := exec.Command("bash", append([]string{filepath.Join(s.repo, "install.sh")}, args...)...)
	cmd.Env = append(os.Environ(), s.env...)
	cmd.Dir = s.root
	out, err := cmd.CombinedOutput()
	code := 0
	if err != nil {
		if exitErr, ok := err.(*exec.ExitError); ok {
			code = exitErr.ExitCode()
		} else {
			s.t.Fatalf("run install.sh: %v", err)
		}
	}
	logged, _ := os.ReadFile(s.log)
	return string(out), code, string(logged)
}

func (s *sandbox) path(parts ...string) string {
	return filepath.Join(append([]string{s.home}, parts...)...)
}

// stateOf returns the identifiers a fake host CLI recorded as installed.
func (s *sandbox) stateOf(host string) string {
	s.t.Helper()
	data, err := os.ReadFile(filepath.Join(s.state, host+".state"))
	if err != nil {
		if os.IsNotExist(err) {
			return ""
		}
		s.t.Fatalf("read %s state: %v", host, err)
	}
	return string(data)
}

func (s *sandbox) mustExist(rel ...string) string {
	s.t.Helper()
	p := s.path(rel...)
	if _, err := os.Stat(p); err != nil {
		s.t.Fatalf("expected %s to exist: %v", p, err)
	}
	return p
}

func (s *sandbox) mustNotExist(rel ...string) {
	s.t.Helper()
	p := s.path(rel...)
	if _, err := os.Stat(p); err == nil {
		s.t.Fatalf("expected %s to not exist", p)
	}
}

func (s *sandbox) writeScript(path, body string) {
	s.t.Helper()
	if err := os.WriteFile(path, []byte(body), 0o755); err != nil {
		s.t.Fatalf("write %s: %v", path, err)
	}
}

func (s *sandbox) copyFile(src, dst string, mode os.FileMode) {
	s.t.Helper()
	data, err := os.ReadFile(src)
	if err != nil {
		s.t.Fatalf("read %s: %v", src, err)
	}
	if err := os.WriteFile(dst, data, mode); err != nil {
		s.t.Fatalf("write %s: %v", dst, err)
	}
}

// addReleaseFixtures publishes a fake release asset plus its checksums so the
// binary component exercises download, checksum and verification.
func (s *sandbox) addReleaseFixtures() {
	asset := releaseAssetName()
	body := []byte(fakeRforgeScript)
	s.writeScript(filepath.Join(s.fixtures, asset), string(body))
	sum := sha256.Sum256(body)
	// Released checksums.txt lists build paths ("dist/<asset>"), not bare names.
	checksums := hex.EncodeToString(sum[:]) + "  dist/" + asset + "\n"
	if err := os.WriteFile(filepath.Join(s.fixtures, "checksums.txt"), []byte(checksums), 0o644); err != nil {
		s.t.Fatalf("write checksums: %v", err)
	}
}

func releaseAssetName() string {
	goos := runtime.GOOS
	goarch := runtime.GOARCH
	if goos == "windows" {
		return "rforge-" + goos + "-" + goarch + ".exe"
	}
	return "rforge-" + goos + "-" + goarch
}

func copyTree(t *testing.T, src, dst string) {
	t.Helper()
	err := filepath.Walk(src, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}
		rel, err := filepath.Rel(src, path)
		if err != nil {
			return err
		}
		target := filepath.Join(dst, rel)
		if info.IsDir() {
			return os.MkdirAll(target, 0o755)
		}
		data, err := os.ReadFile(path)
		if err != nil {
			return err
		}
		return os.WriteFile(target, data, info.Mode().Perm())
	})
	if err != nil {
		t.Fatalf("copy %s: %v", src, err)
	}
}

// snapshot records every path under HOME with its content hash so dry-run can
// be proven to change nothing.
func snapshot(t *testing.T, root string) map[string]string {
	t.Helper()
	state := map[string]string{}
	err := filepath.Walk(root, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}
		rel, err := filepath.Rel(root, path)
		if err != nil {
			return err
		}
		if info.IsDir() {
			state[rel] = "dir"
			return nil
		}
		data, err := os.ReadFile(path)
		if err != nil {
			return err
		}
		sum := sha256.Sum256(data)
		state[rel] = hex.EncodeToString(sum[:])
		return nil
	})
	if err != nil {
		t.Fatalf("snapshot %s: %v", root, err)
	}
	return state
}

func diffSnapshots(before, after map[string]string) []string {
	var changed []string
	for path, state := range after {
		if before[path] != state {
			changed = append(changed, path)
		}
	}
	for path := range before {
		if _, ok := after[path]; !ok {
			changed = append(changed, path)
		}
	}
	sort.Strings(changed)
	return changed
}

func wantContains(t *testing.T, out, want, label string) {
	t.Helper()
	if !strings.Contains(out, want) {
		t.Errorf("%s: output missing %q:\n%s", label, want, out)
	}
}

func wantNotContains(t *testing.T, out, unwanted, label string) {
	t.Helper()
	if strings.Contains(out, unwanted) {
		t.Errorf("%s: output unexpectedly contains %q:\n%s", label, unwanted, out)
	}
}

// --- fresh installation ----------------------------------------------------------

func TestFreshInstallInstallsEveryComponent(t *testing.T) {
	s := newSandbox(t, allHosts...)
	out, code, log := s.run()
	if code != 0 {
		t.Fatalf("exit code = %d; output:\n%s", code, out)
	}
	rforge := s.mustExist(".local", "bin", "rforge")
	verify, err := exec.Command(rforge, "version").CombinedOutput()
	if err != nil {
		t.Fatalf("installed rforge does not run: %v\n%s", err, verify)
	}
	wantContains(t, string(verify), "rforge 0.1.22", "installed binary")

	for _, skill := range []string{"research-forge", "lit", "paper", "brief", "compare", "replicate"} {
		s.mustExist(".agents", "skills", skill, "SKILL.md")
	}
	// Developer-only skills must never be published.
	entries, err := os.ReadDir(s.path(".agents", "skills"))
	if err != nil {
		t.Fatalf("read installed skills: %v", err)
	}
	if len(entries) != 6 {
		t.Errorf("installed %d public skills, want 6", len(entries))
	}

	for _, want := range []string{
		"claude plugin marketplace add",
		"claude plugin install research-forge@research-forge",
		"codex plugin marketplace add",
		"codex plugin add research-forge@research-forge",
		"pi install git:github.com/TrebuchetDynamics/research-forge",
		"hermes plugins install https://github.com/TrebuchetDynamics/research-forge",
		"hermes plugins enable research-forge",
	} {
		wantContains(t, log, want, "host invocations")
	}
	for _, component := range []string{"binary", "skills", "claude", "codex", "pi", "hermes"} {
		wantContains(t, out, component, "summary")
	}
	wantContains(t, out, "6 installed", "summary counts")
}

func TestOnlySelectsExactlyTheNamedComponents(t *testing.T) {
	s := newSandbox(t, allHosts...)
	out, code, log := s.run("--only", "binary,skills")
	if code != 0 {
		t.Fatalf("exit code = %d; output:\n%s", code, out)
	}
	s.mustExist(".local", "bin", "rforge")
	s.mustExist(".agents", "skills", "research-forge", "SKILL.md")
	// Detection probes are read-only; no host component may be installed.
	for _, host := range allHosts {
		if state := s.stateOf(host); state != "" {
			t.Errorf("--only binary,skills installed %s: %s", host, state)
		}
	}
	for _, unwanted := range []string{"Claude Code plugin", "Codex plugin", "Pi package", "Hermes native plugin"} {
		wantNotContains(t, out, unwanted, "--only binary,skills summary")
	}
	if strings.Contains(log, "plugin add") || strings.Contains(log, "plugins install") {
		t.Errorf("--only binary,skills ran host install commands:\n%s", log)
	}
}

func TestDryRunChangesNothing(t *testing.T) {
	s := newSandbox(t, allHosts...)
	before := snapshot(t, s.home)
	out, code, _ := s.run("--dry-run")
	if code != 0 {
		t.Fatalf("exit code = %d; output:\n%s", code, out)
	}
	after := snapshot(t, s.home)
	if changed := diffSnapshots(before, after); len(changed) > 0 {
		t.Fatalf("--dry-run changed %d path(s): %v", len(changed), changed)
	}
	wantContains(t, out, "dry-run", "dry-run output")
	wantContains(t, out, "claude plugin install research-forge@research-forge", "dry-run commands")
	if state, err := os.ReadFile(filepath.Join(s.state, "claude.state")); err == nil && len(state) > 0 {
		t.Errorf("--dry-run mutated host state: %s", state)
	}
}

// --- availability and failure isolation -------------------------------------------

func TestMissingHostIsSkippedByDefault(t *testing.T) {
	s := newSandbox(t, "codex", "pi", "hermes")
	out, code, _ := s.run()
	if code != 0 {
		t.Fatalf("exit code = %d; output:\n%s", code, out)
	}
	wantContains(t, out, "claude", "summary")
	wantContains(t, out, "claude not found", "skip reason")
	s.mustExist(".local", "bin", "rforge")
}

func TestExplicitlyRequestedMissingHostFails(t *testing.T) {
	s := newSandbox(t, "codex")
	out, code, _ := s.run("--only", "claude")
	if code == 0 {
		t.Fatalf("expected nonzero exit when --only names a missing host:\n%s", out)
	}
	wantContains(t, out, "claude not found", "failure reason")
}

func TestOneFailingComponentDoesNotStopTheRest(t *testing.T) {
	s := newSandbox(t, allHosts...)
	s.failOn("codex", "plugin add")
	out, code, _ := s.run()
	if code == 0 {
		t.Fatalf("expected nonzero exit when a component fails:\n%s", out)
	}
	wantContains(t, out, "codex", "summary")
	wantContains(t, out, "failed", "summary")
	s.mustExist(".local", "bin", "rforge")
	s.mustExist(".agents", "skills", "research-forge", "SKILL.md")
	if _, err := os.Stat(filepath.Join(s.state, "claude.state")); err != nil {
		t.Errorf("claude was not installed after codex failed: %v", err)
	}
}

// --- idempotency and update --------------------------------------------------------

func TestRepeatedInstallIsIdempotent(t *testing.T) {
	s := newSandbox(t, allHosts...)
	if out, code, _ := s.run(); code != 0 {
		t.Fatalf("first install failed (%d):\n%s", code, out)
	}
	before := snapshot(t, s.home)
	out, code, _ := s.run()
	if code != 0 {
		t.Fatalf("second install failed (%d):\n%s", code, out)
	}
	if changed := diffSnapshots(before, snapshot(t, s.home)); len(changed) > 0 {
		t.Errorf("reinstall changed %d path(s): %v", len(changed), changed)
	}
	wantContains(t, out, "current", "idempotent skills")
}

func TestUpdateOnlyTouchesInstalledComponents(t *testing.T) {
	s := newSandbox(t, allHosts...)
	// Pretend claude and pi are installed; codex and hermes are not.
	s.seed("claude", "research-forge")
	s.seed("pi", "research-forge")
	out, code, log := s.run("update")
	if code != 0 {
		t.Fatalf("exit code = %d; output:\n%s", code, out)
	}
	wantContains(t, log, "claude plugin update research-forge@research-forge", "claude update")
	wantContains(t, log, "pi update git:github.com/TrebuchetDynamics/research-forge", "pi update")
	wantNotContains(t, log, "codex plugin add", "codex was not installed")
	wantNotContains(t, log, "hermes plugins install", "hermes was not installed")
	wantContains(t, out, "not installed", "skipped reason")
}

// --- Hermes profile selection -------------------------------------------------------

func TestHermesDefaultProfileNeedsNoFlag(t *testing.T) {
	s := newSandbox(t, "hermes")
	out, code, log := s.run("--only", "hermes")
	if code != 0 {
		t.Fatalf("exit code = %d; output:\n%s", code, out)
	}
	wantContains(t, log, "hermes plugins install", "hermes install")
	wantNotContains(t, log, "hermes --profile", "default profile must not be forced")
	wantContains(t, s.stateOf("hermes"), "research-forge@default", "default profile state")
}

func TestHermesExplicitProfileIsUsedAndVerified(t *testing.T) {
	s := newSandbox(t, "hermes")
	out, code, log := s.run("--only", "hermes", "--hermes-profile", "research-forge")
	if code != 0 {
		t.Fatalf("exit code = %d; output:\n%s", code, out)
	}
	wantContains(t, log, "hermes --profile research-forge plugins install", "profile flag")
	wantContains(t, s.stateOf("hermes"), "research-forge@research-forge", "named profile state")
}

func TestHermesUnknownProfileFailsWithChoices(t *testing.T) {
	s := newSandbox(t, "hermes")
	out, code, _ := s.run("--only", "hermes", "--hermes-profile", "nope")
	if code == 0 {
		t.Fatalf("expected failure for an unknown profile:\n%s", out)
	}
	wantContains(t, out, "nope", "failure names the profile")
	wantContains(t, out, "research-forge", "failure lists detected profiles")
}

// --- PATH and shadowing --------------------------------------------------------------

func TestInstallDirOffPathWarnsWithExportLine(t *testing.T) {
	s := newSandbox(t, "pi")
	out, code, _ := s.run("--only", "binary")
	if code != 0 {
		t.Fatalf("exit code = %d; output:\n%s", code, out)
	}
	wantContains(t, out, "not on your PATH", "PATH warning")
	wantContains(t, out, "export PATH=", "PATH guidance")
}

func TestLegacySkillCopyIsQuarantinedWhenPluginInstalls(t *testing.T) {
	s := newSandbox(t, "claude")
	legacy := s.path(".claude", "skills", "research-forge")
	if err := os.MkdirAll(legacy, 0o755); err != nil {
		t.Fatalf("mkdir legacy skill: %v", err)
	}
	if err := os.WriteFile(filepath.Join(legacy, "SKILL.md"), []byte("old copy\n"), 0o644); err != nil {
		t.Fatalf("write legacy skill: %v", err)
	}
	out, code, _ := s.run("--only", "claude")
	if code != 0 {
		t.Fatalf("exit code = %d; output:\n%s", code, out)
	}
	if _, err := os.Stat(legacy); err == nil {
		t.Errorf("legacy skill copy still shadows the plugin at %s", legacy)
	}
	if _, err := os.Stat(legacy + ".bak"); err != nil {
		t.Errorf("legacy copy was not preserved as a backup: %v", err)
	}
	wantContains(t, out, "bak", "quarantine report")
}

// --- skills source modes -------------------------------------------------------------

func TestPiPackageSkipsDuplicateGenericSkills(t *testing.T) {
	s := newSandbox(t, "pi")
	s.seed("pi", "research-forge")
	out, code, _ := s.run("--only", "skills")
	if code != 0 {
		t.Fatalf("exit code = %d; output:\n%s", code, out)
	}
	s.mustNotExist(".agents", "skills", "research-forge", "SKILL.md")
	wantContains(t, out, "Pi package", "duplicate reason")
}

func TestPipedInstallFetchesSkillsFromRawURLs(t *testing.T) {
	s := newSandbox(t)
	s.setEnv("FAKE_CURL_ECHO=1")
	// No checkout: only the script itself is available.
	if err := os.RemoveAll(filepath.Join(s.repo, "adapters")); err != nil {
		t.Fatalf("remove adapters: %v", err)
	}
	out, code, log := s.run("--only", "skills")
	if code != 0 {
		t.Fatalf("exit code = %d; output:\n%s", code, out)
	}
	wantContains(t, log, "raw.githubusercontent.com", "raw fetch")
	body, err := os.ReadFile(s.path(".agents", "skills", "brief", "SKILL.md"))
	if err != nil {
		t.Fatalf("read fetched skill: %v", err)
	}
	wantContains(t, string(body), "adapters/skills/brief/SKILL.md", "fetched skill content")
}

// --- surface -------------------------------------------------------------------------

func TestHelpAndListDescribeTheInstaller(t *testing.T) {
	s := newSandbox(t, "pi")
	help, code, _ := s.run("--help")
	if code != 0 {
		t.Fatalf("--help exit code = %d", code)
	}
	for _, want := range []string{"--menu", "--dry-run", "--only", "--for", "--all", "--verbose", "--quiet", "update", "--from-source", "--version", "--dir"} {
		wantContains(t, help, want, "--help")
	}
	for _, component := range []string{"binary", "skills", "claude", "codex", "pi", "hermes"} {
		wantContains(t, help, component, "--help components")
	}

	list, code, _ := s.run("--list")
	if code != 0 {
		t.Fatalf("--list exit code = %d", code)
	}
	wantContains(t, list, "available", "--list")
	wantContains(t, list, "claude not found", "--list availability")
}

func TestUnknownComponentIsRejected(t *testing.T) {
	s := newSandbox(t, "pi")
	out, code, _ := s.run("--only", "binary,nope")
	if code == 0 {
		t.Fatalf("expected failure for an unknown component:\n%s", out)
	}
	wantContains(t, out, "nope", "error names the component")
	wantContains(t, out, "binary", "error lists valid components")
}

// Released checksums.txt lists build paths ("dist/rforge-linux-amd64"), so the
// installer must match by basename and actually verify the download.
func TestReleaseChecksumIsVerified(t *testing.T) {
	s := newSandbox(t)
	out, code, _ := s.run("--only", "binary")
	if code != 0 {
		t.Fatalf("exit code = %d; output:\n%s", code, out)
	}
	wantContains(t, out, "checksum verified", "checksum")
}

func TestChecksumMismatchAbortsBinaryInstall(t *testing.T) {
	s := newSandbox(t)
	asset := releaseAssetName()
	bad := strings.Repeat("0", 64) + "  dist/" + asset + "\n"
	if err := os.WriteFile(filepath.Join(s.fixtures, "checksums.txt"), []byte(bad), 0o644); err != nil {
		t.Fatalf("write bad checksums: %v", err)
	}
	out, code, _ := s.run("--only", "binary")
	if code == 0 {
		t.Fatalf("expected failure on checksum mismatch:\n%s", out)
	}
	wantContains(t, out, "checksum mismatch", "checksum failure")
	s.mustNotExist(".local", "bin", "rforge")
}

// --- interactive selection and verbosity ---------------------------------------------

// runInput drives the text menu by feeding stdin.
func (s *sandbox) runInput(input string, args ...string) (string, int, string) {
	s.t.Helper()
	cmd := exec.Command("bash", append([]string{filepath.Join(s.repo, "install.sh")}, args...)...)
	cmd.Env = append(os.Environ(), s.env...)
	cmd.Env = append(cmd.Env, "RFORGE_INSTALL_UI=text")
	cmd.Dir = s.root
	cmd.Stdin = strings.NewReader(input)
	out, err := cmd.CombinedOutput()
	code := 0
	if err != nil {
		if exitErr, ok := err.(*exec.ExitError); ok {
			code = exitErr.ExitCode()
		} else {
			s.t.Fatalf("run install.sh: %v", err)
		}
	}
	logged, _ := os.ReadFile(s.log)
	return string(out), code, string(logged)
}

func TestMenuToggleSelectsComponents(t *testing.T) {
	s := newSandbox(t, "pi")
	out, code, _ := s.runInput("3\n\n", "--menu")
	if code != 0 {
		t.Fatalf("exit code = %d; output:\n%s", code, out)
	}
	// Component 3 is claude, which is not installed here, so toggling it must not
	// fail the run; the remaining defaults (binary, skills, pi, hermes) install.
	wantContains(t, out, "[x] 1 binary", "menu rendering")
	s.mustExist(".local", "bin", "rforge")
	wantContains(t, out, "installed", "menu summary")
}

func TestMenuQuitInstallsNothing(t *testing.T) {
	s := newSandbox(t, "pi")
	out, code, _ := s.runInput("q\n", "--menu")
	if code != 0 {
		t.Fatalf("exit code = %d; output:\n%s", code, out)
	}
	wantContains(t, out, "Nothing installed", "quit")
	s.mustNotExist(".local", "bin", "rforge")
}

func TestQuietHidesCommandsAndVerboseShowsThem(t *testing.T) {
	s := newSandbox(t, "pi")
	quiet, code, _ := s.run("--only", "skills", "--quiet")
	if code != 0 {
		t.Fatalf("quiet exit code = %d:\n%s", code, quiet)
	}
	wantNotContains(t, quiet, "$ cp", "quiet output")

	verbose, code, _ := s.run("--only", "skills", "--verbose", "--dry-run")
	if code != 0 {
		t.Fatalf("verbose exit code = %d:\n%s", code, verbose)
	}
	wantContains(t, verbose, "$ cp", "verbose output")
}

// --- install profiles ---------------------------------------------------------------

func TestProfileCLIInstallsOnlyTheEngine(t *testing.T) {
	s := newSandbox(t, allHosts...)
	out, code, log := s.run("--for", "cli")
	if code != 0 {
		t.Fatalf("exit code = %d; output:\n%s", code, out)
	}
	s.mustExist(".local", "bin", "rforge")
	s.mustNotExist(".agents", "skills", "research-forge", "SKILL.md")
	for _, host := range allHosts {
		if state := s.stateOf(host); state != "" {
			t.Errorf("--for cli installed %s: %s", host, state)
		}
	}
	wantNotContains(t, out, "Claude Code plugin", "--for cli summary")
	if strings.Contains(log, "plugins install") || strings.Contains(log, "plugin add") {
		t.Errorf("--for cli ran host install commands:\n%s", log)
	}
}

func TestProfileSkillsInstallsSkillSurfacesWithoutTheEngine(t *testing.T) {
	s := newSandbox(t, allHosts...)
	out, code, _ := s.run("--for", "skills")
	if code != 0 {
		t.Fatalf("exit code = %d; output:\n%s", code, out)
	}
	s.mustExist(".agents", "skills", "research-forge", "SKILL.md")
	for _, host := range allHosts {
		wantContains(t, s.stateOf(host), "research-forge", "--for skills installed "+host)
	}
	s.mustNotExist(".local", "bin", "rforge")
	wantNotContains(t, out, "rforge engine binary", "--for skills summary")
}

func TestProfileMCPInstallsEngineAndHostRegistration(t *testing.T) {
	s := newSandbox(t, allHosts...)
	out, code, _ := s.run("--for", "mcp")
	if code != 0 {
		t.Fatalf("exit code = %d; output:\n%s", code, out)
	}
	s.mustExist(".local", "bin", "rforge")
	for _, host := range allHosts {
		wantContains(t, s.stateOf(host), "research-forge", "--for mcp registered "+host)
	}
	// The native plugins carry the skills, so no duplicate generic copy is needed.
	s.mustNotExist(".agents", "skills", "research-forge", "SKILL.md")
}

func TestProfileAllMatchesTheDefault(t *testing.T) {
	s := newSandbox(t, allHosts...)
	out, code, _ := s.run("--for", "all")
	if code != 0 {
		t.Fatalf("exit code = %d; output:\n%s", code, out)
	}
	s.mustExist(".local", "bin", "rforge")
	s.mustExist(".agents", "skills", "research-forge", "SKILL.md")
	for _, host := range allHosts {
		wantContains(t, s.stateOf(host), "research-forge", "--for all installed "+host)
	}
}

func TestProfileRejectsUnknownValue(t *testing.T) {
	s := newSandbox(t, "pi")
	out, code, _ := s.run("--for", "everything")
	if code == 0 {
		t.Fatalf("expected failure for an unknown profile:\n%s", out)
	}
	wantContains(t, out, "everything", "error names the profile")
	for _, want := range []string{"cli", "skills", "mcp"} {
		wantContains(t, out, want, "error lists profiles")
	}
}

func TestProfileConflictsWithOnly(t *testing.T) {
	s := newSandbox(t, "pi")
	out, code, _ := s.run("--for", "cli", "--only", "binary")
	if code == 0 {
		t.Fatalf("expected failure when --for and --only are combined:\n%s", out)
	}
	wantContains(t, out, "--only", "conflict message")
}

func TestProfileAllowsSkip(t *testing.T) {
	s := newSandbox(t, allHosts...)
	out, code, _ := s.run("--for", "mcp", "--skip", "codex")
	if code != 0 {
		t.Fatalf("exit code = %d; output:\n%s", code, out)
	}
	if state := s.stateOf("codex"); state != "" {
		t.Errorf("--skip codex still installed codex: %s", state)
	}
	wantContains(t, s.stateOf("claude"), "research-forge", "--for mcp --skip codex kept claude")
}

// A profile is a convenience selection, not an explicit per-component demand: a
// host that is missing must be skipped, exactly like the default suite.
func TestProfileMissingHostIsSkippedNotFailed(t *testing.T) {
	s := newSandbox(t, "pi")
	out, code, _ := s.run("--for", "mcp")
	if code != 0 {
		t.Fatalf("exit code = %d; output:\n%s", code, out)
	}
	wantContains(t, out, "claude not found", "skip reason")
}

// --- source builds --------------------------------------------------------------

const fakeGoScript = `#!/usr/bin/env bash
[ -n "${SANDBOX_LOG:-}" ] && printf 'go %s cwd=%s\n' "$*" "$PWD" >> "$SANDBOX_LOG"
out=""; prev=""
for a in "$@"; do
  if [ "$prev" = "-o" ]; then out="$a"; fi
  prev="$a"
done
if [ -n "$out" ]; then
  cp "$FIXTURES/rforge-dev" "$out" && chmod +x "$out"
elif [ "${1:-}" = "install" ] && [ -n "${GOBIN:-}" ]; then
  cp "$FIXTURES/rforge-dev" "$GOBIN/rforge" && chmod +x "$GOBIN/rforge"
fi
exit 0
`

// --from-source run inside a checkout must build that checkout (a dev engine
// has the capabilities a release has not shipped yet), not fetch @latest from
// the module proxy.
func TestFromSourceBuildsTheLocalCheckout(t *testing.T) {
	s := newSandbox(t)
	s.writeScript(filepath.Join(s.bin, "go"), fakeGoScript)
	s.writeScript(filepath.Join(s.fixtures, "rforge-dev"), fakeRforgeScript)
	if err := os.MkdirAll(filepath.Join(s.repo, "cmd", "rforge"), 0o755); err != nil {
		t.Fatalf("mkdir cmd/rforge: %v", err)
	}
	if err := os.WriteFile(filepath.Join(s.repo, "cmd", "rforge", "main.go"), []byte("package main\n"), 0o644); err != nil {
		t.Fatalf("write checkout marker: %v", err)
	}

	out, code, log := s.run("--only", "binary", "--from-source")
	if code != 0 {
		t.Fatalf("exit code = %d; output:\n%s", code, out)
	}
	wantContains(t, log, "go build", "checkout build")
	wantContains(t, log, "./cmd/rforge", "checkout package")
	if strings.Contains(log, "go install") {
		t.Errorf("checkout build used go install @latest instead of the checkout:\n%s", log)
	}
	if !strings.Contains(log, s.repo) {
		t.Errorf("go build did not run in the checkout (%s):\n%s", s.repo, log)
	}
	// Provenance records the engine version, so a source build must carry it.
	for _, want := range []string{"-ldflags", "-X main.commit=", "-X main.date="} {
		wantContains(t, log, want, "source build metadata")
	}
	// An untagged checkout stays "dev": capabilities, not a version label, decide
	// whether a dev engine is accepted.
	wantContains(t, log, "-X main.version=dev", "untagged version")
	s.mustExist(".local", "bin", "rforge")
}

// Outside a checkout (piped install) --from-source keeps the module install.
func TestFromSourceWithoutCheckoutUsesGoInstall(t *testing.T) {
	s := newSandbox(t)
	s.writeScript(filepath.Join(s.bin, "go"), fakeGoScript)
	s.writeScript(filepath.Join(s.fixtures, "rforge-dev"), fakeRforgeScript)

	out, code, log := s.run("--only", "binary", "--from-source")
	if code != 0 {
		t.Fatalf("exit code = %d; output:\n%s", code, out)
	}
	wantContains(t, log, "go install", "module install")
	wantContains(t, log, "@latest", "module install ref")
}

// A checkout that is exactly at a release tag is that release.
func TestFromSourceStampsExactReleaseTag(t *testing.T) {
	s := newSandbox(t)
	s.writeScript(filepath.Join(s.bin, "go"), fakeGoScript)
	s.writeScript(filepath.Join(s.fixtures, "rforge-dev"), fakeRforgeScript)
	if err := os.MkdirAll(filepath.Join(s.repo, "cmd", "rforge"), 0o755); err != nil {
		t.Fatalf("mkdir cmd/rforge: %v", err)
	}
	if err := os.WriteFile(filepath.Join(s.repo, "cmd", "rforge", "main.go"), []byte("package main\n"), 0o644); err != nil {
		t.Fatalf("write checkout marker: %v", err)
	}
	runGit(t, s.repo, "init", "-q")
	runGit(t, s.repo, "config", "user.email", "test@example.com")
	runGit(t, s.repo, "config", "user.name", "Test")
	runGit(t, s.repo, "add", ".")
	runGit(t, s.repo, "commit", "-q", "-m", "checkout")
	runGit(t, s.repo, "tag", "v9.9.9")

	out, code, log := s.run("--only", "binary", "--from-source")
	if code != 0 {
		t.Fatalf("exit code = %d; output:\n%s", code, out)
	}
	wantContains(t, log, "-X main.version=v9.9.9", "tagged version")
}

// runGit runs git in dir without inheriting repository selectors from the test
// process (the repository's own pre-commit hook may be running).
func runGit(t *testing.T, dir string, args ...string) {
	t.Helper()
	cmd := exec.Command("git", args...)
	cmd.Dir = dir
	for _, entry := range os.Environ() {
		switch {
		case strings.HasPrefix(entry, "GIT_DIR="),
			strings.HasPrefix(entry, "GIT_WORK_TREE="),
			strings.HasPrefix(entry, "GIT_INDEX_FILE="),
			strings.HasPrefix(entry, "GIT_COMMON_DIR="),
			strings.HasPrefix(entry, "GIT_OBJECT_DIRECTORY="),
			strings.HasPrefix(entry, "GIT_ALTERNATE_OBJECT_DIRECTORIES="):
			continue
		}
		cmd.Env = append(cmd.Env, entry)
	}
	cmd.Env = append(cmd.Env, "GIT_CONFIG_GLOBAL=/dev/null", "GIT_CONFIG_SYSTEM=/dev/null")
	if out, err := cmd.CombinedOutput(); err != nil {
		t.Fatalf("git %v: %v\n%s", args, err, out)
	}
}

// A checkout past a tag is not that release: git describe would yield a version
// like v0.1.21-162-gabc that fails the capabilities gate on a version compare.
// A dev build must say "dev" so the gate judges capabilities instead.
func TestFromSourceStaysDevPastATag(t *testing.T) {
	s := newSandbox(t)
	s.writeScript(filepath.Join(s.bin, "go"), fakeGoScript)
	s.writeScript(filepath.Join(s.fixtures, "rforge-dev"), fakeRforgeScript)
	if err := os.MkdirAll(filepath.Join(s.repo, "cmd", "rforge"), 0o755); err != nil {
		t.Fatalf("mkdir cmd/rforge: %v", err)
	}
	if err := os.WriteFile(filepath.Join(s.repo, "cmd", "rforge", "main.go"), []byte("package main\n"), 0o644); err != nil {
		t.Fatalf("write checkout marker: %v", err)
	}
	runGit(t, s.repo, "init", "-q")
	runGit(t, s.repo, "config", "user.email", "test@example.com")
	runGit(t, s.repo, "config", "user.name", "Test")
	runGit(t, s.repo, "add", ".")
	runGit(t, s.repo, "commit", "-q", "-m", "checkout")
	runGit(t, s.repo, "tag", "v0.1.21")
	runGit(t, s.repo, "commit", "-q", "--allow-empty", "-m", "later work")

	out, code, log := s.run("--only", "binary", "--from-source")
	if code != 0 {
		t.Fatalf("exit code = %d; output:\n%s", code, out)
	}
	wantContains(t, log, "-X main.version=dev", "version past a tag")
	if strings.Contains(log, "-X main.version=v0.1.21-") {
		t.Errorf("source build stamped a describe version that fails the gate:\n%s", log)
	}
}

// Installing into a path that is a file must fail with the real cause and must
// never print a success line for a binary that was not written.
func TestBinaryInstallIntoFilePathFailsClearly(t *testing.T) {
	s := newSandbox(t)
	file := filepath.Join(s.root, "notadir")
	if err := os.WriteFile(file, []byte("x"), 0o644); err != nil {
		t.Fatalf("write file: %v", err)
	}
	out, code, _ := s.run("--only", "binary", "--dir", file)
	if code == 0 {
		t.Fatalf("expected failure installing into a file path:\n%s", out)
	}
	wantNotContains(t, out, "Binary →", "must not claim success")
	wantContains(t, out, "cannot create directory", "clear cause")
	wantNotContains(t, out, "installed binary does not run", "misleading cause")
}

// --- capability-gate recovery -----------------------------------------------

const incompatibleRforgeScript = `#!/usr/bin/env bash
case "${1:-}" in
  version) printf 'rforge v0.1.21 (old, old)\n' ;;
  capabilities) printf '{"ok":false,"error":{"code":"unknown_command"}}\n'; exit 1 ;;
esac
exit 0
`

// When the release binary fails the capability gate but this script runs from a
// buildable checkout with Go, build the checkout instead of leaving the user
// with a broken install and no path forward.
func TestCapabilityGateFallsBackToCheckoutBuild(t *testing.T) {
	s := newSandbox(t)
	s.writeScript(filepath.Join(s.bin, "go"), fakeGoScript)
	s.writeScript(filepath.Join(s.fixtures, "rforge-dev"), fakeRforgeScript)
	// The release asset is the old engine that fails the gate.
	s.writeScript(filepath.Join(s.fixtures, releaseAssetName()), incompatibleRforgeScript)
	sum := sha256.Sum256([]byte(incompatibleRforgeScript))
	if err := os.WriteFile(filepath.Join(s.fixtures, "checksums.txt"),
		[]byte(hex.EncodeToString(sum[:])+"  dist/"+releaseAssetName()+"\n"), 0o644); err != nil {
		t.Fatalf("write checksums: %v", err)
	}
	if err := os.MkdirAll(filepath.Join(s.repo, "cmd", "rforge"), 0o755); err != nil {
		t.Fatalf("mkdir cmd/rforge: %v", err)
	}
	if err := os.WriteFile(filepath.Join(s.repo, "cmd", "rforge", "main.go"), []byte("package main\n"), 0o644); err != nil {
		t.Fatalf("write checkout marker: %v", err)
	}

	out, code, log := s.run("--only", "binary")
	if code != 0 {
		t.Fatalf("exit code = %d; output:\n%s", code, out)
	}
	wantContains(t, out, "does not satisfy the capability gate", "fallback notice")
	// The release failure no longer describes the installed engine.
	wantNotContains(t, out, "does not meet the required rforge", "stale release failure")
	wantContains(t, log, "go build", "checkout build")
	wantContains(t, log, "-X main.version=dev", "checkout build metadata")
	s.mustExist(".local", "bin", "rforge")
}

// Without a buildable checkout the gate still fails closed with guidance.
func TestCapabilityGateWithoutCheckoutFailsClosed(t *testing.T) {
	s := newSandbox(t)
	s.writeScript(filepath.Join(s.bin, "go"), fakeGoScript)
	s.writeScript(filepath.Join(s.fixtures, releaseAssetName()), incompatibleRforgeScript)
	sum := sha256.Sum256([]byte(incompatibleRforgeScript))
	if err := os.WriteFile(filepath.Join(s.fixtures, "checksums.txt"),
		[]byte(hex.EncodeToString(sum[:])+"  dist/"+releaseAssetName()+"\n"), 0o644); err != nil {
		t.Fatalf("write checksums: %v", err)
	}

	out, code, _ := s.run("--only", "binary")
	if code == 0 {
		t.Fatalf("expected the capability gate to fail closed:\n%s", out)
	}
	wantContains(t, out, "does not meet the required rforge", "gate failure")
}
