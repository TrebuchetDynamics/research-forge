package parsing

// ComparisonReport summarizes differences between parser outputs for the same paper.
type ComparisonReport struct {
	PaperID        string             `json:"paperId"`
	Documents      []ParserRunSummary `json:"documents"`
	Candidates     []ParserCandidate  `json:"candidates"`
	SectionDelta   int                `json:"sectionDelta"`
	PassageDelta   int                `json:"passageDelta"`
	ReferenceDelta int                `json:"referenceDelta"`
	WarningCount   int                `json:"warningCount"`
	TitleMismatch  bool               `json:"titleMismatch"`
	RecommendedUse string             `json:"recommendedUse"`
}

// ParserRunSummary records comparable parser output counts.
type ParserRunSummary struct {
	ParserName    string `json:"parserName"`
	ParserVersion string `json:"parserVersion"`
	Title         string `json:"title"`
	Sections      int    `json:"sections"`
	Passages      int    `json:"passages"`
	References    int    `json:"references"`
	Warnings      int    `json:"warnings"`
}

// ParserCandidate scores parser output coverage and fallback risk for reviewer triage.
type ParserCandidate struct {
	ParserName          string `json:"parserName"`
	CoverageScore       int    `json:"coverageScore"`
	MaintenanceRisk     string `json:"maintenanceRisk"`
	DependencyFootprint string `json:"dependencyFootprint"`
	LicensePolicy       string `json:"licensePolicy"`
}

// CompareParsedDocuments creates a deterministic parser comparison/fallback report.
func CompareParsedDocuments(docs []ParsedDocument) ComparisonReport {
	report := ComparisonReport{RecommendedUse: "review-required"}
	if len(docs) == 0 {
		return report
	}
	report.PaperID = docs[0].PaperID
	minSections, maxSections := -1, 0
	minPassages, maxPassages := -1, 0
	minReferences, maxReferences := -1, 0
	firstTitle := docs[0].Title
	bestIndex := 0
	bestScore := -1
	for i, doc := range docs {
		sections := len(doc.Sections)
		passages := countPassages(doc)
		references := len(doc.References)
		warnings := len(doc.Warnings)
		report.WarningCount += warnings
		if doc.Title != firstTitle {
			report.TitleMismatch = true
		}
		report.Documents = append(report.Documents, ParserRunSummary{ParserName: doc.ParserName, ParserVersion: doc.ParserVersion, Title: doc.Title, Sections: sections, Passages: passages, References: references, Warnings: warnings})
		report.Candidates = append(report.Candidates, parserCandidate(doc.ParserName, sections+passages+references-warnings))
		minSections, maxSections = minMax(minSections, maxSections, sections)
		minPassages, maxPassages = minMax(minPassages, maxPassages, passages)
		minReferences, maxReferences = minMax(minReferences, maxReferences, references)
		score := sections + passages + references - warnings
		if score > bestScore {
			bestScore = score
			bestIndex = i
		}
	}
	report.SectionDelta = maxSections - minSections
	report.PassageDelta = maxPassages - minPassages
	report.ReferenceDelta = maxReferences - minReferences
	if len(report.Documents) == 1 || (!report.TitleMismatch && report.WarningCount == 0 && report.SectionDelta == 0 && report.PassageDelta == 0 && report.ReferenceDelta == 0) {
		report.RecommendedUse = report.Documents[bestIndex].ParserName
	}
	return report
}

func parserCandidate(parserName string, coverageScore int) ParserCandidate {
	candidate := ParserCandidate{ParserName: parserName, CoverageScore: coverageScore, MaintenanceRisk: "unknown", DependencyFootprint: "external", LicensePolicy: "review-required"}
	switch parserName {
	case "grobid":
		candidate.MaintenanceRisk = "active-service"
		candidate.DependencyFootprint = "java-service"
		candidate.LicensePolicy = "adapter-only"
	case "s2orc-doc2json", "s2orc":
		candidate.MaintenanceRisk = "external-derived-json"
		candidate.DependencyFootprint = "offline-json"
		candidate.LicensePolicy = "adapter-only"
	case "anystyle":
		candidate.MaintenanceRisk = "optional-external"
		candidate.DependencyFootprint = "ruby-command"
		candidate.LicensePolicy = "adapter-only"
	case "cermine":
		candidate.MaintenanceRisk = "fallback-candidate"
		candidate.DependencyFootprint = "java-command"
		candidate.LicensePolicy = "adapter-only"
	case "science-parse":
		candidate.MaintenanceRisk = "stale-reference"
		candidate.DependencyFootprint = "historical-service"
		candidate.LicensePolicy = "pattern-reference"
	case "tex":
		candidate.MaintenanceRisk = "builtin"
		candidate.DependencyFootprint = "none"
		candidate.LicensePolicy = "builtin"
	}
	return candidate
}

func countPassages(doc ParsedDocument) int {
	count := 0
	for _, section := range doc.Sections {
		count += len(section.Passages)
	}
	return count
}

func minMax(minValue, maxValue, value int) (int, int) {
	if minValue < 0 || value < minValue {
		minValue = value
	}
	if value > maxValue {
		maxValue = value
	}
	return minValue, maxValue
}
