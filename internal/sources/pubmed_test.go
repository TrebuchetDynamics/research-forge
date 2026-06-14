package sources

import (
	"context"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"
)

func TestPubMedSearchAddsNCBIIdentificationWithoutLeakingAPIKeyToRawRef(t *testing.T) {
	seen := map[string]bool{}
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		seen[r.URL.Path] = true
		query := r.URL.Query()
		if query.Get("api_key") != "secret-key" || query.Get("tool") != "research-forge-tests" || query.Get("email") != "dev@example.org" {
			t.Fatalf("missing NCBI identification params on %s: %s", r.URL.Path, r.URL.RawQuery)
		}
		switch r.URL.Path {
		case "/entrez/eutils/esearch.fcgi":
			_, _ = w.Write([]byte(`{"esearchresult":{"idlist":["123456"]}}`))
		case "/entrez/eutils/esummary.fcgi":
			_, _ = w.Write([]byte(`{"result":{"uids":["123456"],"123456":{"uid":"123456","title":"PubMed fixture","articleids":[]}}}`))
		case "/entrez/eutils/efetch.fcgi":
			_, _ = w.Write([]byte(`<PubmedArticleSet><PubmedArticle><MedlineCitation><PMID>123456</PMID></MedlineCitation></PubmedArticle></PubmedArticleSet>`))
		default:
			t.Fatalf("unexpected path: %s", r.URL.Path)
		}
	}))
	defer server.Close()
	connector := NewPubMedConnectorWithOptions(NewHTTPClient(HTTPClientOptions{BaseURL: server.URL, Timeout: time.Second}), PubMedOptions{APIKey: " secret-key ", Tool: " research-forge-tests ", Email: " dev@example.org "})
	response, err := connector.Search(context.Background(), SourceQuery{Terms: "machine learning", Limit: 1})
	if err != nil {
		t.Fatalf("Search returned error: %v", err)
	}
	if !seen["/entrez/eutils/esearch.fcgi"] || !seen["/entrez/eutils/esummary.fcgi"] || !seen["/entrez/eutils/efetch.fcgi"] {
		t.Fatalf("expected both ESearch and ESummary, saw %#v", seen)
	}
	if strings.Contains(response.RawRef, "secret-key") || !strings.Contains(response.RawRef, "tool=research-forge-tests") || !strings.Contains(response.RawRef, "email=dev%40example.org") {
		t.Fatalf("RawRef should preserve non-secret NCBI provenance only: %s", response.RawRef)
	}
}

func TestPubMedSearchUsesESearchAndESummary(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case "/entrez/eutils/esearch.fcgi":
			if r.URL.Query().Get("term") != "machine learning" || r.URL.Query().Get("retmax") != "2" {
				t.Fatalf("esearch query = %s", r.URL.RawQuery)
			}
			_, _ = w.Write([]byte(`{"esearchresult":{"idlist":["123456"]}}`))
		case "/entrez/eutils/esummary.fcgi":
			if r.URL.Query().Get("id") != "123456" {
				t.Fatalf("esummary query = %s", r.URL.RawQuery)
			}
			_, _ = w.Write([]byte(`{"result":{"uids":["123456"],"123456":{"uid":"123456","title":"PubMed machine learning fixture","fulljournalname":"Journal of PubMed Fixtures","pubdate":"2026 Jun","articleids":[{"idtype":"doi","value":"10.1000/pubmed"},{"idtype":"pmc","value":"PMC1234567"}]}}}`))
		case "/entrez/eutils/efetch.fcgi":
			if r.URL.Query().Get("retmode") != "xml" || r.URL.Query().Get("id") != "123456" {
				t.Fatalf("efetch query = %s", r.URL.RawQuery)
			}
			_, _ = w.Write([]byte(`<PubmedArticleSet><PubmedArticle><MedlineCitation><PMID>123456</PMID><MeshHeadingList><MeshHeading><DescriptorName>Machine Learning</DescriptorName></MeshHeading><MeshHeading><DescriptorName>Biomedical Research</DescriptorName></MeshHeading></MeshHeadingList></MedlineCitation></PubmedArticle></PubmedArticleSet>`))
		default:
			t.Fatalf("unexpected path: %s", r.URL.Path)
		}
	}))
	defer server.Close()
	connector := NewPubMedConnector(NewHTTPClient(HTTPClientOptions{BaseURL: server.URL, Timeout: time.Second}))
	response, err := connector.Search(context.Background(), SourceQuery{Terms: "machine learning", Limit: 2})
	if err != nil {
		t.Fatalf("Search returned error: %v", err)
	}
	if len(response.Records) != 1 {
		t.Fatalf("records = %d, want 1", len(response.Records))
	}
	record := response.Records[0]
	if record.Source != "pubmed" || record.Identifiers.PMID != "123456" || record.Identifiers.PMCID != "PMC1234567" || record.Identifiers.DOI != "10.1000/pubmed" || record.Year != 2026 || record.Metadata["mesh_terms"] != "Machine Learning; Biomedical Research" {
		t.Fatalf("record = %#v", record)
	}
	papers, err := PaperRecords(response)
	if err != nil {
		t.Fatalf("PaperRecords returned error: %v", err)
	}
	if papers[0].Identifiers.PMID != "123456" || papers[0].Identifiers.PMCID != "PMC1234567" {
		t.Fatalf("papers = %#v", papers)
	}
}
