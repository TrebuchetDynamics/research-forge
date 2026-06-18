package sources

import (
	"context"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"
)

func TestOpenCitationsSearchNormalizesRecords(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case "/index/v1/citations/10.1162/qss_a_00018":
			// real API: citing field is a plain DOI, no "doi:" prefix
			_, _ = w.Write([]byte(`[
				{"oci":"01-02","citing":"10.1234/citing1","cited":"10.1162/qss_a_00018","creation":"2021","timespan":"P2Y","journal_sc":"no","author_sc":"no"},
				{"oci":"01-03","citing":"10.1234/citing2","cited":"10.1162/qss_a_00018","creation":"2022","timespan":"P3Y","journal_sc":"no","author_sc":"no"}
			]`))
		case "/meta/v1/metadata/doi:10.1234/citing1__doi:10.1234/citing2":
			// real API: id field has "doi:10.xxx omid:br/..." format; venue/publisher have annotations
			_, _ = w.Write([]byte(`[
				{"id":"doi:10.1234/citing1 omid:br/0001","title":"Citing Paper One","author":"Smith, J","pub_date":"2021-06","venue":"Journal of Science [issn:0000-0001 omid:br/99]","volume":"1","issue":"2","page":"1-10","type":"journal article","publisher":"MIT Press [crossref:123 omid:ra/1]","editor":""},
				{"id":"doi:10.1234/citing2 omid:br/0002","title":"Citing Paper Two","author":"Brown, B","pub_date":"2022-01","venue":"Nature","volume":"5","issue":"1","page":"20-30","type":"journal article","publisher":"Springer","editor":""}
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
	if response.RawRef != "opencitations:/index/v1/citations/10.1162/qss_a_00018" {
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
	if r0.Title != "Citing Paper One" {
		t.Fatalf("Title = %q", r0.Title)
	}
	if r0.Year != 2021 {
		t.Fatalf("Year = %d, want 2021", r0.Year)
	}
	if r0.Venue != "Journal of Science" {
		t.Fatalf("Venue = %q (should strip [issn:...] annotation)", r0.Venue)
	}
	if r0.Publisher != "MIT Press" {
		t.Fatalf("Publisher = %q (should strip [crossref:...] annotation)", r0.Publisher)
	}
	if r0.Metadata["creation"] != "2021" {
		t.Fatalf("creation = %q", r0.Metadata["creation"])
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
		if r.URL.Path != "/index/v1/citations/10.9999/nodoi" {
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
	if response.RawRef != "opencitations:/index/v1/citations/10.9999/nodoi" {
		t.Fatalf("RawRef = %q", response.RawRef)
	}
}

func TestOpenCitationsSearchDefaultLimit(t *testing.T) {
	citationsBody := `[`
	for i := 0; i < 30; i++ {
		if i > 0 {
			citationsBody += ","
		}
		citationsBody += `{"oci":"","citing":"10.1234/c` + itoa(i) + `","cited":"10.0000/x","creation":"2020","timespan":"P1Y","journal_sc":"no","author_sc":"no"}`
	}
	citationsBody += `]`

	metaCalled := false
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path == "/index/v1/citations/10.0000/x" {
			_, _ = w.Write([]byte(citationsBody))
			return
		}
		// metadata call — verify batch size is 25 via __ separator count
		metaCalled = true
		batch := strings.TrimPrefix(r.URL.Path, "/meta/v1/metadata/")
		count := strings.Count(batch, "__") + 1
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
