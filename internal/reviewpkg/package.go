package reviewpkg

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"time"

	"github.com/TrebuchetDynamics/research-forge/internal/documents"
	"github.com/TrebuchetDynamics/research-forge/internal/library"
)

type Options struct {
	CreatedBy string
	Question  string
	Clock     func() time.Time
}

type Package struct {
	Manifest        Manifest        `json:"manifest"`
	RedactionReport RedactionReport `json:"redactionReport"`
}

type Manifest struct {
	SchemaVersion              string   `json:"schemaVersion"`
	PackageID                  string   `json:"packageId"`
	CreatedAt                  string   `json:"createdAt"`
	CreatedBy                  string   `json:"createdBy"`
	ResearchForgeVersion       string   `json:"researchForgeVersion"`
	ProjectTitle               string   `json:"projectTitle,omitempty"`
	Question                   string   `json:"question,omitempty"`
	MetaAnalysisSpineVersion   string   `json:"metaAnalysisSpineVersion"`
	PackageRole                string   `json:"packageRole"`
	ProjectManifestRef         string   `json:"projectManifestRef"`
	SourcePlanRefs             []string `json:"sourcePlanRefs,omitempty"`
	LockfileRef                string   `json:"lockfileRef"`
	LockfileRefs               []string `json:"lockfileRefs,omitempty"`
	ProvenanceRef              string   `json:"provenanceRef,omitempty"`
	DedupeDecisionRef          string   `json:"dedupeDecisionRef,omitempty"`
	SourceRecordRefs           []string `json:"sourceRecordRefs,omitempty"`
	ImportReceiptRefs          []string `json:"importReceiptRefs,omitempty"`
	ReferenceManagerReportRefs []string `json:"referenceManagerReportRefs,omitempty"`
	LegalAcquisitionRef        string   `json:"legalAcquisitionRef,omitempty"`
	DocumentAssetRefs          []string `json:"documentAssetRefs,omitempty"`
	ScreeningAuditRef          string   `json:"screeningAuditRef,omitempty"`
	ExtractionSchemaRef        string   `json:"extractionSchemaRef,omitempty"`
	AcceptedEvidenceRef        string   `json:"acceptedEvidenceRef,omitempty"`
	ParserManifestRefs         []string `json:"parserManifestRefs,omitempty"`
	AnalysisArtifactRefs       []string `json:"analysisArtifactRefs,omitempty"`
	ReportRefs                 []string `json:"reportRefs,omitempty"`
	RedactionReportRef         string   `json:"redactionReportRef"`
	ChecksumManifestRef        string   `json:"checksumManifestRef"`
	ReplayCommand              string   `json:"replayCommand"`
	AuditCommand               string   `json:"auditCommand"`
	Warnings                   []string `json:"warnings,omitempty"`
}

type RedactionReport struct {
	SchemaVersion string          `json:"schemaVersion"`
	Policy        string          `json:"policy"`
	Items         []RedactionItem `json:"items"`
}

type RedactionItem struct {
	Path   string `json:"path"`
	Action string `json:"action"`
	Reason string `json:"reason"`
}

var errSkipCopy = errors.New("skip non-regular package input")

func Create(projectPath, packagePath string, opts Options) (Package, error) {
	if strings.TrimSpace(projectPath) == "" || strings.TrimSpace(packagePath) == "" {
		return Package{}, fmt.Errorf("project and package paths are required")
	}
	if err := guardPackageOutputPath(projectPath, packagePath); err != nil {
		return Package{}, err
	}
	if err := guardReferenceManagerPrivacyGate(projectPath); err != nil {
		return Package{}, err
	}
	if err := os.RemoveAll(packagePath); err != nil {
		return Package{}, err
	}
	if err := os.MkdirAll(filepath.Join(packagePath, "project"), 0o755); err != nil {
		return Package{}, err
	}
	clock := opts.Clock
	if clock == nil {
		clock = time.Now
	}
	now := clock().UTC()
	manifest := Manifest{SchemaVersion: "1", PackageID: "rforgepkg-" + now.Format("20060102T150405Z"), CreatedAt: now.Format(time.RFC3339), CreatedBy: strings.TrimSpace(opts.CreatedBy), ResearchForgeVersion: "dev", Question: strings.TrimSpace(opts.Question), MetaAnalysisSpineVersion: "1", PackageRole: "meta-analysis-spine-first-done-artifact", ProjectManifestRef: "project/rforge.project.toml", LockfileRef: "project/rforge.lock.json", LockfileRefs: []string{"project/rforge.lock.json"}, RedactionReportRef: "redaction-report.json", ChecksumManifestRef: "checksums.sha256", ReplayCommand: "rforge package replay .", AuditCommand: "rforge package audit ."}
	redaction := RedactionReport{SchemaVersion: "1", Policy: "exclude private local paths, credentials, restricted documents, reviewer-private notes, caches"}
	copyPlan := []string{"rforge.project.toml", "rforge.lock.json", "data/provenance.jsonl", "data/forge-state.json", "data/connector-capabilities.json", "data/library.json", "data/privacy-licensing-review.json", "data/legal-acquisition-queue.json", "data/document-assets.json", "data/identity-decisions.jsonl", "data/screening-audit.jsonl", "data/evidence.schemas.json", "data/evidence.items.json", "data/claim-trace.json"}
	for _, rel := range copyPlan {
		_ = copyIfExists(projectPath, packagePath, rel)
	}
	copyGlob(projectPath, packagePath, "data/*.lock.json", func(rel string) { manifest.LockfileRefs = append(manifest.LockfileRefs, "project/"+rel) })
	copyGlob(projectPath, packagePath, "data/source-plans/*", func(rel string) { manifest.SourcePlanRefs = append(manifest.SourcePlanRefs, "project/"+rel) })
	copyGlob(projectPath, packagePath, "data/import-receipts/*", func(rel string) { manifest.ImportReceiptRefs = append(manifest.ImportReceiptRefs, "project/"+rel) })
	copyGlob(projectPath, packagePath, "data/source-cache/*", func(rel string) { manifest.SourceRecordRefs = append(manifest.SourceRecordRefs, "project/"+rel) })
	copyGlob(projectPath, packagePath, "data/reference-manager/*", func(rel string) {
		manifest.ReferenceManagerReportRefs = append(manifest.ReferenceManagerReportRefs, "project/"+rel)
	})
	copyGlob(projectPath, packagePath, "data/parser-manifests/*", func(rel string) { manifest.ParserManifestRefs = append(manifest.ParserManifestRefs, "project/"+rel) })
	copyGlob(projectPath, packagePath, "analysis/*", func(rel string) {
		manifest.AnalysisArtifactRefs = append(manifest.AnalysisArtifactRefs, "project/"+rel)
	})
	copyGlob(projectPath, packagePath, "reports/*", func(rel string) { manifest.ReportRefs = append(manifest.ReportRefs, "project/"+rel) })
	_ = copyDirFiles(projectPath, packagePath, "documents/open-access")
	_ = copyDirFiles(projectPath, packagePath, "parsed")
	if exists(filepath.Join(packagePath, "project", "data", "provenance.jsonl")) {
		manifest.ProvenanceRef = "project/data/provenance.jsonl"
	}
	if exists(filepath.Join(packagePath, "project", "data", "identity-decisions.jsonl")) {
		manifest.DedupeDecisionRef = "project/data/identity-decisions.jsonl"
	}
	if exists(filepath.Join(packagePath, "project", "data", "library.json")) {
		manifest.SourceRecordRefs = append(manifest.SourceRecordRefs, "project/data/library.json")
	}
	if exists(filepath.Join(packagePath, "project", "data", "legal-acquisition-queue.json")) {
		manifest.LegalAcquisitionRef = "project/data/legal-acquisition-queue.json"
	}
	if exists(filepath.Join(packagePath, "project", "data", "document-assets.json")) {
		manifest.DocumentAssetRefs = append(manifest.DocumentAssetRefs, "project/data/document-assets.json")
	}
	if exists(filepath.Join(packagePath, "project", "documents", "open-access")) {
		manifest.DocumentAssetRefs = append(manifest.DocumentAssetRefs, "project/documents/open-access")
	}
	if exists(filepath.Join(packagePath, "project", "data", "screening-audit.jsonl")) {
		manifest.ScreeningAuditRef = "project/data/screening-audit.jsonl"
	}
	if exists(filepath.Join(packagePath, "project", "data", "evidence.schemas.json")) {
		manifest.ExtractionSchemaRef = "project/data/evidence.schemas.json"
	}
	if exists(filepath.Join(packagePath, "project", "data", "evidence.items.json")) {
		manifest.AcceptedEvidenceRef = "project/data/evidence.items.json"
	}
	if !exists(filepath.Join(packagePath, manifest.ProjectManifestRef)) {
		manifest.Warnings = append(manifest.Warnings, "missing project manifest")
	}
	if !exists(filepath.Join(packagePath, manifest.LockfileRef)) {
		manifest.Warnings = append(manifest.Warnings, "missing lockfile")
	}
	redaction.Items = append(redaction.Items, RedactionItem{Path: "documents/local/", Action: "excluded", Reason: "local/restricted document assets require separate shareability approval"}, RedactionItem{Path: "documents/open-access/", Action: "included-if-approved", Reason: "only approved open-access fixture/shareable assets are copied"}, RedactionItem{Path: "cache/", Action: "excluded", Reason: "cache files are local/private state"})
	pkg := Package{Manifest: manifest, RedactionReport: redaction}
	if err := writeJSON(filepath.Join(packagePath, "redaction-report.json"), redaction); err != nil {
		return Package{}, err
	}
	if err := writeJSON(filepath.Join(packagePath, "manifest.json"), manifest); err != nil {
		return Package{}, err
	}
	if err := writePackageHelperFiles(packagePath); err != nil {
		return Package{}, err
	}
	if err := writeChecksums(packagePath); err != nil {
		return Package{}, err
	}
	return pkg, nil
}

func guardPackageOutputPath(projectPath, packagePath string) error {
	projectAbs, err := filepath.Abs(projectPath)
	if err != nil {
		return err
	}
	packageAbs, err := filepath.Abs(packagePath)
	if err != nil {
		return err
	}
	cwd, err := os.Getwd()
	if err != nil {
		return err
	}
	cwdAbs, err := filepath.Abs(cwd)
	if err != nil {
		return err
	}
	volumeRoot := filepath.VolumeName(packageAbs) + string(os.PathSeparator)
	if packageAbs == volumeRoot || packageAbs == cwdAbs || packageAbs == projectAbs || isAncestor(packageAbs, projectAbs) {
		return fmt.Errorf("refusing to overwrite unsafe package output path %s", packagePath)
	}
	return nil
}

func isAncestor(parent, child string) bool {
	rel, err := filepath.Rel(parent, child)
	return err == nil && rel != "." && rel != ".." && !strings.HasPrefix(rel, ".."+string(os.PathSeparator))
}

func guardReferenceManagerPrivacyGate(projectPath string) error {
	libraryPath := filepath.Join(projectPath, "data", "library.json")
	if !exists(libraryPath) {
		return nil
	}
	data, err := os.ReadFile(libraryPath)
	if err != nil {
		return err
	}
	var records []library.PaperRecord
	if err := json.Unmarshal(data, &records); err != nil {
		return err
	}
	review := documents.ReviewPrivacyLicensing(documents.PrivacyLicensingReviewInput{Records: records})
	if len(review.Issues) == 0 {
		return nil
	}
	reviewPath := filepath.Join(projectPath, "data", "privacy-licensing-review.json")
	data, err = os.ReadFile(reviewPath)
	if err != nil {
		return documents.GuardPrivacyLicensing(review)
	}
	var approved documents.PrivacyLicensingReview
	if err := json.Unmarshal(data, &approved); err != nil {
		return err
	}
	if err := documents.GuardPrivacyLicensing(approved); err != nil {
		return err
	}
	return nil
}

func copyGlob(projectPath, packagePath, pattern string, onCopy func(string)) {
	matches, _ := filepath.Glob(filepath.Join(projectPath, filepath.FromSlash(pattern)))
	sort.Strings(matches)
	for _, match := range matches {
		info, err := os.Lstat(match)
		if err != nil || !info.Mode().IsRegular() {
			continue
		}
		rel, _ := filepath.Rel(projectPath, match)
		if copyIfExists(projectPath, packagePath, filepath.ToSlash(rel)) == nil {
			onCopy(filepath.ToSlash(rel))
		}
	}
}

func copyDirFiles(projectPath, packagePath, dir string) error {
	root := filepath.Join(projectPath, dir)
	if !exists(root) {
		return nil
	}
	return filepath.WalkDir(root, func(path string, d os.DirEntry, err error) error {
		if err != nil || d.IsDir() {
			return err
		}
		if d.Type()&os.ModeSymlink != 0 {
			return nil
		}
		rel, _ := filepath.Rel(projectPath, path)
		if err := copyIfExists(projectPath, packagePath, filepath.ToSlash(rel)); errors.Is(err, errSkipCopy) {
			return nil
		} else {
			return err
		}
	})
}

func copyIfExists(projectPath, packagePath, rel string) error {
	src := filepath.Join(projectPath, filepath.FromSlash(rel))
	info, err := os.Lstat(src)
	if os.IsNotExist(err) {
		return nil
	}
	if err != nil {
		return err
	}
	if !info.Mode().IsRegular() {
		return errSkipCopy
	}
	dst := filepath.Join(packagePath, "project", filepath.FromSlash(rel))
	if err := os.MkdirAll(filepath.Dir(dst), 0o755); err != nil {
		return err
	}
	in, err := os.Open(src)
	if err != nil {
		return err
	}
	defer in.Close()
	out, err := os.Create(dst)
	if err != nil {
		return err
	}
	defer out.Close()
	_, err = io.Copy(out, in)
	return err
}

func writePackageHelperFiles(packagePath string) error {
	if err := os.WriteFile(filepath.Join(packagePath, "replay.sh"), []byte("#!/bin/sh\nset -eu\nrforge package replay .\n"), 0o755); err != nil {
		return err
	}
	return writeJSON(filepath.Join(packagePath, "audit-report.json"), map[string]any{"schemaVersion": "1", "status": "created", "message": "run rforge package audit . to verify checksums and replay gates"})
}

func writeJSON(path string, value any) error {
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		return err
	}
	data, err := json.MarshalIndent(value, "", "  ")
	if err != nil {
		return err
	}
	return os.WriteFile(path, data, 0o644)
}

func writeChecksums(packagePath string) error {
	lines := []string{}
	err := filepath.WalkDir(packagePath, func(path string, d os.DirEntry, err error) error {
		if err != nil || d.IsDir() {
			return err
		}
		rel, _ := filepath.Rel(packagePath, path)
		rel = filepath.ToSlash(rel)
		if rel == "checksums.sha256" {
			return nil
		}
		data, err := os.ReadFile(path)
		if err != nil {
			return err
		}
		sum := sha256.Sum256(data)
		lines = append(lines, hex.EncodeToString(sum[:])+"  "+rel)
		return nil
	})
	if err != nil {
		return err
	}
	sort.Strings(lines)
	return os.WriteFile(filepath.Join(packagePath, "checksums.sha256"), []byte(strings.Join(lines, "\n")+"\n"), 0o644)
}

func exists(path string) bool { _, err := os.Stat(path); return err == nil }
