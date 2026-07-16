package reviewpkg

import (
	"bufio"
	"bytes"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/TrebuchetDynamics/research-forge/internal/documents"
	"github.com/TrebuchetDynamics/research-forge/internal/evidence"
	reportmodel "github.com/TrebuchetDynamics/research-forge/internal/report"
	"github.com/TrebuchetDynamics/research-forge/internal/screening"
	"github.com/TrebuchetDynamics/research-forge/internal/security"
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
	add("manifest_schema", manifest.SchemaVersion == packageSchemaVersion, "manifest schema version is supported")
	manifestFieldsOK, manifestFieldsMsg := auditManifestFields(manifest)
	add("manifest_fields", manifestFieldsOK, manifestFieldsMsg)
	refs := requiredRefs(manifest)
	referencePathsOK, referencePathsMsg := validatePackageReferences(packagePath, refs)
	add("reference_paths", referencePathsOK, referencePathsMsg)
	if !referencePathsOK {
		return report, nil
	}
	for _, ref := range refs {
		add("referenced_file", exists(filepath.Join(packagePath, filepath.FromSlash(ref))), ref)
	}
	add("redaction_report", manifest.RedactionReportRef != "" && exists(filepath.Join(packagePath, manifest.RedactionReportRef)), "redaction report present")
	checksumOK, checksumMsg := verifyChecksums(packagePath, manifest.ChecksumManifestRef)
	add("checksums", checksumOK, checksumMsg)
	contentPrivacyOK, contentPrivacyMsg := auditPackageContentPrivacy(packagePath)
	add("content_privacy", contentPrivacyOK, contentPrivacyMsg)
	replayScriptOK, replayScriptMsg := auditReplayScript(packagePath)
	add("replay_script", replayScriptOK, replayScriptMsg)
	add("lockfile", manifest.LockfileRef != "" && exists(filepath.Join(packagePath, filepath.FromSlash(manifest.LockfileRef))), "lockfile present")
	add("analysis_inputs", len(manifest.AnalysisArtifactRefs) > 0, "analysis artifacts referenced")
	sensitivityOK, sensitivityMsg := auditAnalysisSensitivity(packagePath, manifest.AnalysisArtifactRefs)
	add("analysis_sensitivity", sensitivityOK, sensitivityMsg)
	analysisEvidenceOK, analysisEvidenceMsg := auditAnalysisEvidence(packagePath, manifest.AcceptedEvidenceRef, manifest.AnalysisArtifactRefs)
	add("analysis_evidence", analysisEvidenceOK, analysisEvidenceMsg)
	parserManifestsOK, parserManifestsMsg := auditParserManifests(packagePath, manifest.ParserManifestRefs)
	add("parser_manifests", parserManifestsOK, parserManifestsMsg)
	add("report_outputs", len(manifest.ReportRefs) > 0, "report outputs referenced")
	reportClaimsOK, reportClaimsMsg := auditReportClaims(packagePath)
	add("report_claims", reportClaimsOK, reportClaimsMsg)
	add("provenance_links", manifest.ProvenanceRef != "" || len(manifest.SourcePlanRefs) > 0, "provenance or source plans referenced")
	screeningOK, screeningMsg := auditScreeningConflicts(packagePath, manifest.ScreeningAuditRef)
	add("screening_conflicts", screeningOK, screeningMsg)
	if len(manifest.SourcePlanRefs) > 0 {
		add("source_records", len(manifest.SourceRecordRefs) > 0 && len(manifest.ImportReceiptRefs) > 0, "source plans include imported source records and import receipts")
	}
	if manifest.LegalAcquisitionRef != "" {
		ok, msg := auditLegalAcquisition(filepath.Join(packagePath, filepath.FromSlash(manifest.LegalAcquisitionRef)))
		add("legal_acquisition", ok, msg)
	}
	if len(manifest.DocumentAssetRefs) > 0 {
		ok, msg := auditDocumentAssets(filepath.Join(packagePath, "project", "data", "document-assets.json"))
		add("document_assets", ok, msg)
	}
	if manifest.AcceptedEvidenceRef != "" {
		ok, msg := auditAcceptedEvidenceSupport(filepath.Join(packagePath, filepath.FromSlash(manifest.AcceptedEvidenceRef)))
		add("accepted_evidence_support", ok, msg)
	}
	return report, nil
}

func Replay(packagePath string) (AuditReport, error) { return Audit(packagePath) }

func auditLegalAcquisition(path string) (bool, string) {
	data, err := os.ReadFile(path)
	if err != nil {
		return false, err.Error()
	}
	var queue documents.LegalAcquisitionQueue
	if err := json.Unmarshal(data, &queue); err != nil {
		return false, err.Error()
	}
	if len(queue.Items) == 0 {
		return false, "legal acquisition queue has no items"
	}
	for _, item := range queue.Items {
		if err := documents.GuardAcquisition(item, documents.AcquisitionUseArchive); err != nil {
			return false, fmt.Sprintf("%s: %v", item.ID, err)
		}
	}
	return true, "legal acquisition queue approved for archive/shareability"
}

func auditDocumentAssets(path string) (bool, string) {
	data, err := os.ReadFile(path)
	if err != nil {
		return false, err.Error()
	}
	var assets []documents.DocumentAsset
	if err := json.Unmarshal(data, &assets); err != nil {
		return false, err.Error()
	}
	if len(assets) == 0 {
		return false, "document asset list is empty"
	}
	for _, asset := range assets {
		if err := documents.GuardExport(asset); err != nil {
			return false, fmt.Sprintf("%s: %v", asset.PaperID, err)
		}
	}
	return true, "document assets are export-safe"
}

func auditAcceptedEvidenceSupport(path string) (bool, string) {
	data, err := os.ReadFile(path)
	if err != nil {
		return false, err.Error()
	}
	var items []evidence.EvidenceItem
	if err := json.Unmarshal(data, &items); err != nil {
		return false, err.Error()
	}
	issues := evidence.Audit(items)
	if len(issues) > 0 {
		return false, fmt.Sprintf("%d accepted evidence items lack source support", len(issues))
	}
	return true, "accepted evidence has source support"
}

func auditReplayScript(packagePath string) (bool, string) {
	const ref = "replay.sh"
	if err := security.ValidatePathWithinRoot(packagePath, ref); err != nil {
		return false, fmt.Sprintf("invalid replay script path: %v", err)
	}
	info, err := os.Lstat(filepath.Join(packagePath, ref))
	if err != nil {
		return false, err.Error()
	}
	if !info.Mode().IsRegular() {
		return false, "replay script is not a regular file"
	}
	if info.Mode().Perm()&0o111 == 0 {
		return false, "replay script is not executable"
	}
	data, err := os.ReadFile(filepath.Join(packagePath, ref))
	if err != nil {
		return false, err.Error()
	}
	if string(data) != packageReplayScript {
		return false, "replay script does not match the canonical replay command"
	}
	return true, "replay script matches the canonical replay command"
}

func auditPackageContentPrivacy(packagePath string) (bool, string) {
	const (
		chunkSize = 64 * 1024
		overlap   = 512
	)
	filesScanned := 0
	err := filepath.WalkDir(packagePath, func(path string, entry os.DirEntry, err error) error {
		if err != nil || entry.IsDir() {
			return err
		}
		if !entry.Type().IsRegular() {
			return nil
		}
		file, err := os.Open(path)
		if err != nil {
			return err
		}
		defer file.Close()
		filesScanned++
		buffer := make([]byte, chunkSize)
		tail := ""
		for {
			n, readErr := file.Read(buffer)
			if n > 0 {
				text := tail + string(buffer[:n])
				if kind, found := security.DetectShareabilityLeak(text); found {
					rel, relErr := filepath.Rel(packagePath, path)
					if relErr != nil {
						return relErr
					}
					return fmt.Errorf("detected %s in %s", kind, filepath.ToSlash(rel))
				}
				if len(text) > overlap {
					tail = text[len(text)-overlap:]
				} else {
					tail = text
				}
			}
			if readErr == io.EOF {
				break
			}
			if readErr != nil {
				return readErr
			}
		}
		return nil
	})
	if err != nil {
		return false, err.Error()
	}
	return true, fmt.Sprintf("%d artifact(s) contain no detected credentials or private local paths", filesScanned)
}

func auditParserManifests(packagePath string, refs []string) (bool, string) {
	const parsedRef = "project/parsed"
	if err := security.ValidatePathWithinRoot(packagePath, parsedRef); err != nil {
		return false, fmt.Sprintf("invalid parsed output path: %v", err)
	}
	parsedFiles := make([]string, 0)
	err := filepath.WalkDir(filepath.Join(packagePath, filepath.FromSlash(parsedRef)), func(path string, entry os.DirEntry, err error) error {
		if err != nil || entry.IsDir() {
			return err
		}
		if !entry.Type().IsRegular() {
			return fmt.Errorf("unsupported non-regular parsed output %s", entry.Name())
		}
		rel, err := filepath.Rel(packagePath, path)
		if err != nil {
			return err
		}
		parsedFiles = append(parsedFiles, filepath.ToSlash(rel))
		return nil
	})
	if os.IsNotExist(err) {
		return true, "no parsed outputs require parser manifests"
	}
	if err != nil {
		return false, err.Error()
	}
	if len(parsedFiles) == 0 {
		return true, "no parsed outputs require parser manifests"
	}
	if len(refs) == 0 {
		return false, fmt.Sprintf("%d parsed output(s) have no parser manifest references", len(parsedFiles))
	}
	included := make(map[string]struct{}, len(parsedFiles))
	for _, rel := range parsedFiles {
		included[rel] = struct{}{}
	}
	covered := make(map[string]struct{}, len(parsedFiles))
	for _, ref := range refs {
		data, err := os.ReadFile(filepath.Join(packagePath, filepath.FromSlash(ref)))
		if err != nil {
			return false, fmt.Sprintf("read parser manifest %s: %v", ref, err)
		}
		var parserManifest struct {
			SchemaVersion  string `json:"schemaVersion"`
			PaperID        string `json:"paperId"`
			ParserName     string `json:"parserName"`
			ParserVersion  string `json:"parserVersion"`
			InputChecksum  string `json:"inputChecksum"`
			OutputChecksum string `json:"outputChecksum"`
			ParsedPath     string `json:"parsedPath"`
		}
		if err := json.Unmarshal(data, &parserManifest); err != nil {
			return false, fmt.Sprintf("parse parser manifest %s: %v", ref, err)
		}
		for _, field := range []struct {
			name  string
			value string
		}{
			{name: "paperId", value: parserManifest.PaperID},
			{name: "parserName", value: parserManifest.ParserName},
			{name: "parserVersion", value: parserManifest.ParserVersion},
			{name: "inputChecksum", value: parserManifest.InputChecksum},
			{name: "outputChecksum", value: parserManifest.OutputChecksum},
			{name: "parsedPath", value: parserManifest.ParsedPath},
		} {
			if strings.TrimSpace(field.value) == "" {
				return false, fmt.Sprintf("parser manifest %s is missing %s", ref, field.name)
			}
		}
		if parserManifest.SchemaVersion != packageSchemaVersion {
			return false, fmt.Sprintf("parser manifest %s has unsupported schema version %q", ref, parserManifest.SchemaVersion)
		}
		parsedPath := filepath.ToSlash(filepath.Clean(filepath.FromSlash(parserManifest.ParsedPath)))
		if !strings.HasPrefix(parsedPath, "project/") {
			parsedPath = "project/" + parsedPath
		}
		if err := security.ValidatePathWithinRoot(packagePath, parsedPath); err != nil {
			return false, fmt.Sprintf("parser manifest %s has invalid parsedPath: %v", ref, err)
		}
		if _, ok := included[parsedPath]; !ok {
			return false, fmt.Sprintf("parser manifest %s references non-included parsed output %s", ref, parsedPath)
		}
		covered[parsedPath] = struct{}{}
	}
	for _, parsedPath := range parsedFiles {
		if _, ok := covered[parsedPath]; !ok {
			return false, fmt.Sprintf("parsed output %s has no parser manifest", parsedPath)
		}
	}
	return true, fmt.Sprintf("%d parsed output(s) have parser manifest provenance", len(parsedFiles))
}

func auditAnalysisSensitivity(packagePath string, refs []string) (bool, string) {
	referenced := make(map[string]struct{}, len(refs))
	for _, ref := range refs {
		referenced[filepath.ToSlash(ref)] = struct{}{}
	}
	requiredRuns := 0
	for _, ref := range refs {
		ref = filepath.ToSlash(ref)
		if !strings.HasPrefix(ref, "project/analysis/") || !strings.HasSuffix(ref, ".json") {
			continue
		}
		data, err := os.ReadFile(filepath.Join(packagePath, filepath.FromSlash(ref)))
		if err != nil {
			return false, fmt.Sprintf("inspect analysis artifact %s: %v", ref, err)
		}
		var run struct {
			ID        string
			InputRows []struct{ ViSource string }
		}
		if err := json.Unmarshal(data, &run); err != nil || run.InputRows == nil {
			continue
		}
		hasFloor := false
		for _, row := range run.InputRows {
			if row.ViSource == "floor" {
				hasFloor = true
				break
			}
		}
		if !hasFloor {
			continue
		}
		run.ID = strings.TrimSpace(run.ID)
		if run.ID == "" || strings.ContainsAny(run.ID, `/\`) {
			return false, fmt.Sprintf("floor-imputed analysis run %s has an invalid ID", ref)
		}
		requiredRuns++
		sensitivityRef := "project/analysis/" + run.ID + "-excl-floor-result.json"
		if _, ok := referenced[sensitivityRef]; !ok || !exists(filepath.Join(packagePath, filepath.FromSlash(sensitivityRef))) {
			return false, fmt.Sprintf("missing no-floor sensitivity artifact %s for %s", sensitivityRef, ref)
		}
	}
	if requiredRuns == 0 {
		return true, "no floor-imputed analysis run requires a sensitivity artifact"
	}
	return true, fmt.Sprintf("%d floor-imputed analysis run(s) include no-floor sensitivity artifacts", requiredRuns)
}

func auditAnalysisEvidence(packagePath, evidenceRef string, refs []string) (bool, string) {
	inputPaperIDs := map[string]struct{}{}
	for _, ref := range refs {
		ref = filepath.ToSlash(ref)
		if !strings.HasPrefix(ref, "project/analysis/") || !strings.HasSuffix(ref, ".json") {
			continue
		}
		data, err := os.ReadFile(filepath.Join(packagePath, filepath.FromSlash(ref)))
		if err != nil {
			return false, fmt.Sprintf("inspect analysis artifact %s: %v", ref, err)
		}
		var run struct {
			InputRows []struct{ PaperID string }
		}
		if err := json.Unmarshal(data, &run); err != nil || run.InputRows == nil {
			continue
		}
		for _, row := range run.InputRows {
			paperID := strings.TrimSpace(row.PaperID)
			if paperID == "" {
				return false, fmt.Sprintf("analysis artifact %s has an input row without PaperID", ref)
			}
			inputPaperIDs[paperID] = struct{}{}
		}
	}
	if len(inputPaperIDs) == 0 {
		return true, "no analysis input rows require accepted-evidence matching"
	}
	if strings.TrimSpace(evidenceRef) == "" {
		return false, "analysis input rows are present without an accepted evidence reference"
	}
	data, err := os.ReadFile(filepath.Join(packagePath, filepath.FromSlash(evidenceRef)))
	if err != nil {
		return false, fmt.Sprintf("read accepted evidence %s: %v", evidenceRef, err)
	}
	var items []evidence.EvidenceItem
	if err := json.Unmarshal(data, &items); err != nil {
		return false, fmt.Sprintf("parse accepted evidence %s: %v", evidenceRef, err)
	}
	acceptedPaperIDs := map[string]struct{}{}
	for _, item := range items {
		if item.Status == evidence.StatusAccepted {
			if paperID := strings.TrimSpace(item.PaperID); paperID != "" {
				acceptedPaperIDs[paperID] = struct{}{}
			}
		}
	}
	for paperID := range inputPaperIDs {
		if _, ok := acceptedPaperIDs[paperID]; !ok {
			return false, fmt.Sprintf("analysis input PaperID %s has no matching accepted evidence", paperID)
		}
	}
	return true, fmt.Sprintf("%d analysis input paper(s) match accepted evidence", len(inputPaperIDs))
}

func auditScreeningConflicts(packagePath, ref string) (bool, string) {
	if strings.TrimSpace(ref) == "" {
		return true, "no screening audit included"
	}
	data, err := os.ReadFile(filepath.Join(packagePath, filepath.FromSlash(ref)))
	if err != nil {
		return false, fmt.Sprintf("read screening audit %s: %v", ref, err)
	}
	events := []screening.DecisionEvent{}
	trimmed := bytes.TrimSpace(data)
	if len(trimmed) > 0 && trimmed[0] == '[' {
		if err := json.Unmarshal(trimmed, &events); err != nil {
			return false, fmt.Sprintf("parse screening audit %s: %v", ref, err)
		}
	} else {
		decoder := json.NewDecoder(bytes.NewReader(trimmed))
		for {
			var event screening.DecisionEvent
			if err := decoder.Decode(&event); err != nil {
				if err == io.EOF {
					break
				}
				return false, fmt.Sprintf("parse screening audit %s after %d event(s): %v", ref, len(events), err)
			}
			events = append(events, event)
		}
	}
	type screeningKey struct {
		paperID string
		stage   screening.Stage
	}
	type decisions struct {
		include bool
		exclude bool
	}
	byPaper := map[screeningKey]decisions{}
	adjudicated := map[screeningKey]bool{}
	for i, event := range events {
		eventNumber := i + 1
		event.PaperID = strings.TrimSpace(event.PaperID)
		event.Stage = screening.Stage(strings.TrimSpace(string(event.Stage)))
		if event.PaperID == "" || event.Stage == "" {
			return false, fmt.Sprintf("screening audit %s event %d is missing paperId or stage", ref, eventNumber)
		}
		key := screeningKey{paperID: event.PaperID, stage: event.Stage}
		if event.Adjudicated {
			adjudicated[key] = true
		}
		state := byPaper[key]
		switch event.Decision {
		case screening.DecisionInclude:
			state.include = true
		case screening.DecisionExclude:
			state.exclude = true
		case screening.DecisionUncertain:
		default:
			return false, fmt.Sprintf("screening audit %s event %d has unsupported decision %q", ref, eventNumber, event.Decision)
		}
		byPaper[key] = state
	}
	for key, state := range byPaper {
		if state.include && state.exclude && !adjudicated[key] {
			return false, fmt.Sprintf("unresolved screening conflict for PaperID %s at stage %s", key.paperID, key.stage)
		}
	}
	return true, fmt.Sprintf("%d screening decision event(s) have no unresolved conflicts", len(events))
}

func auditReportClaims(packagePath string) (bool, string) {
	const ref = "project/data/claim-trace.json"
	if err := security.ValidatePathWithinRoot(packagePath, ref); err != nil {
		if os.IsNotExist(err) {
			return true, "no claim trace included"
		}
		return false, fmt.Sprintf("invalid claim trace path: %v", err)
	}
	data, err := os.ReadFile(filepath.Join(packagePath, filepath.FromSlash(ref)))
	if err != nil {
		if os.IsNotExist(err) {
			return true, "no claim trace included"
		}
		return false, fmt.Sprintf("read claim trace %s: %v", ref, err)
	}
	var trace reportmodel.CitationEvidenceTraceView
	if err := json.Unmarshal(data, &trace); err != nil {
		return false, fmt.Sprintf("parse claim trace %s: %v", ref, err)
	}
	if trace.SchemaVersion != packageSchemaVersion {
		return false, fmt.Sprintf("claim trace %s has unsupported schema version %q", ref, trace.SchemaVersion)
	}
	panel := reportmodel.BuildClaimTraceabilityPanel(trace)
	if err := reportmodel.GuardFinalReportExport(panel); err != nil {
		return false, err.Error()
	}
	return true, fmt.Sprintf("%d included report claim(s) have complete traceability", len(trace.Claims))
}

func auditManifestFields(manifest Manifest) (bool, string) {
	required := []struct {
		name  string
		value string
	}{
		{name: "packageId", value: manifest.PackageID},
		{name: "createdAt", value: manifest.CreatedAt},
		{name: "researchForgeVersion", value: manifest.ResearchForgeVersion},
		{name: "metaAnalysisSpineVersion", value: manifest.MetaAnalysisSpineVersion},
		{name: "packageRole", value: manifest.PackageRole},
		{name: "projectManifestRef", value: manifest.ProjectManifestRef},
		{name: "lockfileRef", value: manifest.LockfileRef},
		{name: "redactionReportRef", value: manifest.RedactionReportRef},
		{name: "checksumManifestRef", value: manifest.ChecksumManifestRef},
		{name: "replayCommand", value: manifest.ReplayCommand},
		{name: "auditCommand", value: manifest.AuditCommand},
	}
	for _, field := range required {
		if strings.TrimSpace(field.value) == "" {
			return false, fmt.Sprintf("required manifest field %s is missing", field.name)
		}
	}
	if len(manifest.LockfileRefs) == 0 {
		return false, "required manifest field lockfileRefs is empty"
	}
	primaryLockfile := filepath.ToSlash(filepath.Clean(filepath.FromSlash(manifest.LockfileRef)))
	hasPrimaryLockfile := false
	for _, ref := range manifest.LockfileRefs {
		if filepath.ToSlash(filepath.Clean(filepath.FromSlash(ref))) == primaryLockfile {
			hasPrimaryLockfile = true
			break
		}
	}
	if !hasPrimaryLockfile {
		return false, "manifest field lockfileRefs does not include lockfileRef"
	}
	if _, err := time.Parse(time.RFC3339, manifest.CreatedAt); err != nil {
		return false, fmt.Sprintf("manifest field createdAt is not RFC3339: %v", err)
	}
	supported := []struct {
		name     string
		actual   string
		expected string
	}{
		{name: "metaAnalysisSpineVersion", actual: manifest.MetaAnalysisSpineVersion, expected: metaAnalysisSpineVersion},
		{name: "packageRole", actual: manifest.PackageRole, expected: metaAnalysisPackageRole},
		{name: "replayCommand", actual: manifest.ReplayCommand, expected: packageReplayCommand},
		{name: "auditCommand", actual: manifest.AuditCommand, expected: packageAuditCommand},
	}
	for _, field := range supported {
		if field.actual != field.expected {
			return false, fmt.Sprintf("unsupported manifest field %s: %q", field.name, field.actual)
		}
	}
	return true, "required manifest fields are present"
}

func validatePackageReferences(packagePath string, refs []string) (bool, string) {
	for _, ref := range refs {
		if err := security.ValidatePathWithinRoot(packagePath, ref); err != nil {
			return false, fmt.Sprintf("invalid package reference %s: %v", ref, err)
		}
	}
	return true, "manifest references stay within the package root"
}

func readManifest(packagePath string) (Manifest, error) {
	if err := security.ValidatePathWithinRoot(packagePath, "manifest.json"); err != nil {
		return Manifest{}, fmt.Errorf("validate package manifest path: %w", err)
	}
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
	refs = append(refs, manifest.LockfileRefs...)
	refs = append(refs, manifest.SourcePlanRefs...)
	refs = append(refs, manifest.SourceRecordRefs...)
	refs = append(refs, manifest.ImportReceiptRefs...)
	refs = append(refs, manifest.ReferenceManagerReportRefs...)
	if manifest.LegalAcquisitionRef != "" {
		refs = append(refs, manifest.LegalAcquisitionRef)
	}
	refs = append(refs, manifest.DocumentAssetRefs...)
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
	if err := security.ValidatePathWithinRoot(packagePath, checksumRef); err != nil {
		return false, fmt.Sprintf("invalid checksum manifest path %s: %v", checksumRef, err)
	}
	file, err := os.Open(filepath.Join(packagePath, filepath.FromSlash(checksumRef)))
	if err != nil {
		return false, err.Error()
	}
	defer file.Close()
	scanner := bufio.NewScanner(file)
	verified := map[string]struct{}{}
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
		if err := security.ValidatePathWithinRoot(packagePath, rel); err != nil {
			return false, fmt.Sprintf("invalid checksum target %s: %v", rel, err)
		}
		data, err := os.ReadFile(filepath.Join(packagePath, filepath.FromSlash(rel)))
		if err != nil {
			return false, fmt.Sprintf("missing checksum target %s", rel)
		}
		sum := sha256.Sum256(data)
		if hex.EncodeToString(sum[:]) != expected {
			return false, fmt.Sprintf("checksum mismatch %s", rel)
		}
		verified[filepath.ToSlash(filepath.Clean(filepath.FromSlash(rel)))] = struct{}{}
	}
	if err := scanner.Err(); err != nil {
		return false, err.Error()
	}
	checksumRef = filepath.ToSlash(filepath.Clean(filepath.FromSlash(checksumRef)))
	if err := filepath.WalkDir(packagePath, func(path string, entry os.DirEntry, err error) error {
		if err != nil || entry.IsDir() {
			return err
		}
		rel, err := filepath.Rel(packagePath, path)
		if err != nil {
			return err
		}
		rel = filepath.ToSlash(rel)
		if rel == checksumRef {
			return nil
		}
		if !entry.Type().IsRegular() {
			return fmt.Errorf("unsupported non-regular package entry %s", rel)
		}
		if _, ok := verified[rel]; !ok {
			return fmt.Errorf("missing checksum for %s", rel)
		}
		return nil
	}); err != nil {
		return false, err.Error()
	}
	return true, "checksums verified"
}
