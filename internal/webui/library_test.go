package webui

import (
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/TrebuchetDynamics/research-forge/internal/library"
	"github.com/TrebuchetDynamics/research-forge/internal/ui"
)

func TestLibraryHandlerRendersHTMXLibraryScreen(t *testing.T) {
	state := ui.NewLibraryViewModel([]ui.PaperRow{{Title: "Solar fuels review"}, {Title: "Artificial photosynthesis catalyst"}})

	req := httptest.NewRequest("GET", "/library", nil)
	rec := httptest.NewRecorder()

	NewLibraryHandler(state).ServeHTTP(rec, req)

	if rec.Code != 200 {
		t.Fatalf("status = %d", rec.Code)
	}
	body := rec.Body.String()
	for _, want := range []string{
		"Library",
		"hx-get=\"/library/rows\"",
		"role=\"table\"",
		"Solar fuels review",
		"Artificial photosynthesis catalyst",
		"Paper title",
	} {
		if !strings.Contains(body, want) {
			t.Fatalf("library screen missing %q:\n%s", want, body)
		}
	}
	if strings.Contains(body, "No papers yet") {
		t.Fatalf("populated library should not render empty state:\n%s", body)
	}
}

func TestLibraryHandlerRendersEmptyState(t *testing.T) {
	req := httptest.NewRequest("GET", "/library", nil)
	rec := httptest.NewRecorder()

	NewLibraryHandler(ui.NewLibraryViewModel(nil)).ServeHTTP(rec, req)

	body := rec.Body.String()
	if !strings.Contains(body, "No papers yet") || !strings.Contains(body, "Import or search for papers") {
		t.Fatalf("empty library state missing:\n%s", body)
	}
}

func TestBuildLibraryViewModelEmptyProjectDoesNotReadWorkingDirectory(t *testing.T) {
	cwd, err := os.Getwd()
	if err != nil {
		t.Fatal(err)
	}
	dir := t.TempDir()
	if err := os.Chdir(dir); err != nil {
		t.Fatal(err)
	}
	defer func() {
		if err := os.Chdir(cwd); err != nil {
			t.Fatalf("restore cwd: %v", err)
		}
	}()

	store, err := library.OpenStore(filepath.Join(dir, "data", "library.json"))
	if err != nil {
		t.Fatalf("OpenStore: %v", err)
	}
	if err := store.Create(library.PaperRecord{Title: "cwd leak", Identifiers: library.Identifiers{DOI: "10.1000/cwd-leak"}}); err != nil {
		t.Fatalf("Create: %v", err)
	}

	vm, err := BuildLibraryViewModel("")
	if err != nil {
		t.Fatalf("BuildLibraryViewModel: %v", err)
	}
	if !vm.Empty || len(vm.Rows) != 0 {
		t.Fatalf("empty project read cwd library: %#v", vm)
	}
}
