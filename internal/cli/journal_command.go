package cli

import (
	"bufio"
	"encoding/json"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"
	"time"
)

type journalEntry struct {
	Timestamp string `json:"timestamp"`
	Entry     string `json:"entry"`
}

func executeJournal(args []string, stdout, stderr io.Writer, opts globalOptions) int {
	if len(args) == 0 {
		return writeError(stdout, stderr, opts, 2, "usage", "usage: rforge journal append|read")
	}
	switch args[0] {
	case "append":
		return executeJournalAppend(args[1:], stdout, stderr, opts)
	case "read":
		return executeJournalRead(args[1:], stdout, stderr, opts)
	default:
		return writeError(stdout, stderr, opts, 2, "unknown_journal_subcommand", fmt.Sprintf("unknown journal subcommand %q", args[0]))
	}
}

func executeJournalAppend(args []string, stdout, stderr io.Writer, opts globalOptions) int {
	if opts.Project == "" {
		return writeError(stdout, stderr, opts, 2, "missing_project", "--project is required for journal append")
	}
	entry := ""
	for i := 0; i < len(args)-1; i++ {
		if args[i] == "--entry" {
			entry = args[i+1]
		}
	}
	entry = strings.TrimSpace(entry)
	if entry == "" {
		return writeError(stdout, stderr, opts, 2, "usage", "usage: rforge journal append --entry <text>")
	}
	if err := journalAppend(opts.Project, entry); err != nil {
		return writeError(stdout, stderr, opts, 1, "journal_append_failed", err.Error())
	}
	if opts.JSON {
		return writeJSON(stdout, 0, map[string]any{"action": "journal.append", "entry": entry})
	}
	fmt.Fprintln(stdout, "entry recorded")
	return 0
}

func executeJournalRead(args []string, stdout, stderr io.Writer, opts globalOptions) int {
	if opts.Project == "" {
		return writeError(stdout, stderr, opts, 2, "missing_project", "--project is required for journal read")
	}
	_ = args
	entries, err := journalRead(opts.Project)
	if err != nil {
		return writeError(stdout, stderr, opts, 1, "journal_read_failed", err.Error())
	}
	if opts.JSON {
		return writeJSON(stdout, 0, map[string]any{"entries": entries})
	}
	for _, e := range entries {
		fmt.Fprintf(stdout, "[%s] %s\n", e.Timestamp, e.Entry)
	}
	return 0
}

func journalAppend(projectPath, entry string) error {
	dir := filepath.Join(projectPath, "journal")
	if err := os.MkdirAll(dir, 0o755); err != nil {
		return err
	}
	e := journalEntry{
		Timestamp: time.Now().UTC().Format(time.RFC3339),
		Entry:     entry,
	}
	line, err := json.Marshal(e)
	if err != nil {
		return err
	}
	f, err := os.OpenFile(filepath.Join(dir, "entries.jsonl"), os.O_CREATE|os.O_APPEND|os.O_WRONLY, 0o644)
	if err != nil {
		return err
	}
	defer f.Close()
	_, err = fmt.Fprintf(f, "%s\n", line)
	return err
}

func journalRead(projectPath string) ([]journalEntry, error) {
	path := filepath.Join(projectPath, "journal", "entries.jsonl")
	f, err := os.Open(path)
	if os.IsNotExist(err) {
		return []journalEntry{}, nil
	}
	if err != nil {
		return nil, err
	}
	defer f.Close()
	var entries []journalEntry
	scanner := bufio.NewScanner(f)
	for scanner.Scan() {
		line := strings.TrimSpace(scanner.Text())
		if line == "" {
			continue
		}
		var e journalEntry
		if json.Unmarshal([]byte(line), &e) == nil {
			entries = append(entries, e)
		}
	}
	return entries, scanner.Err()
}
