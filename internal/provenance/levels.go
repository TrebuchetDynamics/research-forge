package provenance

import (
	"crypto/rand"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"reflect"
	"strings"
	"time"

	"github.com/TrebuchetDynamics/research-forge/internal/automation"
	"github.com/TrebuchetDynamics/research-forge/internal/filetxn"
)

// Level describes mechanical provenance, never the truth of a research claim.
type Level string

const (
	Unverified     Level = "unverified"
	AgentReported  Level = "agent_reported"
	EngineVerified Level = "engine_verified"
)

type Output struct {
	Path   string `json:"path"`
	SHA256 string `json:"sha256"`
}

type Validation struct {
	Level    Level    `json:"provenance_level"`
	RunID    string   `json:"run_id,omitempty"`
	Warnings []string `json:"warnings"`
}

func minimum(a, b Level) Level {
	if a == Unverified || b == Unverified {
		return Unverified
	}
	if a == AgentReported || b == AgentReported {
		return AgentReported
	}
	return EngineVerified
}

func validLevel(l Level) bool { return l == EngineVerified || l == AgentReported || l == Unverified }
func validRunID(id string) bool {
	if id == "" {
		return false
	}
	for _, r := range id {
		if !(r >= 'a' && r <= 'z' || r >= 'A' && r <= 'Z' || r >= '0' && r <= '9' || r == '-' || r == '_') {
			return false
		}
	}
	return true
}

// Validate computes the level using output bytes, the immutable run receipt and
// its engine-kind ledger event. It does not trust a declared engine level.
func Validate(path string) (Validation, error) {
	root := filepath.Dir(path)
	if filepath.Base(root) == "runs" && filepath.Base(filepath.Dir(root)) == "provenance" {
		root = filepath.Dir(filepath.Dir(root))
	}
	data, err := filetxn.ReadRegular(path)
	if err != nil {
		return Validation{Level: Unverified}, err
	}
	return validate(root, data)
}

func validate(root string, data []byte) (Validation, error) {
	result := Validation{Level: Unverified, Warnings: []string{}}
	var record map[string]any
	if err := json.Unmarshal(data, &record); err != nil {
		return result, fmt.Errorf("invalid JSON: %w", err)
	}
	if problems := validateRecord(record); len(problems) > 0 {
		return result, fmt.Errorf("%s", strings.Join(problems, "; "))
	}
	if record["schema_version"] == "1" {
		result.Warnings = append(result.Warnings, "legacy schema v1 provenance is unverified")
		return result, nil
	}
	declared, _ := record["provenance_level"].(string)
	if !validLevel(Level(declared)) {
		return result, fmt.Errorf("provenance_level must be engine_verified, agent_reported or unverified")
	}
	outputs, err := recordOutputs(record)
	if err != nil {
		return result, err
	}
	id, _ := record["run_id"].(string)
	level := Unverified
	if declared == string(AgentReported) {
		level = AgentReported
	}
	if id != "" {
		if !validRunID(id) {
			return result, fmt.Errorf("invalid run_id %q", id)
		}
		if len(outputs) == 0 {
			return result, fmt.Errorf("engine run %s has no outputs", id)
		}
		canonical, err := readRoot(root, filepath.Join("provenance", "runs", id+".json"))
		if err != nil {
			return result, fmt.Errorf("run_id %s: %w", id, err)
		}
		var saved map[string]any
		if err := json.Unmarshal(canonical, &saved); err != nil || !reflect.DeepEqual(record, saved) {
			return result, fmt.Errorf("run_id %s does not match immutable receipt", id)
		}
		events, err := Read(root)
		if err != nil {
			return result, fmt.Errorf("engine event: %w", err)
		}
		matched := false
		for _, event := range events {
			if matchesRun(event, id, canonical) {
				matched = true
				break
			}
		}
		if !matched {
			return result, fmt.Errorf("run_id %s has no matching engine event", id)
		}
		for _, output := range outputs {
			hash, err := hashOutput(root, output.Path)
			if err != nil {
				return result, fmt.Errorf("output %s: %w", output.Path, err)
			}
			if hash != output.SHA256 {
				return result, fmt.Errorf("output checksum mismatch: %s", output.Path)
			}
		}
		level = EngineVerified
		result.RunID = id
	}
	if rawRuns, exists := record["runs"]; exists {
		runs, ok := rawRuns.([]any)
		if !ok {
			return result, fmt.Errorf("runs must be an array of run IDs")
		}
		if id != "" {
			return result, fmt.Errorf("a run receipt cannot also be an index")
		}
		if len(runs) > 0 && len(outputs) == 0 {
			level = EngineVerified
		}
		for _, rawID := range runs {
			runID, ok := rawID.(string)
			if !ok || !validRunID(runID) {
				return result, fmt.Errorf("invalid run_id %v", rawID)
			}
			bytes, err := readRoot(root, filepath.Join("provenance", "runs", runID+".json"))
			if err != nil {
				return result, fmt.Errorf("run_id %s: %w", runID, err)
			}
			var childRecord map[string]any
			if err := json.Unmarshal(bytes, &childRecord); err != nil {
				return result, err
			}
			if childRecord["run_id"] != runID {
				return result, fmt.Errorf("referenced run_id %s does not match receipt", runID)
			}
			// A referenced receipt must have its own run_id, and receipts may
			// not also be indexes; recursion is therefore bounded to one hop.
			child, err := validate(root, bytes)
			if err != nil {
				return result, err
			}
			level = minimum(level, child.Level)
			result.Warnings = append(result.Warnings, child.Warnings...)
		}
	}
	if rawEntries, exists := record["entries"]; exists {
		entries, ok := rawEntries.([]any)
		if !ok {
			return result, fmt.Errorf("entries must be an array")
		}
		for _, raw := range entries {
			entry, ok := raw.(map[string]any)
			if !ok {
				return result, fmt.Errorf("entry must be an object")
			}
			own, _ := entry["provenance_level"].(string)
			if own != string(AgentReported) && own != string(Unverified) {
				return result, fmt.Errorf("manual entries must be agent_reported or unverified")
			}
			level = minimum(level, Level(own))
		}
	}
	result.Level = level
	if declared == string(EngineVerified) && level != EngineVerified {
		return result, fmt.Errorf("declared engine_verified but computed %s", level)
	}
	return result, nil
}

func recordOutputs(record map[string]any) ([]Output, error) {
	values, ok := record["outputs"].([]any)
	if !ok {
		return nil, fmt.Errorf("outputs must be an array")
	}
	outputs := make([]Output, 0, len(values))
	for _, raw := range values {
		entry, ok := raw.(map[string]any)
		if !ok {
			return nil, fmt.Errorf("v2 outputs require path and sha256")
		}
		path, _ := entry["path"].(string)
		hash, _ := entry["sha256"].(string)
		bytes, err := hex.DecodeString(hash)
		if path == "" || filepath.IsAbs(path) || !filepath.IsLocal(path) || err != nil || len(bytes) != sha256.Size {
			return nil, fmt.Errorf("output requires a local path and sha256 checksum")
		}
		outputs = append(outputs, Output{Path: path, SHA256: hash})
	}
	return outputs, nil
}

func readRoot(root, name string) ([]byte, error) {
	file, err := filetxn.OpenRegularInRoot(root, name)
	if err != nil {
		return nil, err
	}
	defer file.Close()
	return io.ReadAll(file)
}
func hashOutput(root, name string) (string, error) {
	if !filepath.IsLocal(name) {
		return "", fmt.Errorf("output path must be local: %s", name)
	}
	file, err := filetxn.OpenRegularInRoot(root, name)
	if err != nil {
		return "", err
	}
	defer file.Close()
	hash := sha256.New()
	if _, err := io.Copy(hash, file); err != nil {
		return "", err
	}
	return hex.EncodeToString(hash.Sum(nil)), nil
}

// WriteRun writes an exclusive, immutable receipt and binds it to an engine
// event in the output directory (no initialized project is necessary).
func WriteRun(root, action string, metadata map[string]any, paths []string, aliases ...string) (Validation, error) {
	result := Validation{Level: EngineVerified, Warnings: []string{}}
	random := make([]byte, 16)
	if _, err := rand.Read(random); err != nil {
		return result, err
	}
	result.RunID = hex.EncodeToString(random)
	outputs := make([]Output, 0, len(paths))
	absRoot, err := filepath.Abs(root)
	if err != nil {
		return result, err
	}
	for _, path := range paths {
		abs, err := filepath.Abs(path)
		if err != nil {
			return result, err
		}
		name, err := filepath.Rel(absRoot, abs)
		if err != nil {
			return result, err
		}
		hash, err := hashOutput(root, name)
		if err != nil {
			return result, err
		}
		outputs = append(outputs, Output{Path: filepath.ToSlash(name), SHA256: hash})
	}
	if len(outputs) == 0 {
		return result, fmt.Errorf("engine run requires outputs")
	}
	record := map[string]any{}
	for k, v := range metadata {
		record[k] = v
	}
	record["schema_version"] = "2"
	record["provenance_level"] = EngineVerified
	record["run_id"] = result.RunID
	record["outputs"] = outputs
	record["timestamp"] = time.Now().UTC().Format(time.RFC3339Nano)
	data, err := json.MarshalIndent(record, "", "  ")
	if err != nil {
		return result, err
	}
	data = append(data, '\n')
	var normalized map[string]any
	_ = json.Unmarshal(data, &normalized)
	if problems := validateRecord(normalized); len(problems) > 0 {
		return result, fmt.Errorf("%s", strings.Join(problems, "; "))
	}
	runPath := filepath.Join(root, "provenance", "runs", result.RunID+".json")
	if err := os.MkdirAll(filepath.Dir(runPath), 0755); err != nil {
		return result, err
	}
	// OpenRoot confines creation even when a parent was replaced with a symlink.
	directory, err := os.OpenRoot(root)
	if err != nil {
		return result, err
	}
	defer directory.Close()
	file, err := directory.OpenFile(filepath.Join("provenance", "runs", result.RunID+".json"), os.O_WRONLY|os.O_CREATE|os.O_EXCL, 0444)
	if err != nil {
		return result, err
	}
	_, writeErr := file.Write(data)
	closeErr := file.Close()
	if writeErr != nil || closeErr != nil {
		_ = os.Remove(runPath)
		if writeErr != nil {
			return result, writeErr
		}
		return result, closeErr
	}
	aliasOutputs := make([]filetxn.Output, 0, len(aliases))
	for _, alias := range aliases {
		aliasOutputs = append(aliasOutputs, filetxn.Output{Path: alias, Data: data, Mode: 0644})
	}
	// Install compatibility aliases before the final event append. If either
	// fails, restore the aliases and remove the unpublished immutable receipt.
	err = filetxn.ReplaceAllThen(aliasOutputs, func() error {
		return Append(root, Event{SchemaVersion: "1", ID: "evt_" + result.RunID, RunID: result.RunID, Timestamp: record["timestamp"].(string), Actor: "rforge", ActorKind: automation.ActorEngine, Action: action, Target: filepath.ToSlash(filepath.Join("provenance", "runs", result.RunID+".json")), Inputs: map[string]any{}, Outputs: map[string]any{"record_sha256": fmt.Sprintf("%x", sha256.Sum256(data))}, Warnings: []string{}})
	})
	if err != nil {
		_ = os.Remove(runPath)
		return result, err
	}
	return result, nil
}

func matchesRun(event Event, id string, data []byte) bool {
	return validateEvent(event) == nil && event.ActorKind == automation.ActorEngine && event.RunID == id && event.Target == filepath.ToSlash(filepath.Join("provenance", "runs", id+".json")) && event.Outputs["record_sha256"] == fmt.Sprintf("%x", sha256.Sum256(data))
}

// DirectoryLevel is the fail-closed display level for current project outputs.
// A topic index takes precedence and checks every referenced run. Without an
// index, a newer checksum-bound engine event supersedes an older receipt only
// when its entire output set is covered by newer, validated runs. Historical
// receipts remain immutable and can still be validated explicitly.
func DirectoryLevel(root string) Level {
	index := filepath.Join(root, "provenance.json")
	if _, err := os.Lstat(index); err == nil {
		result, err := Validate(index)
		if err != nil {
			return Unverified
		}
		return result.Level
	} else if !os.IsNotExist(err) {
		return Unverified
	}
	paths, err := filepath.Glob(filepath.Join(root, "provenance", "runs", "*.json"))
	if err != nil || len(paths) == 0 {
		return Unverified
	}
	events, err := Read(root)
	if err != nil {
		return Unverified
	}
	type receipt struct {
		path    string
		data    []byte
		outputs []Output
	}
	pending := map[string]receipt{}
	for _, path := range paths {
		data, err := filetxn.ReadRegular(path)
		if err != nil {
			return Unverified
		}
		var record map[string]any
		if err := json.Unmarshal(data, &record); err != nil {
			return Unverified
		}
		id, _ := record["run_id"].(string)
		if !validRunID(id) || filepath.Base(path) != id+".json" {
			return Unverified
		}
		outputs, err := recordOutputs(record)
		if err != nil || len(outputs) == 0 {
			return Unverified
		}
		pending[id] = receipt{path, data, outputs}
	}
	covered := map[string]bool{}
	level := EngineVerified
	for i := len(events) - 1; i >= 0; i-- {
		event := events[i]
		record, ok := pending[event.RunID]
		if !ok || !matchesRun(event, event.RunID, record.data) {
			continue
		}
		delete(pending, event.RunID)
		superseded := true
		for _, output := range record.outputs {
			if !covered[output.Path] {
				superseded = false
			}
		}
		if superseded {
			continue
		}
		result, err := Validate(record.path)
		if err != nil {
			return Unverified
		}
		level = minimum(level, result.Level)
		for _, output := range record.outputs {
			covered[output.Path] = true
		}
	}
	if len(pending) > 0 {
		return Unverified
	}
	return level
}
