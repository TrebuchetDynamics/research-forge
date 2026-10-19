package benchmarks

import (
	"encoding/json"
	"os"
	"path/filepath"
	"testing"
)

// approvedPoolVariant copies the real candidate pool, approves it with the
// given waivers and re-seals set_checksum, so only the waiver is under test.
func approvedPoolVariant(t *testing.T, waivers []any) string {
	t.Helper()
	src := filepath.Join("..", "..", "benchmarks", "sets", "candidates", "pool-a")
	dir := t.TempDir()
	for _, name := range []string{"review.md", "set.json"} {
		b, err := os.ReadFile(filepath.Join(src, name))
		if err != nil {
			t.Skipf("candidate pool not present: %v", err)
		}
		if err := os.WriteFile(filepath.Join(dir, name), b, 0o644); err != nil {
			t.Fatal(err)
		}
	}
	b, err := os.ReadFile(filepath.Join(dir, "set.json"))
	if err != nil {
		t.Fatal(err)
	}
	parsed, err := parseJSON(b)
	if err != nil {
		t.Fatal(err)
	}
	set := parsed.(map[string]any)
	approval := map[string]any{"approver": "Owner", "approved_at": "2026-10-19T16:00:00Z"}
	if waivers != nil {
		approval["waivers"] = waivers
	}
	set["state"] = "approved"
	set["approval"] = approval
	delete(set, "set_checksum")
	c, err := canonical(set)
	if err != nil {
		t.Fatal(err)
	}
	set["set_checksum"] = digest(c)
	out, err := json.Marshal(set)
	if err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(dir, "set.json"), out, 0o644); err != nil {
		t.Fatal(err)
	}
	return dir
}

func TestApprovalWaiverForAbsentArtificialPhotosynthesis(t *testing.T) {
	cases := []struct {
		name    string
		waivers []any
		wantErr string
	}{
		{"approved pool without AP needs a waiver", nil, "invalid_set"},
		{"owner waiver admits the documented AP fallback", []any{"ap_domain_absent"}, ""},
		{"unknown waivers are rejected", []any{"ap_domain_absent", "skip_leakage_checks"}, "invalid_set"},
		{"duplicate waivers are rejected", []any{"ap_domain_absent", "ap_domain_absent"}, "invalid_set"},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			_, err := loadSet(approvedPoolVariant(t, tc.waivers), true)
			got := ""
			if err != nil {
				got = err.Error()
			}
			if got != tc.wantErr {
				t.Fatalf("loadSet(record) error = %q, want %q", got, tc.wantErr)
			}
		})
	}
}
