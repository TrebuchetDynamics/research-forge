package evidence

import (
	"encoding/csv"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"
)

func ExportCSV(path string, items []EvidenceItem) error {
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		return err
	}
	file, err := os.Create(path)
	if err != nil {
		return err
	}
	defer file.Close()
	w := csv.NewWriter(file)
	defer w.Flush()
	if err := w.Write([]string{"paper_id", "schema", "status", "support_kind", "support_ref", "values"}); err != nil {
		return err
	}
	for _, item := range items {
		if err := w.Write([]string{item.PaperID, item.SchemaName, string(item.Status), string(item.Support.Kind), item.Support.Ref, formatValues(item.Values)}); err != nil {
			return err
		}
	}
	return w.Error()
}
func ExportJSON(path string, items []EvidenceItem) error {
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		return err
	}
	data, err := json.MarshalIndent(items, "", "  ")
	if err != nil {
		return err
	}
	data = append(data, '\n')
	return os.WriteFile(path, data, 0o644)
}
func ExportMarkdown(path string, items []EvidenceItem) error {
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		return err
	}
	var b strings.Builder
	b.WriteString("| paper | schema | status | support |\n| --- | --- | --- | --- |\n")
	for _, item := range items {
		fmt.Fprintf(&b, "| %s | %s | %s | %s:%s |\n", item.PaperID, item.SchemaName, item.Status, item.Support.Kind, item.Support.Ref)
	}
	return os.WriteFile(path, []byte(b.String()), 0o644)
}
func formatValues(values map[string]string) string {
	keys := make([]string, 0, len(values))
	for key := range values {
		keys = append(keys, key)
	}
	sort.Strings(keys)
	parts := []string{}
	for _, key := range keys {
		parts = append(parts, key+"="+values[key])
	}
	return strings.Join(parts, ";")
}
