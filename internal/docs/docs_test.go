package docs

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestCLIReferenceDocumentsDecisionAuditModes(t *testing.T) {
	root := filepath.Join("..", "..")
	data, err := os.ReadFile(filepath.Join(root, "docs", "cli.md"))
	if err != nil {
		t.Fatalf("read CLI reference: %v", err)
	}
	text := string(data)
	for _, want := range []string{"rforge decisions --check TODO.md", "rforge decisions --markdown", "rforge decisions --issue-body <decision-id>"} {
		if !strings.Contains(text, want) {
			t.Fatalf("CLI reference missing %q", want)
		}
	}
}

func TestUncheckedTodosPointToDecisionCommands(t *testing.T) {
	root := filepath.Join("..", "..")
	data, err := os.ReadFile(filepath.Join(root, "TODO.md"))
	if err != nil {
		t.Fatalf("read TODO.md: %v", err)
	}
	for _, line := range strings.Split(string(data), "\n") {
		line = strings.TrimSpace(line)
		if !strings.HasPrefix(line, "- [ ] ") {
			continue
		}
		if !strings.Contains(line, "rforge decisions") && !strings.Contains(line, "Fyne desktop build decision") {
			t.Fatalf("unchecked TODO does not point to decision command or Fyne decision: %s", line)
		}
	}
}

func TestRemainingTodoAuditDocumentsExecutableAuditCommand(t *testing.T) {
	root := filepath.Join("..", "..")
	auditData, err := os.ReadFile(filepath.Join(root, "docs", "remaining-todo-audit.md"))
	if err != nil {
		t.Fatalf("read remaining audit: %v", err)
	}
	audit := string(auditData)
	for _, want := range []string{"make todo-audit", "verify decision line references", "decision-resolution-checklist.md", "go test ./...", "go vet ./...", "git diff --check"} {
		if !strings.Contains(audit, want) {
			t.Fatalf("remaining TODO audit missing %q", want)
		}
	}
}

func TestRemainingTodoAuditCoversUncheckedTodos(t *testing.T) {
	root := filepath.Join("..", "..")
	todoData, err := os.ReadFile(filepath.Join(root, "TODO.md"))
	if err != nil {
		t.Fatalf("read TODO.md: %v", err)
	}
	auditData, err := os.ReadFile(filepath.Join(root, "docs", "remaining-todo-audit.md"))
	if err != nil {
		t.Fatalf("read remaining audit: %v", err)
	}
	audit := string(auditData)
	for _, line := range strings.Split(string(todoData), "\n") {
		line = strings.TrimSpace(line)
		if !strings.HasPrefix(line, "- [ ] ") {
			continue
		}
		item := strings.TrimPrefix(line, "- [ ] ")
		if idx := strings.Index(item, " _("); idx >= 0 {
			item = strings.TrimSpace(item[:idx])
		}
		item = strings.TrimSuffix(item, ".")
		if !strings.Contains(audit, item) {
			t.Fatalf("remaining TODO %q not covered by docs/remaining-todo-audit.md", item)
		}
	}
}
