// Package parsing is the seam for structured document parsing.
//
// The implementation is split by responsibility into subpackages:
//
//	model       parsed document model, offsets, manifests and enrichment
//	text        string helpers shared by two or more parsers
//	adapters    one adapter per external parser or parser output format
//	references  reference extraction, normalization and adjudication
//	quality     parser comparison, arbitration and quality reporting
//
// The declarations below re-export the subpackages so callers keep the current
// package path and symbol names.
package parsing

import (
	"github.com/TrebuchetDynamics/research-forge/internal/parsing/adapters"
	"github.com/TrebuchetDynamics/research-forge/internal/parsing/model"
	"github.com/TrebuchetDynamics/research-forge/internal/parsing/quality"
	"github.com/TrebuchetDynamics/research-forge/internal/parsing/references"
	"github.com/TrebuchetDynamics/research-forge/internal/parsing/text"
)

type AdjudicatedReferenceItem = references.AdjudicatedReferenceItem

var AmbiguousReferences = references.AmbiguousReferences

type AnyStyleReferenceParser = adapters.AnyStyleReferenceParser

var AppendReferenceAdjudication = references.AppendReferenceAdjudication
var AppendReferenceAdjudicationThen = references.AppendReferenceAdjudicationThen
var ApplyReferenceAdjudications = references.ApplyReferenceAdjudications
var ArbitrateParserOutputs = quality.ArbitrateParserOutputs

type ArbitrationDecisionInput = quality.ArbitrationDecisionInput

var BuildParserMaintenanceRiskReport = quality.BuildParserMaintenanceRiskReport
var BuildParserQualityReport = quality.BuildParserQualityReport

type CERMINEXMLParser = adapters.CERMINEXMLParser

var CanonicalParserName = quality.CanonicalParserName

type CitationSpan = model.CitationSpan
type CommandRunner = adapters.CommandRunner

var CompactParsingSpace = text.CompactParsingSpace
var CompactSpace = text.CompactSpace
var CompareParsedDocuments = quality.CompareParsedDocuments

type ComparisonReport = quality.ComparisonReport

var DefaultParserBenchmarkFixtures = quality.DefaultParserBenchmarkFixtures
var DefaultParserFallbackPolicy = adapters.DefaultParserFallbackPolicy
var DefaultParserOutputPolicies = model.DefaultParserOutputPolicies
var EnrichParsedDocumentModel = model.EnrichParsedDocumentModel

type ExecCommandRunner = adapters.ExecCommandRunner

var ExportReferenceAmbiguityQueue = references.ExportReferenceAmbiguityQueue

type FieldComparison = quality.FieldComparison
type FieldScore = quality.FieldScore
type GROBIDClient = adapters.GROBIDClient
type GROBIDClientOptions = adapters.GROBIDClientOptions
type LayeredAnnotation = model.LayeredAnnotation

var LoadReferenceAdjudications = references.LoadReferenceAdjudications

type MultiSourceReferenceNormalizationReport = references.MultiSourceReferenceNormalizationReport

var NewGROBIDClient = adapters.NewGROBIDClient
var NewParserRunManifest = model.NewParserRunManifest
var NewParserRunManifestWithOutput = model.NewParserRunManifestWithOutput
var NewReferenceAdjudication = references.NewReferenceAdjudication
var NormalizeParsedReferences = references.NormalizeParsedReferences
var NormalizeParsedReferencesAcrossConnectors = references.NormalizeParsedReferencesAcrossConnectors
var NormalizeReferenceDOI = text.NormalizeReferenceDOI

type PaperMageJSONParser = adapters.PaperMageJSONParser
type ParseOptions = model.ParseOptions
type ParsedAuthor = model.ParsedAuthor
type ParsedDocument = model.ParsedDocument
type ParserAdapter = model.ParserAdapter
type ParserArbitrationDecision = quality.ParserArbitrationDecision
type ParserArbitrationReport = quality.ParserArbitrationReport
type ParserBenchmarkFixture = quality.ParserBenchmarkFixture
type ParserCandidate = quality.ParserCandidate
type ParserConflictReviewItem = quality.ParserConflictReviewItem
type ParserFallbackDecision = adapters.ParserFallbackDecision
type ParserFallbackPolicy = adapters.ParserFallbackPolicy
type ParserFieldValue = quality.ParserFieldValue
type ParserMaintenanceRisk = quality.ParserMaintenanceRisk
type ParserMaintenanceRiskReport = quality.ParserMaintenanceRiskReport
type ParserOutputPolicy = model.ParserOutputPolicy
type ParserOutputPolicyRegistry = model.ParserOutputPolicyRegistry
type ParserQualityConflict = quality.ParserQualityConflict
type ParserQualityReport = quality.ParserQualityReport
type ParserQualityRun = quality.ParserQualityRun
type ParserReconciliationAlt = model.ParserReconciliationAlt
type ParserReconciliationOutput = model.ParserReconciliationOutput
type ParserRunManifest = model.ParserRunManifest
type ParserRunSummary = quality.ParserRunSummary
type ParserWarnings = quality.ParserWarnings
type Passage = model.Passage
type Reference = model.Reference
type ReferenceAdjudication = references.ReferenceAdjudication
type ReferenceAdjudicationReport = references.ReferenceAdjudicationReport

const ReferenceAdjudicationSchemaVersion = references.ReferenceAdjudicationSchemaVersion

type ReferenceAmbiguityExport = references.ReferenceAmbiguityExport
type ReferenceAmbiguityQueueItem = references.ReferenceAmbiguityQueueItem
type ReferenceCorrection = references.ReferenceCorrection
type ReferenceMatch = references.ReferenceMatch
type ReferenceNormalizationReport = references.ReferenceNormalizationReport
type ReferenceReviewItem = references.ReferenceReviewItem
type ReferenceReviewReport = references.ReferenceReviewReport
type S2ORCJSONParser = adapters.S2ORCJSONParser
type Section = model.Section
type TeXParser = adapters.TeXParser
type TextOffset = model.TextOffset

var ValidReferenceAdjudicationDecision = references.ValidReferenceAdjudicationDecision
