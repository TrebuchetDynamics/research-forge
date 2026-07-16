package library

import (
	"bytes"
	"errors"
	"os"
	"path/filepath"
	"strings"
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
	if err := os.Chmod(path, 0o600); err != nil {
		t.Fatalf("chmod identity log: %v", err)
	}
	if err := AppendIdentityDecision(path, split); err != nil {
		t.Fatalf("append split: %v", err)
	}
	if err := AppendIdentityConflict(path, conflict); err != nil {
		t.Fatalf("append conflict: %v", err)
	}
	info, err := os.Stat(path)
	if err != nil {
		t.Fatalf("stat identity log: %v", err)
	}
	if got := info.Mode().Perm(); got != 0o600 {
		t.Fatalf("identity log mode = %o, want 600", got)
	}
	files, err := os.ReadDir(filepath.Dir(path))
	if err != nil {
		t.Fatalf("read identity log directory: %v", err)
	}
	if len(files) != 1 || files[0].Name() != "identity-decisions.jsonl" {
		t.Fatalf("identity log directory entries = %v, want only identity-decisions.jsonl", files)
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

func TestIdentityDecisionLogReadsLargeAppendedDecision(t *testing.T) {
	path := filepath.Join(t.TempDir(), "identity-decisions.jsonl")
	largeReason := strings.Repeat("x", 128*1024)
	decision := IdentityDecision{
		ID:        "merge-large",
		ClusterID: "identity-cluster-large",
		Action:    IdentityDecisionMerge,
		Reason:    largeReason,
	}

	if err := AppendIdentityDecision(path, decision); err != nil {
		t.Fatalf("AppendIdentityDecision returned error: %v", err)
	}
	log, err := ReadIdentityDecisionLog(path)
	if err != nil {
		t.Fatalf("ReadIdentityDecisionLog returned error: %v", err)
	}
	if len(log.Decisions) != 1 {
		t.Fatalf("len(log.Decisions) = %d, want 1", len(log.Decisions))
	}
	if got := log.Decisions[0].Reason; got != largeReason {
		t.Fatalf("reason length = %d, want %d", len(got), len(largeReason))
	}
}

func TestAppendIdentityDecisionThenRemovesNewLogWhenCommitFails(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "identity-decisions.jsonl")
	decision := IdentityDecision{ID: "merge-1", ClusterID: "cluster-1", Action: IdentityDecisionMerge}
	commitErr := errors.New("commit failed")

	err := AppendIdentityDecisionThen(path, decision, func() error { return commitErr })
	if !errors.Is(err, commitErr) {
		t.Fatalf("AppendIdentityDecisionThen error = %v, want commit error", err)
	}
	if _, statErr := os.Stat(path); !os.IsNotExist(statErr) {
		t.Fatalf("identity log remains after rollback: err=%v", statErr)
	}
	entries, readErr := os.ReadDir(dir)
	if readErr != nil {
		t.Fatalf("read identity log directory: %v", readErr)
	}
	if len(entries) != 0 {
		t.Fatalf("identity log rollback left debris: %v", entries)
	}
}

func TestAppendIdentityDecisionDoesNotWriteThroughSymlink(t *testing.T) {
	outsidePath := filepath.Join(t.TempDir(), "outside.jsonl")
	outsideBefore := []byte("outside identity log must remain unchanged\n")
	if err := os.WriteFile(outsidePath, outsideBefore, 0o640); err != nil {
		t.Fatalf("write outside identity log: %v", err)
	}
	path := filepath.Join(t.TempDir(), "identity-decisions.jsonl")
	if err := os.Symlink(outsidePath, path); err != nil {
		t.Skipf("symlink unsupported: %v", err)
	}
	decision := IdentityDecision{ID: "merge-1", ClusterID: "cluster-1", Action: IdentityDecisionMerge}

	if err := AppendIdentityDecision(path, decision); err == nil {
		t.Fatal("AppendIdentityDecision succeeded with a symlinked identity log")
	}
	outsideAfter, err := os.ReadFile(outsidePath)
	if err != nil {
		t.Fatalf("read outside identity log: %v", err)
	}
	if !bytes.Equal(outsideAfter, outsideBefore) {
		t.Fatalf("AppendIdentityDecision wrote through symlink: got %q, want %q", outsideAfter, outsideBefore)
	}
	info, err := os.Lstat(path)
	if err != nil {
		t.Fatalf("lstat identity log: %v", err)
	}
	if info.Mode()&os.ModeSymlink == 0 {
		t.Fatalf("AppendIdentityDecision replaced symlink despite rejecting it: mode=%v", info.Mode())
	}
}

func TestAppendIdentityDecisionRejectsSymlinkedDirectory(t *testing.T) {
	projectDir := t.TempDir()
	outsideDir := t.TempDir()
	dataDir := filepath.Join(projectDir, "data")
	if err := os.Symlink(outsideDir, dataDir); err != nil {
		t.Skipf("symlink unsupported: %v", err)
	}
	path := filepath.Join(dataDir, "identity-decisions.jsonl")
	decision := IdentityDecision{ID: "merge-1", ClusterID: "cluster-1", Action: IdentityDecisionMerge}

	if err := AppendIdentityDecision(path, decision); err == nil {
		t.Fatal("AppendIdentityDecision succeeded through a symlinked directory")
	}
	files, err := os.ReadDir(outsideDir)
	if err != nil {
		t.Fatalf("read outside directory: %v", err)
	}
	if len(files) != 0 {
		t.Fatalf("AppendIdentityDecision wrote through symlinked directory: entries=%v", files)
	}
	info, err := os.Lstat(dataDir)
	if err != nil {
		t.Fatalf("lstat data directory: %v", err)
	}
	if info.Mode()&os.ModeSymlink == 0 {
		t.Fatalf("AppendIdentityDecision replaced directory symlink despite rejecting it: mode=%v", info.Mode())
	}
}

func TestReadIdentityDecisionLogDoesNotReadThroughSymlink(t *testing.T) {
	outsidePath := filepath.Join(t.TempDir(), "outside.jsonl")
	payload := []byte("{\"type\":\"decision\",\"decision\":{\"schemaVersion\":\"1\",\"id\":\"outside\",\"clusterId\":\"private\",\"action\":\"merge\",\"reversible\":true}}\n")
	if err := os.WriteFile(outsidePath, payload, 0o640); err != nil {
		t.Fatalf("write outside identity log: %v", err)
	}
	path := filepath.Join(t.TempDir(), "identity-decisions.jsonl")
	if err := os.Symlink(outsidePath, path); err != nil {
		t.Skipf("symlink unsupported: %v", err)
	}

	if _, err := ReadIdentityDecisionLog(path); err == nil {
		t.Fatal("ReadIdentityDecisionLog succeeded with a symlinked identity log")
	}
	info, err := os.Lstat(path)
	if err != nil {
		t.Fatalf("lstat identity log: %v", err)
	}
	if info.Mode()&os.ModeSymlink == 0 {
		t.Fatalf("ReadIdentityDecisionLog replaced symlink despite rejecting it: mode=%v", info.Mode())
	}
}

func TestReadIdentityDecisionLogRejectsSymlinkedDirectory(t *testing.T) {
	projectDir := t.TempDir()
	outsideDir := t.TempDir()
	outsidePath := filepath.Join(outsideDir, "identity-decisions.jsonl")
	payload := []byte("{\"type\":\"decision\",\"decision\":{\"schemaVersion\":\"1\",\"id\":\"outside\",\"clusterId\":\"private\",\"action\":\"merge\",\"reversible\":true}}\n")
	if err := os.WriteFile(outsidePath, payload, 0o640); err != nil {
		t.Fatalf("write outside identity log: %v", err)
	}
	dataDir := filepath.Join(projectDir, "data")
	if err := os.Symlink(outsideDir, dataDir); err != nil {
		t.Skipf("symlink unsupported: %v", err)
	}

	if _, err := ReadIdentityDecisionLog(filepath.Join(dataDir, "identity-decisions.jsonl")); err == nil {
		t.Fatal("ReadIdentityDecisionLog succeeded through a symlinked directory")
	}
	info, err := os.Lstat(dataDir)
	if err != nil {
		t.Fatalf("lstat data directory: %v", err)
	}
	if info.Mode()&os.ModeSymlink == 0 {
		t.Fatalf("ReadIdentityDecisionLog replaced directory symlink despite rejecting it: mode=%v", info.Mode())
	}
}

func TestReadIdentityDecisionLogReportsIncompleteEntry(t *testing.T) {
	path := filepath.Join(t.TempDir(), "identity-decisions.jsonl")
	incomplete := []byte("{\"type\":\"decision\"}\n")
	if err := os.WriteFile(path, incomplete, 0o600); err != nil {
		t.Fatalf("write incomplete identity log: %v", err)
	}

	if _, err := ReadIdentityDecisionLog(path); err == nil {
		t.Fatal("ReadIdentityDecisionLog silently discarded an incomplete decision entry")
	}
	got, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("read incomplete identity log after call: %v", err)
	}
	if !bytes.Equal(got, incomplete) {
		t.Fatalf("ReadIdentityDecisionLog changed incomplete log: got %q, want %q", got, incomplete)
	}
}

func TestReadIdentityDecisionLogReportsUnknownEntryType(t *testing.T) {
	path := filepath.Join(t.TempDir(), "identity-decisions.jsonl")
	if err := os.WriteFile(path, []byte("{\"type\":\"unexpected\"}\n"), 0o600); err != nil {
		t.Fatalf("write unknown identity log entry: %v", err)
	}

	if _, err := ReadIdentityDecisionLog(path); err == nil {
		t.Fatal("ReadIdentityDecisionLog silently discarded an unknown entry type")
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
