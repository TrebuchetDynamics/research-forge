package provenance

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestRunPublicationRequiresReadableLedger(t *testing.T) {
	for _, tc := range []struct {
		name      string
		malformed bool
	}{{"missing final newline", false}, {"malformed ledger", true}} {
		t.Run(tc.name, func(t *testing.T) {
			root := t.TempDir()
			if err := Note(root, "observation", "reviewer"); err != nil {
				t.Fatal(err)
			}
			ledger := filepath.Join(root, EventsRelativePath)
			data, err := os.ReadFile(ledger)
			if err != nil {
				t.Fatal(err)
			}
			data = []byte(strings.TrimSuffix(string(data), "\n"))
			if tc.malformed {
				data = []byte("{truncated")
			}
			if err := os.WriteFile(ledger, data, 0600); err != nil {
				t.Fatal(err)
			}
			output := filepath.Join(root, "result.md")
			if err := os.WriteFile(output, []byte("result"), 0600); err != nil {
				t.Fatal(err)
			}
			run, err := WriteRun(root, "test.run", map[string]any{"question": "test", "depth": "standard", "sources": []string{}, "errors": []string{}, "rforge_version": map[string]string{"version": "dev", "commit": "test", "date": "test"}}, []string{output})
			if tc.malformed {
				if err == nil {
					t.Fatal("engine run published over unreadable ledger")
				}
				after, _ := os.ReadFile(ledger)
				if string(after) != string(data) {
					t.Fatal("failed run changed ledger")
				}
			} else {
				if err != nil {
					t.Fatal(err)
				}
				result, err := Validate(filepath.Join(root, "provenance", "runs", run.RunID+".json"))
				if err != nil || result.Level != EngineVerified {
					t.Fatalf("published run not verifiable: %+v (%v)", result, err)
				}
			}
		})
	}
}
