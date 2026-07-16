package evidence

import (
	"bytes"
	"encoding/csv"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"

	"github.com/TrebuchetDynamics/research-forge/internal/filetxn"
)

func ExportCSV(path string, items []EvidenceItem) error {
	var buffer bytes.Buffer
	w := csv.NewWriter(&buffer)
	if err := w.Write([]string{"paper_id", "schema", "status", "support_kind", "support_ref", "values"}); err != nil {
		return err
	}
	for _, item := range items {
		if err := w.Write([]string{item.PaperID, item.SchemaName, string(item.Status), string(item.Support.Kind), item.Support.Ref, formatValues(item.Values)}); err != nil {
			return err
		}
	}
	w.Flush()
	if err := w.Error(); err != nil {
		return err
	}
	return writeExport(path, buffer.Bytes())
}
func ExportJSON(path string, items []EvidenceItem) error {
	data, err := json.MarshalIndent(items, "", "  ")
	if err != nil {
		return err
	}
	data = append(data, '\n')
	return writeExport(path, data)
}
func ExportMarkdown(path string, items []EvidenceItem) error {
	var b strings.Builder
	b.WriteString("| paper | schema | status | support |\n| --- | --- | --- | --- |\n")
	for _, item := range items {
		fmt.Fprintf(&b, "| %s | %s | %s | %s:%s |\n", item.PaperID, item.SchemaName, item.Status, item.Support.Kind, item.Support.Ref)
	}
	return writeExport(path, []byte(b.String()))
}
func writeExport(path string, data []byte) error {
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		return err
	}
	return filetxn.Replace(path, data, 0o644)
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
