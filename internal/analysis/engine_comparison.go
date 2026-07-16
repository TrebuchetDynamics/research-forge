package analysis

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"math"
)

type EngineResult struct {
	Engine        string            `json:"engine"`
	Estimate      float64           `json:"estimate"`
	Variance      float64           `json:"variance"`
	InputHash     string            `json:"inputHash"`
	Versions      map[string]string `json:"versions"`
	Warnings      []string          `json:"warnings,omitempty"`
	ModelSettings map[string]string `json:"modelSettings"`
}

type EngineComparisonReport struct {
	SchemaVersion      string                 `json:"schemaVersion"`
	RunID              string                 `json:"runId"`
	PrimaryEngine      string                 `json:"primaryEngine"`
	SecondaryEngine    string                 `json:"secondaryEngine"`
	InputHash          string                 `json:"inputHash"`
	EnvironmentLocks   []EnvironmentLock      `json:"environmentLocks"`
	ModelSettingParity bool                   `json:"modelSettingParity"`
	ModelSettings      map[string]string      `json:"modelSettings"`
	Warnings           []EngineWarning        `json:"warnings,omitempty"`
	OutputDeltas       EngineOutputDeltas     `json:"outputDeltas"`
	Disagreement       DisagreementResolution `json:"disagreement"`
}

type EnvironmentLock struct {
	Engine   string            `json:"engine"`
	Versions map[string]string `json:"versions"`
}

type EngineWarning struct {
	Engine  string `json:"engine"`
	Message string `json:"message"`
}

type EngineOutputDeltas struct {
	EstimateDelta float64 `json:"estimateDelta"`
	VarianceDelta float64 `json:"varianceDelta"`
}

type DisagreementResolution struct {
	RequiresReview bool    `json:"requiresReview"`
	Tolerance      float64 `json:"tolerance"`
	Reason         string  `json:"reason,omitempty"`
}

func DefaultEngineModelSettings() map[string]string {
	return map[string]string{"model": "fixed-effect", "effectScale": "generic-inverse-variance", "tau2": "not-estimated"}
}

func BuildMetaforFixtureResult(run AnalysisRun, result AnalysisResult) (EngineResult, error) {
	estimate, variance, _, err := pooledEstimate(run.InputRows)
	if err != nil {
		return EngineResult{}, err
	}
	return EngineResult{Engine: "metafor", Estimate: estimate, Variance: variance, InputHash: hashJSON(run), Versions: cloneVersions(result.Versions), Warnings: append([]string{}, result.Warnings...), ModelSettings: DefaultEngineModelSettings()}, nil
}

func BuildPyMAREFixtureResult(run AnalysisRun, estimateDelta float64) (EngineResult, error) {
	estimate, variance, _, err := pooledEstimate(run.InputRows)
	if err != nil {
		return EngineResult{}, err
	}
	return EngineResult{Engine: "pymare-fixture", Estimate: estimate + estimateDelta, Variance: variance, InputHash: hashJSON(run), Versions: map[string]string{"python": "fixture", "pymare": "fixture-adapter"}, ModelSettings: DefaultEngineModelSettings()}, nil
}

func CompareAnalysisEngines(run AnalysisRun, primary, secondary EngineResult, tolerance float64) EngineComparisonReport {
	if tolerance <= 0 {
		tolerance = 1e-6
	}
	inputHash := hashJSON(run)
	deltas := EngineOutputDeltas{EstimateDelta: secondary.Estimate - primary.Estimate, VarianceDelta: secondary.Variance - primary.Variance}
	report := EngineComparisonReport{SchemaVersion: "1", RunID: run.ID, PrimaryEngine: primary.Engine, SecondaryEngine: secondary.Engine, InputHash: inputHash, EnvironmentLocks: []EnvironmentLock{{Engine: primary.Engine, Versions: cloneVersions(primary.Versions)}, {Engine: secondary.Engine, Versions: cloneVersions(secondary.Versions)}}, ModelSettingParity: sameStringMap(primary.ModelSettings, secondary.ModelSettings), ModelSettings: cloneVersions(primary.ModelSettings), OutputDeltas: deltas, Disagreement: DisagreementResolution{Tolerance: tolerance}}
	for _, warning := range primary.Warnings {
		report.Warnings = append(report.Warnings, EngineWarning{Engine: primary.Engine, Message: warning})
	}
	for _, warning := range secondary.Warnings {
		report.Warnings = append(report.Warnings, EngineWarning{Engine: secondary.Engine, Message: warning})
	}
	if !finiteEngineResult(primary) || !finiteEngineResult(secondary) {
		report.Disagreement.RequiresReview = true
		report.Disagreement.Reason = "engine output contains an invalid estimate or variance"
	} else if primary.InputHash != "" && secondary.InputHash != "" && primary.InputHash != secondary.InputHash {
		report.Disagreement.RequiresReview = true
		report.Disagreement.Reason = "engine input hashes differ"
	} else if !report.ModelSettingParity {
		report.Disagreement.RequiresReview = true
		report.Disagreement.Reason = "engine model settings differ"
	} else if math.Abs(deltas.EstimateDelta) > tolerance || math.Abs(deltas.VarianceDelta) > tolerance {
		report.Disagreement.RequiresReview = true
		report.Disagreement.Reason = fmt.Sprintf("engine outputs differ beyond tolerance %.6g", tolerance)
	}
	return report
}

func finiteEngineResult(result EngineResult) bool {
	return !math.IsNaN(result.Estimate) && !math.IsInf(result.Estimate, 0) && !math.IsNaN(result.Variance) && !math.IsInf(result.Variance, 0) && result.Variance >= 0
}

func hashJSON(value any) string {
	data, _ := json.Marshal(value)
	sum := sha256.Sum256(data)
	return hex.EncodeToString(sum[:])
}

func sameStringMap(left, right map[string]string) bool {
	if len(left) != len(right) {
		return false
	}
	for key, value := range left {
		if right[key] != value {
			return false
		}
	}
	return true
}
