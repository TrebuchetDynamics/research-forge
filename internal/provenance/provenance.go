package provenance

import (
	"bufio"
	"encoding/json"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"reflect"
	"strings"
	"time"

	"github.com/TrebuchetDynamics/research-forge/internal/filetxn"
)

const eventsRelativePath = "provenance/events.jsonl"

// Event is one append-only Provenance record in a ResearchForge project.
type Event struct {
	SchemaVersion string         `json:"schemaVersion"`
	ID            string         `json:"id"`
	Timestamp     string         `json:"timestamp"`
	Actor         string         `json:"actor"`
	Action        string         `json:"action"`
	Target        string         `json:"target"`
	Inputs        map[string]any `json:"inputs"`
	Outputs       map[string]any `json:"outputs"`
	Warnings      []string       `json:"warnings"`
}

// Note appends a researcher annotation event to the project provenance log.
// It is the CLI-safe path for recording human observations without a full search run.
func Note(projectPath, message, actorName string) error {
	message = strings.TrimSpace(message)
	if message == "" {
		return fmt.Errorf("note message must not be empty")
	}
	if strings.TrimSpace(actorName) == "" {
		actorName = "rforge"
	}
	now := time.Now().UTC()
	return Append(projectPath, Event{
		SchemaVersion: "1",
		ID:            "evt_" + now.Format("20060102T150405Z") + "_note",
		Timestamp:     now.Format(time.RFC3339),
		Actor:         actorName,
		Action:        "provenance.researcher.note",
		Target:        projectPath,
		Inputs:        map[string]any{"message": message},
		Outputs:       map[string]any{},
		Warnings:      []string{},
	})
}

// Append records one Provenance event in the project JSONL ledger.
func Append(projectPath string, event Event) error {
	if err := validateEvent(event); err != nil {
		return err
	}
	eventBytes, err := json.Marshal(event)
	if err != nil {
		return err
	}
	eventBytes = append(eventBytes, '\n')
	if err := os.MkdirAll(filepath.Join(projectPath, "provenance"), 0o755); err != nil {
		return err
	}
	return filetxn.Append(filepath.Join(projectPath, eventsRelativePath), eventBytes, 0o644)
}

func validateEvent(event Event) error {
	if event.SchemaVersion != "1" {
		return fmt.Errorf("unsupported provenance schema version %q", event.SchemaVersion)
	}
	for _, field := range []struct {
		name  string
		value string
	}{
		{name: "ID", value: event.ID},
		{name: "actor", value: event.Actor},
		{name: "action", value: event.Action},
		{name: "target", value: event.Target},
	} {
		if strings.TrimSpace(field.value) == "" {
			return fmt.Errorf("provenance event %s is required", field.name)
		}
	}
	if _, err := time.Parse(time.RFC3339, event.Timestamp); err != nil {
		return fmt.Errorf("invalid provenance event timestamp %q: %w", event.Timestamp, err)
	}
	if event.Inputs == nil {
		return fmt.Errorf("provenance event inputs are required")
	}
	if event.Outputs == nil {
		return fmt.Errorf("provenance event outputs are required")
	}
	if event.Warnings == nil {
		return fmt.Errorf("provenance event warnings are required")
	}
	return nil
}

// Read returns all Provenance events from the project JSONL ledger.
func Read(projectPath string) ([]Event, error) {
	file, err := filetxn.OpenRegular(filepath.Join(projectPath, eventsRelativePath))
	if err != nil {
		return nil, err
	}
	defer file.Close()

	events := []Event{}
	reader := bufio.NewReader(file)
	for {
		text, readErr := reader.ReadString('\n')
		line := strings.TrimSpace(text)
		if line == "" {
			if readErr == io.EOF {
				break
			}
			if readErr != nil {
				return nil, readErr
			}
			continue
		}
		var event Event
		if err := json.Unmarshal([]byte(line), &event); err != nil {
			return nil, err
		}
		events = append(events, event)
		if readErr == io.EOF {
			break
		}
		if readErr != nil {
			return nil, readErr
		}
	}
	return events, nil
}

// LastOutputEquals reports whether the latest event with action has output key equal to expected.
func LastOutputEquals(projectPath, action, key string, expected any) (bool, error) {
	events, err := Read(projectPath)
	if err != nil {
		return false, err
	}
	expectedNormalized, err := normalizeJSONValue(expected)
	if err != nil {
		return false, err
	}
	for i := len(events) - 1; i >= 0; i-- {
		event := events[i]
		if event.Action != action {
			continue
		}
		actual, ok := event.Outputs[key]
		if !ok {
			return false, nil
		}
		actualNormalized, err := normalizeJSONValue(actual)
		if err != nil {
			return false, err
		}
		return reflect.DeepEqual(actualNormalized, expectedNormalized), nil
	}
	return false, nil
}

func normalizeJSONValue(value any) (any, error) {
	data, err := json.Marshal(value)
	if err != nil {
		return nil, err
	}
	var normalized any
	if err := json.Unmarshal(data, &normalized); err != nil {
		return nil, err
	}
	return normalized, nil
}
