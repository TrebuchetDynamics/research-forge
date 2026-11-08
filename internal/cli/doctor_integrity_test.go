package cli

import (
	"bytes"
	"database/sql"
	"encoding/binary"
	"encoding/json"
	"io"
	"os"
	"path/filepath"
	"strings"
	"testing"

	_ "modernc.org/sqlite"
)

func TestExecuteDoctorIntegrityCheck(t *testing.T) {
	dir := filepath.Join(t.TempDir(), "demo")
	if code := Execute([]string{"project", "create", dir, "--title", "Demo Review"}, new(bytes.Buffer), new(bytes.Buffer)); code != 0 {
		t.Fatalf("project create exit code = %d", code)
	}
	corruptDoctorDatabase(t, filepath.Join(dir, "data", "rforge.sqlite"))

	stdout := new(bytes.Buffer)
	stderr := new(bytes.Buffer)
	if code := Execute([]string{"--json", "--project", dir, "doctor"}, stdout, stderr); code != 0 {
		t.Fatalf("json exit code = %d, stderr = %s", code, stderr.String())
	}
	var envelope map[string]any
	if err := json.Unmarshal(stdout.Bytes(), &envelope); err != nil {
		t.Fatalf("stdout is not JSON: %v\n%s", err, stdout.String())
	}
	data := envelope["data"].(map[string]any)
	var sqliteCheck map[string]any
	for _, raw := range data["checks"].([]any) {
		check := raw.(map[string]any)
		if check["name"] == "sqlite" {
			sqliteCheck = check
		}
	}
	if sqliteCheck == nil {
		t.Fatalf("missing sqlite check in %#v", data["checks"])
	}
	if sqliteCheck["ok"] != false {
		t.Fatalf("sqlite ok = %v, want false for corrupt database: %#v", sqliteCheck["ok"], sqliteCheck)
	}
	message, _ := sqliteCheck["message"].(string)
	if message == "" {
		t.Fatalf("sqlite message is empty: %#v", sqliteCheck)
	}
	action, _ := sqliteCheck["action"].(string)
	if action == "" || action == "No action needed." {
		t.Fatalf("sqlite action = %q, want actionable repair guidance", action)
	}

	stdout.Reset()
	stderr.Reset()
	if code := Execute([]string{"--project", dir, "doctor"}, stdout, stderr); code != 0 {
		t.Fatalf("text exit code = %d, stderr = %s", code, stderr.String())
	}
	out := stdout.String()
	if !strings.Contains(out, "sqlite: fail") {
		t.Fatalf("text output missing sqlite failure:\n%s", out)
	}
	if !strings.Contains(out, message) || !strings.Contains(out, action) {
		t.Fatalf("text output missing sqlite message or action:\n%s", out)
	}
}

// corruptDoctorDatabase seeds enough rows to span multiple pages and corrupts
// one later b-tree page in place, keeping page 1 and the header intact.
func corruptDoctorDatabase(t *testing.T, path string) {
	t.Helper()
	db, err := sql.Open("sqlite", path)
	if err != nil {
		t.Fatalf("open project database: %v", err)
	}
	tx, err := db.Begin()
	if err != nil {
		t.Fatalf("begin seed transaction: %v", err)
	}
	if _, err := tx.Exec(`CREATE TABLE integrity_probe (id INTEGER PRIMARY KEY, payload TEXT NOT NULL)`); err != nil {
		t.Fatalf("create probe table: %v", err)
	}
	stmt, err := tx.Prepare(`INSERT INTO integrity_probe(payload) VALUES (?)`)
	if err != nil {
		t.Fatalf("prepare seed insert: %v", err)
	}
	payload := strings.Repeat("x", 512)
	for i := 0; i < 4000; i++ {
		if _, err := stmt.Exec(payload); err != nil {
			t.Fatalf("seed row %d: %v", i, err)
		}
	}
	if err := stmt.Close(); err != nil {
		t.Fatalf("close seed statement: %v", err)
	}
	if err := tx.Commit(); err != nil {
		t.Fatalf("commit seed rows: %v", err)
	}
	if err := db.Close(); err != nil {
		t.Fatalf("close seeded database: %v", err)
	}

	file, err := os.OpenFile(path, os.O_RDWR, 0)
	if err != nil {
		t.Fatalf("open database for corruption: %v", err)
	}
	defer file.Close()
	header := make([]byte, 100)
	if _, err := io.ReadFull(file, header); err != nil {
		t.Fatalf("read database header: %v", err)
	}
	pageSize := int(binary.BigEndian.Uint16(header[16:18]))
	if pageSize == 1 {
		pageSize = 65536
	}
	garbage := bytes.Repeat([]byte{0xa5}, pageSize)
	if _, err := file.WriteAt(garbage, 99*int64(pageSize)); err != nil {
		t.Fatalf("corrupt page 100: %v", err)
	}
}
