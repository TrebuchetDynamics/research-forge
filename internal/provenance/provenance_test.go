package provenance

import (
	"path/filepath"
	"testing"
)

func TestAppendAndReadEvents(t *testing.T) {
	projectPath := t.TempDir()
	event := Event{
		SchemaVersion: "1",
		ID:            "evt_test",
		Timestamp:     "2026-06-08T00:00:00Z",
		Actor:         "rforge",
		Action:        "project.create",
		Target:        projectPath,
		Inputs:        map[string]any{"title": "Demo"},
		Outputs:       map[string]any{"manifest": "rforge.project.toml"},
		Warnings:      []string{},
	}

	if err := Append(projectPath, event); err != nil {
		t.Fatalf("Append returned error: %v", err)
	}
	events, err := Read(projectPath)
	if err != nil {
		t.Fatalf("Read returned error: %v", err)
	}
	if len(events) != 1 {
		t.Fatalf("len(events) = %d, want 1", len(events))
	}
	if events[0].ID != "evt_test" || events[0].Action != "project.create" {
		t.Fatalf("event = %#v", events[0])
	}
}

func TestAppendCreatesProvenanceDirectory(t *testing.T) {
	projectPath := filepath.Join(t.TempDir(), "demo")

	if err := Append(projectPath, Event{SchemaVersion: "1", ID: "evt_test", Action: "project.create"}); err != nil {
		t.Fatalf("Append returned error: %v", err)
	}
	events, err := Read(projectPath)
	if err != nil {
		t.Fatalf("Read returned error: %v", err)
	}
	if len(events) != 1 {
		t.Fatalf("len(events) = %d, want 1", len(events))
	}
}

func TestNoteAppendsResearcherAnnotation(t *testing.T) {
	projectPath := t.TempDir()

	if err := Note(projectPath, "promoted 99 elementary CA rules", "alice"); err != nil {
		t.Fatalf("Note returned error: %v", err)
	}

	events, err := Read(projectPath)
	if err != nil {
		t.Fatalf("Read returned error: %v", err)
	}
	if len(events) != 1 {
		t.Fatalf("len(events) = %d, want 1", len(events))
	}
	ev := events[0]
	if ev.Action != "provenance.researcher.note" {
		t.Errorf("Action = %q, want %q", ev.Action, "provenance.researcher.note")
	}
	if ev.Actor != "alice" {
		t.Errorf("Actor = %q, want %q", ev.Actor, "alice")
	}
	if msg, _ := ev.Inputs["message"].(string); msg != "promoted 99 elementary CA rules" {
		t.Errorf("Inputs[message] = %q, want %q", msg, "promoted 99 elementary CA rules")
	}
	if ev.ID == "" {
		t.Error("ID must not be empty")
	}
	if ev.Timestamp == "" {
		t.Error("Timestamp must not be empty")
	}
}

func TestNoteRejectsEmptyMessage(t *testing.T) {
	if err := Note(t.TempDir(), "", "alice"); err == nil {
		t.Fatal("Note with empty message should return an error")
	}
}

func TestNoteDefaultsActorToRforge(t *testing.T) {
	projectPath := t.TempDir()
	if err := Note(projectPath, "some observation", ""); err != nil {
		t.Fatalf("Note returned error: %v", err)
	}
	events, err := Read(projectPath)
	if err != nil {
		t.Fatalf("Read returned error: %v", err)
	}
	if events[0].Actor != "rforge" {
		t.Errorf("Actor = %q, want %q", events[0].Actor, "rforge")
	}
}

func TestLastOutputEquals(t *testing.T) {
	projectPath := t.TempDir()
	first := []map[string]any{{"path": "old.pdf", "kind": "pdf", "imported": false}}
	latest := []map[string]any{{"path": "paper.pdf", "kind": "pdf", "imported": false}}
	for _, outputs := range []map[string]any{
		{"assets": first},
		{"assets": latest},
	} {
		if err := Append(projectPath, Event{SchemaVersion: "1", ID: "evt", Action: "project.assets.discover", Outputs: outputs}); err != nil {
			t.Fatalf("Append returned error: %v", err)
		}
	}

	matches, err := LastOutputEquals(projectPath, "project.assets.discover", "assets", latest)
	if err != nil {
		t.Fatalf("LastOutputEquals returned error: %v", err)
	}
	if !matches {
		t.Fatalf("LastOutputEquals = false, want true for latest matching output")
	}
	matches, err = LastOutputEquals(projectPath, "project.assets.discover", "assets", first)
	if err != nil {
		t.Fatalf("LastOutputEquals returned error: %v", err)
	}
	if matches {
		t.Fatalf("LastOutputEquals = true, want false for stale output")
	}
}
