package reviewpkg

import (
	"encoding/json"
	"os"
	"path/filepath"
	"time"

	"github.com/TrebuchetDynamics/research-forge/internal/documents"
)

const ArtificialPhotosynthesisQuestion = "Do artificial photosynthesis catalysts improve solar fuel generation outcomes?"

var fixtureTime = time.Date(2026, 1, 2, 3, 4, 5, 0, time.UTC)

// CreateArtificialPhotosynthesisFixturePackage writes an offline, deterministic
// Reproducible review package using fake-backed Meta-analysis spine artifacts.
func CreateArtificialPhotosynthesisFixturePackage(packagePath string, opts Options) (Package, error) {
	projectPath, err := os.MkdirTemp("", "rforge-artificial-photosynthesis-fixture-*")
	if err != nil {
		return Package{}, err
	}
	defer os.RemoveAll(projectPath)
	if err := WriteArtificialPhotosynthesisFixtureProject(projectPath); err != nil {
		return Package{}, err
	}
	if opts.Question == "" {
		opts.Question = ArtificialPhotosynthesisQuestion
	}
	if opts.CreatedBy == "" {
		opts.CreatedBy = "rforge fixture"
	}
	opts.Clock = func() time.Time { return fixtureTime }
	return Create(projectPath, packagePath, opts)
}

func WriteArtificialPhotosynthesisFixtureSourceImport(projectPath string) error {
	files := map[string]string{
		"data/connector-capabilities.json": `{
  "schemaVersion": "1",
  "connectors": [
    {"id":"fake-openalex","network":"none","supportsCursor":true,"supportsCitations":true},
    {"id":"fake-arxiv","network":"none","supportsCursor":false,"supportsFullTextLeads":true}
  ]
}
`,
		"data/source-plans/artificial-photosynthesis.json": `{
  "schemaVersion": "1",
  "question": "Do artificial photosynthesis catalysts improve solar fuel generation outcomes?",
  "sources": [
    {"id":"fake-openalex","query":"artificial photosynthesis catalyst solar fuel","reviewerApproved":true},
    {"id":"fake-arxiv","query":"artificial photosynthesis water splitting","reviewerApproved":true}
  ],
  "warnings": ["offline deterministic fixture; no live network calls"]
}
`,
		"data/import-receipts/fake-sources.json": `{
  "schemaVersion": "1",
  "sourcePlanRef": "data/source-plans/artificial-photosynthesis.json",
  "imports": [
    {"source":"fake-openalex","query":"artificial photosynthesis catalyst solar fuel","recordsReturned":1,"recordsImported":1,"rawPayloadRef":"data/source-cache/fake-openalex-artificial-photosynthesis.json"},
    {"source":"fake-arxiv","query":"artificial photosynthesis water splitting","recordsReturned":1,"recordsImported":1,"rawPayloadRef":"data/source-cache/fake-arxiv-artificial-photosynthesis.json"}
  ],
  "warnings": ["offline deterministic fixture; no live APIs called"]
}
`,
		"data/source-cache/fake-openalex-artificial-photosynthesis.json": `{"id":"https://openalex.org/W-FIXTURE-1","doi":"https://doi.org/10.0000/ap.fixture","title":"Fixture artificial photosynthesis catalyst review","publication_year":2026}`,
		"data/source-cache/fake-arxiv-artificial-photosynthesis.json":    `{"id":"2401.00001","title":"Fixture artificial photosynthesis water splitting preprint","doi":"10.0000/ap.preprint","year":2026}`,
		"data/library.json": `[
  {
    "Title": "Fixture artificial photosynthesis catalyst review",
    "Identifiers": {"DOI":"10.0000/ap.fixture","OpenAlexID":"W-FIXTURE-1"},
    "Authors": [{"Given":"Ada","Family":"Fixture"}],
    "Abstract": "Fixture metadata record imported from fake OpenAlex for package source provenance tests.",
    "Year": 2026,
    "Venue": "Fixture Journal",
    "OpenAccess": true,
    "SourceRefs": [{"Source":"fake-openalex","RawPayloadRef":"data/source-cache/fake-openalex-artificial-photosynthesis.json","RetrievedAt":"2026-01-02T03:04:05Z","Metadata":{"query":"artificial photosynthesis catalyst solar fuel"}}]
  },
  {
    "Title": "Fixture artificial photosynthesis water splitting preprint",
    "Identifiers": {"DOI":"10.0000/ap.preprint","ArXivID":"2401.00001"},
    "Authors": [{"Given":"Grace","Family":"Fixture"}],
    "Abstract": "Fixture metadata record imported from fake arXiv for package source provenance tests.",
    "Year": 2026,
    "Venue": "arXiv",
    "OpenAccess": true,
    "SourceRefs": [{"Source":"fake-arxiv","RawPayloadRef":"data/source-cache/fake-arxiv-artificial-photosynthesis.json","RetrievedAt":"2026-01-02T03:04:05Z","Metadata":{"query":"artificial photosynthesis water splitting"}}]
  }
]
`,
	}
	return writeFixtureFiles(projectPath, files)
}

func writeFixtureFiles(projectPath string, files map[string]string) error {
	for rel, content := range files {
		path := filepath.Join(projectPath, filepath.FromSlash(rel))
		if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
			return err
		}
		if err := os.WriteFile(path, []byte(content), 0o644); err != nil {
			return err
		}
	}
	return nil
}

func WriteArtificialPhotosynthesisReferenceManagerFixture(projectPath string) error {
	if err := WriteArtificialPhotosynthesisFixtureSourceImport(projectPath); err != nil {
		return err
	}
	files := map[string]string{
		"data/library.json": `[
  {
    "Title": "Fixture artificial photosynthesis catalyst review",
    "Identifiers": {"DOI":"10.0000/ap.fixture","OpenAlexID":"W-FIXTURE-1"},
    "Authors": [{"Given":"Ada","Family":"Fixture"}],
    "Abstract": "Fixture metadata record imported from fake OpenAlex for package source provenance tests.",
    "Year": 2026,
    "Venue": "Fixture Journal",
    "OpenAccess": true,
    "SourceRefs": [{"Source":"fake-openalex","RawPayloadRef":"data/source-cache/fake-openalex-artificial-photosynthesis.json","RetrievedAt":"2026-01-02T03:04:05Z","Metadata":{"query":"artificial photosynthesis catalyst solar fuel"}}]
  },
  {
    "Title": "Fixture artificial photosynthesis water splitting preprint",
    "Identifiers": {"DOI":"10.0000/ap.preprint","ArXivID":"2401.00001"},
    "Authors": [{"Given":"Grace","Family":"Fixture"}],
    "Abstract": "Fixture metadata record imported from fake arXiv for package source provenance tests.",
    "Year": 2026,
    "Venue": "arXiv",
    "OpenAccess": true,
    "SourceRefs": [{"Source":"fake-arxiv","RawPayloadRef":"data/source-cache/fake-arxiv-artificial-photosynthesis.json","RetrievedAt":"2026-01-02T03:04:05Z","Metadata":{"query":"artificial photosynthesis water splitting"}}]
  },
  {
    "Title": "Zotero fixture artificial photosynthesis annotations",
    "Identifiers": {"DOI":"10.0000/ap.zotero","ZoteroItemKey":"ZTAP2026"},
    "Authors": [{"Given":"Zoe","Family":"Zotero"}],
    "Abstract": "Fixture Zotero record preserving collection, tags, note, annotation, citation key, and redacted attachment basename.",
    "Year": 2026,
    "Venue": "Zotero Fixture Library",
    "OpenAccess": true,
    "SourceRefs": [{"Source":"zotero-rdf","RawPayloadRef":"data/source-cache/zotero-rdf-artificial-photosynthesis.xml","RetrievedAt":"2026-01-02T03:04:05Z","Metadata":{"collections":"Artificial Photosynthesis/Included","tags":"solar fuels; catalysts","note":"reviewer-approved fixture note","annotations":"highlighted catalyst outcome passage","citation_key":"zotero2026ap","attachment_files":"zotero-paper.pdf","linked_file_privacy_check":"redacted-local-paths"}}]
  },
  {
    "Title": "JabRef fixture artificial photosynthesis library context",
    "Identifiers": {"DOI":"10.0000/ap.jabref"},
    "Authors": [{"Given":"Jay","Family":"JabRef"}],
    "Abstract": "Fixture JabRef record preserving citation key, groups, tags, annotations, cleanup diff, and redacted attachment basename.",
    "Year": 2026,
    "Venue": "JabRef Fixture Journal",
    "OpenAccess": true,
    "SourceRefs": [{"Source":"jabref-bibtex","RawPayloadRef":"data/source-cache/jabref-artificial-photosynthesis.bib","RetrievedAt":"2026-01-02T03:04:05Z","Metadata":{"groups":"Screened/In; Included","tags":"photoelectrochemical; water splitting","note":"reviewer-approved JabRef note","annotations":"JabRef annotation fixture","citation_key":"JabRef2026AP","cleanup_diff":"doi: HTTPS://DOI.ORG/10.0000/AP.JABREF -> 10.0000/ap.jabref","attachment_files":"jabref-paper.pdf","linked_file_privacy_check":"redacted-local-paths"}}]
  }
]
`,
		"data/source-cache/zotero-rdf-artificial-photosynthesis.xml": `<rdf:RDF><z:item rdf:about="ZTAP2026"><dc:title>Zotero fixture artificial photosynthesis annotations</dc:title><better-bibtex:citekey>zotero2026ap</better-bibtex:citekey></z:item></rdf:RDF>`,
		"data/source-cache/jabref-artificial-photosynthesis.bib":     `@article{JabRef2026AP,title={JabRef fixture artificial photosynthesis library context},doi={HTTPS://DOI.ORG/10.0000/AP.JABREF},groups={Screened/In; Included},file={:/Users/alice/Zotero/storage/ABC/jabref-paper.pdf:PDF}}`,
		"data/reference-manager/fidelity.json": `{
  "schemaVersion": "1",
  "records": [
    {"title":"Zotero fixture artificial photosynthesis annotations","doi":"10.0000/ap.zotero","fields":{"collections":true,"groups":false,"tags":true,"notes":true,"annotations":true,"citation_keys":true,"bibtex_cleanup_diffs":false,"linked_file_privacy_checks":true}},
    {"title":"JabRef fixture artificial photosynthesis library context","doi":"10.0000/ap.jabref","fields":{"collections":false,"groups":true,"tags":true,"notes":true,"annotations":true,"citation_keys":true,"bibtex_cleanup_diffs":true,"linked_file_privacy_checks":true}}
  ]
}
`,
		"data/reference-manager/interchange-matrix.json": `{
  "schemaVersion": "1",
  "recordCount": 4,
  "fieldsPresent": {
    "better_bibtex_citation_key": 2,
    "tags": 2,
    "notes": 2,
    "annotations": 2,
    "collections": 1,
    "groups": 1,
    "bibtex_cleanup_diffs": 1,
    "redacted_attachments": 2
  }
}
`,
		"data/privacy-licensing-review.json": `{
  "schemaVersion": "1",
  "issues": [
    {"kind":"imported_attachment","severity":"warning","target":"Zotero fixture artificial photosynthesis annotations","message":"imported attachment metadata requires privacy/licensing review"},
    {"kind":"imported_note","severity":"warning","target":"Zotero fixture artificial photosynthesis annotations","message":"imported private notes require reviewer approval before sharing"},
    {"kind":"imported_annotation","severity":"warning","target":"Zotero fixture artificial photosynthesis annotations","message":"imported annotations require reviewer approval before sharing"},
    {"kind":"local_path","severity":"warning","target":"Zotero fixture artificial photosynthesis annotations","message":"linked local file paths were redacted and require review"},
    {"kind":"imported_attachment","severity":"warning","target":"JabRef fixture artificial photosynthesis library context","message":"imported attachment metadata requires privacy/licensing review"},
    {"kind":"imported_note","severity":"warning","target":"JabRef fixture artificial photosynthesis library context","message":"imported private notes require reviewer approval before sharing"},
    {"kind":"imported_annotation","severity":"warning","target":"JabRef fixture artificial photosynthesis library context","message":"imported annotations require reviewer approval before sharing"},
    {"kind":"local_path","severity":"warning","target":"JabRef fixture artificial photosynthesis library context","message":"linked local file paths were redacted and require review"}
  ],
  "blocked": false,
  "approved": true,
  "reviewer": "fixture",
  "approvalReason": "fixture metadata contains only redacted attachment basenames and reviewer-approved notes/annotations"
}
`,
	}
	return writeFixtureFiles(projectPath, files)
}

func WriteArtificialPhotosynthesisAcquisitionFixture(projectPath string) error {
	if err := WriteArtificialPhotosynthesisReferenceManagerFixture(projectPath); err != nil {
		return err
	}
	assetPath := filepath.Join(projectPath, "documents", "open-access", "ap-fixture.txt")
	if err := os.MkdirAll(filepath.Dir(assetPath), 0o755); err != nil {
		return err
	}
	if err := os.WriteFile(assetPath, []byte("Open-access artificial photosynthesis fixture text; no copyrighted full text.\n"), 0o644); err != nil {
		return err
	}
	asset, err := documents.NewDocumentAsset(documents.DocumentAssetInput{
		PaperID:           "doi:10.0000/ap.fixture",
		AcquisitionSource: "fixture-unpaywall",
		License:           "CC-BY-4.0",
		OAStatus:          "gold",
		LocalPath:         assetPath,
		MIMEType:          "text/plain",
	})
	if err != nil {
		return err
	}
	asset.LocalPath = "documents/open-access/ap-fixture.txt"
	assetData, err := json.MarshalIndent([]documents.DocumentAsset{asset}, "", "  ")
	if err != nil {
		return err
	}
	files := map[string]string{
		"data/legal-acquisition-queue.json": `{
  "schemaVersion": "1",
  "items": [
    {
      "id": "acq-001",
      "paperTitle": "Fixture artificial photosynthesis catalyst review",
      "doi": "10.0000/ap.fixture",
      "source": "fixture-unpaywall",
      "sourceUrl": "https://example.org/open-access/ap-fixture.txt",
      "expectedLocalPath": "documents/open-access/ap-fixture.txt",
      "license": "CC-BY-4.0",
      "oaStatus": "gold",
      "restricted": false,
      "shareable": true,
      "reviewerApprovalRequired": true,
      "reviewerApproved": true,
      "reviewer": "fixture",
      "approvalReason": "fixture open-access metadata and harmless text approved for archive",
      "provenance": "fixture open-access candidate",
      "attribution": "fixture",
      "rateLimitPolicy": "none; offline fixture",
      "apiProvenance": "no live API call"
    }
  ]
}
`,
		"data/document-assets.json": string(assetData) + "\n",
	}
	return writeFixtureFiles(projectPath, files)
}

func WriteArtificialPhotosynthesisFixtureProject(projectPath string) error {
	files := map[string]string{
		"rforge.project.toml": "schema_version = \"1\"\ntitle = \"Artificial photosynthesis fixture review\"\nstorage_mode = \"sqlite\"\n",
		"rforge.lock.json": `{
  "schemaVersion": "1",
  "createdAt": "2026-01-02T03:04:05Z",
  "tools": {
    "source": "fake-source-adapter",
    "parser": "fake-parser-adapter",
    "analysis": "fake-metafor-adapter"
  }
}
`,
		"data/provenance.jsonl": `{"schemaVersion":"1","id":"evt_fixture_protocol","timestamp":"2026-01-02T03:04:05Z","actor":"rforge fixture","action":"protocol.plan.approved","target":"source-plan","inputs":{"question":"Do artificial photosynthesis catalysts improve solar fuel generation outcomes?"},"outputs":{"sourcePlan":"data/source-plans/artificial-photosynthesis.json"},"warnings":[]}
{"schemaVersion":"1","id":"evt_fixture_package_ready","timestamp":"2026-01-02T03:04:05Z","actor":"rforge fixture","action":"package.audit.fixture_ready","target":"review-package","inputs":{},"outputs":{"report":"reports/report.md"},"warnings":[]}
`,
		"data/forge-state.json": `{
  "schemaVersion": "1",
  "currentState": "package_export",
  "question": "Do artificial photosynthesis catalysts improve solar fuel generation outcomes?",
  "validationReceipts": ["offline fixture artifacts present"]
}
`,
		"data/connector-capabilities.json": `{
  "schemaVersion": "1",
  "connectors": [
    {"id":"fake-openalex","network":"none","supportsCursor":true,"supportsCitations":true},
    {"id":"fake-arxiv","network":"none","supportsCursor":false,"supportsFullTextLeads":true}
  ]
}
`,
		"data/source-plans/artificial-photosynthesis.json": `{
  "schemaVersion": "1",
  "question": "Do artificial photosynthesis catalysts improve solar fuel generation outcomes?",
  "sources": [
    {"id":"fake-openalex","query":"artificial photosynthesis catalyst solar fuel","reviewerApproved":true},
    {"id":"fake-arxiv","query":"artificial photosynthesis water splitting","reviewerApproved":true}
  ],
  "warnings": ["offline deterministic fixture; no live network calls"]
}
`,
		"data/import-receipts/fake-sources.json": `{
  "schemaVersion": "1",
  "sourcePlanRef": "data/source-plans/artificial-photosynthesis.json",
  "imports": [
    {"source":"fake-openalex","query":"artificial photosynthesis catalyst solar fuel","recordsReturned":1,"recordsImported":1,"rawPayloadRef":"data/source-cache/fake-openalex-artificial-photosynthesis.json"},
    {"source":"fake-arxiv","query":"artificial photosynthesis water splitting","recordsReturned":1,"recordsImported":1,"rawPayloadRef":"data/source-cache/fake-arxiv-artificial-photosynthesis.json"}
  ],
  "warnings": ["offline deterministic fixture; no live APIs called"]
}
`,
		"data/source-cache/fake-openalex-artificial-photosynthesis.json": `{"id":"https://openalex.org/W-FIXTURE-1","doi":"https://doi.org/10.0000/ap.fixture","title":"Fixture artificial photosynthesis catalyst review","publication_year":2026}`,
		"data/source-cache/fake-arxiv-artificial-photosynthesis.json":    `{"id":"2401.00001","title":"Fixture artificial photosynthesis water splitting preprint","doi":"10.0000/ap.preprint","year":2026}`,
		"data/library.json": `[
  {
    "Title": "Fixture artificial photosynthesis catalyst review",
    "Identifiers": {"DOI":"10.0000/ap.fixture","OpenAlexID":"W-FIXTURE-1"},
    "Authors": [{"Given":"Ada","Family":"Fixture"}],
    "Abstract": "Fixture metadata record imported from fake OpenAlex for package source provenance tests.",
    "Year": 2026,
    "Venue": "Fixture Journal",
    "OpenAccess": true,
    "SourceRefs": [{"Source":"fake-openalex","RawPayloadRef":"data/source-cache/fake-openalex-artificial-photosynthesis.json","RetrievedAt":"2026-01-02T03:04:05Z","Metadata":{"query":"artificial photosynthesis catalyst solar fuel"}}]
  },
  {
    "Title": "Fixture artificial photosynthesis water splitting preprint",
    "Identifiers": {"DOI":"10.0000/ap.preprint","ArXivID":"2401.00001"},
    "Authors": [{"Given":"Grace","Family":"Fixture"}],
    "Abstract": "Fixture metadata record imported from fake arXiv for package source provenance tests.",
    "Year": 2026,
    "Venue": "arXiv",
    "OpenAccess": true,
    "SourceRefs": [{"Source":"fake-arxiv","RawPayloadRef":"data/source-cache/fake-arxiv-artificial-photosynthesis.json","RetrievedAt":"2026-01-02T03:04:05Z","Metadata":{"query":"artificial photosynthesis water splitting"}}]
  }
]
`,
		"data/identity-decisions.jsonl": `{"schemaVersion":"1","clusterId":"cluster-ap-001","decision":"merge","reviewer":"fixture","records":["doi:10.0000/ap.fixture","openalex:W-FIXTURE-1"],"note":"same fixture paper"}
`,
		"data/parser-manifests/fake-parser.json": `{
  "schemaVersion": "1",
  "parser": "fake-parser-adapter",
  "version": "fixture-1",
  "inputChecksum": "fixture-input",
  "outputChecksum": "fixture-output",
  "warnings": []
}
`,
		"parsed/artificial-photosynthesis-passages.json": `[
  {"id":"passage:ap-001:intro","paperId":"doi:10.0000/ap.fixture","section":"Abstract","text":"Fixture passage describing artificial photosynthesis catalyst outcomes for audit and replay tests."}
]
`,
		"data/screening-audit.jsonl": `{"schemaVersion":"1","paperId":"doi:10.0000/ap.fixture","stage":"title_abstract","decision":"include","reviewer":"fixture","reason":"matches artificial photosynthesis catalyst question"}
`,
		"data/evidence.schemas.json": `[
  {"schemaVersion":"1","name":"solar_fuel_outcome","fields":[{"Name":"outcome","Type":"string"},{"Name":"direction","Type":"string"}]}
]
`,
		"data/evidence.items.json": `[
  {
    "PaperID": "doi:10.0000/ap.fixture",
    "SchemaName": "solar_fuel_outcome",
    "Values": {"outcome":"solar fuel generation","direction":"improved in fixture evidence"},
    "Support": {"Kind":"passage","Ref":"passage:ap-001:intro"},
    "Status": "accepted",
    "History": [{"Status":"accepted","Reviewer":"fixture","Note":"accepted for deterministic package replay fixture"}]
  }
]
`,
		"analysis/run1-artifact-manifest.json": `{
  "schemaVersion": "1",
  "runId": "fixture-analysis-1",
  "engine": "fake-metafor-adapter",
  "inputEvidenceRef": "data/evidence.items.json",
  "outputs": [{"path":"analysis/forest-plot.txt","sha256":"fixture"}],
  "warnings": ["fixture analysis; no statistical claim for real research use"]
}
`,
		"analysis/forest-plot.txt": "fixture forest plot placeholder; not a real statistical result\n",
		"data/claim-trace.json": `{
  "schemaVersion": "1",
  "claims": [
    {"claimId":"claim-1","text":"The fixture package is replayable offline.","supportRefs":["passage:ap-001:intro"],"status":"supported"}
  ]
}
`,
		"reports/report.md": "# Artificial photosynthesis fixture review\n\nThis offline fixture report exists to validate package create, audit, and replay. It makes no real scientific performance claim.\n\nSupported fixture claim: the package includes a source-supported accepted evidence item (`passage:ap-001:intro`).\n",
	}
	if err := writeFixtureFiles(projectPath, files); err != nil {
		return err
	}
	return WriteArtificialPhotosynthesisAcquisitionFixture(projectPath)
}
