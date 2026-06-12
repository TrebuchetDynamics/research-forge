package watch

import "testing"

func TestWatchedSearchRefreshCreatesInboxAndRequiresPDFApproval(t *testing.T) {
	watched, err := NewWatchedSearch(Input{Name: "ap", Source: "openalex", Query: "artificial photosynthesis", Interval: "daily"})
	if err != nil {
		t.Fatalf("NewWatchedSearch: %v", err)
	}
	inbox := NewInbox()
	run := Refresh(watched, []Paper{{ID: "p1", Title: "New catalyst"}}, inbox)
	if run.WatchedName != "ap" || run.NewCount != 1 || len(inbox.List()) != 1 {
		t.Fatalf("run=%#v inbox=%#v", run, inbox.List())
	}
	if err := inbox.ApprovePDF("p1"); err != nil {
		t.Fatalf("ApprovePDF: %v", err)
	}
	if !inbox.List()[0].PDFApproved {
		t.Fatalf("approval not stored")
	}
	if event := run.ProvenanceEvent(); event.Action != "watch.refresh" || event.Outputs["newCount"] != 1 {
		t.Fatalf("event=%#v", event)
	}
}
