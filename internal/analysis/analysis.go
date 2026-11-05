// Package analysis is the seam for the meta-analysis pipeline.
//
// The implementation is split by responsibility into subpackages:
//
//	core         analysis run model, row validation, preparation and metafor execution
//	estimators   one statistical estimate or diagnostic per file
//	comparison   comparison of the estimators on the same run
//	artifacts    analysis artifact manifest and provenance versions
//	workbench    method comparison workbench for research-method choices
//
// The declarations below re-export the subpackages so callers keep the current
// package path and symbol names.
package analysis

import (
	"github.com/TrebuchetDynamics/research-forge/internal/analysis/artifacts"
	"github.com/TrebuchetDynamics/research-forge/internal/analysis/comparison"
	"github.com/TrebuchetDynamics/research-forge/internal/analysis/core"
	"github.com/TrebuchetDynamics/research-forge/internal/analysis/estimators"
	"github.com/TrebuchetDynamics/research-forge/internal/analysis/workbench"
)

type AnalysisArtifactManifest = artifacts.AnalysisArtifactManifest
type AnalysisResult = core.AnalysisResult
type AnalysisRun = core.AnalysisRun
type Artifact = core.Artifact
type BayesianEngine = estimators.BayesianEngine
type BayesianEngineOptions = estimators.BayesianEngineOptions

var BayesianNormalApproximation = estimators.BayesianNormalApproximation

type BayesianReport = estimators.BayesianReport

var BeggRankCorrelation = estimators.BeggRankCorrelation
var BenchmarkingReadiness = core.BenchmarkingReadiness
var BuildMetaforFixtureResult = comparison.BuildMetaforFixtureResult
var BuildPyMAREFixtureResult = comparison.BuildPyMAREFixtureResult
var CloneVersions = artifacts.CloneVersions
var CompareAnalysisEngines = comparison.CompareAnalysisEngines
var DefaultEngineModelSettings = comparison.DefaultEngineModelSettings
var DefaultMethodComparisonWorkbench = workbench.DefaultMethodComparisonWorkbench

type DisagreementResolution = comparison.DisagreementResolution
type EffectSizeCalculator = core.EffectSizeCalculator

var EggerRegression = estimators.EggerRegression

type EngineComparisonReport = comparison.EngineComparisonReport
type EngineOutputDeltas = comparison.EngineOutputDeltas
type EngineResult = comparison.EngineResult
type EngineWarning = comparison.EngineWarning
type EnvironmentLock = comparison.EnvironmentLock

var ExcludeByViSource = core.ExcludeByViSource

type FakeRunner = core.FakeRunner
type FisherZCorrelation = core.FisherZCorrelation

var ForestPlotSVG = core.ForestPlotSVG
var FunnelPlotSVG = core.FunnelPlotSVG
var GenerateMetaforScript = core.GenerateMetaforScript

type GridBayesianEngine = estimators.GridBayesianEngine
type HeterogeneityMetrics = core.HeterogeneityMetrics

var InfluenceDiagnostics = estimators.InfluenceDiagnostics

type InfluenceReport = estimators.InfluenceReport
type InfluenceRow = estimators.InfluenceRow
type InputRow = core.InputRow

var LeaveOneOut = estimators.LeaveOneOut

type LeaveOneOutRow = estimators.LeaveOneOutRow
type LogOddsRatio = core.LogOddsRatio
type MeanDifference = core.MeanDifference

var MetaRegression = estimators.MetaRegression

type MetaRegressionReport = estimators.MetaRegressionReport

var MetaRegressionValuesFromEvidence = estimators.MetaRegressionValuesFromEvidence

type MethodCategory = workbench.MethodCategory
type MethodComparisonReport = workbench.MethodComparisonReport
type MethodComparisonWorkbench = workbench.MethodComparisonWorkbench
type MethodLockSelection = workbench.MethodLockSelection
type MethodOption = workbench.MethodOption
type MethodSelectionInput = workbench.MethodSelectionInput
type ModeratorField = estimators.ModeratorField
type ModeratorPreview = estimators.ModeratorPreview

var ModeratorPreviewFromEvidence = estimators.ModeratorPreviewFromEvidence
var NewAnalysisArtifactManifest = artifacts.NewAnalysisArtifactManifest
var ParseHeterogeneity = core.ParseHeterogeneity

type PlotArtifactManifest = artifacts.PlotArtifactManifest

var PooledEstimate = estimators.PooledEstimate
var Prepare = core.Prepare
var PrepareRawContinuous = core.PrepareRawContinuous
var PrepareWithCalculator = core.PrepareWithCalculator

type PublicationBiasReport = estimators.PublicationBiasReport
type RawContinuousOutcome = core.RawContinuousOutcome
type RawContinuousResult = core.RawContinuousResult
type ReadinessIssue = core.ReadinessIssue
type ReadinessReport = core.ReadinessReport
type ReportEmbeddingMetadata = artifacts.ReportEmbeddingMetadata
type RiskDifference = core.RiskDifference
type RiskRatio = core.RiskRatio
type RscriptRunner = core.RscriptRunner

var RunBayesianEngine = estimators.RunBayesianEngine
var RunMetafor = core.RunMetafor

type RunOutput = core.RunOutput
type Runner = core.Runner
type Scaffold = core.Scaffold
type ScriptArtifactManifest = artifacts.ScriptArtifactManifest
type SensitivityReport = estimators.SensitivityReport
type StandardizedMeanDifference = core.StandardizedMeanDifference

var SubgroupAnalysis = estimators.SubgroupAnalysis

type SubgroupEstimate = estimators.SubgroupEstimate
type SubgroupReport = estimators.SubgroupReport

var SubgroupValuesFromEvidence = estimators.SubgroupValuesFromEvidence
var ValidateAnalysisRows = core.ValidateAnalysisRows
var WriteAnalysisArtifactManifest = artifacts.WriteAnalysisArtifactManifest
