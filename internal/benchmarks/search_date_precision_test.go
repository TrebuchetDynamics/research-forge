package benchmarks

import (
	"encoding/json"
	"os"
	"path/filepath"
	"testing"
)

// writeTinyVariant copies fixture-tiny, lets edit change the first question's
// review and gold, and re-seals set_checksum so only the edit is under test.
func writeTinyVariant(t *testing.T, edit func(review, gold0 map[string]any)) string {
	return writeTinyVariantPlan(t, edit, "")
}

// writeTinyVariantPlan also re-caps the first question's search plan when capDate is set.
func writeTinyVariantPlan(t *testing.T, edit func(review, gold0 map[string]any), capDate string) string {
	t.Helper()
	dir := t.TempDir()
	for _, name := range []string{"review.md", "set.json"} {
		b, err := os.ReadFile(filepath.Join("testdata", "fixture-tiny", name))
		if err != nil {
			t.Fatal(err)
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
	q := set["questions"].([]any)[0].(map[string]any)
	edit(q["gold_source_review"].(map[string]any), q["gold"].([]any)[0].(map[string]any))
	if capDate != "" {
		for _, call := range q["search_plan"].([]any) {
			call.(map[string]any)["filters"].(map[string]any)["filter"] = "to_publication_date:" + capDate
		}
	}
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

func TestSearchDatePrecision(t *testing.T) {
	cases := []struct {
		name    string
		edit    func(review, gold0 map[string]any)
		capDate string
		wantErr string
	}{
		{"day precision is the default", func(review, _ map[string]any) {}, "", ""},
		{"explicit day precision", func(review, _ map[string]any) { review["search_date_precision"] = "day" }, "", ""},
		{"month precision caps at the first day of the month", func(review, _ map[string]any) {
			review["search_date_precision"] = "month"
			review["search_date"] = "2020-06-01"
		}, "2020-06-01", ""},
		{"month precision still requires plans capped at the first day", func(review, _ map[string]any) {
			review["search_date_precision"] = "month"
			review["search_date"] = "2020-06-01"
		}, "", "date_cap_unenforceable"},
		{"month precision rejects a day other than the first", func(review, _ map[string]any) {
			review["search_date_precision"] = "month"
			review["search_date"] = "2020-06-15"
		}, "2020-06-15", "invalid_set"},
		{"year precision is not conforming", func(review, _ map[string]any) {
			review["search_date_precision"] = "year"
			review["search_date"] = "2020-01-01"
		}, "2020-01-01", "invalid_set"},
		{"gold published within the capped month must be dropped", func(review, gold0 map[string]any) {
			review["search_date_precision"] = "month"
			review["search_date"] = "2019-01-01"
			gold0["publication_date"] = "2019-01"
		}, "2019-01-01", "invalid_set"},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			_, err := loadSet(writeTinyVariantPlan(t, tc.edit, tc.capDate), false)
			got := ""
			if err != nil {
				got = err.Error()
			}
			if got != tc.wantErr {
				t.Fatalf("loadSet error = %q, want %q", got, tc.wantErr)
			}
		})
	}
}
