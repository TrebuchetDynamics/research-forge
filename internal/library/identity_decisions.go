package library

import (
	"bufio"
	"encoding/json"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"reflect"
	"strings"

	"github.com/TrebuchetDynamics/research-forge/internal/filetxn"
)

const (
	IdentityDecisionMerge = "merge"
	IdentityDecisionSplit = "split"
)

type IdentityDecision struct {
	SchemaVersion string        `json:"schemaVersion"`
	ID            string        `json:"id"`
	ClusterID     string        `json:"clusterId"`
	Action        string        `json:"action"`
	Reviewer      string        `json:"reviewer,omitempty"`
	Reason        string        `json:"reason"`
	Reversible    bool          `json:"reversible"`
	Before        []PaperRecord `json:"before"`
	After         []PaperRecord `json:"after"`
}

type IdentityConflictRecord struct {
	SchemaVersion string `json:"schemaVersion"`
	ID            string `json:"id"`
	ClusterID     string `json:"clusterId"`
	Severity      string `json:"severity"`
	Reason        string `json:"reason"`
	RecordIndexes []int  `json:"recordIndexes"`
	Resolved      bool   `json:"resolved"`
}

type IdentityDecisionLog struct {
	SchemaVersion string                   `json:"schemaVersion"`
	Decisions     []IdentityDecision       `json:"decisions"`
	Conflicts     []IdentityConflictRecord `json:"conflicts"`
}

type identityLogEntry struct {
	Type     string                  `json:"type"`
	Decision *IdentityDecision       `json:"decision,omitempty"`
	Conflict *IdentityConflictRecord `json:"conflict,omitempty"`
}

func AppendIdentityDecision(path string, decision IdentityDecision) error {
	return appendIdentityDecision(path, decision, nil)
}

// AppendIdentityDecisionThen appends a decision, runs commit, and restores the
// prior log if commit fails.
func AppendIdentityDecisionThen(path string, decision IdentityDecision, commit func() error) error {
	if commit == nil {
		return fmt.Errorf("identity decision commit callback is required")
	}
	return appendIdentityDecision(path, decision, commit)
}

func appendIdentityDecision(path string, decision IdentityDecision, commit func() error) error {
	if strings.TrimSpace(decision.ID) == "" || strings.TrimSpace(decision.ClusterID) == "" {
		return fmt.Errorf("identity decision id and cluster id are required")
	}
	if decision.Action != IdentityDecisionMerge && decision.Action != IdentityDecisionSplit {
		return fmt.Errorf("identity decision action must be merge or split")
	}
	decision.SchemaVersion = "1"
	decision.Reversible = true
	return appendIdentityLogEntryThen(path, identityLogEntry{Type: "decision", Decision: &decision}, commit)
}

func AppendIdentityConflict(path string, conflict IdentityConflictRecord) error {
	if strings.TrimSpace(conflict.ID) == "" || strings.TrimSpace(conflict.ClusterID) == "" {
		return fmt.Errorf("identity conflict id and cluster id are required")
	}
	conflict.SchemaVersion = "1"
	return appendIdentityLogEntry(path, identityLogEntry{Type: "conflict", Conflict: &conflict})
}

// ApplyIdentityDecision replaces decision.Before with decision.After within
// records, leaving every unrelated record untouched. It previously ignored
// records entirely and returned decision.After alone, which silently
// dropped every library record outside the decision's own cluster.
func ApplyIdentityDecision(records []PaperRecord, decision IdentityDecision) ([]PaperRecord, error) {
	if decision.Action != IdentityDecisionMerge && decision.Action != IdentityDecisionSplit {
		return nil, fmt.Errorf("identity decision action must be merge or split")
	}
	if len(decision.After) == 0 {
		return nil, fmt.Errorf("identity decision after state is required")
	}
	remaining := append([]PaperRecord{}, records...)
	for _, before := range decision.Before {
		index := -1
		for i, record := range remaining {
			if reflect.DeepEqual(record, before) {
				index = i
				break
			}
		}
		if index == -1 {
			return nil, fmt.Errorf("identity decision before record %q not found in current library", before.Title)
		}
		remaining = append(remaining[:index], remaining[index+1:]...)
	}
	out := append(remaining, decision.After...)
	return out, nil
}

func ReadIdentityDecisionLog(path string) (IdentityDecisionLog, error) {
	dir := filepath.Dir(path)
	dirInfo, err := os.Lstat(dir)
	if os.IsNotExist(err) {
		return IdentityDecisionLog{SchemaVersion: "1"}, nil
	}
	if err != nil {
		return IdentityDecisionLog{}, err
	}
	if !dirInfo.IsDir() || dirInfo.Mode()&os.ModeSymlink != 0 {
		return IdentityDecisionLog{}, fmt.Errorf("identity log path is not a directory: %s", dir)
	}
	info, err := os.Lstat(path)
	if os.IsNotExist(err) {
		return IdentityDecisionLog{SchemaVersion: "1"}, nil
	}
	if err != nil {
		return IdentityDecisionLog{}, err
	}
	if !info.Mode().IsRegular() {
		return IdentityDecisionLog{}, fmt.Errorf("identity log is not a regular file: %s", path)
	}
	file, err := os.Open(path)
	if err != nil {
		return IdentityDecisionLog{}, err
	}
	defer file.Close()
	log := IdentityDecisionLog{SchemaVersion: "1"}
	reader := bufio.NewReader(file)
	lineNumber := 0
	for {
		text, readErr := reader.ReadString('\n')
		if text != "" {
			lineNumber++
		}
		line := strings.TrimSpace(text)
		if line == "" {
			if readErr == io.EOF {
				break
			}
			if readErr != nil {
				return IdentityDecisionLog{}, readErr
			}
			continue
		}
		var entry identityLogEntry
		if err := json.Unmarshal([]byte(line), &entry); err != nil {
			return IdentityDecisionLog{}, fmt.Errorf("decode identity log line %d: %w", lineNumber, err)
		}
		switch entry.Type {
		case "decision":
			if entry.Decision == nil {
				return IdentityDecisionLog{}, fmt.Errorf("decode identity log line %d: decision payload is required", lineNumber)
			}
			log.Decisions = append(log.Decisions, *entry.Decision)
		case "conflict":
			if entry.Conflict == nil {
				return IdentityDecisionLog{}, fmt.Errorf("decode identity log line %d: conflict payload is required", lineNumber)
			}
			log.Conflicts = append(log.Conflicts, *entry.Conflict)
		default:
			return IdentityDecisionLog{}, fmt.Errorf("decode identity log line %d: unknown entry type %q", lineNumber, entry.Type)
		}
		if readErr == io.EOF {
			break
		}
		if readErr != nil {
			return IdentityDecisionLog{}, readErr
		}
	}
	return log, nil
}

func DetectIdentityConflicts(report IdentityResolutionReport, records []PaperRecord) []IdentityConflictRecord {
	conflicts := []IdentityConflictRecord{}
	for _, cluster := range report.Clusters {
		if len(cluster.RecordIndexes) < 2 {
			continue
		}
		if reason := conflictingClusterReason(cluster, records); reason != "" {
			conflicts = append(conflicts, IdentityConflictRecord{SchemaVersion: "1", ID: cluster.ID + "-conflict-1", ClusterID: cluster.ID, Severity: "high", Reason: reason, RecordIndexes: append([]int{}, cluster.RecordIndexes...), Resolved: false})
		}
	}
	return conflicts
}

func appendIdentityLogEntry(path string, entry identityLogEntry) error {
	return appendIdentityLogEntryThen(path, entry, nil)
}

func appendIdentityLogEntryThen(path string, entry identityLogEntry, commit func() error) error {
	if strings.TrimSpace(path) == "" {
		return fmt.Errorf("identity log path is required")
	}
	dir := filepath.Dir(path)
	if err := os.MkdirAll(dir, 0o755); err != nil {
		return err
	}
	dirInfo, err := os.Lstat(dir)
	if err != nil {
		return err
	}
	if !dirInfo.IsDir() || dirInfo.Mode()&os.ModeSymlink != 0 {
		return fmt.Errorf("identity log path is not a directory: %s", dir)
	}
	existing := []byte{}
	mode := os.FileMode(0o644)
	if info, err := os.Lstat(path); err == nil {
		if !info.Mode().IsRegular() {
			return fmt.Errorf("identity log is not a regular file: %s", path)
		}
		mode = info.Mode().Perm()
		existing, err = os.ReadFile(path)
		if err != nil {
			return err
		}
	} else if !os.IsNotExist(err) {
		return err
	}
	payload, err := json.Marshal(entry)
	if err != nil {
		return err
	}
	payload = append(payload, '\n')
	data := append(existing, payload...)
	output := []filetxn.Output{{Path: path, Data: data, Mode: mode}}
	if commit != nil {
		return filetxn.ReplaceAllThen(output, commit)
	}
	return filetxn.ReplaceAll(output)
}

func conflictingClusterReason(cluster IdentityCluster, records []PaperRecord) string {
	year := 0
	title := ""
	for _, index := range cluster.RecordIndexes {
		if index < 0 || index >= len(records) {
			continue
		}
		record := records[index]
		if year != 0 && record.Year != 0 && absInt(year-record.Year) > 2 {
			return "matching identifiers but publication years differ by more than two years"
		}
		if year == 0 {
			year = record.Year
		}
		if title != "" && tokenJaccard(title, record.Title) < 0.2 {
			return "matching identifiers but titles have low token overlap"
		}
		if title == "" {
			title = record.Title
		}
	}
	return ""
}

func absInt(value int) int {
	if value < 0 {
		return -value
	}
	return value
}
