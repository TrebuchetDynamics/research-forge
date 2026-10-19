package cli

import (
	"bytes"
	"encoding/json"
	"reflect"
	"strings"
	"testing"
)

// These tests exercise the public CLI seam: omitted contracts, lexical version
// comparisons, and treating dev as a release must not silently admit an engine.
func TestExecuteCapabilitiesReport(t *testing.T) {
	oldVersion := Version
	Version = "v2.3.4"
	t.Cleanup(func() { Version = oldVersion })
	for _, args := range [][]string{{"capabilities", "--json"}, {"--json", "capabilities"}} {
		var out, errOut bytes.Buffer
		if code := Execute(args, &out, &errOut); code != 0 {
			t.Fatalf("%v: exit %d: %s%s", args, code, out.String(), errOut.String())
		}
		var result struct {
			OK   bool `json:"ok"`
			Data struct {
				Version      string         `json:"version"`
				Capabilities map[string]int `json:"capabilities"`
				ErrorCodes   []string       `json:"error_codes"`
			} `json:"data"`
		}
		if err := json.Unmarshal(out.Bytes(), &result); err != nil {
			t.Fatal(err)
		}
		want := map[string]int{"search": 1, "paper": 1, "brief": 1, "compare": 1, "provenance": 1, "mcp": 1}
		if !result.OK || result.Data.Version != "v2.3.4" || !reflect.DeepEqual(result.Data.Capabilities, want) {
			t.Fatalf("unexpected report: %s", out.String())
		}
		for _, code := range []string{"usage", "version_invalid", "version_too_old", "capability_missing", "capability_too_old"} {
			if !strings.Contains(" "+strings.Join(result.Data.ErrorCodes, " ")+" ", " "+code+" ") {
				t.Errorf("report missing error code %q", code)
			}
		}
		if errOut.Len() != 0 {
			t.Fatalf("JSON command wrote stderr: %s", errOut.String())
		}
	}
}

func TestExecuteCapabilitiesCompatibility(t *testing.T) {
	oldVersion := Version
	t.Cleanup(func() { Version = oldVersion })
	for _, tt := range []struct {
		name, version, minimum, requirement, wantCode, gap string
	}{
		{"release pass", "v1.10.0", "1.9.0", "search=1", "", ""},
		{"dev with capabilities pass", "dev", "999.0.0", "mcp=1", "", ""},
		{"dev missing capability fail", "dev", "1.0.0", "claims=1", "capability_missing", "claims"},
		{"older capability fail", "dev", "1.0.0", "search=2", "capability_too_old", "search"},
		{"release missing capability fail", "v2.0.0", "1.0.0", "claims=1", "capability_missing", "claims"},
		{"old version fail", "1.9.0", "v1.10.0", "mcp=1", "version_too_old", "1.10.0"},
		{"equal release with metadata pass", "v1.2.3+build.1", "1.2.3", "compare=1", "", ""},
		{"prerelease below release fail", "v1.2.3-rc.1", "1.2.3", "provenance=1", "version_too_old", "1.2.3"},
		{"numeric prerelease pass", "1.2.3-rc.10", "1.2.3-rc.2", "search=1", "", ""},
		{"unknown build fails closed", "abcdef0-dirty", "1.0.0", "search=1", "version_invalid", "abcdef0-dirty"},
	} {
		t.Run(tt.name, func(t *testing.T) {
			Version = tt.version
			var out, errOut bytes.Buffer
			code := Execute([]string{"capabilities", "--min-rforge-version", tt.minimum, "--require", "provenance=1", "--require", tt.requirement, "--require", "brief=1", "--json"}, &out, &errOut)
			var result struct {
				OK    bool `json:"ok"`
				Error struct {
					Code       string `json:"code"`
					Message    string `json:"message"`
					Capability string `json:"capability"`
					Required   string `json:"required"`
					Actual     string `json:"actual"`
					Install    string `json:"install"`
				} `json:"error"`
			}
			if err := json.Unmarshal(out.Bytes(), &result); err != nil {
				t.Fatalf("invalid JSON: %v; output=%s; stderr=%s", err, out.String(), errOut.String())
			}
			if tt.wantCode == "" {
				if code != 0 || !result.OK {
					t.Fatalf("expected compatible engine: exit %d: %s%s", code, out.String(), errOut.String())
				}
			} else {
				if code != 1 || result.OK || result.Error.Code != tt.wantCode {
					t.Fatalf("expected %s: exit %d: %s", tt.wantCode, code, out.String())
				}
				if !strings.Contains(result.Error.Message, tt.gap) || !strings.Contains(result.Error.Message, "install.sh") || result.Error.Install != "install.sh" || result.Error.Required == "" || result.Error.Actual == "" {
					t.Fatalf("error missing gap or repair guidance: %s", out.String())
				}
				if strings.HasPrefix(tt.wantCode, "capability_") && result.Error.Capability != tt.gap {
					t.Fatalf("error missing capability name: %s", out.String())
				}
			}
			if errOut.Len() != 0 {
				t.Fatalf("JSON command wrote stderr: %s", errOut.String())
			}
		})
	}
}

func TestExecuteCapabilitiesRejectsInvalidRequirements(t *testing.T) {
	for _, args := range [][]string{
		{"--bogus"}, {"--require"}, {"--require", "search"}, {"--require", "=1"},
		{"--require", "search=0"}, {"--require", "search=-1"}, {"--require", "search=one"},
		{"--min-rforge-version"}, {"--min-rforge-version", "dev"},
		{"--min-rforge-version", "1.02.3"}, {"--min-rforge-version", "1.2.3-01"},
	} {
		var out, errOut bytes.Buffer
		command := append([]string{"--json", "capabilities"}, args...)
		if code := Execute(command, &out, &errOut); code != 2 {
			t.Fatalf("%v: expected usage exit 2, got %d: %s%s", args, code, out.String(), errOut.String())
		}
		var result struct {
			Error struct{ Code string } `json:"error"`
		}
		if err := json.Unmarshal(out.Bytes(), &result); err != nil || result.Error.Code != "usage" {
			t.Fatalf("%v: expected structured usage error: %s", args, out.String())
		}
	}
}

func TestCapabilitiesInJSONHelp(t *testing.T) {
	var out, errOut bytes.Buffer
	if code := Execute([]string{"help", "--json"}, &out, &errOut); code != 0 {
		t.Fatalf("help: exit %d: %s", code, errOut.String())
	}
	var result struct {
		Data struct {
			Sections []struct {
				Commands []struct{ Usage string } `json:"commands"`
			} `json:"sections"`
		} `json:"data"`
	}
	if err := json.Unmarshal(out.Bytes(), &result); err != nil {
		t.Fatalf("help --json did not return JSON: %v", err)
	}
	for _, section := range result.Data.Sections {
		for _, command := range section.Commands {
			if strings.HasPrefix(command.Usage, "rforge capabilities") {
				return
			}
		}
	}
	t.Fatal("JSON help does not list capabilities")
}
