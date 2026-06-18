package sources

import (
	"context"
	"fmt"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"
)

func TestPubChemSearchNormalizesRecords(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case "/rest/pug/compound/name/aspirin/cids/JSON":
			_, _ = w.Write([]byte(`{"IdentifierList":{"CID":[2244,1983]}}`))
		case "/rest/pug/compound/cid/2244,1983/property/IUPACName,MolecularFormula,MolecularWeight,InChIKey,CanonicalSMILES/JSON":
			_, _ = w.Write([]byte(`{"PropertyTable":{"Properties":[
				{"CID":2244,"IUPACName":"2-acetoxybenzoic acid","MolecularFormula":"C9H8O4","MolecularWeight":"180.16","InChIKey":"BSYNRYMUTXBXSQ-UHFFFAOYSA-N","CanonicalSMILES":"CC(=O)Oc1ccccc1C(=O)O"},
				{"CID":1983,"IUPACName":"acetylsalicylate","MolecularFormula":"C9H7O4","MolecularWeight":"179.15","InChIKey":"ABCDEFGHIJKLMN-UHFFFAOYSA-N","CanonicalSMILES":"CC(=O)Oc1ccccc1C(=O)[O-]"}
			]}}`))
		default:
			t.Fatalf("unexpected path: %s", r.URL.Path)
		}
	}))
	defer server.Close()

	connector := NewPubChemConnector(NewHTTPClient(HTTPClientOptions{BaseURL: server.URL, Timeout: time.Second}))
	response, err := connector.Search(context.Background(), SourceQuery{Terms: "aspirin", Limit: 10})
	if err != nil {
		t.Fatalf("Search error: %v", err)
	}
	wantRawRef := "pubchem:/rest/pug/compound/name/aspirin/cids/JSON"
	if response.RawRef != wantRawRef {
		t.Fatalf("RawRef = %q, want %q", response.RawRef, wantRawRef)
	}
	if len(response.Records) != 2 {
		t.Fatalf("records = %d, want 2", len(response.Records))
	}
	r := response.Records[0]
	if r.Source != "pubchem" {
		t.Fatalf("Source = %q", r.Source)
	}
	if r.SourceID != "2244" {
		t.Fatalf("SourceID = %q, want 2244", r.SourceID)
	}
	if r.Identifiers.CrossrefID != "pubchem:2244" {
		t.Fatalf("CrossrefID = %q", r.Identifiers.CrossrefID)
	}
	if r.Identifiers.DOI != "" {
		t.Fatalf("DOI = %q, want empty", r.Identifiers.DOI)
	}
	if r.Title != "2-acetoxybenzoic acid" {
		t.Fatalf("Title = %q", r.Title)
	}
	if !r.OpenAccess {
		t.Fatal("OpenAccess = false, want true")
	}
	if len(r.URLs) != 1 || r.URLs[0] != "https://pubchem.ncbi.nlm.nih.gov/compound/2244" {
		t.Fatalf("URLs = %v", r.URLs)
	}
	if r.Metadata["molecular_formula"] != "C9H8O4" {
		t.Fatalf("molecular_formula = %q", r.Metadata["molecular_formula"])
	}
	if r.Metadata["inchikey"] != "BSYNRYMUTXBXSQ-UHFFFAOYSA-N" {
		t.Fatalf("inchikey = %q", r.Metadata["inchikey"])
	}
	papers, err := PaperRecords(response)
	if err != nil {
		t.Fatalf("PaperRecords error: %v", err)
	}
	if papers[0].Title != "2-acetoxybenzoic acid" {
		t.Fatalf("papers[0].Title = %q", papers[0].Title)
	}
}

func TestPubChemSearchEmptyResults(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		_, _ = w.Write([]byte(`{"IdentifierList":{"CID":[]}}`))
	}))
	defer server.Close()

	connector := NewPubChemConnector(NewHTTPClient(HTTPClientOptions{BaseURL: server.URL, Timeout: time.Second}))
	response, err := connector.Search(context.Background(), SourceQuery{Terms: "xyznotacompound"})
	if err != nil {
		t.Fatalf("Search error: %v", err)
	}
	if len(response.Records) != 0 {
		t.Fatalf("records = %d, want 0", len(response.Records))
	}
}

func TestPubChemSearchDefaultLimit(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case "/rest/pug/compound/name/ibuprofen/cids/JSON":
			// Return 30 CIDs to verify limit is applied.
			cids := "["
			for i := 1; i <= 30; i++ {
				if i > 1 {
					cids += ","
				}
				cids += fmt.Sprintf("%d", 3000+i)
			}
			cids += "]"
			_, _ = w.Write([]byte(`{"IdentifierList":{"CID":` + cids + `}}`))
		default:
			// Properties batch for 25 CIDs (default limit).
			props := `{"PropertyTable":{"Properties":[`
			for i := 0; i < 25; i++ {
				if i > 0 {
					props += ","
				}
				cid := 3001 + i
				props += fmt.Sprintf(`{"CID":%d,"IUPACName":"compound%d","MolecularFormula":"C1H1","MolecularWeight":"13.0","InChIKey":"ABC","CanonicalSMILES":"C"}`, cid, cid)
			}
			props += `]}}`
			_, _ = w.Write([]byte(props))
		}
	}))
	defer server.Close()

	connector := NewPubChemConnector(NewHTTPClient(HTTPClientOptions{BaseURL: server.URL, Timeout: time.Second}))
	response, err := connector.Search(context.Background(), SourceQuery{Terms: "ibuprofen"})
	if err != nil {
		t.Fatalf("Search error: %v", err)
	}
	if len(response.Records) != 25 {
		t.Fatalf("records = %d, want 25 (default limit applied)", len(response.Records))
	}
}
