package sources

import "testing"

func TestOpenAlexFilterPresetsProvideHigherLevelWorkflows(t *testing.T) {
	for _, preset := range []string{"systematic-review", "open-access-review", "recent-domain-map"} {
		filters, ok := OpenAlexFilterPreset(preset)
		if !ok || filters["filter"] == "" {
			t.Fatalf("preset %s = %#v ok=%t", preset, filters, ok)
		}
	}
}
