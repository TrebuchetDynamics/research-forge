package cli

import (
	"bytes"
	"encoding/json"
	"path/filepath"
	"testing"

	"github.com/TrebuchetDynamics/research-forge/internal/screening"
)

func TestExecuteScreenDecidePersistsCanonicalTextFields(t *testing.T) {
	project := filepath.Join(t.TempDir(), "review")
	if code := Execute([]string{"project", "create", project, "--title", "Review"}, new(bytes.Buffer), new(bytes.Buffer)); code != 0 {
		t.Fatalf("project create exit code = %d", code)
	}
	if code := Execute([]string{"--project", project, "screen", "configure", "--reason", "off-topic"}, new(bytes.Buffer), new(bytes.Buffer)); code != 0 {
		t.Fatalf("screen configure exit code = %d", code)
	}
	stderr := new(bytes.Buffer)
	if code := Execute([]string{
		"--project", project, "screen", "decide",
		"--paper", "  paper-1  ",
		"--stage", "title_abstract",
		"--decision", "exclude",
		"--reason", "  off-topic  ",
		"--reviewer", "  ada  ",
	}, new(bytes.Buffer), stderr); code != 0 {
		t.Fatalf("screen decide exit code = %d, stderr = %s", code, stderr.String())
	}

	var events []screening.DecisionEvent
	if err := readJSONFile(screenEventsPath(project), &events); err != nil {
		t.Fatalf("read screening events: %v", err)
	}
	if len(events) != 1 {
		t.Fatalf("screening events length = %d, want 1: %#v", len(events), events)
	}
	if got := events[0]; got.PaperID != "paper-1" || got.Reason != "off-topic" || got.Reviewer != "ada" {
		t.Fatalf("persisted decision was not canonicalized: %#v", got)
	}
}

func TestExecuteScreenProgressCanonicalizesLegacyDecisionHistory(t *testing.T) {
	project := filepath.Join(t.TempDir(), "review")
	if code := Execute([]string{"project", "create", project, "--title", "Review"}, new(bytes.Buffer), new(bytes.Buffer)); code != 0 {
		t.Fatalf("project create exit code = %d", code)
	}
	if code := Execute([]string{"--project", project, "screen", "configure", "--reason", "off-topic"}, new(bytes.Buffer), new(bytes.Buffer)); code != 0 {
		t.Fatalf("screen configure exit code = %d", code)
	}
	if err := writeJSONFile(screenEventsPath(project), []screening.DecisionEvent{
		{PaperID: "paper-1", Stage: screening.StageTitleAbstract, Decision: screening.DecisionInclude, Reviewer: "ada"},
		{PaperID: "  paper-1  ", Stage: screening.StageTitleAbstract, Decision: screening.DecisionUncertain, Reviewer: "  ada  "},
	}); err != nil {
		t.Fatalf("write legacy screening events: %v", err)
	}

	stdout := new(bytes.Buffer)
	stderr := new(bytes.Buffer)
	if code := Execute([]string{"--json", "--project", project, "screen", "progress", "--stage", "title_abstract"}, stdout, stderr); code != 0 {
		t.Fatalf("screen progress exit code = %d, stderr = %s", code, stderr.String())
	}
	var response struct {
		Data struct {
			Progress screening.ProgressReport `json:"progress"`
		} `json:"data"`
	}
	if err := json.Unmarshal(stdout.Bytes(), &response); err != nil {
		t.Fatalf("decode screen progress: %v: %s", err, stdout.String())
	}
	progress := response.Data.Progress
	if progress.ScreenedRecords != 1 || len(progress.Reviewers) != 1 || progress.Reviewers[0].Reviewer != "ada" || progress.Reviewers[0].Decisions != 2 {
		t.Fatalf("legacy decision history was not canonicalized: %#v", progress)
	}
}
