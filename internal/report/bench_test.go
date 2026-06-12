package report

import "testing"

func BenchmarkReportGeneration(b *testing.B) {
	data := Data{Title: "Benchmark", Citations: []Citation{{ID: "p1", Title: "Paper"}}, EvidenceRows: []EvidenceRow{{PaperID: "p1", Summary: "Evidence"}}}
	for i := 0; i < b.N; i++ {
		_ = BuildMarkdown(data)
	}
}
