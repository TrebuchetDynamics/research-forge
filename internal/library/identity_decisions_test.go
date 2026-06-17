package library

import (
	"path/filepath"
	"testing"
)

func TestIdentityDecisionLogRecordsReversibleMergeSplitAndConflicts(t *testing.T) {
	path := filepath.Join(t.TempDir(), "identity-decisions.jsonl")
	merge := IdentityDecision{
		ID: "merge-1", ClusterID: "identity-cluster-1", Action: IdentityDecisionMerge,
		Reviewer: "reviewer-a", Reason: "same DOI", Reversible: true,
		Before: []PaperRecord{{Title: "Left", Identifiers: Identifiers{DOI: "10.1000/a"}}, {Title: "Right", Identifiers: Identifiers{CrossrefID: "10.1000/a"}}},
		After:  []PaperRecord{{Title: "Merged", Identifiers: Identifiers{DOI: "10.1000/a", CrossrefID: "10.1000/a"}}},
	}
	split := IdentityDecision{ID: "split-1", ClusterID: "identity-cluster-1", Action: IdentityDecisionSplit, Reviewer: "reviewer-a", Reason: "false positive", Reversible: true, Before: merge.After, After: merge.Before}
	conflict := IdentityConflictRecord{ID: "conflict-1", ClusterID: "identity-cluster-1", Severity: "high", Reason: "same DOI but conflicting titles", RecordIndexes: []int{0, 1}, Resolved: false}
	if err := AppendIdentityDecision(path, merge); err != nil {
		t.Fatalf("append merge: %v", err)
	}
	if err := AppendIdentityDecision(path, split); err != nil {
		t.Fatalf("append split: %v", err)
	}
	if err := AppendIdentityConflict(path, conflict); err != nil {
		t.Fatalf("append conflict: %v", err)
	}
	log, err := ReadIdentityDecisionLog(path)
	if err != nil {
		t.Fatalf("read log: %v", err)
	}
	if len(log.Decisions) != 2 || len(log.Conflicts) != 1 {
		t.Fatalf("log = %#v", log)
	}
	if !log.Decisions[0].Reversible || len(log.Decisions[0].Before) != 2 || len(log.Decisions[0].After) != 1 {
		t.Fatalf("merge not reversible: %#v", log.Decisions[0])
	}
	if log.Conflicts[0].Resolved {
		t.Fatalf("conflict should be unresolved: %#v", log.Conflicts[0])
	}
}

func TestApplyIdentityDecisionMergeAndSplitAreReversible(t *testing.T) {
	records := []PaperRecord{{Title: "Left", Identifiers: Identifiers{DOI: "10.1000/a"}}, {Title: "Right", Identifiers: Identifiers{OpenAlexID: "W1", ZoteroItemKey: "ZOT-1"}}}
	merge := IdentityDecision{ID: "merge-1", ClusterID: "identity-cluster-1", Action: IdentityDecisionMerge, Before: records, After: []PaperRecord{{Title: "Left", Identifiers: Identifiers{DOI: "10.1000/a", OpenAlexID: "W1", ZoteroItemKey: "ZOT-1"}}}}
	merged, err := ApplyIdentityDecision(records, merge)
	if err != nil {
		t.Fatalf("merge: %v", err)
	}
	if len(merged) != 1 || merged[0].Identifiers.OpenAlexID != "W1" || merged[0].Identifiers.ZoteroItemKey != "ZOT-1" {
		t.Fatalf("merged = %#v", merged)
	}
	split := IdentityDecision{ID: "split-1", ClusterID: "identity-cluster-1", Action: IdentityDecisionSplit, Before: merged, After: records}
	restored, err := ApplyIdentityDecision(merged, split)
	if err != nil {
		t.Fatalf("split: %v", err)
	}
	if len(restored) != 2 || restored[1].Identifiers.OpenAlexID != "W1" {
		t.Fatalf("restored = %#v", restored)
	}
}

func TestDetectIdentityConflictsFlagsConflictingClusterMetadata(t *testing.T) {
	records := []PaperRecord{
		{Title: "Catalyst A", Identifiers: Identifiers{DOI: "10.1000/same"}, Year: 2020},
		{Title: "Unrelated title", Identifiers: Identifiers{DOI: "10.1000/same"}, Year: 2024},
	}
	report := ResolveIdentityClusters(records)
	conflicts := DetectIdentityConflicts(report, records)
	if len(conflicts) != 1 {
		t.Fatalf("conflicts = %#v", conflicts)
	}
	if conflicts[0].ClusterID != report.Clusters[0].ID || conflicts[0].Reason == "" || conflicts[0].Resolved {
		t.Fatalf("bad conflict: %#v", conflicts[0])
	}
}
