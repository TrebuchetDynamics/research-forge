package cli

import (
	"encoding/json"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"time"
)

// paperNote is a per-paper annotation stored in <project>/data/paper-notes.json.
type paperNote struct {
	PaperID   string `json:"paperID"`
	Note      string `json:"note"`
	Actor     string `json:"actor,omitempty"`
	Timestamp string `json:"timestamp"`
}

// executeNote manages per-paper annotations (Feynman `alpha annotate` analog).
func executeNote(args []string, stdout, stderr io.Writer, opts globalOptions) int {
	if opts.Project == "" {
		return writeError(stdout, stderr, opts, 2, "missing_project", "--project is required for note commands")
	}
	notesPath := filepath.Join(opts.Project, "data", "paper-notes.json")
	if err := os.MkdirAll(filepath.Dir(notesPath), 0o755); err != nil {
		return writeError(stdout, stderr, opts, 1, "note_mkdir_failed", err.Error())
	}
	if len(args) == 0 {
		return writeError(stdout, stderr, opts, 2, "usage", "usage: rforge note add|list|clear --project <path>")
	}
	switch args[0] {
	case "add":
		paperID, note, actor, ok := parseNoteAdd(args[1:])
		if !ok {
			return writeError(stdout, stderr, opts, 2, "usage", "usage: rforge note add --paper <id> --note <text> [--actor <name>]")
		}
		var notes []paperNote
		if err := readJSONFile(notesPath, &notes); err != nil && !os.IsNotExist(err) {
			return writeError(stdout, stderr, opts, 1, "note_read_failed", err.Error())
		}
		notes = append(notes, paperNote{
			PaperID:   paperID,
			Note:      note,
			Actor:     actor,
			Timestamp: time.Now().UTC().Format(time.RFC3339),
		})
		if err := writeJSONFile(notesPath, notes); err != nil {
			return writeError(stdout, stderr, opts, 1, "note_write_failed", err.Error())
		}
		if opts.JSON {
			return writeJSON(stdout, 0, map[string]any{"action": "note.add", "paper": paperID, "note": note})
		}
		fmt.Fprintf(stdout, "note added for %s\n", paperID)
		return 0

	case "list":
		paperFilter := ""
		for i := 1; i < len(args); i++ {
			if args[i] == "--paper" && i+1 < len(args) {
				paperFilter = args[i+1]
				i++
			}
		}
		var notes []paperNote
		if err := readJSONFile(notesPath, &notes); err != nil && !os.IsNotExist(err) {
			return writeError(stdout, stderr, opts, 1, "note_read_failed", err.Error())
		}
		filtered := make([]paperNote, 0)
		for _, n := range notes {
			if paperFilter == "" || n.PaperID == paperFilter {
				filtered = append(filtered, n)
			}
		}
		if opts.JSON {
			return writeJSON(stdout, 0, map[string]any{"notes": filtered, "count": len(filtered)})
		}
		for _, n := range filtered {
			fmt.Fprintf(stdout, "%s\t%s\n  %s\n", n.PaperID, n.Timestamp, n.Note)
		}
		return 0

	case "clear":
		paperID, ok := parseNoteClear(args[1:])
		if !ok {
			return writeError(stdout, stderr, opts, 2, "usage", "usage: rforge note clear --paper <id>")
		}
		var notes []paperNote
		if err := readJSONFile(notesPath, &notes); err != nil && !os.IsNotExist(err) {
			return writeError(stdout, stderr, opts, 1, "note_read_failed", err.Error())
		}
		kept := make([]paperNote, 0, len(notes))
		removed := 0
		for _, n := range notes {
			if n.PaperID == paperID {
				removed++
				continue
			}
			kept = append(kept, n)
		}
		if err := writeJSONFile(notesPath, kept); err != nil {
			return writeError(stdout, stderr, opts, 1, "note_write_failed", err.Error())
		}
		if opts.JSON {
			return writeJSON(stdout, 0, map[string]any{"action": "note.clear", "paper": paperID, "removed": removed})
		}
		fmt.Fprintf(stdout, "removed %d note(s) for %s\n", removed, paperID)
		return 0
	}
	return writeError(stdout, stderr, opts, 2, "usage", "usage: rforge note add|list|clear --project <path>")
}

func parseNoteAdd(args []string) (string, string, string, bool) {
	paperID, note, actor := "", "", ""
	for i := 0; i < len(args); i++ {
		switch args[i] {
		case "--paper", "--note", "--actor":
			if i+1 >= len(args) {
				return "", "", "", false
			}
			switch args[i] {
			case "--paper":
				paperID = args[i+1]
			case "--note":
				note = args[i+1]
			case "--actor":
				actor = args[i+1]
			}
			i++
		}
	}
	return paperID, note, actor, paperID != "" && note != ""
}

func parseNoteClear(args []string) (string, bool) {
	for i := 0; i < len(args); i++ {
		if args[i] == "--paper" && i+1 < len(args) {
			return args[i+1], true
		}
	}
	return "", false
}

var _ = json.Marshal
var _ = io.Discard
