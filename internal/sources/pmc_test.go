package sources

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"
)

func TestPMCSearchNormalizesRecords(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case "/entrez/eutils/esearch.fcgi":
			if r.URL.Query().Get("db") != "pmc" {
				t.Fatalf("esearch db = %q, want pmc", r.URL.Query().Get("db"))
			}
			if r.URL.Query().Get("term") == "" {
				t.Fatal("esearch missing term param")
			}
			_, _ = w.Write([]byte(`{
				"esearchresult": {
					"count": "1",
					"idlist": ["13275012"]
				}
			}`))
		case "/entrez/eutils/esummary.fcgi":
			if r.URL.Query().Get("db") != "pmc" {
				t.Fatalf("esummary db = %q, want pmc", r.URL.Query().Get("db"))
			}
			_, _ = w.Write([]byte(`{
				"result": {
					"uids": ["13275012"],
					"13275012": {
						"title": "Deep Learning for Protein Structure Prediction",
						"pubdate": "2024 Mar 15",
						"source": "Nat Methods",
						"volume": "21",
						"issue": "3",
						"pages": "345-356",
						"authors": [
							{"name": "Smith J", "authtype": "Author"},
							{"name": "Lee K", "authtype": "Author"},
							{"name": "ORCID", "authtype": "ORCID"}
						],
						"articleids": [
							{"idtype": "pmid", "value": "38472920"},
							{"idtype": "doi", "value": "10.1038/s41592-024-02222-x"},
							{"idtype": "pmcid", "value": "PMC13275012"}
						]
					}
				}
			}`))
		default:
			t.Fatalf("unexpected path: %q", r.URL.Path)
		}
	}))
	defer server.Close()

	connector := NewPMCConnector(NewHTTPClient(HTTPClientOptions{BaseURL: server.URL, Timeout: time.Second}))
	response, err := connector.Search(context.Background(), SourceQuery{Terms: "protein structure", Limit: 5})
	if err != nil {
		t.Fatalf("Search error: %v", err)
	}
	if len(response.Records) != 1 {
		t.Fatalf("records = %d, want 1", len(response.Records))
	}
	r := response.Records[0]
	if r.Source != "pmc" {
		t.Fatalf("Source = %q, want pmc", r.Source)
	}
	if r.SourceID != "13275012" {
		t.Fatalf("SourceID = %q, want 13275012", r.SourceID)
	}
	if r.Identifiers.DOI != "10.1038/s41592-024-02222-x" {
		t.Fatalf("DOI = %q, want 10.1038/s41592-024-02222-x", r.Identifiers.DOI)
	}
	if r.Identifiers.CrossrefID != "" {
		t.Fatalf("CrossrefID = %q, want empty when DOI set", r.Identifiers.CrossrefID)
	}
	if r.Title != "Deep Learning for Protein Structure Prediction" {
		t.Fatalf("Title = %q", r.Title)
	}
	if r.Year != 2024 {
		t.Fatalf("Year = %d, want 2024", r.Year)
	}
	if r.Venue != "Nat Methods" {
		t.Fatalf("Venue = %q, want Nat Methods", r.Venue)
	}
	if !r.OpenAccess {
		t.Fatal("OpenAccess = false, want true")
	}
	// Only Author-type authors should be included (not ORCID entries).
	if r.Metadata["authors"] != "Smith J; Lee K" {
		t.Fatalf("authors = %q, want Smith J; Lee K", r.Metadata["authors"])
	}
	if r.Metadata["pmc_id"] != "PMC13275012" {
		t.Fatalf("pmc_id = %q, want PMC13275012", r.Metadata["pmc_id"])
	}
	if r.Metadata["volume"] != "21" {
		t.Fatalf("volume = %q, want 21", r.Metadata["volume"])
	}

	papers, err := PaperRecords(response)
	if err != nil {
		t.Fatalf("PaperRecords error: %v", err)
	}
	if len(papers) != 1 || papers[0].Title != "Deep Learning for Protein Structure Prediction" {
		t.Fatalf("papers round-trip failed")
	}
}

func TestPMCSearchCrossrefIDFallback(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case "/entrez/eutils/esearch.fcgi":
			_, _ = w.Write([]byte(`{"esearchresult":{"count":"1","idlist":["9999999"]}}`))
		case "/entrez/eutils/esummary.fcgi":
			_, _ = w.Write([]byte(`{
				"result": {
					"uids": ["9999999"],
					"9999999": {
						"title": "No DOI Article",
						"pubdate": "2022",
						"source": "Test Journal",
						"authors": [{"name": "Doe J", "authtype": "Author"}],
						"articleids": [
							{"idtype": "pmid", "value": "99999999"},
							{"idtype": "pmcid", "value": "PMC9999999"}
						]
					}
				}
			}`))
		}
	}))
	defer server.Close()

	connector := NewPMCConnector(NewHTTPClient(HTTPClientOptions{BaseURL: server.URL, Timeout: time.Second}))
	response, err := connector.Search(context.Background(), SourceQuery{Terms: "test"})
	if err != nil {
		t.Fatalf("Search error: %v", err)
	}
	if len(response.Records) != 1 {
		t.Fatalf("records = %d, want 1", len(response.Records))
	}
	r := response.Records[0]
	if r.Identifiers.DOI != "" {
		t.Fatalf("DOI = %q, want empty", r.Identifiers.DOI)
	}
	if r.Identifiers.CrossrefID != "pmc:PMC9999999" {
		t.Fatalf("CrossrefID = %q, want pmc:PMC9999999", r.Identifiers.CrossrefID)
	}
	if r.Year != 2022 {
		t.Fatalf("Year = %d, want 2022", r.Year)
	}
}

func TestPMCSearchEmptyIDList(t *testing.T) {
	calls := 0
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		calls++
		if r.URL.Path == "/entrez/eutils/esearch.fcgi" {
			_, _ = w.Write([]byte(`{"esearchresult":{"count":"0","idlist":[]}}`))
		} else {
			t.Fatalf("esummary should not be called when idlist is empty")
		}
	}))
	defer server.Close()

	connector := NewPMCConnector(NewHTTPClient(HTTPClientOptions{BaseURL: server.URL, Timeout: time.Second}))
	response, err := connector.Search(context.Background(), SourceQuery{Terms: "xyzzy"})
	if err != nil {
		t.Fatalf("Search error: %v", err)
	}
	if len(response.Records) != 0 {
		t.Fatalf("records = %d, want 0", len(response.Records))
	}
	if calls != 1 {
		t.Fatalf("HTTP calls = %d, want 1 (esummary skipped for empty idlist)", calls)
	}
}

func TestPMCSearchSkipsBlankTitles(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case "/entrez/eutils/esearch.fcgi":
			_, _ = w.Write([]byte(`{"esearchresult":{"count":"2","idlist":["111","222"]}}`))
		case "/entrez/eutils/esummary.fcgi":
			_, _ = w.Write([]byte(`{
				"result": {
					"uids": ["111","222"],
					"111": {"title": "", "pubdate": "2023", "source": "J Test", "authors": [], "articleids": [{"idtype": "doi", "value": "10.0/blank"}]},
					"222": {"title": "Valid PMC Paper", "pubdate": "2023", "source": "J Test", "authors": [], "articleids": [{"idtype": "doi", "value": "10.0/valid"}]}
				}
			}`))
		}
	}))
	defer server.Close()

	connector := NewPMCConnector(NewHTTPClient(HTTPClientOptions{BaseURL: server.URL, Timeout: time.Second}))
	response, err := connector.Search(context.Background(), SourceQuery{Terms: "test"})
	if err != nil {
		t.Fatalf("Search error: %v", err)
	}
	if len(response.Records) != 1 {
		t.Fatalf("records = %d, want 1 (blank title skipped)", len(response.Records))
	}
	if response.Records[0].Title != "Valid PMC Paper" {
		t.Fatalf("Title = %q, want Valid PMC Paper", response.Records[0].Title)
	}
}

func TestPMCSearchDefaultLimit(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path == "/entrez/eutils/esearch.fcgi" {
			if r.URL.Query().Get("retmax") != "25" {
				t.Fatalf("default retmax = %q, want 25", r.URL.Query().Get("retmax"))
			}
			_, _ = w.Write([]byte(`{"esearchresult":{"count":"0","idlist":[]}}`))
		}
	}))
	defer server.Close()

	connector := NewPMCConnector(NewHTTPClient(HTTPClientOptions{BaseURL: server.URL, Timeout: time.Second}))
	_, err := connector.Search(context.Background(), SourceQuery{Terms: "biology", Limit: 0})
	if err != nil {
		t.Fatalf("Search error: %v", err)
	}
}
