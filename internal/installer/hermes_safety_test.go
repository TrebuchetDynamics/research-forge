package installer_test

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// Installing Hermes must never quarantine customized standalone skills, even
// when native selection differs from the installer's old ~/.hermes guess.
func TestHermesPreservesStandaloneSkills(t *testing.T) {
	for _, tc := range []struct {
		name, envProfile, sticky, home, flag, selected string
	}{
		{name: "current", selected: "default"},
		{name: "environment", envProfile: "research-forge", selected: "research-forge"},
		{name: "sticky", sticky: "research-forge", selected: "research-forge"},
		{name: "explicit-default", envProfile: "research-forge", flag: "default", selected: "default"},
		{name: "named", flag: "research-forge", selected: "research-forge"},
		{name: "native-name-not-in-list", flag: "native-only", selected: "native-only"},
		{name: "custom-home", home: "custom-hermes", sticky: "research-forge", selected: "research-forge"},
	} {
		t.Run(tc.name, func(t *testing.T) {
			s := newSandbox(t, "hermes")
			s.setEnv("HERMES_PROFILE="+tc.envProfile, "HERMES_HOME=")
			hermesHome := s.path(".hermes")
			if tc.home != "" {
				hermesHome = s.path(tc.home)
				s.setEnv("HERMES_HOME=" + hermesHome)
			}
			for _, root := range []string{s.path(".agents"), s.path(".hermes"), s.path(".hermes", "profiles", "default"), s.path(".hermes", "profiles", "research-forge"), s.path(".hermes", "profiles", "unrelated"), hermesHome} {
				for _, skill := range []string{"research-forge", "lit", "paper", "brief", "compare", "replicate", "custom"} {
					dir := filepath.Join(root, "skills", skill)
					if err := os.MkdirAll(dir, 0o755); err != nil {
						t.Fatal(err)
					}
					s.writeScript(filepath.Join(dir, "SKILL.md"), "custom standalone "+skill+"\n")
				}
			}
			if tc.sticky != "" {
				s.writeScript(filepath.Join(hermesHome, "fake-current-profile"), tc.sticky+"\n")
			}
			s.writeScript(filepath.Join(s.state, "hermes.state.unrelated"), "enabled git pinned@untouched 1.0 research-forge\n")
			before := snapshot(t, s.home)
			args, prefix := []string{"--only", "hermes"}, "hermes"
			if tc.flag != "" {
				args = append(args, "--hermes-profile", tc.flag)
				prefix += " --profile " + tc.flag
			}
			out, code, log := s.run(args...)
			if code != 0 {
				t.Fatalf("install failed (%d):\n%s", code, out)
			}
			if changed := diffSnapshots(before, snapshot(t, s.home)); len(changed) != 0 {
				t.Fatalf("Hermes install changed standalone/profile files: %v", changed)
			}
			wantContains(t, log, prefix+" plugins install", "native selection")
			if tc.flag == "" {
				wantNotContains(t, log, "hermes --profile", "current selection stays native")
			}
			wantContains(t, s.stateOf("hermes"), "research-forge@"+tc.selected+"\n", "selected profile")
			data, err := os.ReadFile(filepath.Join(s.state, "hermes.state.unrelated"))
			if err != nil || string(data) != "enabled git pinned@untouched 1.0 research-forge\n" {
				t.Fatalf("unrelated plugin changed: %q, %v", data, err)
			}
		})
	}
}

// Repeat install must preserve a native source pin, not replace it with main;
// neither a similar name nor a successful enable establishes enabled identity.
func TestHermesIdempotentPinnedLifecycle(t *testing.T) {
	const pin = "git pinned@c98d77ca 0.1.0 research-forge\n"
	for _, tc := range []struct {
		name, rows, command, fail, wantRows, wantMutation, message string
		noopEnable, failEnabledList, wantFailure                   bool
	}{
		{name: "enabled-pin", rows: "enabled " + pin, wantRows: "enabled " + pin, message: "already enabled"},
		{name: "disabled-pin", rows: "disabled " + pin, wantRows: "enabled " + pin, wantMutation: "enable research-forge"},
		{name: "similar-name-is-absent", rows: "enabled git pinned@other 1.0 research-forge-extra\n", wantRows: "enabled git pinned@other 1.0 research-forge-extra\nenabled git 0.1.0 research-forge\n", wantMutation: "install https://github.com/TrebuchetDynamics/research-forge/tree/main/hermes-plugin --no-deps"},
		{name: "enable-does-not-activate", rows: "disabled " + pin, wantRows: "disabled " + pin, wantMutation: "enable research-forge", noopEnable: true, wantFailure: true, message: "not enabled"},
		{name: "enable-error", rows: "disabled " + pin, wantRows: "disabled " + pin, wantMutation: "enable research-forge", fail: "plugins enable", wantFailure: true, message: "simulated failure"},
		{name: "install-error", wantMutation: "install https://github.com/TrebuchetDynamics/research-forge/tree/main/hermes-plugin --no-deps", fail: "plugins install", wantFailure: true, message: "simulated failure"},
		{name: "verification-list-error", rows: "disabled " + pin, wantRows: "enabled " + pin, wantMutation: "enable research-forge", failEnabledList: true, wantFailure: true, message: "simulated post-enable list failure"},
		{name: "pinned-update-refused", rows: "enabled " + pin, wantRows: "enabled " + pin, command: "update", wantMutation: "update research-forge", wantFailure: true, message: "explicit --ref"},
	} {
		t.Run(tc.name, func(t *testing.T) {
			s := newSandbox(t, "hermes")
			rowsPath := filepath.Join(s.state, "hermes.state.default")
			s.writeScript(rowsPath, tc.rows)
			s.writeScript(filepath.Join(s.state, "hermes.state.unrelated"), "enabled git pinned@untouched 1.0 research-forge\n")
			beforeHome, beforeState := snapshot(t, s.home), snapshot(t, s.state)
			if tc.noopEnable {
				s.writeScript(filepath.Join(s.state, "hermes.state.enable-noop"), "")
			}
			if tc.failEnabledList {
				s.writeScript(filepath.Join(s.state, "hermes.state.fail-enabled-list"), "")
			}
			if tc.fail != "" {
				s.failOn("hermes", tc.fail)
			}
			args := []string{"--only", "hermes", "--hermes-profile", "default"}
			if tc.command != "" {
				args = append([]string{tc.command}, args...)
				args = append(args, "--quiet")
			}
			out, code, log := s.run(args...)
			if (code != 0) != tc.wantFailure {
				t.Fatalf("exit = %d, want failure %v:\n%s", code, tc.wantFailure, out)
			}
			wantContains(t, log, "hermes --profile default plugins list --plain --no-bundled", "native exact listing")
			var mutations []string
			for _, line := range strings.Split(log, "\n") {
				if strings.Contains(line, " plugins ") && !strings.Contains(line, " plugins list") {
					mutations = append(mutations, line)
				}
			}
			wantMutations := ""
			if tc.wantMutation != "" {
				wantMutations = "hermes --profile default plugins " + tc.wantMutation
				if strings.HasPrefix(tc.wantMutation, "install ") && tc.fail != "plugins install" {
					wantMutations += "\nhermes --profile default plugins enable research-forge"
				}
			}
			if got := strings.Join(mutations, "\n"); got != wantMutations {
				t.Errorf("mutations = %q, want %q", got, wantMutations)
			}
			data, err := os.ReadFile(rowsPath)
			if err != nil || string(data) != tc.wantRows {
				t.Errorf("plugin rows = %q, want %q (error %v)", data, tc.wantRows, err)
			}
			if tc.message != "" {
				wantContains(t, out, tc.message, "lifecycle result")
			}
			if changed := diffSnapshots(beforeHome, snapshot(t, s.home)); len(changed) != 0 {
				t.Errorf("user files changed: %v", changed)
			}
			afterState := snapshot(t, s.state)
			if afterState["hermes.state.unrelated"] != beforeState["hermes.state.unrelated"] {
				t.Error("unrelated profile pin changed")
			}
			if tc.wantMutation == "" || tc.command == "update" {
				if changed := diffSnapshots(beforeState, afterState); len(changed) != 0 {
					t.Errorf("no-op/refused update changed state: %v", changed)
				}
			}
		})
	}
}

// Validate every target and probe before the first mutation; dry-run must use
// the same read-only preflight, but never execute the planned writes.
func TestHermesPreflightAndDryRun(t *testing.T) {
	for _, tc := range []struct {
		name, profiles, fail, rows, planned string
		dry, wantFailure                    bool
	}{
		{name: "profile-probe-error", profiles: "default", fail: "profile show", wantFailure: true},
		{name: "invalid-later-profile", profiles: "default,nope", wantFailure: true},
		{name: "unknown-profile-is-not-current", profiles: "__current__", wantFailure: true},
		{name: "profile-is-not-shell-split", profiles: "default research-forge", wantFailure: true},
		{name: "blank-later-profile", profiles: "default, ", wantFailure: true},
		{name: "list-probe-error", profiles: "default", fail: "plugins list", wantFailure: true},
		{name: "later-list-probe-error", profiles: "default,research-forge", fail: "--profile research-forge plugins list", wantFailure: true},
		{name: "dry-run-absent", profiles: "default", dry: true, planned: "plugins install https://github.com/TrebuchetDynamics/research-forge/tree/main/hermes-plugin --no-deps"},
		{name: "dry-run-disabled", profiles: "default", dry: true, rows: "disabled git pinned@c98d77ca 0.1.0 research-forge\n", planned: "plugins enable research-forge"},
		{name: "dry-run-failed-probe", profiles: "default", dry: true, fail: "plugins list", wantFailure: true},
	} {
		t.Run(tc.name, func(t *testing.T) {
			s := newSandbox(t, "hermes")
			dir := s.path(".hermes", "skills", "research-forge")
			if err := os.MkdirAll(dir, 0o755); err != nil {
				t.Fatal(err)
			}
			s.writeScript(filepath.Join(dir, "SKILL.md"), "custom standalone methodology\n")
			s.writeScript(s.path(".hermes", "config.yaml"), "model: unchanged\n")
			if tc.rows != "" {
				s.writeScript(filepath.Join(s.state, "hermes.state.default"), tc.rows)
			}
			if tc.fail != "" {
				s.failOn("hermes", tc.fail)
			}
			beforeHome, beforeState := snapshot(t, s.home), snapshot(t, s.state)
			args := []string{"--only", "hermes", "--hermes-profiles", tc.profiles}
			if tc.dry {
				args = append(args, "--dry-run")
			}
			out, code, log := s.run(args...)
			if (code != 0) != tc.wantFailure {
				t.Errorf("exit = %d, want failure %v:\n%s", code, tc.wantFailure, out)
			}
			for _, command := range []string{"install", "enable", "update"} {
				wantNotContains(t, log, "plugins "+command, "preflight/dry-run must not mutate")
			}
			if tc.planned != "" {
				wantContains(t, out, "hermes --profile default "+tc.planned, "planned native command")
			}
			if tc.fail != "" {
				wantContains(t, out, "simulated failure", "visible native probe failure")
			}
			if changed := diffSnapshots(beforeHome, snapshot(t, s.home)); len(changed) != 0 {
				t.Errorf("user files changed: %v", changed)
			}
			if changed := diffSnapshots(beforeState, snapshot(t, s.state)); len(changed) != 0 {
				t.Errorf("plugin/profile state changed: %v", changed)
			}
		})
	}
}
