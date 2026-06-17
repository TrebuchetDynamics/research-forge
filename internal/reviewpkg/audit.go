package reviewpkg

import (
	"bufio"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"
)

type AuditReport struct {
	SchemaVersion string       `json:"schemaVersion"`
	OK            bool         `json:"ok"`
	Checks        []AuditCheck `json:"checks"`
}

type AuditCheck struct {
	Code    string `json:"code"`
	OK      bool   `json:"ok"`
	Message string `json:"message"`
}

func Audit(packagePath string) (AuditReport, error) {
	report := AuditReport{SchemaVersion: "1", OK: true}
	manifest, err := readManifest(packagePath)
	if err != nil {
		return AuditReport{}, err
	}
	add := func(code string, ok bool, message string) {
		report.Checks = append(report.Checks, AuditCheck{Code: code, OK: ok, Message: message})
		if !ok {
			report.OK = false
		}
	}
	add("manifest_schema", manifest.SchemaVersion == "1", "manifest schema version is supported")
	for _, ref := range requiredRefs(manifest) {
		add("referenced_file", exists(filepath.Join(packagePath, filepath.FromSlash(ref))), ref)
	}
	add("redaction_report", manifest.RedactionReportRef != "" && exists(filepath.Join(packagePath, manifest.RedactionReportRef)), "redaction report present")
	checksumOK, checksumMsg := verifyChecksums(packagePath, manifest.ChecksumManifestRef)
	add("checksums", checksumOK, checksumMsg)
	add("lockfile", manifest.LockfileRef != "" && exists(filepath.Join(packagePath, filepath.FromSlash(manifest.LockfileRef))), "lockfile present")
	add("analysis_inputs", len(manifest.AnalysisArtifactRefs) > 0, "analysis artifacts referenced")
	add("report_outputs", len(manifest.ReportRefs) > 0, "report outputs referenced")
	add("provenance_links", manifest.ProvenanceRef != "" || len(manifest.SourcePlanRefs) > 0, "provenance or source plans referenced")
	return report, nil
}

func Replay(packagePath string) (AuditReport, error) { return Audit(packagePath) }

func readManifest(packagePath string) (Manifest, error) {
	data, err := os.ReadFile(filepath.Join(packagePath, "manifest.json"))
	if err != nil {
		return Manifest{}, err
	}
	var manifest Manifest
	if err := json.Unmarshal(data, &manifest); err != nil {
		return Manifest{}, err
	}
	return manifest, nil
}

func requiredRefs(manifest Manifest) []string {
	refs := []string{manifest.ProjectManifestRef, manifest.LockfileRef, manifest.RedactionReportRef, manifest.ChecksumManifestRef}
	refs = append(refs, manifest.SourcePlanRefs...)
	refs = append(refs, manifest.ParserManifestRefs...)
	refs = append(refs, manifest.AnalysisArtifactRefs...)
	refs = append(refs, manifest.ReportRefs...)
	for _, ref := range []string{manifest.ProvenanceRef, manifest.DedupeDecisionRef, manifest.ScreeningAuditRef, manifest.ExtractionSchemaRef, manifest.AcceptedEvidenceRef} {
		if ref != "" {
			refs = append(refs, ref)
		}
	}
	out := []string{}
	for _, ref := range refs {
		if strings.TrimSpace(ref) != "" {
			out = append(out, ref)
		}
	}
	return out
}

func verifyChecksums(packagePath, checksumRef string) (bool, string) {
	if checksumRef == "" {
		checksumRef = "checksums.sha256"
	}
	file, err := os.Open(filepath.Join(packagePath, filepath.FromSlash(checksumRef)))
	if err != nil {
		return false, err.Error()
	}
	defer file.Close()
	scanner := bufio.NewScanner(file)
	for scanner.Scan() {
		line := strings.TrimSpace(scanner.Text())
		if line == "" {
			continue
		}
		parts := strings.Fields(line)
		if len(parts) < 2 {
			return false, "invalid checksum line"
		}
		expected, rel := parts[0], parts[1]
		data, err := os.ReadFile(filepath.Join(packagePath, filepath.FromSlash(rel)))
		if err != nil {
			return false, fmt.Sprintf("missing checksum target %s", rel)
		}
		sum := sha256.Sum256(data)
		if hex.EncodeToString(sum[:]) != expected {
			return false, fmt.Sprintf("checksum mismatch %s", rel)
		}
	}
	if err := scanner.Err(); err != nil {
		return false, err.Error()
	}
	return true, "checksums verified"
}
