package library

import (
	"fmt"
	"testing"
)

func benchmarkRecords(n int) []PaperRecord {
	out := make([]PaperRecord, 0, n)
	for i := 0; i < n; i++ {
		p, _ := NewPaperRecord(PaperRecordInput{Title: "Artificial photosynthesis benchmark", Identifiers: Identifiers{DOI: fmt.Sprintf("10.1000/bench-%d", i)}, Year: 2026})
		out = append(out, p)
	}
	return out
}

func BenchmarkDeduplication10(b *testing.B) {
	records := benchmarkRecords(10)
	for i := 0; i < b.N; i++ {
		_ = ScoreDuplicate(records[0], records[1])
	}
}

func BenchmarkDeduplication1000(b *testing.B) {
	records := benchmarkRecords(1000)
	for i := 0; i < b.N; i++ {
		_ = ScoreDuplicate(records[0], records[999])
	}
}

func BenchmarkDataset100000(b *testing.B) {
	for i := 0; i < b.N; i++ {
		_ = benchmarkRecords(100000)
	}
}

func BenchmarkImportsExports(b *testing.B) {
	records := benchmarkRecords(10)
	for i := 0; i < b.N; i++ {
		path := b.TempDir() + "/x.json"
		_ = ExportJSON(path, records)
		_, _ = ImportJSON(path)
	}
}
