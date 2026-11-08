package cli

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
	"reflect"
	"strings"
	"testing"
)

func TestSearchStatsFailuresReadStrict(t *testing.T) {
	const first = `{"source":"openalex","query":"first query","error":"timeout"}`
	const second = `{"source":"unregistered","query":"second query","error":"source unavailable"}`
	rows := []searchBatchFailure{
		{Source: "openalex", Query: "first query", Error: "timeout"},
		{Source: "unregistered", Query: "second query", Error: "source unavailable"},
	}
	for _, mode := range []string{"text", "json"} {
		for _, tc := range []struct {
			name       string
			content    string
			diagnostic string
			want       []searchBatchFailure
		}{
			{name: "directory", diagnostic: "not a regular file"},
			{name: "malformed", content: "garbage\n", diagnostic: "decode failures.jsonl line 1"},
			{name: "partial", content: first + "\n{\n", diagnostic: "decode failures.jsonl line 2"},
			{name: "symlink", diagnostic: "not a regular file"},
			{name: "dangling symlink", diagnostic: "not a regular file"},
			{name: "missing directory"},
			{name: "missing ledger"},
			{name: "empty"},
			{name: "blank", content: "\n \t\r\n"},
			{name: "valid rows", content: first + "\n\n" + second + "\n", want: rows},
		} {
			t.Run(mode+"/"+tc.name, func(t *testing.T) {
				root := t.TempDir()
				dir := filepath.Join(root, "topic")
				if tc.name != "missing directory" {
					if err := os.Mkdir(dir, 0o755); err != nil {
						t.Fatal(err)
					}
				}
				path := filepath.Join(dir, "failures.jsonl")
				switch tc.name {
				case "missing directory", "missing ledger":
				case "directory":
					if err := os.Mkdir(path, 0o755); err != nil {
						t.Fatal(err)
					}
				case "symlink", "dangling symlink":
					target := filepath.Join(root, "sentinel.jsonl")
					if tc.name == "symlink" {
						if err := os.WriteFile(target, []byte(first+"\n"), 0o600); err != nil {
							t.Fatal(err)
						}
					}
					if err := os.Symlink(target, path); err != nil {
						t.Fatal(err)
					}
				default:
					writeSearchFailures(t, dir, tc.content)
				}

				// Include directories, links and the sentinel: stats must not create
				// jobs/locks or change any topic artifacts or symlink target bytes.
				snapshot := func() map[string]string {
					t.Helper()
					entries := map[string]string{}
					err := filepath.WalkDir(root, func(path string, entry fs.DirEntry, err error) error {
						if err != nil {
							return err
						}
						info, err := entry.Info()
						if err != nil {
							return err
						}
						var content string
						if entry.Type()&os.ModeSymlink != 0 {
							content, err = os.Readlink(path)
						} else if !entry.IsDir() {
							var data []byte
							data, err = os.ReadFile(path)
							content = string(data)
						}
						entries[path] = info.Mode().String() + ":" + content
						return err
					})
					if err != nil {
						t.Fatal(err)
					}
					return entries
				}
				before := snapshot()
				t.Cleanup(func() {
					if after := snapshot(); !reflect.DeepEqual(after, before) {
						t.Errorf("stats changed topic tree or sentinel: before=%v after=%v", before, after)
					}
				})

				args := []string{"search", "stats", "--dir", dir}
				if mode == "json" {
					args = append([]string{"--json"}, args...)
				}
				stdout, stderr := new(bytes.Buffer), new(bytes.Buffer)
				code := Execute(args, stdout, stderr)
				wantCode := 0
				if tc.diagnostic != "" {
					wantCode = 1
				}
				if code != wantCode {
					t.Fatalf("exit=%d, want %d; stdout=%s stderr=%s", code, wantCode, stdout, stderr)
				}
				if mode == "json" {
					var envelope struct {
						OK    bool            `json:"ok"`
						Data  json.RawMessage `json:"data"`
						Error struct {
							Code    string `json:"code"`
							Message string `json:"message"`
						} `json:"error"`
					}
					if err := json.Unmarshal(stdout.Bytes(), &envelope); err != nil {
						t.Fatalf("invalid JSON: %v; stdout=%s", err, stdout)
					}
					if stderr.Len() != 0 || envelope.OK != (wantCode == 0) {
						t.Fatalf("unexpected envelope/streams: stdout=%s stderr=%s", stdout, stderr)
					}
					if wantCode == 1 {
						if len(envelope.Data) != 0 || envelope.Error.Code != "search_stats_failures_read_failed" ||
							!strings.HasPrefix(envelope.Error.Message, "read failures.jsonl: ") || !strings.Contains(envelope.Error.Message, tc.diagnostic) {
							t.Fatalf("error must name ledger/line and omit partial stats: %s", stdout)
						}
					} else {
						var data struct {
							Failures []searchBatchFailure `json:"failures"`
						}
						if err := json.Unmarshal(envelope.Data, &data); err != nil {
							t.Fatal(err)
						}
						if data.Failures == nil || len(data.Failures) != len(tc.want) || (len(tc.want) > 0 && !reflect.DeepEqual(data.Failures, tc.want)) {
							t.Fatalf("failures=%#v, want non-null array %#v", data.Failures, tc.want)
						}
					}
				} else if wantCode == 1 {
					if stdout.Len() != 0 || !strings.Contains(stderr.String(), "read failures.jsonl: ") || !strings.Contains(stderr.String(), tc.diagnostic) {
						t.Fatalf("error must name ledger/line without successful stats: stdout=%s stderr=%s", stdout, stderr)
					}
				} else {
					if stderr.Len() != 0 || !strings.Contains(stdout.String(), "Source coverage for "+dir) {
						t.Fatalf("missing successful stats: stdout=%s stderr=%s", stdout, stderr)
					}
					last := -1
					for _, row := range tc.want {
						line := fmt.Sprintf("[%s] %q — %s", row.Source, row.Query, row.Error)
						index := strings.Index(stdout.String(), line)
						if index <= last {
							t.Fatalf("missing or reordered failure %q: %s", line, stdout)
						}
						last = index
					}
					if len(tc.want) == 0 && strings.Contains(stdout.String(), "Failed queries") {
						t.Fatalf("unexpected failures section: %s", stdout)
					}
				}
			})
		}
	}
}
