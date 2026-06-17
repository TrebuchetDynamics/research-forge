package cli

import (
	"bytes"
	"encoding/json"
	"strings"
	"testing"
)

func TestExecuteForgeGuidedWorkflowE2E(t *testing.T) {
	projectPath := t.TempDir()
	var stdout, stderr bytes.Buffer
	code := Execute([]string{"--json", "forge", "init", "--project", projectPath, "--question", "Do catalysts improve hydrogen evolution?", "--sources", "openalex,semantic-scholar", "--tools", "grobid,qdrant"}, &stdout, &stderr)
	if code != 0 {
		t.Fatalf("init code=%d stderr=%s stdout=%s", code, stderr.String(), stdout.String())
	}
	var env struct {
		OK   bool `json:"ok"`
		Data struct {
			State struct {
				CurrentState       string `json:"currentState"`
				BlockedReviewGates []struct {
					Gate string `json:"gate"`
				} `json:"blockedReviewGates"`
				SourceChoices       []string `json:"sourceChoices"`
				ToolChoices         []string `json:"toolChoices"`
				PrivacyLegalPreview []string `json:"privacyLegalPreview"`
			} `json:"state"`
		} `json:"data"`
	}
	if err := json.Unmarshal(stdout.Bytes(), &env); err != nil {
		t.Fatalf("json: %v\n%s", err, stdout.String())
	}
	if !env.OK || env.Data.State.CurrentState != "question_draft" || len(env.Data.State.BlockedReviewGates) == 0 || len(env.Data.State.SourceChoices) != 2 || len(env.Data.State.ToolChoices) != 2 || len(env.Data.State.PrivacyLegalPreview) == 0 {
		t.Fatalf("unexpected init: %#v", env)
	}

	stdout.Reset()
	stderr.Reset()
	code = Execute([]string{"--json", "forge", "next", "--project", projectPath}, &stdout, &stderr)
	if code == 0 || !strings.Contains(stdout.String(), "blocked review gate") {
		t.Fatalf("next should be blocked code=%d out=%s err=%s", code, stdout.String(), stderr.String())
	}

	for _, gate := range []string{"question approval", "protocol approval"} {
		stdout.Reset()
		stderr.Reset()
		code = Execute([]string{"--json", "forge", "approve", "--project", projectPath, "--gate", gate, "--note", "approved by reviewer"}, &stdout, &stderr)
		if code != 0 {
			t.Fatalf("approve %s code=%d stderr=%s stdout=%s", gate, code, stderr.String(), stdout.String())
		}
	}
	stdout.Reset()
	stderr.Reset()
	code = Execute([]string{"--json", "forge", "status", "--project", projectPath}, &stdout, &stderr)
	if code != 0 {
		t.Fatalf("status code=%d stderr=%s stdout=%s", code, stderr.String(), stdout.String())
	}
	if !strings.Contains(stdout.String(), "source_plan") || !strings.Contains(stdout.String(), "network/API approval") || !strings.Contains(stdout.String(), "rforge protocol") {
		t.Fatalf("status missing gates/actions: %s", stdout.String())
	}
}

func TestExecuteForgeRunDAGWritesCheckpoints(t *testing.T) {
	projectPath := t.TempDir()
	var stdout, stderr bytes.Buffer
	code := Execute([]string{"--json", "forge", "run-dag", "--project", projectPath, "--question", "Q?", "--max-steps", "2"}, &stdout, &stderr)
	if code != 0 {
		t.Fatalf("run-dag code=%d stderr=%s stdout=%s", code, stderr.String(), stdout.String())
	}
	if !strings.Contains(stdout.String(), "checkpoints") || !strings.Contains(stdout.String(), "discovery") || !strings.Contains(stdout.String(), "import") {
		t.Fatalf("run-dag output: %s", stdout.String())
	}
}

func TestExecuteForgeReopenRecordsReviewerReason(t *testing.T) {
	projectPath := t.TempDir()
	var stdout, stderr bytes.Buffer
	if code := Execute([]string{"forge", "init", "--project", projectPath, "--question", "Q?"}, &stdout, &stderr); code != 0 {
		t.Fatalf("init: %s", stderr.String())
	}
	stdout.Reset()
	stderr.Reset()
	code := Execute([]string{"--json", "forge", "reopen", "--project", projectPath, "--state", "question_draft", "--reason", "criteria changed"}, &stdout, &stderr)
	if code != 0 {
		t.Fatalf("reopen code=%d stderr=%s stdout=%s", code, stderr.String(), stdout.String())
	}
	if !strings.Contains(stdout.String(), "criteria changed") || !strings.Contains(stdout.String(), "question_draft") {
		t.Fatalf("reopen output: %s", stdout.String())
	}
}
