package reviewpkg

import (
	"crypto/sha256"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestAuditAndReplayVerifyChecksumsAndRequiredPackageLinks(t *testing.T) {
	pkgDir := createBasicAuditablePackage(t)
	report, err := Audit(pkgDir)
	if err != nil {
		t.Fatalf("Audit: %v", err)
	}
	if !report.OK || len(report.Checks) == 0 {
		t.Fatalf("report = %#v", report)
	}
	replay, err := Replay(pkgDir)
	if err != nil || !replay.OK {
		t.Fatalf("Replay report=%#v err=%v", replay, err)
	}
	if err := os.WriteFile(filepath.Join(pkgDir, "project", "reports", "report.md"), []byte("tampered"), 0o644); err != nil {
		t.Fatal(err)
	}
	report, err = Audit(pkgDir)
	if err != nil {
		t.Fatalf("Audit tampered: %v", err)
	}
	if report.OK {
		t.Fatalf("expected checksum failure, got %#v", report)
	}
}

func TestAuditRequiresSensitivityArtifactForFloorImputedRun(t *testing.T) {
	for _, tt := range []struct {
		name               string
		includeSensitivity bool
		wantOK             bool
	}{
		{name: "missing", wantOK: false},
		{name: "present", includeSensitivity: true, wantOK: true},
	} {
		t.Run(tt.name, func(t *testing.T) {
			project := t.TempDir()
			write(t, filepath.Join(project, "rforge.project.toml"), "title='Review'\n")
			write(t, filepath.Join(project, "rforge.lock.json"), `{"version":"1"}`)
			write(t, filepath.Join(project, "data", "provenance.jsonl"), `{"action":"test"}`)
			write(t, filepath.Join(project, "data", "evidence.items.json"), `[
  {"PaperID":"p1","Status":"accepted","Support":{"Kind":"passage","Ref":"passage:p1:1"}},
  {"PaperID":"p2","Status":"accepted","Support":{"Kind":"passage","Ref":"passage:p2:1"}},
  {"PaperID":"p3","Status":"accepted","Support":{"Kind":"passage","Ref":"passage:p3:1"}}
]`)
			write(t, filepath.Join(project, "analysis", "run-floor.json"), `{
  "SchemaVersion": "1",
  "ID": "run-floor",
  "InputRows": [
    {"PaperID":"p1","EffectSize":1,"Variance":0.1,"ViSource":"ci"},
    {"PaperID":"p2","EffectSize":2,"Variance":0.2,"ViSource":"floor"},
    {"PaperID":"p3","EffectSize":3,"Variance":0.3,"ViSource":"se"}
  ]
}`)
			if tt.includeSensitivity {
				write(t, filepath.Join(project, "analysis", "run-floor-excl-floor-result.json"), `{}`)
			}
			write(t, filepath.Join(project, "reports", "report.md"), `# Report`)
			pkgDir := filepath.Join(t.TempDir(), "review.rforgepkg")
			if _, err := Create(project, pkgDir, Options{}); err != nil {
				t.Fatalf("Create: %v", err)
			}

			report, err := Audit(pkgDir)
			if err != nil {
				t.Fatalf("Audit: %v", err)
			}
			if report.OK != tt.wantOK {
				t.Fatalf("Audit OK=%t, want %t: %#v", report.OK, tt.wantOK, report)
			}
			for _, check := range report.Checks {
				if check.Code == "analysis_sensitivity" {
					if check.OK != tt.wantOK {
						t.Fatalf("analysis_sensitivity OK=%t, want %t: %#v", check.OK, tt.wantOK, check)
					}
					return
				}
			}
			t.Fatalf("Audit checks missing analysis_sensitivity result: %#v", report.Checks)
		})
	}
}

func TestAuditRejectsAnalysisInputWithoutAcceptedEvidence(t *testing.T) {
	pkgDir := createBasicAuditablePackage(t)
	write(t, filepath.Join(pkgDir, "project", "analysis", "run.json"), `{
  "SchemaVersion":"1",
  "ID":"run",
  "InputRows":[{"PaperID":"paper-without-evidence","EffectSize":1,"Variance":0.1,"ViSource":"ci"}]
}`)
	write(t, filepath.Join(pkgDir, "project", "data", "evidence.items.json"), `[
  {
    "PaperID":"different-paper",
    "SchemaName":"outcome",
    "Values":{"effect":"1"},
    "Support":{"Kind":"passage","Ref":"passage:different-paper:1"},
    "Status":"accepted",
    "History":[]
  }
]`)
	if err := writeChecksums(pkgDir); err != nil {
		t.Fatalf("refresh checksums: %v", err)
	}

	report, err := Audit(pkgDir)
	if err != nil {
		t.Fatalf("Audit: %v", err)
	}
	if report.OK {
		t.Fatalf("Audit accepted analysis input without matching accepted evidence: %#v", report)
	}
	for _, check := range report.Checks {
		if check.Code == "analysis_evidence" && !check.OK {
			return
		}
	}
	t.Fatalf("Audit checks missing failed analysis_evidence result: %#v", report.Checks)
}

func TestAuditRejectsUnresolvedScreeningConflict(t *testing.T) {
	for _, tt := range []struct {
		name         string
		adjudication string
		wantOK       bool
	}{
		{name: "unresolved", wantOK: false},
		{name: "adjudicated", adjudication: `{"paperId":"p1","stage":"title_abstract","decision":"include","reviewer":"lead","adjudicated":true}` + "\n", wantOK: true},
	} {
		t.Run(tt.name, func(t *testing.T) {
			pkgDir := createBasicAuditablePackage(t)
			screeningRef := "project/data/screening-audit.jsonl"
			write(t, filepath.Join(pkgDir, filepath.FromSlash(screeningRef)), ""+
				`{"paperId":"p1","stage":"title_abstract","decision":"include","reviewer":"reviewer-a"}`+"\n"+
				`{"paperId":"p1","stage":"title_abstract","decision":"exclude","reviewer":"reviewer-b"}`+"\n"+
				tt.adjudication)
			manifest, err := readManifest(pkgDir)
			if err != nil {
				t.Fatalf("read manifest: %v", err)
			}
			manifest.ScreeningAuditRef = screeningRef
			if err := writeJSON(filepath.Join(pkgDir, "manifest.json"), manifest); err != nil {
				t.Fatalf("write manifest: %v", err)
			}
			if err := writeChecksums(pkgDir); err != nil {
				t.Fatalf("refresh checksums: %v", err)
			}

			report, err := Audit(pkgDir)
			if err != nil {
				t.Fatalf("Audit: %v", err)
			}
			if report.OK != tt.wantOK {
				t.Fatalf("Audit OK=%t, want %t: %#v", report.OK, tt.wantOK, report)
			}
			for _, check := range report.Checks {
				if check.Code == "screening_conflicts" {
					if check.OK != tt.wantOK {
						t.Fatalf("screening_conflicts OK=%t, want %t: %#v", check.OK, tt.wantOK, check)
					}
					return
				}
			}
			t.Fatalf("Audit checks missing screening_conflicts result: %#v", report.Checks)
		})
	}
}

func TestCreatePackagesProductionScreeningEventsForConflictAudit(t *testing.T) {
	project := t.TempDir()
	write(t, filepath.Join(project, "rforge.project.toml"), "title='Review'\n")
	write(t, filepath.Join(project, "rforge.lock.json"), `{"version":"1"}`)
	write(t, filepath.Join(project, "data", "provenance.jsonl"), `{"action":"test"}`)
	write(t, filepath.Join(project, "data", "evidence.items.json"), `[]`)
	write(t, filepath.Join(project, "data", "screening.events.json"), `[
  {"PaperID":"p1","Stage":"title_abstract","Decision":"include","Reviewer":"reviewer-a"},
  {"PaperID":"p1","Stage":"title_abstract","Decision":"exclude","Reviewer":"reviewer-b"}
]`)
	write(t, filepath.Join(project, "analysis", "run.json"), `{"InputRows":[]}`)
	write(t, filepath.Join(project, "reports", "report.md"), `# Report`)
	pkgDir := filepath.Join(t.TempDir(), "review.rforgepkg")
	if _, err := Create(project, pkgDir, Options{}); err != nil {
		t.Fatalf("Create: %v", err)
	}

	manifest, err := readManifest(pkgDir)
	if err != nil {
		t.Fatalf("read manifest: %v", err)
	}
	const wantRef = "project/data/screening.events.json"
	if manifest.ScreeningAuditRef != wantRef {
		t.Fatalf("ScreeningAuditRef=%q, want %q", manifest.ScreeningAuditRef, wantRef)
	}
	if _, err := os.Stat(filepath.Join(pkgDir, filepath.FromSlash(wantRef))); err != nil {
		t.Fatalf("packaged screening events: %v", err)
	}
	report, err := Audit(pkgDir)
	if err != nil {
		t.Fatalf("Audit: %v", err)
	}
	if report.OK {
		t.Fatalf("Audit accepted packaged production screening conflict: %#v", report)
	}
	for _, check := range report.Checks {
		if check.Code == "screening_conflicts" && !check.OK {
			return
		}
	}
	t.Fatalf("Audit checks missing failed screening_conflicts result: %#v", report.Checks)
}

func TestAuditRejectsUnsupportedIncludedReportClaim(t *testing.T) {
	pkgDir := createBasicAuditablePackage(t)
	write(t, filepath.Join(pkgDir, "project", "data", "claim-trace.json"), `{
  "schemaVersion":"1",
  "claims":[
    {
      "claimId":"claim-1",
      "paperId":"p1",
      "claimText":"An unsupported report claim.",
      "claimStatus":"suggested",
      "effectSizeRows":[],
      "acceptedEvidence":[],
      "passages":[]
    }
  ]
}`)
	if err := writeChecksums(pkgDir); err != nil {
		t.Fatalf("refresh checksums: %v", err)
	}

	report, err := Audit(pkgDir)
	if err != nil {
		t.Fatalf("Audit: %v", err)
	}
	if report.OK {
		t.Fatalf("Audit accepted unsupported included report claim: %#v", report)
	}
	for _, check := range report.Checks {
		if check.Code == "report_claims" && !check.OK {
			return
		}
	}
	t.Fatalf("Audit checks missing failed report_claims result: %#v", report.Checks)
}

func TestAuditRejectsNamedCredentialInIncludedTextArtifact(t *testing.T) {
	pkgDir := createBasicAuditablePackage(t)
	write(t, filepath.Join(pkgDir, "project", "reports", "report.md"), "# Report\n\napi_key = \"live-package-credential-123456\"\n")
	if err := writeChecksums(pkgDir); err != nil {
		t.Fatalf("refresh checksums: %v", err)
	}

	report, err := Audit(pkgDir)
	if err != nil {
		t.Fatalf("Audit: %v", err)
	}
	if report.OK {
		t.Fatalf("Audit accepted an included credential: %#v", report)
	}
	for _, check := range report.Checks {
		if check.Code == "content_privacy" && !check.OK {
			if strings.Contains(check.Message, "live-package-credential") {
				t.Fatalf("content_privacy disclosed the credential: %q", check.Message)
			}
			return
		}
	}
	t.Fatalf("Audit checks missing failed content_privacy result: %#v", report.Checks)
}

func TestAuditRejectsPrivateHomePathInIncludedTextArtifact(t *testing.T) {
	pkgDir := createBasicAuditablePackage(t)
	write(t, filepath.Join(pkgDir, "project", "reports", "report.md"), "# Report\n\nLocal source: /home/alice/research/private-paper.pdf\n")
	if err := writeChecksums(pkgDir); err != nil {
		t.Fatalf("refresh checksums: %v", err)
	}

	report, err := Audit(pkgDir)
	if err != nil {
		t.Fatalf("Audit: %v", err)
	}
	if report.OK {
		t.Fatalf("Audit accepted an included private home path: %#v", report)
	}
	for _, check := range report.Checks {
		if check.Code == "content_privacy" && !check.OK {
			if strings.Contains(check.Message, "alice") {
				t.Fatalf("content_privacy disclosed the private path: %q", check.Message)
			}
			return
		}
	}
	t.Fatalf("Audit checks missing failed content_privacy result: %#v", report.Checks)
}

func TestAuditAllowsRedactedCredentialPlaceholders(t *testing.T) {
	pkgDir := createBasicAuditablePackage(t)
	write(t, filepath.Join(pkgDir, "project", "reports", "report.md"), "# Report\n\napiKey: [redacted]\npassword: ${REPORT_PASSWORD}\nThe paper discusses secreted proteins.\n")
	if err := writeChecksums(pkgDir); err != nil {
		t.Fatalf("refresh checksums: %v", err)
	}

	report, err := Audit(pkgDir)
	if err != nil {
		t.Fatalf("Audit: %v", err)
	}
	if !report.OK {
		t.Fatalf("Audit rejected explicit credential placeholders: %#v", report)
	}
}

func TestAuditScansBinaryArtifactsForEmbeddedPrivatePaths(t *testing.T) {
	pkgDir := createBasicAuditablePackage(t)
	binaryPath := filepath.Join(pkgDir, "project", "analysis", "plot.bin")
	if err := os.WriteFile(binaryPath, []byte{0x00, 0x01, 'x', '=', '/', 'h', 'o', 'm', 'e', '/', 'a', 'l', 'i', 'c', 'e', '/', 'p', 'l', 'o', 't', '.', 'p', 'n', 'g', 0x00}, 0o644); err != nil {
		t.Fatalf("write binary artifact: %v", err)
	}
	if err := writeChecksums(pkgDir); err != nil {
		t.Fatalf("refresh checksums: %v", err)
	}

	report, err := Audit(pkgDir)
	if err != nil {
		t.Fatalf("Audit: %v", err)
	}
	if report.OK {
		t.Fatalf("Audit skipped an embedded private path in a binary artifact: %#v", report)
	}
	for _, check := range report.Checks {
		if check.Code == "content_privacy" && !check.OK {
			return
		}
	}
	t.Fatalf("Audit checks missing failed content_privacy result: %#v", report.Checks)
}

func TestAuditRejectsManifestReferencesOutsidePackage(t *testing.T) {
	pkgDir := createBasicAuditablePackage(t)
	outsidePath := filepath.Join(filepath.Dir(pkgDir), "outside.toml")
	write(t, outsidePath, "title='Outside'\n")
	manifest, err := readManifest(pkgDir)
	if err != nil {
		t.Fatalf("read manifest: %v", err)
	}
	manifest.ProjectManifestRef = "../outside.toml"
	if err := writeJSON(filepath.Join(pkgDir, "manifest.json"), manifest); err != nil {
		t.Fatalf("write traversing manifest: %v", err)
	}
	if err := writeChecksums(pkgDir); err != nil {
		t.Fatalf("refresh checksums: %v", err)
	}

	report, err := Audit(pkgDir)
	if err != nil {
		t.Fatalf("Audit: %v", err)
	}
	if report.OK {
		t.Fatalf("Audit accepted manifest reference outside package: %#v", report)
	}
	for _, check := range report.Checks {
		if check.Code == "reference_paths" && !check.OK {
			return
		}
	}
	t.Fatalf("Audit checks missing failed reference_paths result: %#v", report.Checks)
}

func TestAuditRejectsMissingRequiredManifestField(t *testing.T) {
	pkgDir := createBasicAuditablePackage(t)
	manifest, err := readManifest(pkgDir)
	if err != nil {
		t.Fatalf("read manifest: %v", err)
	}
	manifest.PackageID = ""
	if err := writeJSON(filepath.Join(pkgDir, "manifest.json"), manifest); err != nil {
		t.Fatalf("write incomplete manifest: %v", err)
	}
	if err := writeChecksums(pkgDir); err != nil {
		t.Fatalf("refresh checksums: %v", err)
	}

	report, err := Audit(pkgDir)
	if err != nil {
		t.Fatalf("Audit: %v", err)
	}
	if report.OK {
		t.Fatalf("Audit accepted missing packageId: %#v", report)
	}
	for _, check := range report.Checks {
		if check.Code == "manifest_fields" && !check.OK {
			return
		}
	}
	t.Fatalf("Audit checks missing failed manifest_fields result: %#v", report.Checks)
}

func TestAuditRejectsInvalidManifestTimestamp(t *testing.T) {
	pkgDir := createBasicAuditablePackage(t)
	manifest, err := readManifest(pkgDir)
	if err != nil {
		t.Fatalf("read manifest: %v", err)
	}
	manifest.CreatedAt = "not-a-timestamp"
	if err := writeJSON(filepath.Join(pkgDir, "manifest.json"), manifest); err != nil {
		t.Fatalf("write invalid manifest: %v", err)
	}
	if err := writeChecksums(pkgDir); err != nil {
		t.Fatalf("refresh checksums: %v", err)
	}

	report, err := Audit(pkgDir)
	if err != nil {
		t.Fatalf("Audit: %v", err)
	}
	if report.OK {
		t.Fatalf("Audit accepted invalid createdAt: %#v", report)
	}
	for _, check := range report.Checks {
		if check.Code == "manifest_fields" && !check.OK {
			return
		}
	}
	t.Fatalf("Audit checks missing failed manifest_fields result: %#v", report.Checks)
}

func TestAuditRejectsUnsupportedManifestLifecycleCommand(t *testing.T) {
	pkgDir := createBasicAuditablePackage(t)
	manifest, err := readManifest(pkgDir)
	if err != nil {
		t.Fatalf("read manifest: %v", err)
	}
	manifest.ReplayCommand = "rm -rf /"
	if err := writeJSON(filepath.Join(pkgDir, "manifest.json"), manifest); err != nil {
		t.Fatalf("write unsafe manifest: %v", err)
	}
	if err := writeChecksums(pkgDir); err != nil {
		t.Fatalf("refresh checksums: %v", err)
	}

	report, err := Audit(pkgDir)
	if err != nil {
		t.Fatalf("Audit: %v", err)
	}
	if report.OK {
		t.Fatalf("Audit accepted unsupported replay command: %#v", report)
	}
	for _, check := range report.Checks {
		if check.Code == "manifest_fields" && !check.OK {
			return
		}
	}
	t.Fatalf("Audit checks missing failed manifest_fields result: %#v", report.Checks)
}

func TestAuditRejectsReplayScriptThatDoesNotMatchManifestCommand(t *testing.T) {
	pkgDir := createBasicAuditablePackage(t)
	if err := os.WriteFile(filepath.Join(pkgDir, "replay.sh"), []byte("#!/bin/sh\nrm -rf /\n"), 0o755); err != nil {
		t.Fatalf("write unsafe replay script: %v", err)
	}
	if err := writeChecksums(pkgDir); err != nil {
		t.Fatalf("refresh checksums: %v", err)
	}

	report, err := Audit(pkgDir)
	if err != nil {
		t.Fatalf("Audit: %v", err)
	}
	if report.OK {
		t.Fatalf("Audit accepted replay script that contradicts manifest: %#v", report)
	}
	for _, check := range report.Checks {
		if check.Code == "replay_script" && !check.OK {
			return
		}
	}
	t.Fatalf("Audit checks missing failed replay_script result: %#v", report.Checks)
}

func TestAuditRejectsMissingDeclaredLockfileReference(t *testing.T) {
	pkgDir := createBasicAuditablePackage(t)
	manifest, err := readManifest(pkgDir)
	if err != nil {
		t.Fatalf("read manifest: %v", err)
	}
	manifest.LockfileRefs = append(manifest.LockfileRefs, "project/data/missing.lock.json")
	if err := writeJSON(filepath.Join(pkgDir, "manifest.json"), manifest); err != nil {
		t.Fatalf("write incomplete manifest: %v", err)
	}
	if err := writeChecksums(pkgDir); err != nil {
		t.Fatalf("refresh checksums: %v", err)
	}

	report, err := Audit(pkgDir)
	if err != nil {
		t.Fatalf("Audit: %v", err)
	}
	if report.OK {
		t.Fatalf("Audit accepted missing declared lockfile: %#v", report)
	}
	for _, check := range report.Checks {
		if check.Code == "referenced_file" && !check.OK && strings.Contains(check.Message, "missing.lock.json") {
			return
		}
	}
	t.Fatalf("Audit checks missing failed lockfile referenced_file result: %#v", report.Checks)
}

func TestAuditRequiresPrimaryLockfileInDeclaredLockfileReferences(t *testing.T) {
	pkgDir := createBasicAuditablePackage(t)
	secondaryRef := "project/data/secondary.lock.json"
	write(t, filepath.Join(pkgDir, filepath.FromSlash(secondaryRef)), `{}`)
	manifest, err := readManifest(pkgDir)
	if err != nil {
		t.Fatalf("read manifest: %v", err)
	}
	manifest.LockfileRefs = []string{secondaryRef}
	if err := writeJSON(filepath.Join(pkgDir, "manifest.json"), manifest); err != nil {
		t.Fatalf("write inconsistent manifest: %v", err)
	}
	if err := writeChecksums(pkgDir); err != nil {
		t.Fatalf("refresh checksums: %v", err)
	}

	report, err := Audit(pkgDir)
	if err != nil {
		t.Fatalf("Audit: %v", err)
	}
	if report.OK {
		t.Fatalf("Audit accepted lockfileRefs without primary lockfile: %#v", report)
	}
	for _, check := range report.Checks {
		if check.Code == "manifest_fields" && !check.OK {
			return
		}
	}
	t.Fatalf("Audit checks missing failed manifest_fields result: %#v", report.Checks)
}

func TestAuditRequiresParserManifestForIncludedParsedOutput(t *testing.T) {
	pkgDir := createBasicAuditablePackage(t)
	write(t, filepath.Join(pkgDir, "project", "parsed", "paper.json"), `{"paperId":"p1"}`)
	if err := writeChecksums(pkgDir); err != nil {
		t.Fatalf("refresh checksums: %v", err)
	}

	report, err := Audit(pkgDir)
	if err != nil {
		t.Fatalf("Audit: %v", err)
	}
	if report.OK {
		t.Fatalf("Audit accepted parsed output without parser manifest: %#v", report)
	}
	for _, check := range report.Checks {
		if check.Code == "parser_manifests" && !check.OK {
			return
		}
	}
	t.Fatalf("Audit checks missing failed parser_manifests result: %#v", report.Checks)
}

func TestAuditRequiresParserManifestToCoverIncludedParsedOutput(t *testing.T) {
	pkgDir := createBasicAuditablePackage(t)
	write(t, filepath.Join(pkgDir, "project", "parsed", "paper.json"), `{"paperId":"p1"}`)
	parserManifestRef := "project/data/parser-manifests/parser.json"
	write(t, filepath.Join(pkgDir, filepath.FromSlash(parserManifestRef)), `{
  "schemaVersion":"1",
  "paperId":"p1",
  "parserName":"fake-parser",
  "parserVersion":"1",
  "inputChecksum":"input",
  "outputChecksum":"output",
  "parsedPath":"parsed/other.json"
}`)
	manifest, err := readManifest(pkgDir)
	if err != nil {
		t.Fatalf("read manifest: %v", err)
	}
	manifest.ParserManifestRefs = []string{parserManifestRef}
	if err := writeJSON(filepath.Join(pkgDir, "manifest.json"), manifest); err != nil {
		t.Fatalf("write manifest: %v", err)
	}
	if err := writeChecksums(pkgDir); err != nil {
		t.Fatalf("refresh checksums: %v", err)
	}

	report, err := Audit(pkgDir)
	if err != nil {
		t.Fatalf("Audit: %v", err)
	}
	if report.OK {
		t.Fatalf("Audit accepted parser manifest for a different parsed output: %#v", report)
	}
	for _, check := range report.Checks {
		if check.Code == "parser_manifests" && !check.OK {
			return
		}
	}
	t.Fatalf("Audit checks missing failed parser_manifests result: %#v", report.Checks)
}

func TestAuditRejectsChecksumTargetsOutsidePackage(t *testing.T) {
	pkgDir := createBasicAuditablePackage(t)
	outsideData := []byte("outside package\n")
	write(t, filepath.Join(filepath.Dir(pkgDir), "outside.txt"), string(outsideData))
	checksumPath := filepath.Join(pkgDir, "checksums.sha256")
	checksums, err := os.ReadFile(checksumPath)
	if err != nil {
		t.Fatalf("read checksums: %v", err)
	}
	sum := sha256.Sum256(outsideData)
	checksums = append(checksums, []byte(fmt.Sprintf("%x ../outside.txt\n", sum))...)
	if err := os.WriteFile(checksumPath, checksums, 0o644); err != nil {
		t.Fatalf("write traversing checksum: %v", err)
	}

	report, err := Audit(pkgDir)
	if err != nil {
		t.Fatalf("Audit: %v", err)
	}
	if report.OK {
		t.Fatalf("Audit accepted checksum target outside package: %#v", report)
	}
	for _, check := range report.Checks {
		if check.Code == "checksums" && !check.OK {
			return
		}
	}
	t.Fatalf("Audit checks missing failed checksums result: %#v", report.Checks)
}

func TestAuditRejectsRequiredFileMissingFromChecksumManifest(t *testing.T) {
	pkgDir := createBasicAuditablePackage(t)
	checksumPath := filepath.Join(pkgDir, "checksums.sha256")
	checksums, err := os.ReadFile(checksumPath)
	if err != nil {
		t.Fatalf("read checksums: %v", err)
	}
	kept := make([]string, 0)
	for _, line := range strings.Split(string(checksums), "\n") {
		if line != "" && !strings.HasSuffix(line, "project/reports/report.md") {
			kept = append(kept, line)
		}
	}
	if err := os.WriteFile(checksumPath, []byte(strings.Join(kept, "\n")+"\n"), 0o644); err != nil {
		t.Fatalf("omit report checksum: %v", err)
	}
	if err := os.WriteFile(filepath.Join(pkgDir, "project", "reports", "report.md"), []byte("tampered\n"), 0o644); err != nil {
		t.Fatalf("tamper report: %v", err)
	}

	report, err := Audit(pkgDir)
	if err != nil {
		t.Fatalf("Audit: %v", err)
	}
	if report.OK {
		t.Fatalf("Audit accepted required file omitted from checksums: %#v", report)
	}
	for _, check := range report.Checks {
		if check.Code == "checksums" && !check.OK {
			return
		}
	}
	t.Fatalf("Audit checks missing failed checksums result: %#v", report.Checks)
}

func TestAuditRejectsSymlinkedManifestBeforeReading(t *testing.T) {
	pkgDir := createBasicAuditablePackage(t)
	manifestPath := filepath.Join(pkgDir, "manifest.json")
	manifestData, err := os.ReadFile(manifestPath)
	if err != nil {
		t.Fatalf("read manifest: %v", err)
	}
	outsidePath := filepath.Join(filepath.Dir(pkgDir), "outside-manifest.json")
	if err := os.WriteFile(outsidePath, manifestData, 0o644); err != nil {
		t.Fatalf("write outside manifest: %v", err)
	}
	if err := os.Remove(manifestPath); err != nil {
		t.Fatalf("remove package manifest: %v", err)
	}
	if err := os.Symlink(outsidePath, manifestPath); err != nil {
		t.Skipf("symlinks unavailable: %v", err)
	}

	if _, err := Audit(pkgDir); err == nil || !strings.Contains(err.Error(), "symlink") {
		t.Fatalf("Audit symlinked manifest error=%v, want symlink rejection before read", err)
	}
}

func createBasicAuditablePackage(t *testing.T) string {
	t.Helper()
	project := t.TempDir()
	write(t, filepath.Join(project, "rforge.project.toml"), "title='Review'\n")
	write(t, filepath.Join(project, "rforge.lock.json"), `{"version":"1"}`)
	write(t, filepath.Join(project, "data", "provenance.jsonl"), `{"action":"test"}`)
	write(t, filepath.Join(project, "data", "evidence.items.json"), `[]`)
	write(t, filepath.Join(project, "analysis", "run.json"), `{"InputRows":[]}`)
	write(t, filepath.Join(project, "reports", "report.md"), `# Report`)
	pkgDir := filepath.Join(t.TempDir(), "review.rforgepkg")
	if _, err := Create(project, pkgDir, Options{}); err != nil {
		t.Fatalf("Create: %v", err)
	}
	return pkgDir
}
