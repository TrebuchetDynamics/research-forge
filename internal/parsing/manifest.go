package parsing

import (
	"crypto/sha256"
	"encoding/hex"
	"strings"
)

// ParserRunManifest records auditable parser-run provenance and layer counts.
type ParserRunManifest struct {
	SchemaVersion            string   `json:"schemaVersion"`
	PaperID                  string   `json:"paperId"`
	ParserName               string   `json:"parserName"`
	ParserVersion            string   `json:"parserVersion"`
	ParserSource             string   `json:"parserSource"`
	Command                  []string `json:"command,omitempty"`
	InputChecksum            string   `json:"inputChecksum"`
	OutputChecksum           string   `json:"outputChecksum"`
	ParsedPath               string   `json:"parsedPath"`
	LicenseConstraints       string   `json:"licenseConstraints"`
	Shareability             string   `json:"shareability"`
	ProvenanceFields         []string `json:"provenanceFields"`
	ReviewerApprovalRequired bool     `json:"reviewerApprovalRequired"`
	Sections                 int      `json:"sections"`
	Passages                 int      `json:"passages"`
	References               int      `json:"references"`
	Warnings                 []string `json:"warnings,omitempty"`
}

// NewParserRunManifest summarizes one parser output for reproducibility and comparison.
func NewParserRunManifest(doc ParsedDocument, input []byte, parsedPath string) ParserRunManifest {
	return NewParserRunManifestWithOutput(doc, input, nil, parsedPath, nil)
}

func NewParserRunManifestWithOutput(doc ParsedDocument, input, output []byte, parsedPath string, command []string) ParserRunManifest {
	passages := 0
	for _, section := range doc.Sections {
		passages += len(section.Passages)
	}
	inputSum := sha256.Sum256(input)
	outputSum := sha256.Sum256(output)
	policy := DefaultParserOutputPolicies().MustPolicy(doc.ParserName)
	return ParserRunManifest{SchemaVersion: "1", PaperID: doc.PaperID, ParserName: doc.ParserName, ParserVersion: doc.ParserVersion, ParserSource: policy.ParserSource, Command: command, InputChecksum: hex.EncodeToString(inputSum[:]), OutputChecksum: hex.EncodeToString(outputSum[:]), ParsedPath: parsedPath, LicenseConstraints: policy.LicenseConstraints, Shareability: policy.Shareability, ProvenanceFields: append([]string{}, policy.ProvenanceFields...), ReviewerApprovalRequired: true, Sections: len(doc.Sections), Passages: passages, References: len(doc.References), Warnings: append([]string{}, doc.Warnings...)}
}

type ParserOutputPolicyRegistry struct {
	SchemaVersion string               `json:"schemaVersion"`
	Policies      []ParserOutputPolicy `json:"policies"`
}

type ParserOutputPolicy struct {
	ParserName         string   `json:"parserName"`
	ParserSource       string   `json:"parserSource"`
	LicenseConstraints string   `json:"licenseConstraints"`
	Shareability       string   `json:"shareability"`
	ProvenanceFields   []string `json:"provenanceFields"`
}

func DefaultParserOutputPolicies() ParserOutputPolicyRegistry {
	return ParserOutputPolicyRegistry{SchemaVersion: "1", Policies: []ParserOutputPolicy{
		{ParserName: "grobid", ParserSource: "external-service", LicenseConstraints: "GROBID TEI output is parser-generated from user-supplied PDFs; original PDF copyright/license still governs sharing.", Shareability: "share parsed TEI/JSON only after document acquisition and redaction gates pass", ProvenanceFields: []string{"parser", "version", "endpoint", "input_checksum", "output_checksum", "command"}},
		{ParserName: "s2orc", ParserSource: "fixture-or-imported-json", LicenseConstraints: "S2ORC-style JSON may carry dataset-specific license and redistribution constraints.", Shareability: "share normalized output only when source JSON license permits", ProvenanceFields: []string{"parser", "version", "source_json", "input_checksum", "output_checksum"}},
		{ParserName: "s2orc-doc2json", ParserSource: "fixture-or-imported-json", LicenseConstraints: "S2ORC-style JSON may carry dataset-specific license and redistribution constraints.", Shareability: "share normalized output only when source JSON license permits", ProvenanceFields: []string{"parser", "version", "source_json", "input_checksum", "output_checksum"}},
		{ParserName: "papermage", ParserSource: "fixture-or-imported-json", LicenseConstraints: "PaperMage JSON license follows the producing tool/model and original document.", Shareability: "share normalized output only after parser/source license review", ProvenanceFields: []string{"parser", "version", "source_json", "input_checksum", "output_checksum"}},
		{ParserName: "cermine", ParserSource: "external-tool", LicenseConstraints: "CERMINE output license follows CERMINE/tool packaging and original document rights.", Shareability: "share only after external-tool license and document rights review", ProvenanceFields: []string{"parser", "version", "command", "input_checksum", "output_checksum"}},
		{ParserName: "science-parse", ParserSource: "external-tool", LicenseConstraints: "Science Parse-style metadata output is historical/fallback parser output; verify tool license and document rights.", Shareability: "share metadata only after stale-parser risk and license review", ProvenanceFields: []string{"parser", "version", "command", "input_checksum", "output_checksum"}},
		{ParserName: "anystyle", ParserSource: "external-tool", LicenseConstraints: "Anystyle reference output follows Anystyle command license and source reference text rights.", Shareability: "share parsed references only after source document/license gate", ProvenanceFields: []string{"parser", "version", "command", "input_checksum", "output_checksum"}},
	}}
}

func (r ParserOutputPolicyRegistry) Policy(parserName string) (ParserOutputPolicy, bool) {
	parserName = strings.TrimSpace(parserName)
	for _, policy := range r.Policies {
		if policy.ParserName == parserName {
			return policy, true
		}
	}
	return ParserOutputPolicy{}, false
}

func (r ParserOutputPolicyRegistry) MustPolicy(parserName string) ParserOutputPolicy {
	if policy, ok := r.Policy(parserName); ok {
		return policy
	}
	return ParserOutputPolicy{ParserName: parserName, ParserSource: "unknown", LicenseConstraints: "unknown parser output license; review required", Shareability: "blocked until parser output policy is defined", ProvenanceFields: []string{"parser", "input_checksum", "output_checksum"}}
}
