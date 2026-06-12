package analysis

import (
	"os"
	"testing"
)

func TestOptInRMetaforIntegrationGate(t *testing.T) {
	if os.Getenv("RFORGE_RUN_R_METAFOR_INTEGRATION") != "1" {
		t.Skip("set RFORGE_RUN_R_METAFOR_INTEGRATION=1 to run real R/metafor integration")
	}
	// Real R/metafor execution will be wired behind Runner after local fake coverage remains green.
}
