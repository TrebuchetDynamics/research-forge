package cli

import (
	"bytes"
	"os"
	"path/filepath"
	"testing"
)

func TestBenchmarkReceiptFailurePreservesPreviousOutput(t *testing.T) {
	for _, command := range [][]string{{"benchmark", "cross-tool"}, {"retrieve", "benchmark"}} {
		t.Run(command[0], func(t *testing.T) {
			root := t.TempDir()
			out := filepath.Join(root, "benchmark.json")
			if err := os.WriteFile(out, []byte("previous"), 0600); err != nil {
				t.Fatal(err)
			}
			if err := os.MkdirAll(filepath.Join(root, "provenance", "events.jsonl"), 0755); err != nil {
				t.Fatal(err)
			}
			var stdout, stderr bytes.Buffer
			args := append([]string{"--json"}, command...)
			args = append(args, "--out", out)
			if code := Execute(args, &stdout, &stderr); code == 0 {
				t.Fatal("blocked ledger succeeded")
			}
			data, err := os.ReadFile(out)
			if err != nil || string(data) != "previous" {
				t.Fatalf("failed run replaced output: %s (%v)", data, err)
			}
		})
	}
}
