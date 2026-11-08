package webui

import (
	"encoding/xml"
	"io"
	"net/http"
	"net/http/httptest"
	"net/url"
	"reflect"
	"strings"
	"testing"
)

// TestSearchFormServedTarget pins the form at a real route: /search/results is
// not registered by NewRouter, so the form must submit natively to /search and
// no control may request the missing route.
func TestSearchFormServedTarget(t *testing.T) {
	router := NewRouter(Config{})

	rec := httptest.NewRecorder()
	router.ServeHTTP(rec, httptest.NewRequest(http.MethodGet, "/search/results?query=paper", nil))
	if rec.Code != http.StatusNotFound {
		t.Fatalf("GET /search/results status = %d, want 404", rec.Code)
	}

	rec = httptest.NewRecorder()
	router.ServeHTTP(rec, httptest.NewRequest(http.MethodGet, "/search", nil))
	if rec.Code != http.StatusOK {
		t.Fatalf("GET /search status = %d: %s", rec.Code, rec.Body.String())
	}
	body := rec.Body.String()
	if !strings.Contains(body, `<form method="get" action="/search">`) {
		t.Errorf("search form must submit natively to /search:\n%s", body)
	}
	if strings.Contains(body, "/search/results") {
		t.Errorf("search form still targets the unregistered /search/results route:\n%s", body)
	}
}

// TestSearchFormRoundTrip resubmits the rendered form and checks that the query
// and chosen sources survive, and that query text cannot inject markup.
func TestSearchFormRoundTrip(t *testing.T) {
	router := NewRouter(Config{})
	render := func(t *testing.T, values url.Values) string {
		t.Helper()
		rec := httptest.NewRecorder()
		router.ServeHTTP(rec, httptest.NewRequest(http.MethodGet, "/search?"+values.Encode(), nil))
		if rec.Code != http.StatusOK {
			t.Fatalf("GET /search status = %d: %s", rec.Code, rec.Body.String())
		}
		return rec.Body.String()
	}

	values := url.Values{"query": {"artificial photosynthesis"}, "source": {"openalex"}}
	for submission := 0; submission < 2; submission++ {
		body := render(t, values)
		query, checked := searchFormSubmission(t, body)
		if query != values.Get("query") {
			t.Errorf("submission %d query = %q, want %q", submission, query, values.Get("query"))
		}
		if !reflect.DeepEqual(checked, []string{"openalex"}) {
			t.Errorf("submission %d checked sources = %v, want [openalex]", submission, checked)
		}
		values = url.Values{"query": {query}, "source": checked}
	}

	injected := url.Values{"query": {`" ><input name="injected"><script>alert(1)</script>`}, "source": {"openalex"}}
	body := render(t, injected)
	if query, _ := searchFormSubmission(t, body); query != injected.Get("query") {
		t.Errorf("escaped query = %q, want %q", query, injected.Get("query"))
	}
	if strings.Contains(body, `name="injected"`) || strings.Contains(body, "<script>alert(1)") {
		t.Errorf("query text injected markup:\n%s", body)
	}
}

// searchFormSubmission decodes the native search form and returns the query
// value and the sources whose checkbox is checked.
func searchFormSubmission(t *testing.T, body string) (string, []string) {
	t.Helper()
	start, end := strings.Index(body, "<form"), strings.Index(body, "</form>")
	if start < 0 || end < start {
		t.Fatalf("missing search form in:\n%s", body)
	}
	decoder := xml.NewDecoder(strings.NewReader(body[start : end+len("</form>")]))
	decoder.Strict, decoder.AutoClose, decoder.Entity = false, xml.HTMLAutoClose, xml.HTMLEntity
	var query string
	var checked []string
	for {
		token, err := decoder.Token()
		if err == io.EOF {
			break
		}
		if err != nil {
			t.Fatalf("parse search form: %v", err)
		}
		start, ok := token.(xml.StartElement)
		if !ok || start.Name.Local != "input" {
			continue
		}
		attrs := map[string]string{}
		for _, attr := range start.Attr {
			attrs[attr.Name.Local] = attr.Value
		}
		switch attrs["name"] {
		case "query":
			query = attrs["value"]
		case "source":
			if _, selected := attrs["checked"]; selected {
				checked = append(checked, attrs["value"])
			}
		}
	}
	return query, checked
}
