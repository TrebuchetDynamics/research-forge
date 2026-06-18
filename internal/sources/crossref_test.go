package sources

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"
)

func TestCrossrefConnectorLookupDOIRefreshesMetadata(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/works/10.5555/refresh" {
			t.Fatalf("path = %q", r.URL.Path)
		}
		_, _ = w.Write([]byte(`{"message":{"DOI":"10.5555/REFRESH","title":["Refreshed Crossref metadata"],"publisher":"Refresh Publisher","reference-count":2,"license":[{"URL":"https://license.example"}]}}`))
	}))
	defer server.Close()

	connector := NewCrossrefConnector(NewHTTPClient(HTTPClientOptions{BaseURL: server.URL, Timeout: time.Second}))
	record, rawRef, err := connector.LookupDOI(context.Background(), "10.5555/refresh")
	if err != nil {
		t.Fatalf("LookupDOI returned error: %v", err)
	}
	if rawRef != "crossref:/works/10.5555/refresh" {
		t.Fatalf("rawRef = %q", rawRef)
	}
	if record.Title != "Refreshed Crossref metadata" || record.Publisher != "Refresh Publisher" || record.License != "https://license.example" || record.Metadata["reference_count"] != "2" {
		t.Fatalf("record = %#v", record)
	}
}

func TestCrossrefConnectorReferencesExtractsReferenceList(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/works/10.5555/source" {
			t.Fatalf("path = %q", r.URL.Path)
		}
		_, _ = w.Write([]byte(`{"message":{"DOI":"10.5555/source","reference":[{"DOI":"10.1000/ref-one","article-title":"Reference One","key":"ref1"},{"article-title":"Title only reference","key":"ref2"},{"key":"empty"}]}}`))
	}))
	defer server.Close()

	response, err := NewCrossrefConnector(NewHTTPClient(HTTPClientOptions{BaseURL: server.URL})).References(context.Background(), "10.5555/source")
	if err != nil {
		t.Fatalf("References returned error: %v", err)
	}
	if response.RawRef != "crossref:/works/10.5555/source/references" {
		t.Fatalf("rawRef = %q", response.RawRef)
	}
	if len(response.Records) != 2 {
		t.Fatalf("records = %#v", response.Records)
	}
	if response.Records[0].Identifiers.DOI != "10.1000/ref-one" || response.Records[0].Title != "Reference One" || response.Records[0].Metadata["referenced_by_doi"] != "10.5555/source" {
		t.Fatalf("first record = %#v", response.Records[0])
	}
	if response.Records[1].Title != "Title only reference" || response.Records[1].Metadata["reference_key"] != "ref2" {
		t.Fatalf("second record = %#v", response.Records[1])
	}
}

func TestCrossrefConnectorSearchTranslatesOpenAlexFilters(t *testing.T) {
	cases := []struct {
		name            string
		inputFilter     string
		wantFilterParam string
	}{
		{
			name:            "from_publication_date translates to from-pub-date year",
			inputFilter:     "from_publication_date:2020-01-01",
			wantFilterParam: "from-pub-date:2020",
		},
		{
			name:            "to_publication_date translates to until-pub-date year",
			inputFilter:     "to_publication_date:2022-12-31",
			wantFilterParam: "until-pub-date:2022",
		},
		{
			name:            "OpenAlex type:article maps to journal-article",
			inputFilter:     "type:article",
			wantFilterParam: "type:journal-article",
		},
		{
			name:            "is_oa filter is dropped (no Crossref equivalent)",
			inputFilter:     "is_oa:true",
			wantFilterParam: "",
		},
		{
			name:            "open_access.is_oa filter is dropped",
			inputFilter:     "open_access.is_oa:true",
			wantFilterParam: "",
		},
		{
			name:            "concepts.id filter is dropped",
			inputFilter:     "concepts.id:C41008148",
			wantFilterParam: "",
		},
		{
			name:            "mixed OpenAlex preset translates supported and drops unsupported",
			inputFilter:     "from_publication_date:2020-01-01,type:article,is_oa:true",
			wantFilterParam: "from-pub-date:2020,type:journal-article",
		},
		{
			name:            "native Crossref filter is passed through unchanged",
			wantFilterParam: "from-pub-date:2019,type:journal-article",
			inputFilter:     "from-pub-date:2019,type:journal-article",
		},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				if got := r.URL.Query().Get("filter"); got != tc.wantFilterParam {
					t.Errorf("filter param = %q, want %q", got, tc.wantFilterParam)
				}
				_, _ = w.Write([]byte(`{"message":{"items":[]}}`))
			}))
			defer server.Close()
			_, _ = NewCrossrefConnector(NewHTTPClient(HTTPClientOptions{BaseURL: server.URL})).Search(
				context.Background(),
				SourceQuery{Terms: "test", Limit: 1, Filters: map[string]string{"filter": tc.inputFilter}},
			)
		})
	}
}

func TestCrossrefConnectorSearchPassesWorksFilters(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/works" || r.URL.Query().Get("filter") != "from-pub-date:2020,type:journal-article" {
			t.Fatalf("request = %s?%s", r.URL.Path, r.URL.RawQuery)
		}
		_, _ = w.Write([]byte(`{"message":{"items":[{"DOI":"10.5555/filtered","title":["Filtered work"]}]}}`))
	}))
	defer server.Close()

	response, err := NewCrossrefConnector(NewHTTPClient(HTTPClientOptions{BaseURL: server.URL})).Search(context.Background(), SourceQuery{Terms: "filtered", Limit: 1, Filters: map[string]string{"filter": "from-pub-date:2020,type:journal-article"}})
	if err != nil {
		t.Fatalf("Search returned error: %v", err)
	}
	if len(response.Records) != 1 || response.Records[0].Identifiers.DOI != "10.5555/filtered" {
		t.Fatalf("response = %#v", response)
	}
	if response.RawRef != "crossref:/works?filter=from-pub-date%3A2020%2Ctype%3Ajournal-article&query=filtered&rows=1" {
		t.Fatalf("rawRef = %q", response.RawRef)
	}
}

func TestCrossrefConnectorSearchesAndNormalizesWorks(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/works" {
			t.Fatalf("path = %q, want /works", r.URL.Path)
		}
		if r.URL.Query().Get("query") != "artificial photosynthesis" {
			t.Fatalf("query = %q", r.URL.Query().Get("query"))
		}
		if r.URL.Query().Get("rows") != "1" {
			t.Fatalf("rows = %q", r.URL.Query().Get("rows"))
		}
		_, _ = w.Write([]byte(`{
			"message": {"items": [{
				"DOI": "10.5555/CROSSREF.EXAMPLE",
				"title": ["Artificial photosynthesis Crossref fixture"],
				"abstract": "<jats:p>Deterministic Crossref abstract.</jats:p>",
				"published-print": {"date-parts": [[2026, 1, 2]]},
				"container-title": ["Journal of Test Fixtures"],
				"publisher": "Fixture Publisher",
				"URL": "https://doi.org/10.5555/crossref.example",
				"type": "journal-article",
				"reference-count": 12,
				"reference": [{"DOI":"10.1000/ref-one"}, {"article-title":"no doi"}],
				"funder": [{"name":"Fixture Foundation","award":["FF-1"]}],
				"license": [{"URL":"https://creativecommons.org/licenses/by/4.0/"}],
				"relation": {"is-preprint-of":[{"id-type":"doi","id":"10.5555/version-of-record"}]}
			}]}
		}`))
	}))
	defer server.Close()

	connector := NewCrossrefConnector(NewHTTPClient(HTTPClientOptions{BaseURL: server.URL, UserAgent: "ResearchForge/test", Timeout: time.Second}))
	response, err := connector.Search(context.Background(), SourceQuery{Terms: "artificial photosynthesis", Limit: 1})
	if err != nil {
		t.Fatalf("Search returned error: %v", err)
	}
	if connector.Name() != "crossref" {
		t.Fatalf("Name = %q", connector.Name())
	}
	if response.RawRef != "crossref:/works?query=artificial+photosynthesis&rows=1" {
		t.Fatalf("RawRef = %q", response.RawRef)
	}
	if len(response.Records) != 1 {
		t.Fatalf("len(records) = %d", len(response.Records))
	}
	record := response.Records[0]
	if record.Source != "crossref" || record.SourceID != "10.5555/crossref.example" || record.Title != "Artificial photosynthesis Crossref fixture" {
		t.Fatalf("record = %#v", record)
	}
	if record.Identifiers.DOI != "10.5555/crossref.example" || record.Identifiers.CrossrefID != "10.5555/crossref.example" || record.Year != 2026 {
		t.Fatalf("identifiers/year = %#v", record)
	}
	if record.Abstract != "Deterministic Crossref abstract." || len(record.URLs) != 1 || record.URLs[0] != "https://doi.org/10.5555/crossref.example" {
		t.Fatalf("text/urls = %#v", record)
	}
	if record.License != "https://creativecommons.org/licenses/by/4.0/" || !record.OpenAccess {
		t.Fatalf("license/open access = %#v", record)
	}
	if record.Metadata["type"] != "journal-article" || record.Metadata["reference_count"] != "12" || record.Metadata["reference_dois"] != "10.1000/ref-one" || record.Metadata["funders"] != "Fixture Foundation" || record.Metadata["funder_awards"] != "Fixture Foundation:FF-1" || record.Metadata["relations"] != "is-preprint-of:10.5555/version-of-record" {
		t.Fatalf("metadata = %#v", record.Metadata)
	}
	papers, err := PaperRecords(response)
	if err != nil {
		t.Fatalf("PaperRecords returned error: %v", err)
	}
	if papers[0].Identifiers.DOI != "10.5555/crossref.example" || papers[0].Identifiers.CrossrefID != "10.5555/crossref.example" {
		t.Fatalf("paper identifiers = %#v", papers[0].Identifiers)
	}
	if papers[0].Venue != "Journal of Test Fixtures" || papers[0].Publisher != "Fixture Publisher" {
		t.Fatalf("paper venue/publisher = %#v", papers[0])
	}
}
