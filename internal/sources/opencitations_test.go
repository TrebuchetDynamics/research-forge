package sources

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"
)

func TestOpenCitationsSearchNormalizesRecords(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case "/index/api/v2/citations/10.1162/qss_a_00018":
			_, _ = w.Write([]byte(`[
				{"oci":"01-02","citing":"doi:10.1234/citing1","cited":"doi:10.1162/qss_a_00018","creation":"2021","timespan":"P2Y","journal_sc":"no","author_sc":"no"},
				{"oci":"01-03","citing":"doi:10.1234/citing2","cited":"doi:10.1162/qss_a_00018","creation":"2022","timespan":"P3Y","journal_sc":"no","author_sc":"no"}
			]`))
		case "/index/api/v2/metadata/10.1234/citing1;10.1234/citing2":
			_, _ = w.Write([]byte(`[
				{"id":"doi:10.1234/citing1","title":"Citing Paper One","author":"Smith, J; Doe, A","pub_date":"2021-06","venue":"Journal of Science","volume":"1","issue":"2","page":"1-10","type":"journal article","publisher":"MIT Press","editor":""},
				{"id":"doi:10.1234/citing2","title":"Citing Paper Two","author":"Brown, B","pub_date":"2022-01","venue":"Nature","volume":"5","issue":"1","page":"20-30","type":"journal article","publisher":"Springer","editor":""}
			]`))
		default:
			t.Fatalf("unexpected path: %s", r.URL.Path)
		}
	}))
	defer server.Close()

	connector := NewOpenCitationsConnector(NewHTTPClient(HTTPClientOptions{BaseURL: server.URL, Timeout: time.Second}))
	response, err := connector.Search(context.Background(), SourceQuery{Terms: "10.1162/qss_a_00018", Limit: 25})
	if err != nil {
		t.Fatalf("Search returned error: %v", err)
	}

	if response.RawRef != "opencitations:/index/api/v2/citations/10.1162/qss_a_00018" {
		t.Fatalf("RawRef = %q", response.RawRef)
	}
	if len(response.Records) != 2 {
		t.Fatalf("records = %d, want 2", len(response.Records))
	}

	r0 := response.Records[0]
	if r0.Source != "opencitations" {
		t.Fatalf("Source = %q", r0.Source)
	}
	if r0.Identifiers.DOI != "10.1234/citing1" {
		t.Fatalf("DOI = %q", r0.Identifiers.DOI)
	}
	if r0.SourceID != "10.1234/citing1" {
		t.Fatalf("SourceID = %q", r0.SourceID)
	}
	if r0.Title != "Citing Paper One" {
		t.Fatalf("Title = %q", r0.Title)
	}
	if r0.Year != 2021 {
		t.Fatalf("Year = %d, want 2021", r0.Year)
	}
	if r0.Venue != "Journal of Science" {
		t.Fatalf("Venue = %q", r0.Venue)
	}
	if len(r0.URLs) != 1 || r0.URLs[0] != "https://doi.org/10.1234/citing1" {
		t.Fatalf("URLs = %v", r0.URLs)
	}
	if r0.Metadata["type"] != "journal article" {
		t.Fatalf("Metadata[type] = %q", r0.Metadata["type"])
	}
	if r0.Metadata["creation"] != "2021" {
		t.Fatalf("Metadata[creation] = %q", r0.Metadata["creation"])
	}

	r1 := response.Records[1]
	if r1.Identifiers.DOI != "10.1234/citing2" {
		t.Fatalf("r1 DOI = %q", r1.Identifiers.DOI)
	}
	if r1.Year != 2022 {
		t.Fatalf("r1 Year = %d, want 2022", r1.Year)
	}

	papers, err := PaperRecords(response)
	if err != nil {
		t.Fatalf("PaperRecords error: %v", err)
	}
	if papers[0].Title != "Citing Paper One" {
		t.Fatalf("papers[0].Title = %q", papers[0].Title)
	}
}

func TestOpenCitationsSearchEmptyCitations(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/index/api/v2/citations/10.9999/nodoi" {
			t.Fatalf("unexpected path: %s", r.URL.Path)
		}
		_, _ = w.Write([]byte(`[]`))
	}))
	defer server.Close()

	connector := NewOpenCitationsConnector(NewHTTPClient(HTTPClientOptions{BaseURL: server.URL, Timeout: time.Second}))
	response, err := connector.Search(context.Background(), SourceQuery{Terms: "10.9999/nodoi"})
	if err != nil {
		t.Fatalf("Search error: %v", err)
	}
	if len(response.Records) != 0 {
		t.Fatalf("records = %d, want 0", len(response.Records))
	}
	if response.RawRef != "opencitations:/index/api/v2/citations/10.9999/nodoi" {
		t.Fatalf("RawRef = %q", response.RawRef)
	}
}

func TestOpenCitationsSearchDefaultLimit(t *testing.T) {
	citationsBody := `[`
	for i := 0; i < 30; i++ {
		if i > 0 {
			citationsBody += ","
		}
		citationsBody += `{"oci":"","citing":"doi:10.1234/c` + itoa(i) + `","cited":"doi:10.0000/x","creation":"2020","timespan":"P1Y","journal_sc":"no","author_sc":"no"}`
	}
	citationsBody += `]`

	metaCalled := false
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path == "/index/api/v2/citations/10.0000/x" {
			_, _ = w.Write([]byte(citationsBody))
			return
		}
		// metadata call — just return empty array; we only care that it was called with 25 DOIs
		metaCalled = true
		// count semicolons in path to verify limit
		parts := r.URL.Path[len("/index/api/v2/metadata/"):]
		count := 1
		for _, ch := range parts {
			if ch == ';' {
				count++
			}
		}
		if count != 25 {
			t.Errorf("metadata batch size = %d, want 25", count)
		}
		_, _ = w.Write([]byte(`[]`))
	}))
	defer server.Close()

	connector := NewOpenCitationsConnector(NewHTTPClient(HTTPClientOptions{BaseURL: server.URL, Timeout: time.Second}))
	_, err := connector.Search(context.Background(), SourceQuery{Terms: "10.0000/x"})
	if err != nil {
		t.Fatalf("Search error: %v", err)
	}
	if !metaCalled {
		t.Fatal("metadata endpoint was not called")
	}
}

// itoa is a local helper to avoid importing strconv in the test file.
func itoa(n int) string {
	if n == 0 {
		return "0"
	}
	digits := []byte{}
	for n > 0 {
		digits = append([]byte{byte('0' + n%10)}, digits...)
		n /= 10
	}
	return string(digits)
}
