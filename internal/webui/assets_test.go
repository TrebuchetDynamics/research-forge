package webui

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

func TestShellLoadsHTMXLocallyNotFromCDN(t *testing.T) {
	ts := httptest.NewServer(NewRouter(Config{}))
	defer ts.Close()

	body, status, _ := getURL(t, ts.URL+"/")
	if status != http.StatusOK {
		t.Fatalf("GET / status = %d", status)
	}
	if !strings.Contains(body, `src="/assets/htmx.min.js"`) {
		t.Fatalf("shell does not load vendored htmx: %s", body)
	}
	if strings.Contains(body, "unpkg.com") || strings.Contains(body, "integrity=") {
		t.Fatalf("shell still references a CDN / SRI hash (offline-breaking): %s", body)
	}
}

func TestHTMXAssetServed(t *testing.T) {
	ts := httptest.NewServer(NewRouter(Config{}))
	defer ts.Close()

	body, status, ctype := getURL(t, ts.URL+"/assets/htmx.min.js")
	if status != http.StatusOK {
		t.Fatalf("GET htmx status = %d", status)
	}
	if !strings.Contains(ctype, "javascript") {
		t.Fatalf("htmx content-type = %q", ctype)
	}
	if !strings.Contains(body, "htmx") || len(body) < 1000 {
		t.Fatalf("htmx asset looks wrong (len=%d)", len(body))
	}
}
