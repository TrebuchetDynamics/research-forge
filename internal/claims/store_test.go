package claims_test

import (
	"bytes"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/TrebuchetDynamics/research-forge/internal/claims"
)

// Ingestion must reject structurally invalid records, not persist unusable IDs,
// evidence locations, or incomplete assessments for future mechanical checks.
func TestStoreRejectsInvalidRecords(t *testing.T) {
	for _, tc := range []struct {
		name string
		edit func(*claims.Claim)
	}{
		{"schema", func(c *claims.Claim) { c.SchemaVersion = "future" }},
		{"id", func(c *claims.Claim) { c.ID = " " }},
		{"text", func(c *claims.Claim) { c.Text = " " }},
		{"source scheme", func(c *claims.Claim) { c.Evidence[0].Identifier.Scheme = " " }},
		{"source value", func(c *claims.Claim) { c.Evidence[0].Identifier.Value = " " }},
		{"relation", func(c *claims.Claim) { c.Evidence[0].Relation = "unknown" }},
		{"hash", func(c *claims.Claim) { c.Evidence[0].ContentHash = "sha256:bad" }},
		{"offset bounds", func(c *claims.Claim) {
			c.Evidence[0].Offsets = &claims.TextOffsets{Start: 4, End: 1, ParserName: "parser", ParserVersion: "1"}
		}},
		{"offset parser", func(c *claims.Claim) { c.Evidence[0].Offsets = &claims.TextOffsets{End: 1} }},
		{"assessment", func(c *claims.Claim) { c.Facets.SourceIdentity = &claims.Assessment{} }},
	} {
		t.Run(tc.name, func(t *testing.T) {
			project := t.TempDir()
			record := claims.Claim{SchemaVersion: "0", ID: "c1", Text: "Assertion.", Evidence: []claims.EvidenceReference{{Identifier: claims.Identifier{Scheme: "local", Value: "p1"}, Relation: claims.RelationSupports}}}
			tc.edit(&record)
			if err := claims.Append(project, record); err == nil {
				t.Fatal("accepted invalid claim")
			}
			if _, err := os.Stat(filepath.Join(project, "data")); !os.IsNotExist(err) {
				t.Fatalf("invalid input touched store: %v", err)
			}
		})
	}
}

// Ignoring a broken prior line would hide lost history or concatenate JSON.
func TestStoreRefusesCorruptLedger(t *testing.T) {
	for _, content := range []string{
		"{broken}\n",
		`{"schemaVersion":"0","id":"c1","text":"unterminated","evidence":[]}`,
		`{"schemaVersion":"0","id":"c1","text":"bad attribution","evidence":[],"facets":{"semanticSupport":{"state":"passed","checker":{"kind":"engine","id":"rforge","model":"m1"},"timestamp":"2026-10-19T12:00:00Z"}}}` + "\n",
	} {
		project := t.TempDir()
		path := filepath.Join(project, "data", "claims.jsonl")
		if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(path, []byte(content), 0o600); err != nil {
			t.Fatal(err)
		}
		if _, err := claims.List(project); err == nil {
			t.Fatal("list hid corrupt ledger")
		}
		if _, err := claims.Show(project, "c1"); err == nil {
			t.Fatal("show hid corrupt ledger")
		}
		if err := claims.Append(project, claims.Claim{SchemaVersion: "0", ID: "c2", Text: "New assertion."}); err == nil {
			t.Fatal("appended to corrupt ledger")
		}
		data, err := os.ReadFile(path)
		if err != nil || string(data) != content {
			t.Fatalf("changed corrupt ledger: %q, %v", data, err)
		}
	}
}

// Bypassing filetxn would write through links into another project's ledger.
func TestStoreDoesNotWriteThroughLinks(t *testing.T) {
	for _, kind := range []string{"symlink", "parent symlink", "hardlink"} {
		t.Run(kind, func(t *testing.T) {
			project, outside := t.TempDir(), t.TempDir()
			original := []byte("{\"schemaVersion\":\"0\",\"id\":\"old\",\"text\":\"Original.\",\"evidence\":[]}\n")
			source := filepath.Join(outside, "claims.jsonl")
			if err := os.WriteFile(source, original, 0o600); err != nil {
				t.Fatal(err)
			}
			parent := filepath.Join(project, "data")
			if kind == "parent symlink" {
				if err := os.Symlink(outside, parent); err != nil {
					t.Skip(err)
				}
			} else {
				if err := os.Mkdir(parent, 0o755); err != nil {
					t.Fatal(err)
				}
				link := os.Symlink
				if kind == "hardlink" {
					link = os.Link
				}
				if err := link(source, filepath.Join(parent, "claims.jsonl")); err != nil {
					t.Skip(err)
				}
			}
			err := claims.Append(project, claims.Claim{SchemaVersion: "0", ID: "new", Text: "New assertion."})
			if (err == nil) != (kind == "hardlink") {
				t.Fatalf("append error: %v", err)
			}
			got, err := os.ReadFile(source)
			if err != nil || !bytes.Equal(got, original) {
				t.Fatalf("modified link target: %q, %v", got, err)
			}
		})
	}
}

// A scanner's default token limit would make a successfully appended quote unreadable.
func TestStoreReadsLargeEvidence(t *testing.T) {
	project := t.TempDir()
	quote := strings.Repeat("evidence ", 9000)
	record := claims.Claim{SchemaVersion: "0", ID: "large", Text: "Assertion.", Evidence: []claims.EvidenceReference{{Identifier: claims.Identifier{Scheme: "local", Value: "p1"}, Relation: claims.RelationContext, ExactText: quote}}}
	if err := claims.Append(project, record); err != nil {
		t.Fatal(err)
	}
	got, err := claims.Show(project, "large")
	if err != nil || len(got.Evidence) != 1 || got.Evidence[0].ExactText != quote {
		t.Fatalf("lost large quote: %v", err)
	}
}
