package adapters

import (
	"context"
	"errors"
	"io"
	"net"
	"net/http"
	"net/url"
	"strings"
	"testing"

	"github.com/TrebuchetDynamics/research-forge/internal/sources/core"
	"github.com/TrebuchetDynamics/research-forge/internal/sources/transport"
)

type pubMedErrorTransport func(*http.Request) (*http.Response, error)

func (f pubMedErrorTransport) RoundTrip(r *http.Request) (*http.Response, error) { return f(r) }

func TestPubMedSearchRedactsRequestErrors(t *testing.T) {
	fixtures := map[string]string{
		"esearch":  `{"esearchresult":{"idlist":["123456"]}}`,
		"esummary": `{"result":{"uids":["123456"],"123456":{"uid":"123456","title":"PubMed fixture"}}}`,
	}
	for _, key := range []string{"synthetic-pubmed-key", "synthetic+pubmed/key=with spaces&percent%"} {
		for _, endpoint := range []string{"esearch", "esummary", "efetch"} {
			for _, failure := range []struct {
				name    string
				cause   error
				context string
				timeout bool
			}{
				{"transport", errors.New("fixture transport failure: " + key + " / " + url.QueryEscape(key)), "fixture transport failure", false},
				{"redirect", errors.New("fixture redirect denied: " + key), "fixture redirect denied", false},
				{"canceled", context.Canceled, "context canceled", false},
				{"deadline", context.DeadlineExceeded, "context deadline exceeded", true},
				{"timeout", &net.DNSError{Err: "fixture timeout", IsTimeout: true}, "fixture timeout", true},
			} {
				t.Run(key+"/"+endpoint+"/"+failure.name, func(t *testing.T) {
					reached := false
					client := transport.NewHTTPClient(transport.HTTPClientOptions{
						BaseURL: "https://pubmed.invalid",
						Transport: pubMedErrorTransport(func(r *http.Request) (*http.Response, error) {
							if r.URL.Query().Get("api_key") != key {
								t.Fatal("outbound request lost its configured API key")
							}
							stage := strings.TrimSuffix(strings.TrimPrefix(r.URL.Path, "/entrez/eutils/"), ".fcgi")
							if stage == endpoint {
								reached = true
								if failure.name == "redirect" {
									return &http.Response{StatusCode: http.StatusFound, Header: http.Header{"Location": {r.URL.String() + "&redirected=true"}}, Body: io.NopCloser(strings.NewReader("")), Request: r}, nil
								}
								return nil, failure.cause
							}
							body, ok := fixtures[stage]
							if !ok {
								t.Fatalf("unexpected request stage: %s", stage)
							}
							return &http.Response{StatusCode: http.StatusOK, Header: http.Header{}, Body: io.NopCloser(strings.NewReader(body)), Request: r}, nil
						}),
						CheckRedirect: func(*http.Request, []*http.Request) error { return failure.cause },
					})
					connector := NewPubMedConnectorWithOptions(client, PubMedOptions{APIKey: key})
					_, err := connector.Search(context.Background(), core.SourceQuery{Terms: "fixture query", Limit: 1})
					if !reached || err == nil {
						t.Fatalf("target failure not reached: reached=%v, err=%v", reached, err)
					}
					for _, secret := range []string{key, url.QueryEscape(key), url.PathEscape(key)} {
						if strings.Contains(err.Error(), secret) {
							t.Errorf("request error exposes configured key: %s", err)
						}
					}
					if !strings.Contains(err.Error(), endpoint+".fcgi") || !strings.Contains(err.Error(), failure.context) {
						t.Errorf("request error lost operation/cause context: %s", err)
					}
					if !errors.Is(err, failure.cause) {
						t.Errorf("request error lost cause identity: %v", err)
					}
					var networkError net.Error
					if !errors.As(err, &networkError) || networkError.Timeout() != failure.timeout {
						t.Errorf("request error lost net.Error timeout classification: %v", err)
					}
					if cause, ok := failure.cause.(*net.DNSError); ok {
						var dnsError *net.DNSError
						if !errors.As(err, &dnsError) || dnsError != cause {
							t.Error("request error lost typed underlying cause")
						}
					}
				})
			}
		}
	}
}
