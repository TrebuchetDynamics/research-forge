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

func TestApplyIdentityDecisionDoesNotDropUnrelatedLibraryRecords(t *testing.T) {
	fullLibrary := []PaperRecord{
		{Title: "Left", Identifiers: Identifiers{DOI: "10.1000/a"}},
		{Title: "Right", Identifiers: Identifiers{CrossrefID: "10.1000/a"}},
		{Title: "Unrelated Paper 1", Identifiers: Identifiers{DOI: "10.1000/z1"}},
		{Title: "Unrelated Paper 2", Identifiers: Identifiers{DOI: "10.1000/z2"}},
	}
	merge := IdentityDecision{
		ID: "merge-1", ClusterID: "cluster-1", Action: IdentityDecisionMerge,
		Before: []PaperRecord{fullLibrary[0], fullLibrary[1]},
		After:  []PaperRecord{{Title: "Merged", Identifiers: Identifiers{DOI: "10.1000/a", CrossrefID: "10.1000/a"}}},
	}
	applied, err := ApplyIdentityDecision(fullLibrary, merge)
	if err != nil {
		t.Fatalf("apply: %v", err)
	}
	if len(applied) != 3 {
		t.Fatalf("applied = %#v, want merged record + 2 unrelated papers surviving", applied)
	}
	titles := map[string]bool{}
	for _, record := range applied {
		titles[record.Title] = true
	}
	if !titles["Merged"] || !titles["Unrelated Paper 1"] || !titles["Unrelated Paper 2"] {
		t.Fatalf("applied lost unrelated records: %#v", applied)
	}
}

func TestApplyIdentityDecisionErrorsWhenBeforeRecordMissing(t *testing.T) {
	records := []PaperRecord{{Title: "Kept", Identifiers: Identifiers{DOI: "10.1000/keep"}}}
	decision := IdentityDecision{
		ID: "merge-1", ClusterID: "cluster-1", Action: IdentityDecisionMerge,
		Before: []PaperRecord{{Title: "Not In Library", Identifiers: Identifiers{DOI: "10.1000/missing"}}},
		After:  []PaperRecord{{Title: "Merged", Identifiers: Identifiers{DOI: "10.1000/missing"}}},
	}
	if _, err := ApplyIdentityDecision(records, decision); err == nil {
		t.Fatalf("ApplyIdentityDecision returned nil error for a before record absent from the current library")
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
