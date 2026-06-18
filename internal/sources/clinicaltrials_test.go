package sources

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"
)

func TestClinicalTrialsSearchNormalizesStudies(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/api/v2/studies" {
			t.Fatalf("path = %s", r.URL.Path)
		}
		if r.URL.Query().Get("query.term") != "hypertension drug" {
			t.Fatalf("query.term = %q", r.URL.Query().Get("query.term"))
		}
		if r.URL.Query().Get("pageSize") != "4" {
			t.Fatalf("pageSize = %q", r.URL.Query().Get("pageSize"))
		}
		_, _ = w.Write([]byte(`{"studies":[{"protocolSection":{"identificationModule":{"nctId":"NCT01234567","briefTitle":"A Randomized Study of Drug X","officialTitle":"Full Official Title of Drug X Study"},"descriptionModule":{"briefSummary":"This study evaluates the efficacy of Drug X in hypertension."},"statusModule":{"overallStatus":"Recruiting","startDateStruct":{"date":"2024-01-15"}},"sponsorCollaboratorsModule":{"leadSponsor":{"name":"Example University"}}}}],"nextPageToken":""}`))
	}))
	defer server.Close()
	connector := NewClinicalTrialsConnector(NewHTTPClient(HTTPClientOptions{BaseURL: server.URL, Timeout: time.Second}))
	response, err := connector.Search(context.Background(), SourceQuery{Terms: "hypertension drug", Limit: 4})
	if err != nil {
		t.Fatalf("Search returned error: %v", err)
	}
	if response.RawRef != "clinicaltrials:/api/v2/studies?format=json&pageSize=4&query.term=hypertension+drug" {
		t.Fatalf("RawRef = %q", response.RawRef)
	}
	if len(response.Records) != 1 {
		t.Fatalf("records = %d, want 1", len(response.Records))
	}
	record := response.Records[0]
	if record.Source != "clinicaltrials" {
		t.Fatalf("Source = %q", record.Source)
	}
	if record.SourceID != "NCT01234567" {
		t.Fatalf("SourceID = %q", record.SourceID)
	}
	if record.Title != "A Randomized Study of Drug X" {
		t.Fatalf("Title = %q", record.Title)
	}
	if record.Year != 2024 {
		t.Fatalf("Year = %d", record.Year)
	}
	if record.Abstract != "This study evaluates the efficacy of Drug X in hypertension." {
		t.Fatalf("Abstract = %q", record.Abstract)
	}
	if record.Metadata["sponsor"] != "Example University" {
		t.Fatalf("sponsor = %q", record.Metadata["sponsor"])
	}
	if record.Metadata["overall_status"] != "Recruiting" {
		t.Fatalf("overall_status = %q", record.Metadata["overall_status"])
	}
	if record.Identifiers.CrossrefID != "NCT01234567" {
		t.Fatalf("CrossrefID (NCT ID) = %q", record.Identifiers.CrossrefID)
	}
	papers, err := PaperRecords(response)
	if err != nil {
		t.Fatalf("PaperRecords error: %v", err)
	}
	if papers[0].Title != "A Randomized Study of Drug X" {
		t.Fatalf("papers[0].Title = %q", papers[0].Title)
	}
}

func TestClinicalTrialsFallsBackToOfficialTitle(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		_, _ = w.Write([]byte(`{"studies":[{"protocolSection":{"identificationModule":{"nctId":"NCT99999999","briefTitle":"","officialTitle":"Only Official Title"},"descriptionModule":{},"statusModule":{},"sponsorCollaboratorsModule":{}}}]}`))
	}))
	defer server.Close()
	connector := NewClinicalTrialsConnector(NewHTTPClient(HTTPClientOptions{BaseURL: server.URL, Timeout: time.Second}))
	response, err := connector.Search(context.Background(), SourceQuery{Terms: "test"})
	if err != nil {
		t.Fatalf("Search error: %v", err)
	}
	if response.Records[0].Title != "Only Official Title" {
		t.Fatalf("fallback title = %q", response.Records[0].Title)
	}
}
