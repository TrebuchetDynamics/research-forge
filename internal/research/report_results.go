package research

import (
	"bytes"
	"encoding/json"
	"fmt"
)

// decodeReportResults rejects damaged report inputs without changing the
// permissive ReadPaperResults reader used by legacy consumers.
func decodeReportResults(data []byte, path string) ([]Paper, error) {
	var papers []Paper
	for i, line := range bytes.Split(data, []byte("\n")) {
		line = bytes.TrimSpace(line)
		if len(line) == 0 {
			continue
		}
		var paper Paper
		if err := json.Unmarshal(line, &paper); err != nil {
			return nil, fmt.Errorf("%s:%d: decode paper: %w", path, i+1, err)
		}
		if paper.Title != "" {
			papers = append(papers, paper)
		}
	}
	return papers, nil
}
