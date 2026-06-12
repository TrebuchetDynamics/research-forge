package parsing

import (
	"context"
	"net/http"
	"net/http/httptest"
	"strconv"
	"strings"
	"testing"
	"time"
)

func TestGROBIDClientRejectsOversizedTEIResponse(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Length", strconv.FormatInt(maxGROBIDTEIBytes+1, 10))
		_, _ = w.Write([]byte("<TEI></TEI>"))
	}))
	defer server.Close()
	client := NewGROBIDClient(GROBIDClientOptions{BaseURL: server.URL, Timeout: time.Second, Version: "0.8.0-test"})

	_, err := client.Parse(context.Background(), []byte("%PDF-1.4 fixture"), ParseOptions{PaperID: "paper-1"})
	if err == nil || !strings.Contains(err.Error(), "too large") {
		t.Fatalf("Parse error = %v, want too large", err)
	}
}

func TestGROBIDClientParsesMockTEIIntoParsedDocument(t *testing.T) {
	tei := `<TEI><teiHeader><fileDesc><titleStmt><title>Artificial photosynthesis TEI fixture</title><author><persName><forename>Ada</forename><surname>Lovelace</surname></persName></author></titleStmt><profileDesc><abstract><p>Deterministic abstract.</p></abstract></profileDesc></fileDesc></teiHeader><text><body><div><head>Introduction</head><p>Solar fuel catalysts split water.</p></div><div><head>Methods</head><p>We used deterministic fixtures.</p></div></body><back><listBibl><biblStruct><analytic><title>Referenced work</title></analytic></biblStruct></listBibl></back></text></TEI>`
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/api/processFulltextDocument" {
			t.Fatalf("path = %q", r.URL.Path)
		}
		_, _ = w.Write([]byte(tei))
	}))
	defer server.Close()
	client := NewGROBIDClient(GROBIDClientOptions{BaseURL: server.URL, Timeout: time.Second, Version: "0.8.0-test"})

	doc, err := client.Parse(context.Background(), []byte("%PDF-1.4 fixture"), ParseOptions{PaperID: "paper-1"})
	if err != nil {
		t.Fatalf("Parse returned error: %v", err)
	}
	if doc.SchemaVersion != "1" || doc.PaperID != "paper-1" || doc.ParserName != "grobid" || doc.ParserVersion != "0.8.0-test" {
		t.Fatalf("doc metadata = %#v", doc)
	}
	if doc.Title != "Artificial photosynthesis TEI fixture" || len(doc.Authors) != 1 || doc.Authors[0].Family != "Lovelace" || doc.Abstract != "Deterministic abstract." {
		t.Fatalf("doc header = %#v", doc)
	}
	if len(doc.Sections) != 2 || doc.Sections[0].ID != "paper-1-sec-1" || doc.Sections[0].Passages[0].ID != "paper-1-sec-1-p-1" {
		t.Fatalf("sections = %#v", doc.Sections)
	}
	if len(doc.References) != 1 || doc.References[0].Title != "Referenced work" {
		t.Fatalf("references = %#v", doc.References)
	}
}

func TestGROBIDClientReportsTimeoutAndParserWarnings(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		time.Sleep(50 * time.Millisecond)
	}))
	defer server.Close()
	client := NewGROBIDClient(GROBIDClientOptions{BaseURL: server.URL, Timeout: time.Millisecond, Version: "0.8.0-test"})
	_, err := client.Parse(context.Background(), []byte("%PDF"), ParseOptions{PaperID: "paper-1"})
	if err == nil {
		t.Fatalf("Parse returned nil error, want timeout")
	}
}
