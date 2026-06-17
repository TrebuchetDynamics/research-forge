package parsing

import (
	"fmt"
	"sort"
	"strings"
)

var qualityReportParsers = []string{"grobid", "s2orc-doc2json", "papermage", "cermine", "science-parse", "anystyle"}

type ParserQualityReport struct {
	SchemaVersion      string                  `json:"schemaVersion"`
	PaperID            string                  `json:"paperId"`
	ParserRuns         []ParserQualityRun      `json:"parserRuns"`
	MissingParsers     []string                `json:"missingParsers"`
	FieldConfidence    map[string]float64      `json:"fieldConfidence"`
	Conflicts          []ParserQualityConflict `json:"conflicts"`
	ReviewerRequired   bool                    `json:"reviewerRequired"`
	AutoAcceptedFields bool                    `json:"autoAcceptedFields"`
	RecommendedAction  string                  `json:"recommendedAction"`
}

type ParserQualityRun struct {
	ParserName       string  `json:"parserName"`
	ParserVersion    string  `json:"parserVersion"`
	TitlePresent     bool    `json:"titlePresent"`
	Sections         int     `json:"sections"`
	Passages         int     `json:"passages"`
	References       int     `json:"references"`
	ReferenceDOIs    int     `json:"referenceDois"`
	Warnings         int     `json:"warnings"`
	ParserConfidence float64 `json:"parserConfidence"`
	QualityScore     float64 `json:"qualityScore"`
}

type ParserQualityConflict struct {
	Field  string   `json:"field"`
	Values []string `json:"values"`
	Status string   `json:"status"`
}

func BuildParserQualityReport(docs []ParsedDocument) ParserQualityReport {
	report := ParserQualityReport{SchemaVersion: "1", FieldConfidence: map[string]float64{}, ReviewerRequired: true, AutoAcceptedFields: false, RecommendedAction: "review-conflicting-fields"}
	if len(docs) == 0 {
		report.MissingParsers = append(report.MissingParsers, qualityReportParsers...)
		return report
	}
	report.PaperID = docs[0].PaperID
	seen := map[string]bool{}
	titles := map[string]bool{}
	sectionCounts := map[string]bool{}
	passageCounts := map[string]bool{}
	referenceCounts := map[string]bool{}
	confSums := map[string]float64{}
	confCounts := map[string]int{}
	for _, doc := range docs {
		name := canonicalParserName(doc.ParserName)
		seen[name] = true
		run := ParserQualityRun{ParserName: name, ParserVersion: doc.ParserVersion, TitlePresent: strings.TrimSpace(doc.Title) != "", Sections: len(doc.Sections), Passages: countPassages(doc), References: len(doc.References), ReferenceDOIs: countReferenceDOIs(doc), Warnings: len(doc.Warnings), ParserConfidence: doc.ParserConfidence}
		run.QualityScore = parserQualityScore(run)
		report.ParserRuns = append(report.ParserRuns, run)
		if run.TitlePresent {
			titles[doc.Title] = true
			confSums["title"] += nonzero(doc.ParserConfidence, 0.7)
			confCounts["title"]++
		}
		sectionCounts[intKey(run.Sections)] = true
		passageCounts[intKey(run.Passages)] = true
		referenceCounts[intKey(run.References)] = true
		if run.Sections > 0 {
			confSums["sections"] += nonzero(doc.ParserConfidence, 0.7)
			confCounts["sections"]++
		}
		if run.Passages > 0 {
			confSums["passages"] += nonzero(doc.ParserConfidence, 0.7)
			confCounts["passages"]++
		}
		if run.References > 0 {
			confSums["references"] += referenceConfidence(doc)
			confCounts["references"]++
		}
	}
	for _, parser := range qualityReportParsers {
		if !seen[parser] {
			report.MissingParsers = append(report.MissingParsers, parser)
		}
	}
	if len(titles) > 1 {
		report.Conflicts = append(report.Conflicts, qualityConflict("title", titles))
	}
	if len(sectionCounts) > 1 {
		report.Conflicts = append(report.Conflicts, qualityConflict("sections", sectionCounts))
	}
	if len(passageCounts) > 1 {
		report.Conflicts = append(report.Conflicts, qualityConflict("passages", passageCounts))
	}
	if len(referenceCounts) > 1 {
		report.Conflicts = append(report.Conflicts, qualityConflict("references", referenceCounts))
	}
	for field, sum := range confSums {
		report.FieldConfidence[field] = sum / float64(confCounts[field])
	}
	sort.Slice(report.ParserRuns, func(i, j int) bool { return report.ParserRuns[i].ParserName < report.ParserRuns[j].ParserName })
	sort.Strings(report.MissingParsers)
	sort.Slice(report.Conflicts, func(i, j int) bool { return report.Conflicts[i].Field < report.Conflicts[j].Field })
	return report
}

func (r ParserQualityReport) HasParser(parser string) bool {
	parser = canonicalParserName(parser)
	for _, run := range r.ParserRuns {
		if run.ParserName == parser {
			return true
		}
	}
	return false
}
func canonicalParserName(name string) string {
	name = strings.TrimSpace(strings.ToLower(name))
	if name == "s2orc" {
		return "s2orc-doc2json"
	}
	return name
}
func countReferenceDOIs(doc ParsedDocument) int {
	n := 0
	for _, ref := range doc.References {
		if strings.TrimSpace(ref.DOI) != "" {
			n++
		}
	}
	return n
}
func parserQualityScore(run ParserQualityRun) float64 {
	return float64(boolInt(run.TitlePresent)+run.Sections+run.Passages+run.ReferenceDOIs) + run.ParserConfidence - float64(run.Warnings)
}
func boolInt(v bool) int {
	if v {
		return 1
	}
	return 0
}
func nonzero(v, fallback float64) float64 {
	if v > 0 {
		return v
	}
	return fallback
}
func intKey(v int) string { return fmt.Sprint(v) }
func qualityConflict(field string, values map[string]bool) ParserQualityConflict {
	out := []string{}
	for v := range values {
		out = append(out, v)
	}
	sort.Strings(out)
	return ParserQualityConflict{Field: field, Values: out, Status: "review-required"}
}
func referenceConfidence(doc ParsedDocument) float64 {
	if len(doc.References) == 0 {
		return nonzero(doc.ParserConfidence, 0.5)
	}
	sum := 0.0
	for _, ref := range doc.References {
		sum += nonzero(ref.Confidence, nonzero(doc.ParserConfidence, 0.5))
	}
	return sum / float64(len(doc.References))
}
