package cli

import (
	"bytes"
	"encoding/json"
	"path/filepath"
	"testing"
)

// `rforge help` states that all commands accept --json. It only worked before
// the command name: trailing --json was printed as text or rejected as an
// unknown flag, so scripts following the documented contract broke.
func TestTrailingJSONFlagIsHonored(t *testing.T) {
	missing := filepath.Join(t.TempDir(), "missing")
	cases := [][]string{
		{"version", "--json"},
		{"search", "stats", "--dir", missing, "--json"},
		{"screen", "progress", "--dir", missing, "--json"},
		{"jobs", "list", "--dir", missing, "--json"},
	}
	for _, args := range cases {
		t.Run(args[0], func(t *testing.T) {
			stdout, stderr := new(bytes.Buffer), new(bytes.Buffer)
			code := Execute(args, stdout, stderr)
			if code != 0 {
				t.Fatalf("exit code = %d; stdout = %s stderr = %s", code, stdout.String(), stderr.String())
			}
			var envelope map[string]any
			if err := json.Unmarshal(stdout.Bytes(), &envelope); err != nil {
				t.Fatalf("trailing --json did not produce JSON: %v\n%s", err, stdout.String())
			}
			if ok, _ := envelope["ok"].(bool); !ok {
				t.Errorf("envelope ok = %v, want true: %s", envelope["ok"], stdout.String())
			}
		})
	}
}
