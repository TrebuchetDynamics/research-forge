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

const (
	packageSchemaVersion     = "1"
	metaAnalysisSpineVersion = "1"
	metaAnalysisPackageRole  = "meta-analysis-spine-first-done-artifact"
	packageReplayCommand     = "rforge package replay ."
	packageAuditCommand      = "rforge package audit ."
	packageReplayScript      = "#!/bin/sh\nset -eu\n" + packageReplayCommand + "\n"
)

var errSkipCopy = errors.New("skip non-regular package input")

func Create(projectPath, packagePath string, opts Options) (Package, error) {
	if strings.TrimSpace(projectPath) == "" || strings.TrimSpace(packagePath) == "" {
		return Package{}, fmt.Errorf("project and package paths are required")
	}
	if err := ValidatePackageOutputPath(projectPath, packagePath); err != nil {
		return Package{}, err
	}
	if err := guardReferenceManagerPrivacyGate(projectPath); err != nil {
		return Package{}, err
	}
	if err := validateRequiredProjectArtifacts(projectPath); err != nil {
		return Package{}, err
	}
	output, err := beginPackageOutputTransaction(packagePath)
	if err != nil {
		return Package{}, err
	}
	defer output.cleanup()
	packagePath = output.stagingPath
	if err := os.MkdirAll(filepath.Join(packagePath, "project"), 0o755); err != nil {
		return Package{}, err
	}
	clock := opts.Clock
	if clock == nil {
		clock = time.Now
	}
	now := clock().UTC()
	manifest := Manifest{SchemaVersion: packageSchemaVersion, PackageID: "rforgepkg-" + now.Format("20060102T150405Z"), CreatedAt: now.Format(time.RFC3339), CreatedBy: strings.TrimSpace(opts.CreatedBy), ResearchForgeVersion: "dev", Question: strings.TrimSpace(opts.Question), MetaAnalysisSpineVersion: metaAnalysisSpineVersion, PackageRole: metaAnalysisPackageRole, ProjectManifestRef: "project/rforge.project.toml", LockfileRef: "project/rforge.lock.json", LockfileRefs: []string{"project/rforge.lock.json"}, RedactionReportRef: "redaction-report.json", ChecksumManifestRef: "checksums.sha256", ReplayCommand: packageReplayCommand, AuditCommand: packageAuditCommand}
	redaction := RedactionReport{SchemaVersion: "1", Policy: "exclude private local paths, credentials, restricted documents, reviewer-private notes, caches"}
	copyPlan := []string{"rforge.project.toml", "rforge.lock.json", "data/provenance.jsonl", "data/forge-state.json", "data/connector-capabilities.json", "data/library.json", "data/privacy-licensing-review.json", "data/legal-acquisition-queue.json", "data/document-assets.json", "data/identity-decisions.jsonl", "data/screening.events.json", "data/screening-audit.jsonl", "data/evidence.schemas.json", "data/evidence.items.json", "data/claim-trace.json"}
	for _, rel := range copyPlan {
		if err := copyIfExists(projectPath, packagePath, rel); err != nil {
			return Package{}, fmt.Errorf("copy project artifact %s: %w", rel, err)
		}
	}
	if err := sanitizePackagedForgeState(packagePath); err != nil {
		return Package{}, err
	}
	globPlans := []struct {
		pattern string
		refs    *[]string
	}{
		{pattern: "data/*.lock.json", refs: &manifest.LockfileRefs},
		{pattern: "data/source-plans/*", refs: &manifest.SourcePlanRefs},
		{pattern: "data/import-receipts/*", refs: &manifest.ImportReceiptRefs},
		{pattern: "data/source-cache/*", refs: &manifest.SourceRecordRefs},
		{pattern: "data/reference-manager/*", refs: &manifest.ReferenceManagerReportRefs},
		{pattern: "data/parser-manifests/*", refs: &manifest.ParserManifestRefs},
		{pattern: "analysis/*", refs: &manifest.AnalysisArtifactRefs},
		{pattern: "reports/*", refs: &manifest.ReportRefs},
	}
	for _, plan := range globPlans {
		if err := copyGlob(projectPath, packagePath, plan.pattern, func(rel string) {
			*plan.refs = append(*plan.refs, "project/"+rel)
		}); err != nil {
			return Package{}, fmt.Errorf("copy project artifacts matching %s: %w", plan.pattern, err)
		}
	}
	for _, dir := range []string{"documents/open-access", "parsed"} {
		if err := copyDirFiles(projectPath, packagePath, dir); err != nil {
			return Package{}, fmt.Errorf("copy project directory %s: %w", dir, err)
		}
	}
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
	if exists(filepath.Join(packagePath, "project", "data", "screening.events.json")) {
		manifest.ScreeningAuditRef = "project/data/screening.events.json"
	} else if exists(filepath.Join(packagePath, "project", "data", "screening-audit.jsonl")) {
		manifest.ScreeningAuditRef = "project/data/screening-audit.jsonl"
	}
	if exists(filepath.Join(packagePath, "project", "data", "evidence.schemas.json")) {
		manifest.ExtractionSchemaRef = "project/data/evidence.schemas.json"
	}
	if exists(filepath.Join(packagePath, "project", "data", "evidence.items.json")) {
		manifest.AcceptedEvidenceRef = "project/data/evidence.items.json"
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
	if err := output.commit(); err != nil {
		return Package{}, err
	}
	return pkg, nil
}

type packageOutputTransaction struct {
	finalPath   string
	stagingPath string
	newDirs     []string
	committed   bool
}

func beginPackageOutputTransaction(packagePath string) (*packageOutputTransaction, error) {
	finalPath, err := filepath.Abs(packagePath)
	if err != nil {
		return nil, err
	}
	parent := filepath.Dir(finalPath)
	newDirs, err := missingPackageOutputDirectories(parent)
	if err != nil {
		return nil, err
	}
	if err := os.MkdirAll(parent, 0o755); err != nil {
		removePackageOutputDirectories(newDirs)
		return nil, err
	}
	stagingPath, err := os.MkdirTemp(parent, "."+filepath.Base(finalPath)+".rforge-stage-*")
	if err != nil {
		removePackageOutputDirectories(newDirs)
		return nil, err
	}
	return &packageOutputTransaction{finalPath: finalPath, stagingPath: stagingPath, newDirs: newDirs}, nil
}

func validateRequiredProjectArtifacts(projectPath string) error {
	for _, rel := range []string{"rforge.project.toml", "rforge.lock.json"} {
		info, err := os.Lstat(filepath.Join(projectPath, rel))
		if os.IsNotExist(err) {
			return fmt.Errorf("required project artifact is missing: %s", rel)
		}
		if err != nil {
			return fmt.Errorf("inspect required project artifact %s: %w", rel, err)
		}
		if !info.Mode().IsRegular() {
			return fmt.Errorf("required project artifact is not a regular file: %s", rel)
		}
	}
	return nil
}

func missingPackageOutputDirectories(path string) ([]string, error) {
	missing := make([]string, 0)
	for {
		info, err := os.Stat(path)
		if err == nil {
			if !info.IsDir() {
				return nil, fmt.Errorf("package output parent is not a directory: %s", path)
			}
			return missing, nil
		}
		if !os.IsNotExist(err) {
			return nil, err
		}
		missing = append(missing, path)
		parent := filepath.Dir(path)
		if parent == path {
			return nil, fmt.Errorf("package output has no existing parent directory: %s", path)
		}
		path = parent
	}
}

func removePackageOutputDirectories(paths []string) {
	for _, path := range paths {
		_ = os.Remove(path)
	}
}

func (output *packageOutputTransaction) commit() error {
	if _, err := os.Lstat(output.finalPath); err != nil {
		if !os.IsNotExist(err) {
			return err
		}
		if err := os.Rename(output.stagingPath, output.finalPath); err != nil {
			return err
		}
		output.stagingPath = ""
		output.committed = true
		return nil
	}

	backupPath, err := os.MkdirTemp(filepath.Dir(output.finalPath), "."+filepath.Base(output.finalPath)+".rforge-backup-*")
	if err != nil {
		return err
	}
	if err := os.Remove(backupPath); err != nil {
		return err
	}
	if err := os.Rename(output.finalPath, backupPath); err != nil {
		return err
	}
	if err := os.Rename(output.stagingPath, output.finalPath); err != nil {
		if removeErr := os.RemoveAll(output.finalPath); removeErr != nil {
			return fmt.Errorf("install package: %w; clear failed output before restoring prior package: %v", err, removeErr)
		}
		if restoreErr := os.Rename(backupPath, output.finalPath); restoreErr != nil {
			return fmt.Errorf("install package: %w; restore prior package from %s: %v", err, backupPath, restoreErr)
		}
		return err
	}
	output.stagingPath = ""
	output.committed = true
	_ = os.RemoveAll(backupPath)
	return nil
}

func (output *packageOutputTransaction) cleanup() {
	if output.stagingPath != "" {
		_ = os.RemoveAll(output.stagingPath)
	}
	if !output.committed {
		removePackageOutputDirectories(output.newDirs)
	}
}

// ValidatePackageOutputPath rejects package destinations that could overwrite the project, working directory, or filesystem root.
func ValidatePackageOutputPath(projectPath, packagePath string) error {
	if strings.TrimSpace(projectPath) == "" || strings.TrimSpace(packagePath) == "" {
		return fmt.Errorf("project and package paths are required")
	}
	return guardPackageOutputPath(projectPath, packagePath)
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

func copyGlob(projectPath, packagePath, pattern string, onCopy func(string)) error {
	dir, namePattern := filepath.Split(filepath.FromSlash(pattern))
	entries, err := os.ReadDir(filepath.Join(projectPath, dir))
	if os.IsNotExist(err) {
		return nil
	}
	if err != nil {
		return err
	}
	for _, entry := range entries {
		matches, err := filepath.Match(namePattern, entry.Name())
		if err != nil {
			return err
		}
		if !matches {
			continue
		}
		info, err := entry.Info()
		if err != nil {
			return err
		}
		if !info.Mode().IsRegular() {
			continue
		}
		rel := filepath.ToSlash(filepath.Join(dir, entry.Name()))
		if err := copyIfExists(projectPath, packagePath, rel); errors.Is(err, errSkipCopy) {
			continue
		} else if err != nil {
			return err
		}
		onCopy(rel)
	}
	return nil
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
	return copyAndClose(out, in)
}

func sanitizePackagedForgeState(packagePath string) error {
	path := filepath.Join(packagePath, "project", "data", "forge-state.json")
	data, err := os.ReadFile(path)
	if os.IsNotExist(err) {
		return nil
	}
	if err != nil {
		return fmt.Errorf("read packaged forge state: %w", err)
	}
	var state map[string]json.RawMessage
	if err := json.Unmarshal(data, &state); err != nil {
		return fmt.Errorf("parse packaged forge state: %w", err)
	}
	state["projectPath"] = json.RawMessage(`"project"`)
	if err := writeJSON(path, state); err != nil {
		return fmt.Errorf("sanitize packaged forge state: %w", err)
	}
	return nil
}

// copyAndClose copies src into dst and closes dst, returning the copy error
// if any, otherwise the close error. A bare deferred Close would silently
// discard a flush failure (e.g. disk full) and report a package file as
// copied when it was actually truncated.
func copyAndClose(dst io.WriteCloser, src io.Reader) error {
	_, copyErr := io.Copy(dst, src)
	closeErr := dst.Close()
	if copyErr != nil {
		return copyErr
	}
	return closeErr
}

func writePackageHelperFiles(packagePath string) error {
	if err := os.WriteFile(filepath.Join(packagePath, "replay.sh"), []byte(packageReplayScript), 0o755); err != nil {
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
