package ui

import "testing"

func TestDashboardAndFeatureViewModelsUseSharedServiceState(t *testing.T) {
	dashboard := NewDashboardState("Demo", 3)
	if dashboard.Title != "Demo" || dashboard.LibraryCount != 3 || dashboard.Loading || dashboard.Error != "" {
		t.Fatalf("dashboard = %#v", dashboard)
	}
	search := NewSearchFormState([]string{"openalex", "arxiv"})
	if len(search.Sources) != 2 || search.Query != "" {
		t.Fatalf("search = %#v", search)
	}
	library := NewLibraryViewModel([]PaperRow{{Title: "Paper"}})
	if len(library.Rows) != 1 || library.Empty {
		t.Fatalf("library = %#v", library)
	}
	oss := NewOSSDashboardViewModel([]OSSRow{{Name: "owner/repo"}})
	if len(oss.Repositories) != 1 {
		t.Fatalf("oss = %#v", oss)
	}
	if NewCitationGraphViewModel([]GraphNode{{ID: "p1"}}, nil).Nodes[0].ID != "p1" {
		t.Fatalf("citation vm failed")
	}
	if NewScreeningViewModel([]ScreeningRow{{PaperID: "p1"}}).Rows[0].PaperID != "p1" {
		t.Fatalf("screening vm failed")
	}
	if NewEvidenceViewModel([]EvidenceRow{{PaperID: "p1", SupportRef: "passage:p1"}}).Rows[0].SupportRef != "passage:p1" {
		t.Fatalf("evidence vm failed")
	}
	if NewAnalysisViewModel("run-1", true).Ready != true {
		t.Fatalf("analysis vm failed")
	}
	if NewReportViewModel([]string{"markdown"}).Formats[0] != "markdown" {
		t.Fatalf("report vm failed")
	}
}
