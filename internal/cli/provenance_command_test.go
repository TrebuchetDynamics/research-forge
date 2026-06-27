package cli

import (
	"bytes"
	"encoding/json"
	"strings"
	"testing"

	"github.com/TrebuchetDynamics/research-forge/internal/provenance"
)

func TestProvenanceNoteAppendsEvent(t *testing.T) {
	projectPath := t.TempDir()

	stdout := new(bytes.Buffer)
	stderr := new(bytes.Buffer)
	code := Execute([]string{"--project", projectPath, "provenance", "note", "--message", "promoted 99 CA rules"}, stdout, stderr)
	if code != 0 {
		t.Fatalf("exit code = %d, stderr = %s", code, stderr.String())
	}
	if !strings.Contains(stdout.String(), "note recorded") {
		t.Errorf("stdout = %q, want to contain %q", stdout.String(), "note recorded")
	}

	events, err := provenance.Read(projectPath)
	if err != nil {
		t.Fatalf("Read events: %v", err)
	}
	var found bool
	for _, ev := range events {
		if ev.Action == "provenance.researcher.note" {
			if msg, _ := ev.Inputs["message"].(string); msg == "promoted 99 CA rules" {
				found = true
			}
		}
	}
	if !found {
		t.Fatalf("no researcher note event found in provenance; events = %#v", events)
	}
}

func TestProvenanceNoteJSONOutput(t *testing.T) {
	projectPath := t.TempDir()

	stdout := new(bytes.Buffer)
	code := Execute([]string{"--project", projectPath, "--json", "provenance", "note", "--message", "test observation"}, stdout, new(bytes.Buffer))
	if code != 0 {
		t.Fatalf("exit code = %d", code)
	}
	var envelope map[string]any
	if err := json.Unmarshal(stdout.Bytes(), &envelope); err != nil {
		t.Fatalf("stdout is not JSON: %v\n%s", err, stdout.String())
	}
	data, _ := envelope["data"].(map[string]any)
	if data["action"] != "provenance.researcher.note" {
		t.Errorf("data.action = %v, want %q", data["action"], "provenance.researcher.note")
	}
}

func TestProvenanceNoteWithActor(t *testing.T) {
	projectPath := t.TempDir()

	code := Execute([]string{"--project", projectPath, "provenance", "note", "--message", "my note", "--actor", "alice"}, new(bytes.Buffer), new(bytes.Buffer))
	if code != 0 {
		t.Fatalf("exit code = %d", code)
	}
	events, err := provenance.Read(projectPath)
	if err != nil {
		t.Fatalf("Read events: %v", err)
	}
	for _, ev := range events {
		if ev.Action == "provenance.researcher.note" && ev.Actor == "alice" {
			return
		}
	}
	t.Fatal("no note event with actor=alice found")
}

func TestProvenanceNoteRequiresMessage(t *testing.T) {
	projectPath := t.TempDir()

	stderr := new(bytes.Buffer)
	code := Execute([]string{"--project", projectPath, "provenance", "note"}, new(bytes.Buffer), stderr)
	if code != 2 {
		t.Errorf("exit code = %d, want 2", code)
	}
}

func TestProvenanceNoteRequiresProject(t *testing.T) {
	stderr := new(bytes.Buffer)
	code := Execute([]string{"provenance", "note", "--message", "hello"}, new(bytes.Buffer), stderr)
	if code != 2 {
		t.Errorf("exit code = %d, want 2", code)
	}
}

func TestProvenanceNoteEmptyMessageError(t *testing.T) {
	projectPath := t.TempDir()

	stderr := new(bytes.Buffer)
	code := Execute([]string{"--project", projectPath, "provenance", "note", "--message", "   "}, new(bytes.Buffer), stderr)
	if code == 0 {
		t.Error("exit code = 0, want non-zero for blank message")
	}
}
