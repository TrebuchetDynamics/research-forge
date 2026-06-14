package sources

import (
	"context"
	"os"
	"strings"
	"testing"
	"time"
)

// liveSmokeHTTPClient builds an HTTP client for a live source endpoint, honoring
// the same RFORGE_*_URL override the CLI uses and falling back to the public
// default.
func liveSmokeHTTPClient(baseURLEnv, defaultURL string) HTTPClient {
	baseURL := os.Getenv(baseURLEnv)
	if baseURL == "" {
		baseURL = defaultURL
	}
	return NewHTTPClient(HTTPClientOptions{BaseURL: baseURL, UserAgent: "ResearchForge-smoke/dev", Timeout: 30 * time.Second, MaxRetries: 2})
}

// TestOptInLiveSourceConnectorSmoke runs lightweight live queries against the
// public scholarly source APIs to confirm the connectors work end-to-end against
// real services. It is opt-in (RFORGE_RUN_LIVE_SOURCE_SMOKE=1) and never runs in
// the normal, network-free suite. Base URLs may be overridden with the same
// RFORGE_*_URL env vars the CLI uses; the Unpaywall subtest additionally
// requires RFORGE_UNPAYWALL_EMAIL.
func TestOptInLiveSourceConnectorSmoke(t *testing.T) {
	if os.Getenv("RFORGE_RUN_LIVE_SOURCE_SMOKE") != "1" {
		t.Skip("set RFORGE_RUN_LIVE_SOURCE_SMOKE=1 to run live source connector smoke tests")
	}

	searchConnectors := []struct {
		name       string
		baseEnv    string
		defaultURL string
		build      func(HTTPClient) SourceConnector
	}{
		{"openalex", "RFORGE_OPENALEX_URL", "https://api.openalex.org", func(c HTTPClient) SourceConnector { return NewOpenAlexConnector(c) }},
		{"arxiv", "RFORGE_ARXIV_URL", "https://export.arxiv.org", func(c HTTPClient) SourceConnector { return NewArXivConnector(c) }},
		{"crossref", "RFORGE_CROSSREF_URL", "https://api.crossref.org", func(c HTTPClient) SourceConnector { return NewCrossrefConnector(c) }},
	}
	for _, sc := range searchConnectors {
		sc := sc
		t.Run(sc.name, func(t *testing.T) {
			ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
			defer cancel()
			response, err := sc.build(liveSmokeHTTPClient(sc.baseEnv, sc.defaultURL)).Search(ctx, SourceQuery{Terms: "artificial photosynthesis", Limit: 3})
			if err != nil {
				t.Fatalf("%s live search failed: %v", sc.name, err)
			}
			if len(response.Records) == 0 {
				t.Fatalf("%s live search returned no records", sc.name)
			}
			for _, record := range response.Records {
				if strings.TrimSpace(record.Title) != "" {
					return
				}
			}
			t.Fatalf("%s live search returned records without titles: %#v", sc.name, response.Records)
		})
	}

	t.Run("unpaywall", func(t *testing.T) {
		email := os.Getenv("RFORGE_UNPAYWALL_EMAIL")
		if email == "" {
			t.Skip("set RFORGE_UNPAYWALL_EMAIL to run the Unpaywall live smoke test")
		}
		doi := os.Getenv("RFORGE_UNPAYWALL_E2E_DOI")
		if doi == "" {
			doi = "10.1371/journal.pone.0000308" // a long-standing open-access PLOS ONE article
		}
		ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
		defer cancel()
		record, err := NewUnpaywallConnector(liveSmokeHTTPClient("RFORGE_UNPAYWALL_URL", "https://api.unpaywall.org"), email).LookupDOI(ctx, doi)
		if err != nil {
			t.Fatalf("unpaywall live lookup failed: %v", err)
		}
		if !strings.EqualFold(strings.TrimSpace(record.DOI), strings.TrimSpace(doi)) {
			t.Fatalf("unpaywall returned DOI %q, want %q", record.DOI, doi)
		}
	})
}
