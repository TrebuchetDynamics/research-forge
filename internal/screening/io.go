package screening

import (
	"bytes"
	"encoding/csv"
	"fmt"
	"os"
	"path/filepath"
	"sort"

	"github.com/TrebuchetDynamics/research-forge/internal/filetxn"
)

var screeningCSVHeader = []string{"paper_id", "stage", "decision", "reason", "reviewer"}

func ExportCSV(path string, events []DecisionEvent) error {
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		return err
	}
	var buffer bytes.Buffer
	writer := csv.NewWriter(&buffer)
	if err := writer.Write(screeningCSVHeader); err != nil {
		return err
	}
	for _, event := range events {
		if err := writer.Write([]string{event.PaperID, string(event.Stage), string(event.Decision), event.Reason, event.Reviewer}); err != nil {
			return err
		}
	}
	writer.Flush()
	if err := writer.Error(); err != nil {
		return err
	}
	return filetxn.Replace(path, buffer.Bytes(), 0o644)
}

func ImportCSV(path string) ([]DecisionEvent, error) {
	file, err := os.Open(path)
	if err != nil {
		return nil, err
	}
	defer file.Close()
	rows, err := csv.NewReader(file).ReadAll()
	if err != nil {
		return nil, err
	}
	if len(rows) == 0 {
		return nil, fmt.Errorf("screening CSV is empty")
	}
	requiredColumns := len(screeningCSVHeader)
	if len(rows[0]) < requiredColumns {
		return nil, fmt.Errorf("screening CSV header has %d columns; want at least %d", len(rows[0]), requiredColumns)
	}
	for index, want := range screeningCSVHeader {
		if got := rows[0][index]; got != want {
			return nil, fmt.Errorf("screening CSV column %d is %q; want %q", index+1, got, want)
		}
	}
	var events []DecisionEvent
	for index, row := range rows[1:] {
		if len(row) < requiredColumns {
			return nil, fmt.Errorf("screening CSV row %d has %d columns; want at least %d", index+2, len(row), requiredColumns)
		}
		events = append(events, DecisionEvent{PaperID: row[0], Stage: Stage(row[1]), Decision: Decision(row[2]), Reason: row[3], Reviewer: row[4]})
	}
	return events, nil
}

// PrioritizeActiveLearning is a deterministic scaffold for future ASReview-style ranking.
func PrioritizeActiveLearning(paperIDs []string) []string {
	out := append([]string{}, paperIDs...)
	sort.Strings(out)
	return out
}
