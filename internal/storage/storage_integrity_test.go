package storage

import (
	"bytes"
	"database/sql"
	"encoding/binary"
	"io"
	"net/url"
	"os"
	"path/filepath"
	"strings"
	"testing"

	_ "modernc.org/sqlite"
)

func TestCheckExistingIntegrityCheck(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "rforge.sqlite")
	seedMultiPageSQLite(t, path)
	corruptSQLitePage(t, path, 100)

	if ok, result := rawIntegrityCheck(t, path); ok {
		t.Fatalf("raw PRAGMA integrity_check reported ok for corrupted database: %s", result)
	}
	before := readDatabaseFile(t, path)
	beforeEntries := directoryEntries(t, dir)

	err := CheckExisting(path)
	if err == nil {
		t.Fatal("CheckExisting reported a database with a corrupted page as healthy")
	}
	if !strings.Contains(err.Error(), "sqlite integrity check failed") {
		t.Fatalf("CheckExisting error = %q, want integrity check failure", err)
	}
	if after := readDatabaseFile(t, path); !bytes.Equal(before, after) {
		t.Fatal("CheckExisting modified the corrupt database")
	}
	if afterEntries := directoryEntries(t, dir); !equalStrings(beforeEntries, afterEntries) {
		t.Fatalf("CheckExisting created files: before=%v after=%v", beforeEntries, afterEntries)
	}
}

func TestCheckExistingHealthyControl(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "rforge.sqlite")
	seedMultiPageSQLite(t, path)

	if ok, result := rawIntegrityCheck(t, path); !ok {
		t.Fatalf("raw PRAGMA integrity_check failed for healthy database: %s", result)
	}
	before := readDatabaseFile(t, path)
	beforeEntries := directoryEntries(t, dir)

	if err := CheckExisting(path); err != nil {
		t.Fatalf("CheckExisting returned error for healthy database: %v", err)
	}
	if after := readDatabaseFile(t, path); !bytes.Equal(before, after) {
		t.Fatal("CheckExisting modified the healthy database")
	}
	if afterEntries := directoryEntries(t, dir); !equalStrings(beforeEntries, afterEntries) {
		t.Fatalf("CheckExisting created files: before=%v after=%v", beforeEntries, afterEntries)
	}
}

func seedMultiPageSQLite(t *testing.T, path string) {
	t.Helper()
	db, err := sql.Open("sqlite", path)
	if err != nil {
		t.Fatalf("open sqlite: %v", err)
	}
	tx, err := db.Begin()
	if err != nil {
		t.Fatalf("begin seed transaction: %v", err)
	}
	if _, err := tx.Exec(`CREATE TABLE records (id INTEGER PRIMARY KEY, payload TEXT NOT NULL)`); err != nil {
		t.Fatalf("create seed table: %v", err)
	}
	stmt, err := tx.Prepare(`INSERT INTO records(payload) VALUES (?)`)
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

func rawIntegrityCheck(t *testing.T, path string) (bool, string) {
	t.Helper()
	dsn := (&url.URL{Scheme: "file", Path: path, RawQuery: "mode=ro"}).String()
	db, err := sql.Open("sqlite", dsn)
	if err != nil {
		t.Fatalf("open sqlite read-only: %v", err)
	}
	defer db.Close()
	rows, err := db.Query("PRAGMA integrity_check")
	if err != nil {
		return false, err.Error()
	}
	defer rows.Close()
	var result []string
	for rows.Next() {
		var line string
		if err := rows.Scan(&line); err != nil {
			return false, err.Error()
		}
		result = append(result, line)
	}
	if err := rows.Err(); err != nil {
		return false, err.Error()
	}
	return len(result) == 1 && result[0] == "ok", strings.Join(result, "; ")
}

func readDatabaseFile(t *testing.T, path string) []byte {
	t.Helper()
	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("read database: %v", err)
	}
	return data
}

func directoryEntries(t *testing.T, dir string) []string {
	t.Helper()
	entries, err := os.ReadDir(dir)
	if err != nil {
		t.Fatalf("read directory: %v", err)
	}
	names := make([]string, 0, len(entries))
	for _, entry := range entries {
		names = append(names, entry.Name())
	}
	return names
}

func equalStrings(a, b []string) bool {
	if len(a) != len(b) {
		return false
	}
	for i := range a {
		if a[i] != b[i] {
			return false
		}
	}
	return true
}
