package cli

import (
	"bytes"
	"path/filepath"
	"testing"
	"time"

	"github.com/TrebuchetDynamics/research-forge/internal/protocol"
)

func TestExecuteProtocolLiveSmokeDashboardJSON(t *testing.T) {
	dir := t.TempDir()
	now := time.Date(2026, 6, 17, 12, 0, 0, 0, time.UTC)
	registry := protocol.DefaultConnectorCapabilityRegistry()
	current := protocol.NewLiveSmokeSnapshot(registry, now)
	current.UpsertResult(protocol.ConnectorLiveSmokeResult{ConnectorID: "semantic-scholar", Status: protocol.LiveSmokeFail, CheckedAt: now, Message: "429", ObservedFields: []string{"source"}})
	path := filepath.Join(dir, "snapshot.json")
	if err := protocol.SaveLiveSmokeSnapshot(path, current); err != nil {
		t.Fatal(err)
	}
	var stdout, stderr bytes.Buffer
	code := Execute([]string{"--json", "protocol", "live-smoke-dashboard", "--snapshot", path, "--provenance", "semantic-scholar=data/provenance.jsonl#evt1"}, &stdout, &stderr)
	if code != 0 {
		t.Fatalf("code=%d stderr=%s stdout=%s", code, stderr.String(), stdout.String())
	}
	for _, want := range []string{"sourceApiDriftDashboard", "semantic-scholar", "data/provenance.jsonl#evt1", "failing"} {
		if !bytes.Contains(stdout.Bytes(), []byte(want)) {
			t.Fatalf("missing %s in %s", want, stdout.String())
		}
	}
}
