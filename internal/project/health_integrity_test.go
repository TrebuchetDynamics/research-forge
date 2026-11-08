package project

import (
	"bytes"
	"database/sql"
	"encoding/binary"
	"io"
	"io/fs"
	"net/url"
	"os"
	"path/filepath"
	"strings"
	"testing"

	_ "modernc.org/sqlite"
)

func TestCheckHealthIntegrityCheck(t *testing.T) {
	dir := filepath.Join(t.TempDir(), "demo")
	if _, err := Create(dir, CreateOptions{Title: "Demo Review"}); err != nil {
		t.Fatalf("Create returned error: %v", err)
	}
	dbPath := filepath.Join(dir, "data", "rforge.sqlite")
	integrityResult := corruptProjectDatabase(t, dbPath)

	before := snapshotProject(t, dir)
	report := CheckHealth(dir)

	checks := map[string]HealthCheck{}
	for _, check := range report.Checks {
		checks[check.Name] = check
	}
	sqliteCheck, ok := checks["sqlite"]
	if !ok {
		t.Fatalf("missing sqlite check: %#v", report.Checks)
	}
	if sqliteCheck.OK {
		t.Fatalf("sqlite check OK = true for corrupt database: %#v", sqliteCheck)
	}
	if sqliteCheck.Message == "" || !strings.Contains(sqliteCheck.Message, integrityResult) {
		t.Fatalf("sqlite message = %q, want integrity result %q", sqliteCheck.Message, integrityResult)
	}
	if sqliteCheck.Action == "" || sqliteCheck.Action == "No action needed." {
		t.Fatalf("sqlite action = %q, want actionable repair guidance", sqliteCheck.Action)
	}
	for _, name := range []string{"project_manifest", "project_lockfile"} {
		check, ok := checks[name]
		if !ok || !check.OK {
			t.Fatalf("check %q = %#v, want OK", name, check)
		}
	}
	if after := snapshotProject(t, dir); !equalSnapshots(before, after) {
		t.Fatalf("CheckHealth changed project files: before=%v after=%v", before, after)
	}
}

// corruptProjectDatabase adds enough rows to span multiple pages, corrupts one
// later b-tree page in place, and returns the raw integrity_check result.
func corruptProjectDatabase(t *testing.T, path string) string {
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
	corruptSQLitePage(t, path, 100)
	result := rawIntegrityResult(t, path)
	if result == "ok" {
		t.Fatal("raw PRAGMA integrity_check reported ok for corrupted database")
	}
	return result
}

// corruptSQLitePage overwrites one later b-tree page in place, keeping page 1
// and the database header intact.
func corruptSQLitePage(t *testing.T, path string, page int) {
	t.Helper()
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
	if _, err := file.WriteAt(garbage, int64(page-1)*int64(pageSize)); err != nil {
		t.Fatalf("corrupt page %d: %v", page, err)
	}
}

func rawIntegrityResult(t *testing.T, path string) string {
	t.Helper()
	dsn := (&url.URL{Scheme: "file", Path: path, RawQuery: "mode=ro"}).String()
	db, err := sql.Open("sqlite", dsn)
	if err != nil {
		t.Fatalf("open sqlite read-only: %v", err)
	}
	defer db.Close()
	rows, err := db.Query("PRAGMA integrity_check")
	if err != nil {
		return err.Error()
	}
	defer rows.Close()
	var result []string
	for rows.Next() {
		var line string
		if err := rows.Scan(&line); err != nil {
			t.Fatalf("scan integrity result: %v", err)
		}
		result = append(result, line)
	}
	if err := rows.Err(); err != nil {
		return err.Error()
	}
	return strings.Join(result, "; ")
}

func snapshotProject(t *testing.T, dir string) map[string][]byte {
	t.Helper()
	files := map[string][]byte{}
	err := filepath.WalkDir(dir, func(path string, entry fs.DirEntry, err error) error {
		if err != nil || entry.IsDir() {
			return err
		}
		data, err := os.ReadFile(path)
		if err != nil {
			return err
		}
		rel, err := filepath.Rel(dir, path)
		if err != nil {
			return err
		}
		files[rel] = data
		return nil
	})
	if err != nil {
		t.Fatalf("snapshot project: %v", err)
	}
	return files
}

func equalSnapshots(a, b map[string][]byte) bool {
	if len(a) != len(b) {
		return false
	}
	for name, data := range a {
		other, ok := b[name]
		if !ok || !bytes.Equal(data, other) {
			return false
		}
	}
	return true
}
