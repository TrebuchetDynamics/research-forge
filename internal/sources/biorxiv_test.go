package sources

import (
	"context"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"
)

func TestBioRxivSearchFiltersPreprints(t *testing.T) {
	fixedNow := time.Date(2024, 3, 15, 0, 0, 0, 0, time.UTC)
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if !strings.HasPrefix(r.URL.Path, "/details/biorxiv/") {
			t.Fatalf("path = %s", r.URL.Path)
		}
		if !strings.Contains(r.URL.Path, "2024-02-14/2024-03-15") {
			t.Fatalf("interval not in path: %s", r.URL.Path)
		}
		_, _ = w.Write([]byte(`{"messages":[{"status":"ok","total":2,"cursor":"0","count":2,"interval":"2024-02-14/2024-03-15"}],"collection":[{"doi":"10.1101/2024.03.01.576543","title":"Neural correlates of decision making","authors":"Smith J; Doe A","date":"2024-03-01","version":"1","category":"neuroscience","abstract":"We investigate neural correlates in humans.","server":"biorxiv"},{"doi":"10.1101/2024.02.20.888888","title":"Protein folding mechanisms","authors":"Lee K","date":"2024-02-20","version":"1","category":"biochemistry","abstract":"We study protein folding.","server":"biorxiv"}]}`))
	}))
	defer server.Close()
	connector := newBioRxivConnectorWithClock(
		NewHTTPClient(HTTPClientOptions{BaseURL: server.URL, Timeout: time.Second}),
		func() time.Time { return fixedNow },
	)
	response, err := connector.Search(context.Background(), SourceQuery{Terms: "neural decision"})
	if err != nil {
		t.Fatalf("Search returned error: %v", err)
	}
	if len(response.Records) != 1 {
		t.Fatalf("records = %d, want 1 (filtered)", len(response.Records))
	}
	record := response.Records[0]
	if record.Source != "biorxiv" {
		t.Fatalf("Source = %q", record.Source)
	}
	if record.Identifiers.DOI != "10.1101/2024.03.01.576543" {
		t.Fatalf("DOI = %q", record.Identifiers.DOI)
	}
	if record.Title != "Neural correlates of decision making" {
		t.Fatalf("Title = %q", record.Title)
	}
	if record.Year != 2024 {
		t.Fatalf("Year = %d", record.Year)
	}
	if !record.OpenAccess {
		t.Fatal("OpenAccess = false, want true")
	}
	if record.Metadata["category"] != "neuroscience" {
		t.Fatalf("category = %q", record.Metadata["category"])
	}
}

func TestBioRxivSearchMedRxivServer(t *testing.T) {
	fixedNow := time.Date(2024, 3, 15, 0, 0, 0, 0, time.UTC)
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if !strings.Contains(r.URL.Path, "/details/medrxiv/") {
			t.Fatalf("expected medrxiv in path: %s", r.URL.Path)
		}
		_, _ = w.Write([]byte(`{"messages":[{"status":"ok","total":0,"cursor":"0","count":0}],"collection":[]}`))
	}))
	defer server.Close()
	connector := newBioRxivConnectorWithClock(
		NewHTTPClient(HTTPClientOptions{BaseURL: server.URL, Timeout: time.Second}),
		func() time.Time { return fixedNow },
	)
	_, err := connector.Search(context.Background(), SourceQuery{Terms: "vaccine", Filters: map[string]string{"server": "medrxiv"}})
	if err != nil {
		t.Fatalf("Search error: %v", err)
	}
}

func TestBioRxivSearchLimitRespected(t *testing.T) {
	fixedNow := time.Date(2024, 3, 15, 0, 0, 0, 0, time.UTC)
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		_, _ = w.Write([]byte(`{"messages":[{"status":"ok"}],"collection":[` +
			`{"doi":"10.1101/a","title":"Neural alpha","abstract":"neural alpha study","date":"2024-03-01","server":"biorxiv","authors":"","category":"","version":"1"},` +
			`{"doi":"10.1101/b","title":"Neural beta","abstract":"neural beta study","date":"2024-03-02","server":"biorxiv","authors":"","category":"","version":"1"},` +
			`{"doi":"10.1101/c","title":"Neural gamma","abstract":"neural gamma study","date":"2024-03-03","server":"biorxiv","authors":"","category":"","version":"1"}` +
			`]}`))
	}))
	defer server.Close()
	connector := newBioRxivConnectorWithClock(
		NewHTTPClient(HTTPClientOptions{BaseURL: server.URL, Timeout: time.Second}),
		func() time.Time { return fixedNow },
	)
	response, err := connector.Search(context.Background(), SourceQuery{Terms: "neural", Limit: 2})
	if err != nil {
		t.Fatalf("Search error: %v", err)
	}
	if len(response.Records) != 2 {
		t.Fatalf("records = %d, want 2 (limited)", len(response.Records))
	}
}
