package cli

import (
	"bytes"
	"crypto/sha256"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"testing"
)

// Removing the engine route, substituting pre-normalized records, changing rank
// order, or using today's latency breaks this hand-scored public-command test.
func TestBenchmarkEngineReplay(t *testing.T) {
	args := []string{"benchmark", "run", "--set", "../benchmarks/testdata/fixture-tiny"}
	var first, second, stderr bytes.Buffer
	if code := Execute(args, &first, &stderr); code != 0 {
		t.Fatalf("code=%d: %s %s", code, first.String(), stderr.String())
	}
	if code := Execute(args, &second, &stderr); code != 0 || !bytes.Equal(first.Bytes(), second.Bytes()) {
		t.Fatalf("non-deterministic replay: %d\n%s\n%s", code, first.Bytes(), second.Bytes())
	}
	var r struct {
		Status, Purpose string
		Runs            []struct {
			Returned int
			Recall   struct {
				Numerator, Denominator int
				Value                  float64
			}
			AtK []struct {
				K         int
				Precision struct{ Value float64 } `json:"precision_at_k"`
			} `json:"at_k"`
			True     int    `json:"dedupe_true_merges"`
			False    int    `json:"dedupe_false_merges"`
			Requests int    `json:"http_requests"`
			Wall     int    `json:"wall_clock_ms"`
			Excluded int    `json:"date_excluded"`
			Citation string `json:"citation_status"`
		}
	}
	if err := json.Unmarshal(first.Bytes(), &r); err != nil {
		t.Fatal(err)
	}
	if r.Status != "complete" || r.Purpose != "fixture" || len(r.Runs) != 1 {
		t.Fatalf("bad report: %s", first.Bytes())
	}
	row := r.Runs[0]
	if row.Returned != 3 || row.Recall.Numerator != 1 || row.Recall.Denominator != 3 || row.True != 1 || row.False != 1 || row.Requests != 3 || row.Wall != 1080 || row.Excluded != 2 || row.Citation != "not_implemented" {
		t.Fatalf("wrong hand-scored result: %+v", row)
	}
	if len(row.AtK) != 2 || row.AtK[0].Precision.Value != 1 || row.AtK[1].Precision.Value != .2 {
		t.Fatalf("wrong fixed-K precision: %+v", row.AtK)
	}
	out := filepath.Join(t.TempDir(), "report.json")
	if e := os.WriteFile(out+".audit.json", []byte("prior audit"), 0644); e != nil {
		t.Fatal(e)
	}
	var stdout bytes.Buffer
	if code := Execute(append(args, "--out", out), &stdout, &stderr); code != 0 {
		t.Fatalf("out: %d %s", code, stderr.String())
	}
	data, err := os.ReadFile(out)
	if err != nil || !bytes.Equal(data, first.Bytes()) || stdout.Len() != 0 {
		t.Fatalf("canonical --out mismatch: %v %s", err, stdout.String())
	}
	if info, e := os.Stat(out + ".audit.json"); e != nil || info.Mode().Perm() != 0600 {
		t.Fatalf("audit must be private: %v %v", info, e)
	}
	t.Run("recorded CPU gap exhausts next call", func(t *testing.T) {
		dir := copyBenchmarkFixture(t)
		b, _ := os.ReadFile(filepath.Join(dir, "set.json"))
		b = bytes.ReplaceAll(b, []byte(`"max_wall_clock_ms":5000`), []byte(`"max_wall_clock_ms":1040`))
		if e := os.WriteFile(filepath.Join(dir, "set.json"), b, 0600); e != nil {
			t.Fatal(e)
		}
		b, _ = os.ReadFile(filepath.Join(dir, "snapshots/index.json"))
		var idx map[string]any
		if e := json.Unmarshal(b, &idx); e != nil {
			t.Fatal(e)
		}
		q := idx["questions"].([]any)[0].(map[string]any)
		q["exchanges"] = q["exchanges"].([]any)[:2]
		times := q["timing"].([]any)
		q["timing"] = append(times[:9], times[12:]...)
		q["stop_reason"] = "budget_exhausted"
		b, _ = json.Marshal(idx)
		if e := os.WriteFile(filepath.Join(dir, "snapshots/index.json"), b, 0600); e != nil {
			t.Fatal(e)
		}
		resignBenchmarkFixture(t, dir, "snapshots/index.json")
		var stdout, stderr bytes.Buffer
		if code := Execute([]string{"benchmark", "run", "--set", dir}, &stdout, &stderr); code != 0 {
			t.Fatalf("budget replay: %d %s %s", code, stdout.String(), stderr.String())
		}
		var report struct {
			Runs []struct {
				Requests int    `json:"http_requests"`
				Stop     string `json:"stop_reason"`
			}
		}
		if e := json.Unmarshal(stdout.Bytes(), &report); e != nil {
			t.Fatal(e)
		}
		if len(report.Runs) != 1 || report.Runs[0].Requests != 2 || report.Runs[0].Stop != "budget_exhausted" {
			t.Fatalf("lost partial output: %s", stdout.String())
		}
	})
}

func TestBenchmarkEngineRejectsInvalidInputs(t *testing.T) {
	for _, tc := range []struct{ name, path, old, new, want string }{
		{"leaked review", "snapshots/bodies/RESPONSE.bin", `"10.9999/p1"`, `"10.9999/review"`, "reference_review_leak"},
		{"body tamper", "snapshots/bodies/RESPONSE.bin", `"10.9999/p1"`, `"10.9999/changed"`, "checksum_mismatch"},
		{"manifest tamper", "set.json", `"max_http_requests":10`, `"max_http_requests":1`, "checksum_mismatch"},
		{"uncapped query", "set.json", `to_publication_date:2020-06-30`, `from_publication_date:2020-06-30`, "date_cap_unenforceable"},
		{"request mismatch", "snapshots/index.json", `per-page=10`, `per-page=11`, "snapshot_mismatch"},
		{"question gold hint", "set.json", `How does synthetic energy conversion work?`, `How does 10.9999/p1 work?`, "reference_review_leak"},
		{"gold filter hint", "set.json", `to_publication_date:2020-06-30`, `to_publication_date:2020-06-30,doi:https://doi.org/10.9999/p1`, "reference_review_leak"},
		{"conflicting cap", "set.json", `to_publication_date:2020-06-30`, `to_publication_date:2020-06-30,to_publication_date:2021-01-01`, "date_cap_unenforceable"},
		{"attempt timeout mismatch", "set.json", `"timeout_ms":10000`, `"timeout_ms":1`, "snapshot_mismatch"},
		{"unused backoff", "snapshots/index.json", `"retry_after_ms":0`, `"retry_after_ms":1`, "snapshot_mismatch"},
		{"retry policy mismatch", "set.json", `"max_retries":2`, `"max_retries":0`, "snapshot_mismatch"},
		{"label tamper", "snapshots/index.json", `"other-conflict"`, `"other-changed"`, "checksum_mismatch"},
	} {
		t.Run(tc.name, func(t *testing.T) {
			dir := copyBenchmarkFixture(t)
			path := tc.path
			if path == "snapshots/bodies/RESPONSE.bin" {
				path = fixtureResponsePath(t, dir)
			}
			full := filepath.Join(dir, path)
			data, err := os.ReadFile(full)
			if err != nil {
				t.Fatal(err)
			}
			changed := bytes.ReplaceAll(data, []byte(tc.old), []byte(tc.new))
			if bytes.Equal(data, changed) {
				t.Fatal("mutation did not apply")
			}
			if err := os.WriteFile(full, changed, 0600); err != nil {
				t.Fatal(err)
			}
			if tc.name != "body tamper" && tc.name != "manifest tamper" && tc.name != "label tamper" {
				resignBenchmarkFixture(t, dir, path)
			}
			var stdout, stderr bytes.Buffer
			if code := Execute([]string{"benchmark", "run", "--set", dir}, &stdout, &stderr); code != 1 || !bytes.Contains(stdout.Bytes(), []byte(tc.want)) {
				t.Fatalf("code=%d want=%s: %s %s", code, tc.want, stdout.String(), stderr.String())
			}
			if bytes.Contains(stdout.Bytes(), []byte(`"status":"complete"`)) {
				t.Fatal("invalid baseline published")
			}
		})
	}
}

// Fixture values are ASCII/integer-only, so encoding/json's sorted map keys
// suffice for independent test re-signing (not a general JCS implementation).
func resignBenchmarkFixture(t *testing.T, dir, changed string) {
	t.Helper()
	load := func(path string) map[string]any {
		b, e := os.ReadFile(filepath.Join(dir, path))
		if e != nil {
			t.Fatal(e)
		}
		var v map[string]any
		if e = json.Unmarshal(b, &v); e != nil {
			t.Fatal(e)
		}
		return v
	}
	digest := func(b []byte) string { return fmt.Sprintf("%x", sha256.Sum256(b)) }
	marshal := func(v any) []byte {
		var b bytes.Buffer
		enc := json.NewEncoder(&b)
		enc.SetEscapeHTML(false)
		if e := enc.Encode(v); e != nil {
			t.Fatal(e)
		}
		return bytes.TrimSuffix(b.Bytes(), []byte{'\n'})
	}
	save := func(path string, v any) string {
		b := marshal(v)
		if e := os.WriteFile(filepath.Join(dir, path), append(b, '\n'), 0600); e != nil {
			t.Fatal(e)
		}
		return digest(b)
	}
	m := load("set.json")
	idx := load("snapshots/index.json")
	if filepath.Ext(changed) == ".bin" {
		b, e := os.ReadFile(filepath.Join(dir, changed))
		if e != nil {
			t.Fatal(e)
		}
		hash := digest(b)
		if e = os.WriteFile(filepath.Join(dir, "snapshots/bodies/"+hash+".bin"), b, 0600); e != nil {
			t.Fatal(e)
		}
		idx["questions"].([]any)[0].(map[string]any)["exchanges"].([]any)[1].(map[string]any)["response"].(map[string]any)["body_sha256"] = hash
	}
	m["snapshots_sha256"] = save("snapshots/index.json", idx)
	delete(m, "set_checksum")
	m["set_checksum"] = digest(marshal(m))
	save("set.json", m)
}

func copyBenchmarkFixture(t *testing.T) string {
	t.Helper()
	root := "../benchmarks/testdata/fixture-tiny"
	dir := t.TempDir()
	err := filepath.WalkDir(root, func(path string, d os.DirEntry, err error) error {
		if err != nil {
			return err
		}
		rel, _ := filepath.Rel(root, path)
		dest := filepath.Join(dir, rel)
		if d.IsDir() {
			return os.MkdirAll(dest, 0700)
		}
		b, err := os.ReadFile(path)
		if err != nil {
			return err
		}
		return os.WriteFile(dest, b, 0600)
	})
	if err != nil {
		t.Fatal(err)
	}
	return dir
}

func fixtureResponsePath(t *testing.T, dir string) string {
	t.Helper()
	b, err := os.ReadFile(filepath.Join(dir, "snapshots/index.json"))
	if err != nil {
		t.Fatal(err)
	}
	var idx struct {
		Questions []struct {
			Exchanges []struct {
				Response struct {
					Hash string `json:"body_sha256"`
				}
			}
		}
	}
	if err := json.Unmarshal(b, &idx); err != nil {
		t.Fatal(err)
	}
	return "snapshots/bodies/" + idx.Questions[0].Exchanges[1].Response.Hash + ".bin"
}
