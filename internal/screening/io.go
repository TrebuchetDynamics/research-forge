package screening

import (
	"encoding/csv"
	"os"
	"path/filepath"
	"sort"
)

func ExportCSV(path string, events []DecisionEvent) error {
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		return err
	}
	file, err := os.Create(path)
	if err != nil {
		return err
	}
	defer file.Close()
	writer := csv.NewWriter(file)
	defer writer.Flush()
	if err := writer.Write([]string{"paper_id", "stage", "decision", "reason", "reviewer"}); err != nil {
		return err
	}
	for _, event := range events {
		if err := writer.Write([]string{event.PaperID, string(event.Stage), string(event.Decision), event.Reason, event.Reviewer}); err != nil {
			return err
		}
	}
	return writer.Error()
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
	var events []DecisionEvent
	for _, row := range rows[1:] {
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
