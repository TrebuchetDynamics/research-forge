package provenance

import (
	"bytes"
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"
)

func validTestEvent() Event {
	return Event{SchemaVersion: "1", ID: "evt_test", Timestamp: "2026-06-08T00:00:00Z", Actor: "rforge", Action: "test.action", Target: "test-target", Inputs: map[string]any{}, Outputs: map[string]any{}, Warnings: []string{}}
}

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

func TestReadDoesNotReadThroughSymlinkedLedger(t *testing.T) {
	projectPath := t.TempDir()
	provenanceDir := filepath.Join(projectPath, "provenance")
	if err := os.MkdirAll(provenanceDir, 0o755); err != nil {
		t.Fatalf("create provenance directory: %v", err)
	}
	externalEvent := validTestEvent()
	externalEvent.ID = "evt_external_private"
	externalBytes, err := json.Marshal(externalEvent)
	if err != nil {
		t.Fatalf("marshal external event: %v", err)
	}
	outsidePath := filepath.Join(t.TempDir(), "outside-events.jsonl")
	if err := os.WriteFile(outsidePath, append(externalBytes, '\n'), 0o640); err != nil {
		t.Fatalf("write outside provenance ledger: %v", err)
	}
	ledgerPath := filepath.Join(provenanceDir, "events.jsonl")
	if err := os.Symlink(outsidePath, ledgerPath); err != nil {
		t.Skipf("symlinks are unavailable: %v", err)
	}

	if events, err := Read(projectPath); err == nil {
		t.Fatalf("Read returned external events through a symlink: %#v", events)
	}
	info, err := os.Lstat(ledgerPath)
	if err != nil {
		t.Fatalf("lstat provenance ledger: %v", err)
	}
	if info.Mode()&os.ModeSymlink == 0 {
		t.Fatalf("Read replaced the symlink: mode=%v", info.Mode())
	}
}

func TestAppendDoesNotWriteThroughHardLinkedLedger(t *testing.T) {
	projectPath := t.TempDir()
	provenanceDir := filepath.Join(projectPath, "provenance")
	if err := os.MkdirAll(provenanceDir, 0o755); err != nil {
		t.Fatalf("create provenance directory: %v", err)
	}
	previous := validTestEvent()
	previous.ID = "evt_previous"
	previousBytes, err := json.Marshal(previous)
	if err != nil {
		t.Fatalf("marshal previous event: %v", err)
	}
	previousBytes = append(previousBytes, '\n')
	outsidePath := filepath.Join(t.TempDir(), "outside-events.jsonl")
	if err := os.WriteFile(outsidePath, previousBytes, 0o640); err != nil {
		t.Fatalf("write outside provenance ledger: %v", err)
	}
	ledgerPath := filepath.Join(provenanceDir, "events.jsonl")
	if err := os.Link(outsidePath, ledgerPath); err != nil {
		t.Skipf("hard links are unavailable: %v", err)
	}
	fixedTime := time.Unix(1_600_000_000, 0)
	if err := os.Chtimes(outsidePath, fixedTime, fixedTime); err != nil {
		t.Fatalf("set outside provenance timestamps: %v", err)
	}
	before, err := os.Stat(outsidePath)
	if err != nil {
		t.Fatalf("stat outside provenance ledger before Append: %v", err)
	}

	next := validTestEvent()
	next.ID = "evt_next"
	if err := Append(projectPath, next); err != nil {
		t.Fatalf("Append returned error: %v", err)
	}

	outsideBytes, err := os.ReadFile(outsidePath)
	if err != nil {
		t.Fatalf("read outside provenance ledger: %v", err)
	}
	if !bytes.Equal(outsideBytes, previousBytes) {
		t.Fatalf("outside provenance ledger changed: got %q, want %q", outsideBytes, previousBytes)
	}
	after, err := os.Stat(outsidePath)
	if err != nil {
		t.Fatalf("stat outside provenance ledger after Append: %v", err)
	}
	if !after.ModTime().Equal(before.ModTime()) {
		t.Fatalf("outside provenance ledger mtime changed: got %s, want %s", after.ModTime(), before.ModTime())
	}
	events, err := Read(projectPath)
	if err != nil {
		t.Fatalf("Read returned error: %v", err)
	}
	if len(events) != 2 || events[0].ID != "evt_previous" || events[1].ID != "evt_next" {
		t.Fatalf("events = %#v, want previous then next", events)
	}
}

func TestAppendAndReadLargeEvent(t *testing.T) {
	projectPath := t.TempDir()
	largeValue := strings.Repeat("x", 128*1024)
	event := validTestEvent()
	event.Inputs["metadata"] = largeValue

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
	if got := events[0].Inputs["metadata"]; got != largeValue {
		t.Fatalf("metadata length = %d, want %d", len(got.(string)), len(largeValue))
	}
}

func TestAppendRejectsInvalidEventsBeforeCreatingProvenanceDirectory(t *testing.T) {
	tests := []struct {
		name   string
		want   string
		mutate func(*Event)
	}{
		{name: "unsupported schema", want: "schema version", mutate: func(event *Event) { event.SchemaVersion = "999" }},
		{name: "blank ID", want: "ID is required", mutate: func(event *Event) { event.ID = " \t" }},
		{name: "invalid timestamp", want: "timestamp", mutate: func(event *Event) { event.Timestamp = "not-a-time" }},
		{name: "blank actor", want: "actor is required", mutate: func(event *Event) { event.Actor = " \t" }},
		{name: "blank action", want: "action is required", mutate: func(event *Event) { event.Action = " \t" }},
		{name: "blank target", want: "target is required", mutate: func(event *Event) { event.Target = " \t" }},
		{name: "nil inputs", want: "inputs are required", mutate: func(event *Event) { event.Inputs = nil }},
		{name: "nil outputs", want: "outputs are required", mutate: func(event *Event) { event.Outputs = nil }},
		{name: "nil warnings", want: "warnings are required", mutate: func(event *Event) { event.Warnings = nil }},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			projectPath := filepath.Join(t.TempDir(), "project")
			event := validTestEvent()
			tt.mutate(&event)
			if err := Append(projectPath, event); err == nil || !strings.Contains(err.Error(), tt.want) {
				t.Errorf("Append error = %v, want %q", err, tt.want)
			}
			provenancePath := filepath.Join(projectPath, "provenance")
			if _, err := os.Stat(provenancePath); !os.IsNotExist(err) {
				t.Errorf("invalid event created provenance directory %s: %v", provenancePath, err)
			}
		})
	}
}

func TestAppendCreatesProvenanceDirectory(t *testing.T) {
	projectPath := filepath.Join(t.TempDir(), "demo")
	event := validTestEvent()
	event.Action = "project.create"
	event.Target = projectPath

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
	if ev.Target != projectPath {
		t.Errorf("Target = %q, want %q", ev.Target, projectPath)
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
		event := validTestEvent()
		event.Action = "project.assets.discover"
		event.Target = projectPath
		event.Outputs = outputs
		if err := Append(projectPath, event); err != nil {
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
