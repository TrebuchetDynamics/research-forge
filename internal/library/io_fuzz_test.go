package library

import (
	"os"
	"path/filepath"
	"testing"
)

func FuzzImportCSV(f *testing.F) {
	f.Add("title,doi,year\nFixture,10.1000/fuzz,2026\n")
	f.Fuzz(func(t *testing.T, data string) {
		path := filepath.Join(t.TempDir(), "fuzz.csv")
		if err := os.WriteFile(path, []byte(data), 0o644); err != nil {
			t.Fatal(err)
		}
		_, _, _ = ImportCSV(path)
	})
}
