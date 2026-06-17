package reviewpkg

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"time"
)

type Options struct {
	CreatedBy string
	Question  string
}

type Package struct {
	Manifest        Manifest        `json:"manifest"`
	RedactionReport RedactionReport `json:"redactionReport"`
}

type Manifest struct {
	SchemaVersion            string   `json:"schemaVersion"`
	PackageID                string   `json:"packageId"`
	CreatedAt                string   `json:"createdAt"`
	CreatedBy                string   `json:"createdBy"`
	ResearchForgeVersion     string   `json:"researchForgeVersion"`
	ProjectTitle             string   `json:"projectTitle,omitempty"`
	Question                 string   `json:"question,omitempty"`
	MetaAnalysisSpineVersion string   `json:"metaAnalysisSpineVersion"`
	ProjectManifestRef       string   `json:"projectManifestRef"`
	SourcePlanRefs           []string `json:"sourcePlanRefs,omitempty"`
	LockfileRef              string   `json:"lockfileRef"`
	ProvenanceRef            string   `json:"provenanceRef,omitempty"`
	DedupeDecisionRef        string   `json:"dedupeDecisionRef,omitempty"`
	ScreeningAuditRef        string   `json:"screeningAuditRef,omitempty"`
	ExtractionSchemaRef      string   `json:"extractionSchemaRef,omitempty"`
	AcceptedEvidenceRef      string   `json:"acceptedEvidenceRef,omitempty"`
	ParserManifestRefs       []string `json:"parserManifestRefs,omitempty"`
	AnalysisArtifactRefs     []string `json:"analysisArtifactRefs,omitempty"`
	ReportRefs               []string `json:"reportRefs,omitempty"`
	RedactionReportRef       string   `json:"redactionReportRef"`
	ChecksumManifestRef      string   `json:"checksumManifestRef"`
	ReplayCommand            string   `json:"replayCommand"`
	AuditCommand             string   `json:"auditCommand"`
	Warnings                 []string `json:"warnings,omitempty"`
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

func Create(projectPath, packagePath string, opts Options) (Package, error) {
	if strings.TrimSpace(projectPath) == "" || strings.TrimSpace(packagePath) == "" {
		return Package{}, fmt.Errorf("project and package paths are required")
	}
	if err := os.RemoveAll(packagePath); err != nil {
		return Package{}, err
	}
	if err := os.MkdirAll(filepath.Join(packagePath, "project"), 0o755); err != nil {
		return Package{}, err
	}
	manifest := Manifest{SchemaVersion: "1", PackageID: "rforgepkg-" + time.Now().UTC().Format("20060102T150405Z"), CreatedAt: time.Now().UTC().Format(time.RFC3339), CreatedBy: strings.TrimSpace(opts.CreatedBy), ResearchForgeVersion: "dev", Question: strings.TrimSpace(opts.Question), MetaAnalysisSpineVersion: "1", ProjectManifestRef: "project/rforge.project.toml", LockfileRef: "project/rforge.lock.json", RedactionReportRef: "redaction-report.json", ChecksumManifestRef: "checksums.sha256", ReplayCommand: "rforge package replay .", AuditCommand: "rforge package audit ."}
	redaction := RedactionReport{SchemaVersion: "1", Policy: "exclude private local paths, credentials, restricted documents, reviewer-private notes, caches"}
	copyPlan := []string{"rforge.project.toml", "rforge.lock.json", "data/provenance.jsonl", "data/forge-state.json", "data/connector-capabilities.json", "data/identity-decisions.jsonl", "data/screening-audit.jsonl", "data/evidence.schemas.json", "data/evidence.items.json", "data/claim-trace.json"}
	for _, rel := range copyPlan {
		_ = copyIfExists(projectPath, packagePath, rel)
	}
	copyGlob(projectPath, packagePath, "data/source-plans/*", func(rel string) { manifest.SourcePlanRefs = append(manifest.SourcePlanRefs, "project/"+rel) })
	copyGlob(projectPath, packagePath, "data/parser-manifests/*", func(rel string) { manifest.ParserManifestRefs = append(manifest.ParserManifestRefs, "project/"+rel) })
	copyGlob(projectPath, packagePath, "analysis/*", func(rel string) {
		manifest.AnalysisArtifactRefs = append(manifest.AnalysisArtifactRefs, "project/"+rel)
	})
	copyGlob(projectPath, packagePath, "reports/*", func(rel string) { manifest.ReportRefs = append(manifest.ReportRefs, "project/"+rel) })
	_ = copyDirFiles(projectPath, packagePath, "parsed")
	if exists(filepath.Join(packagePath, "project", "data", "provenance.jsonl")) {
		manifest.ProvenanceRef = "project/data/provenance.jsonl"
	}
	if exists(filepath.Join(packagePath, "project", "data", "identity-decisions.jsonl")) {
		manifest.DedupeDecisionRef = "project/data/identity-decisions.jsonl"
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
	redaction.Items = append(redaction.Items, RedactionItem{Path: "documents/", Action: "excluded-by-default", Reason: "document assets require shareability approval"}, RedactionItem{Path: "cache/", Action: "excluded", Reason: "cache files are local/private state"})
	pkg := Package{Manifest: manifest, RedactionReport: redaction}
	if err := writeJSON(filepath.Join(packagePath, "redaction-report.json"), redaction); err != nil {
		return Package{}, err
	}
	if err := writeJSON(filepath.Join(packagePath, "manifest.json"), manifest); err != nil {
		return Package{}, err
	}
	if err := writeChecksums(packagePath); err != nil {
		return Package{}, err
	}
	return pkg, nil
}

func copyGlob(projectPath, packagePath, pattern string, onCopy func(string)) {
	matches, _ := filepath.Glob(filepath.Join(projectPath, filepath.FromSlash(pattern)))
	sort.Strings(matches)
	for _, match := range matches {
		info, err := os.Stat(match)
		if err != nil || info.IsDir() {
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
		rel, _ := filepath.Rel(projectPath, path)
		return copyIfExists(projectPath, packagePath, filepath.ToSlash(rel))
	})
}

func copyIfExists(projectPath, packagePath, rel string) error {
	src := filepath.Join(projectPath, filepath.FromSlash(rel))
	if !exists(src) {
		return nil
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
