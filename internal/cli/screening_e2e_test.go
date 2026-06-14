package cli

import (
	"encoding/json"
	"testing"

	"github.com/TrebuchetDynamics/research-forge/internal/screening"
)

// TestE2EMultiReviewerScreeningConflictAndUncertainQueue drives a multi-reviewer
// title/abstract screening round through the CLI and asserts the cross-reviewer
// signals the workflow must surface: conflict detection (one paper included by
// one reviewer and excluded by another), the uncertain queue, the include
// queue, and PRISMA counts over every recorded decision.
func TestE2EMultiReviewerScreeningConflictAndUncertainQueue(t *testing.T) {
	proj := t.TempDir() + "/screening"
	mustRunCLI(t, "--json", "project", "create", proj, "--title", "Multi-reviewer Screening Review")
	mustRunCLI(t, "--project", proj, "screen", "configure", "--reason", "off-topic")

	// paper-1: reviewers disagree -> conflict at title_abstract.
	mustRunCLI(t, "--project", proj, "screen", "decide", "--paper", "paper-1", "--stage", "title_abstract", "--decision", "include", "--reviewer", "ada")
	mustRunCLI(t, "--project", proj, "screen", "decide", "--paper", "paper-1", "--stage", "title_abstract", "--decision", "exclude", "--reason", "off-topic", "--reviewer", "linus")
	// paper-2: a single reviewer is uncertain.
	mustRunCLI(t, "--project", proj, "screen", "decide", "--paper", "paper-2", "--stage", "title_abstract", "--decision", "uncertain", "--reviewer", "ada")
	// paper-3: two reviewers agree to include -> no conflict.
	mustRunCLI(t, "--project", proj, "screen", "decide", "--paper", "paper-3", "--stage", "title_abstract", "--decision", "include", "--reviewer", "ada")
	mustRunCLI(t, "--project", proj, "screen", "decide", "--paper", "paper-3", "--stage", "title_abstract", "--decision", "include", "--reviewer", "linus")
	// paper-4: excluded.
	mustRunCLI(t, "--project", proj, "screen", "decide", "--paper", "paper-4", "--stage", "title_abstract", "--decision", "exclude", "--reason", "off-topic", "--reviewer", "ada")

	if got := decodeStringList(t, mustRunCLI(t, "--json", "--project", proj, "screen", "conflicts", "--stage", "title_abstract"), "conflicts"); !equalStrings(got, []string{"paper-1"}) {
		t.Fatalf("conflicts = %v, want [paper-1]", got)
	}
	if got := decodeStringList(t, mustRunCLI(t, "--json", "--project", proj, "screen", "queue", "--stage", "title_abstract", "--decision", "uncertain"), "queue"); !equalStrings(got, []string{"paper-2"}) {
		t.Fatalf("uncertain queue = %v, want [paper-2]", got)
	}
	if got := decodeStringList(t, mustRunCLI(t, "--json", "--project", proj, "screen", "queue", "--stage", "title_abstract", "--decision", "include"), "queue"); !equalStrings(got, []string{"paper-1", "paper-3"}) {
		t.Fatalf("include queue = %v, want [paper-1 paper-3]", got)
	}

	var prisma struct {
		Data struct {
			Counts screening.PRISMACounts `json:"counts"`
		} `json:"data"`
	}
	if err := json.Unmarshal(mustRunCLI(t, "--json", "--project", proj, "prisma", "counts"), &prisma); err != nil {
		t.Fatalf("decode prisma counts: %v", err)
	}
	if prisma.Data.Counts != (screening.PRISMACounts{Included: 3, Excluded: 2, Uncertain: 1}) {
		t.Fatalf("prisma counts = %+v, want included=3 excluded=2 uncertain=1", prisma.Data.Counts)
	}
}

func decodeStringList(t *testing.T, raw []byte, key string) []string {
	t.Helper()
	var env struct {
		Data map[string][]string `json:"data"`
	}
	if err := json.Unmarshal(raw, &env); err != nil {
		t.Fatalf("decode %s list: %v\n%s", key, err, raw)
	}
	return env.Data[key]
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
