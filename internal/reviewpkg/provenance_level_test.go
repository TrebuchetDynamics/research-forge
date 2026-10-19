package reviewpkg

import (
	"path/filepath"
	"testing"

	"github.com/TrebuchetDynamics/research-forge/internal/provenance"
)

func TestPackageComputesProvenanceFromPackagedBytes(t *testing.T) {
	project := t.TempDir()
	write(t, filepath.Join(project, "rforge.project.toml"), "title='Review'\n")
	write(t, filepath.Join(project, "rforge.lock.json"), `{"version":"1"}`)
	output := filepath.Join(project, "reports", "report.md")
	write(t, output, "evidence")
	_, err := provenance.WriteRun(project, "test.run", map[string]any{"question": "test", "depth": "standard", "sources": []string{}, "errors": []string{}, "rforge_version": map[string]string{"version": "dev", "commit": "test", "date": "test"}}, []string{output})
	if err != nil {
		t.Fatal(err)
	}
	for _, tc := range []struct{ name, bytes, want string }{{"intact", "evidence", "engine_verified"}, {"tampered", "altered", "unverified"}} {
		t.Run(tc.name, func(t *testing.T) {
			write(t, output, tc.bytes)
			out := filepath.Join(t.TempDir(), "review.rforgepkg")
			pkg, err := Create(project, out, Options{})
			if err != nil {
				t.Fatal(err)
			}
			// Read the public manifest rather than depending on the internal field type.
			if got := string(pkg.Manifest.ProvenanceLevel); got != tc.want {
				t.Fatalf("manifest level=%s want %s", got, tc.want)
			}
			audit, err := Audit(out)
			if err != nil {
				t.Fatal(err)
			}
			if string(audit.ProvenanceLevel) != tc.want {
				t.Fatalf("audit level=%s want %s", audit.ProvenanceLevel, tc.want)
			}
		})
	}
}
