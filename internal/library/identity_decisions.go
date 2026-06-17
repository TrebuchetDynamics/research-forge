package library

import (
	"bufio"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"
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
	if strings.TrimSpace(decision.ID) == "" || strings.TrimSpace(decision.ClusterID) == "" {
		return fmt.Errorf("identity decision id and cluster id are required")
	}
	if decision.Action != IdentityDecisionMerge && decision.Action != IdentityDecisionSplit {
		return fmt.Errorf("identity decision action must be merge or split")
	}
	decision.SchemaVersion = "1"
	decision.Reversible = true
	return appendIdentityLogEntry(path, identityLogEntry{Type: "decision", Decision: &decision})
}

func AppendIdentityConflict(path string, conflict IdentityConflictRecord) error {
	if strings.TrimSpace(conflict.ID) == "" || strings.TrimSpace(conflict.ClusterID) == "" {
		return fmt.Errorf("identity conflict id and cluster id are required")
	}
	conflict.SchemaVersion = "1"
	return appendIdentityLogEntry(path, identityLogEntry{Type: "conflict", Conflict: &conflict})
}

func ApplyIdentityDecision(records []PaperRecord, decision IdentityDecision) ([]PaperRecord, error) {
	if decision.Action != IdentityDecisionMerge && decision.Action != IdentityDecisionSplit {
		return nil, fmt.Errorf("identity decision action must be merge or split")
	}
	if len(decision.After) == 0 {
		return nil, fmt.Errorf("identity decision after state is required")
	}
	out := append([]PaperRecord{}, decision.After...)
	return out, nil
}

func ReadIdentityDecisionLog(path string) (IdentityDecisionLog, error) {
	file, err := os.Open(path)
	if os.IsNotExist(err) {
		return IdentityDecisionLog{SchemaVersion: "1"}, nil
	}
	if err != nil {
		return IdentityDecisionLog{}, err
	}
	defer file.Close()
	log := IdentityDecisionLog{SchemaVersion: "1"}
	scanner := bufio.NewScanner(file)
	for scanner.Scan() {
		line := strings.TrimSpace(scanner.Text())
		if line == "" {
			continue
		}
		var entry identityLogEntry
		if err := json.Unmarshal([]byte(line), &entry); err != nil {
			return IdentityDecisionLog{}, err
		}
		switch entry.Type {
		case "decision":
			if entry.Decision != nil {
				log.Decisions = append(log.Decisions, *entry.Decision)
			}
		case "conflict":
			if entry.Conflict != nil {
				log.Conflicts = append(log.Conflicts, *entry.Conflict)
			}
		}
	}
	return log, scanner.Err()
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
	if strings.TrimSpace(path) == "" {
		return fmt.Errorf("identity log path is required")
	}
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		return err
	}
	payload, err := json.Marshal(entry)
	if err != nil {
		return err
	}
	payload = append(payload, '\n')
	file, err := os.OpenFile(path, os.O_CREATE|os.O_APPEND|os.O_WRONLY, 0o644)
	if err != nil {
		return err
	}
	defer file.Close()
	_, err = file.Write(payload)
	return err
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
