// Package sources is the seam for scholarly metadata connectors.
//
// The implementation is split by responsibility into subpackages:
//
//	core        normalized record model, search run and graph budget types
//	transport   HTTP client with retry, redirect and response size limits
//	fields      normalization helpers shared by two or more connectors
//	adapters    one connector per scholarly source
//	resolve     identifier-first paper resolution across adapters
//	openaccess  legal open-access candidate and resolve-plan policy
//
// The declarations below re-export the subpackages so callers keep the current
// package path and symbol names.
package sources

import (
	"github.com/TrebuchetDynamics/research-forge/internal/sources/adapters"
	"github.com/TrebuchetDynamics/research-forge/internal/sources/core"
	"github.com/TrebuchetDynamics/research-forge/internal/sources/fields"
	"github.com/TrebuchetDynamics/research-forge/internal/sources/openaccess"
	"github.com/TrebuchetDynamics/research-forge/internal/sources/resolve"
	"github.com/TrebuchetDynamics/research-forge/internal/sources/transport"
)

type ArXivConnector = adapters.ArXivConnector
type BASEConnector = adapters.BASEConnector
type BioRxivConnector = adapters.BioRxivConnector
type BioStudiesConnector = adapters.BioStudiesConnector

var BuildOpenAccessResolvePlan = openaccess.BuildOpenAccessResolvePlan
var BuildOpenAlexDisambiguationQueue = adapters.BuildOpenAlexDisambiguationQueue

type COREConnector = openaccess.COREConnector
type ChemRxivConnector = adapters.ChemRxivConnector
type CiNiiConnector = adapters.CiNiiConnector
type CitationEdge = core.CitationEdge
type CitationGraphExpansion = core.CitationGraphExpansion
type ClinicalTrialsConnector = adapters.ClinicalTrialsConnector

var CompactSpace = fields.CompactSpace
var CompareOpenAccessCandidates = openaccess.CompareOpenAccessCandidates

type CrossrefConnector = adapters.CrossrefConnector
type DBLPConnector = adapters.DBLPConnector
type DOABConnector = adapters.DOABConnector
type DOAJConnector = openaccess.DOAJConnector

var DOIURL = fields.DOIURL

type DataCiteConnector = adapters.DataCiteConnector
type DataverseConnector = adapters.DataverseConnector
type DimensionsConnector = adapters.DimensionsConnector
type DryadConnector = adapters.DryadConnector
type ELifeConnector = adapters.ELifeConnector
type ERICConnector = adapters.ERICConnector

var EstimateGraphExpansionBudget = core.EstimateGraphExpansionBudget

type EuropePMCConnector = adapters.EuropePMCConnector
type FigshareConnector = adapters.FigshareConnector

var FirstNonEmpty = fields.FirstNonEmpty

type GBIFConnector = adapters.GBIFConnector
type GraphExpansionBudget = core.GraphExpansionBudget
type GraphExpansionBudgetEstimate = core.GraphExpansionBudgetEstimate
type HALConnector = adapters.HALConnector

var HTMLTagRe = fields.HTMLTagRe

type HTTPClient = transport.HTTPClient
type HTTPClientOptions = transport.HTTPClientOptions
type HuggingFaceConnector = adapters.HuggingFaceConnector
type Identifiers = core.Identifiers
type InspireHEPConnector = adapters.InspireHEPConnector

var IsHTTPStatus = transport.IsHTTPStatus
var JSONScalarString = fields.JSONScalarString
var LegalOpenAccessResolveSources = openaccess.LegalOpenAccessResolveSources

type LensConnector = adapters.LensConnector
type NASAADSConnector = adapters.NASAADSConnector
type NASACMRConnector = adapters.NASACMRConnector
type NBERConnector = adapters.NBERConnector
type NTRSConnector = adapters.NTRSConnector

var NewArXivConnector = adapters.NewArXivConnector
var NewBASEConnector = adapters.NewBASEConnector
var NewBioRxivConnector = adapters.NewBioRxivConnector
var NewBioStudiesConnector = adapters.NewBioStudiesConnector
var NewCOREConnector = openaccess.NewCOREConnector
var NewChemRxivConnector = adapters.NewChemRxivConnector
var NewCiNiiConnector = adapters.NewCiNiiConnector
var NewClinicalTrialsConnector = adapters.NewClinicalTrialsConnector
var NewCrossrefConnector = adapters.NewCrossrefConnector
var NewDBLPConnector = adapters.NewDBLPConnector
var NewDOABConnector = adapters.NewDOABConnector
var NewDOAJConnector = openaccess.NewDOAJConnector
var NewDataCiteConnector = adapters.NewDataCiteConnector
var NewDataverseConnector = adapters.NewDataverseConnector
var NewDimensionsConnector = adapters.NewDimensionsConnector
var NewDryadConnector = adapters.NewDryadConnector
var NewELifeConnector = adapters.NewELifeConnector
var NewERICConnector = adapters.NewERICConnector
var NewEuropePMCConnector = adapters.NewEuropePMCConnector
var NewFigshareConnector = adapters.NewFigshareConnector
var NewGBIFConnector = adapters.NewGBIFConnector
var NewHALConnector = adapters.NewHALConnector
var NewHTTPClient = transport.NewHTTPClient
var NewHuggingFaceConnector = adapters.NewHuggingFaceConnector
var NewInspireHEPConnector = adapters.NewInspireHEPConnector
var NewLensConnector = adapters.NewLensConnector
var NewNASAADSConnector = adapters.NewNASAADSConnector
var NewNASAADSHTTPClient = adapters.NewNASAADSHTTPClient
var NewNASACMRConnector = adapters.NewNASACMRConnector
var NewNBERConnector = adapters.NewNBERConnector
var NewNTRSConnector = adapters.NewNTRSConnector
var NewOAPenConnector = adapters.NewOAPenConnector
var NewOSFConnector = adapters.NewOSFConnector
var NewOSTIConnector = adapters.NewOSTIConnector
var NewOpenAIREConnector = adapters.NewOpenAIREConnector
var NewOpenAlexConnector = adapters.NewOpenAlexConnector
var NewOpenCitationsConnector = adapters.NewOpenCitationsConnector
var NewOpenLibraryConnector = adapters.NewOpenLibraryConnector
var NewPLOSConnector = adapters.NewPLOSConnector
var NewPMCConnector = adapters.NewPMCConnector
var NewPubChemConnector = adapters.NewPubChemConnector
var NewPubMedConnector = adapters.NewPubMedConnector
var NewPubMedConnectorWithOptions = adapters.NewPubMedConnectorWithOptions
var NewResearchSquareConnector = adapters.NewResearchSquareConnector
var NewResponseCache = core.NewResponseCache
var NewSemanticScholarConnector = adapters.NewSemanticScholarConnector
var NewSemanticScholarGraphRun = adapters.NewSemanticScholarGraphRun
var NewUnpaywallConnector = adapters.NewUnpaywallConnector
var NewZbMATHConnector = adapters.NewZbMATHConnector
var NewZenodoConnector = adapters.NewZenodoConnector
var NonEmptyStrings = fields.NonEmptyStrings
var NormalizeArXivID = fields.NormalizeArXivID
var NormalizeDOI = fields.NormalizeDOI
var NormalizeGraphExpansionBudget = core.NormalizeGraphExpansionBudget

type OAPenConnector = adapters.OAPenConnector
type OAResult = resolve.OAResult
type OSFConnector = adapters.OSFConnector
type OSTIConnector = adapters.OSTIConnector
type OpenAIREConnector = adapters.OpenAIREConnector
type OpenAccessCandidate = openaccess.OpenAccessCandidate
type OpenAccessCandidateComparison = openaccess.OpenAccessCandidateComparison
type OpenAccessRecord = adapters.OpenAccessRecord
type OpenAccessResolvePlan = openaccess.OpenAccessResolvePlan
type OpenAccessResolveSource = openaccess.OpenAccessResolveSource
type OpenAlexConnector = adapters.OpenAlexConnector
type OpenAlexDisambiguationItem = adapters.OpenAlexDisambiguationItem
type OpenAlexDisambiguationQueue = adapters.OpenAlexDisambiguationQueue
type OpenAlexEntity = adapters.OpenAlexEntity

var OpenAlexFilterPreset = adapters.OpenAlexFilterPreset

type OpenAlexGraphQuery = adapters.OpenAlexGraphQuery
type OpenCitationsConnector = adapters.OpenCitationsConnector
type OpenLibraryConnector = adapters.OpenLibraryConnector
type PLOSConnector = adapters.PLOSConnector
type PMCConnector = adapters.PMCConnector

var PaperRecords = core.PaperRecords

type PaperResolution = resolve.PaperResolution
type PubChemConnector = adapters.PubChemConnector
type PubMedConnector = adapters.PubMedConnector
type PubMedOptions = adapters.PubMedOptions

var RedactNASAADSToken = adapters.RedactNASAADSToken

type RequestProvenance = core.RequestProvenance
type ResearchSquareConnector = adapters.ResearchSquareConnector
type ResolutionSource = resolve.ResolutionSource

var ResolvePaper = resolve.ResolvePaper

type ResponseCache = core.ResponseCache
type ResponseProvenance = core.ResponseProvenance

var RunSearch = core.RunSearch

type SearchRun = core.SearchRun
type SemanticScholarConnector = adapters.SemanticScholarConnector

const SemanticScholarDirectionBoth = core.SemanticScholarDirectionBoth
const SemanticScholarDirectionCitations = core.SemanticScholarDirectionCitations
const SemanticScholarDirectionReferences = core.SemanticScholarDirectionReferences

type SemanticScholarGraphDirection = core.SemanticScholarGraphDirection
type SemanticScholarGraphQuery = core.SemanticScholarGraphQuery
type SemanticScholarGraphRun = adapters.SemanticScholarGraphRun
type SemanticScholarGraphRunOptions = adapters.SemanticScholarGraphRunOptions
type SourceConnector = core.SourceConnector
type SourceQuery = core.SourceQuery
type SourceRecord = core.SourceRecord
type SourceResponse = core.SourceResponse

const StopBudgetExhausted = core.StopBudgetExhausted
const StopCompleted = core.StopCompleted
const StopCoverageSaturated = core.StopCoverageSaturated
const StopErrorThreshold = core.StopErrorThreshold
const StopInterrupted = core.StopInterrupted

type StopReason = core.StopReason

var StripHTMLTags = fields.StripHTMLTags

type UnpaywallConnector = adapters.UnpaywallConnector
type UnsupportedFullTextSource = openaccess.UnsupportedFullTextSource

var UnsupportedFullTextSources = openaccess.UnsupportedFullTextSources

type ZbMATHConnector = adapters.ZbMATHConnector
type ZenodoConnector = adapters.ZenodoConnector
