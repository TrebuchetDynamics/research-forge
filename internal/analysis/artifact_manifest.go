package analysis

import (
	"encoding/json"
	"os"
	"path/filepath"
)

type AnalysisArtifactManifest struct {
	SchemaVersion   string                    `json:"schemaVersion"`
	RunID           string                    `json:"runId"`
	Plots           []PlotArtifactManifest    `json:"plots"`
	Script          ScriptArtifactManifest    `json:"script"`
	EngineVersions  map[string]string         `json:"engineVersions"`
	Warnings        []string                  `json:"warnings,omitempty"`
	ReportEmbedding []ReportEmbeddingMetadata `json:"reportEmbedding"`
}

type PlotArtifactManifest struct {
	Kind     string            `json:"kind"`
	Path     string            `json:"path"`
	Checksum string            `json:"checksum"`
	Settings map[string]string `json:"settings"`
}

type ScriptArtifactManifest struct {
	Path     string `json:"path"`
	Checksum string `json:"checksum"`
	Engine   string `json:"engine"`
}

type ReportEmbeddingMetadata struct {
	Kind     string `json:"kind"`
	Path     string `json:"path"`
	Markdown string `json:"markdown"`
	AltText  string `json:"altText"`
}

func NewAnalysisArtifactManifest(run AnalysisRun, result AnalysisResult) AnalysisArtifactManifest {
	plots := []PlotArtifactManifest{
		{Kind: "forest", Path: result.ForestPlot.Path, Checksum: result.ForestPlot.Checksum, Settings: defaultPlotSettings("forest")},
		{Kind: "funnel", Path: result.FunnelPlot.Path, Checksum: result.FunnelPlot.Checksum, Settings: defaultPlotSettings("funnel")},
	}
	embeddings := []ReportEmbeddingMetadata{}
	for _, plot := range plots {
		embeddings = append(embeddings, ReportEmbeddingMetadata{Kind: plot.Kind, Path: plot.Path, Markdown: "![" + plot.Kind + " plot](" + plot.Path + ")", AltText: plot.Kind + " plot for analysis run " + run.ID})
	}
	return AnalysisArtifactManifest{SchemaVersion: "1", RunID: run.ID, Plots: plots, Script: ScriptArtifactManifest{Path: scriptPathFromPlot(result.ForestPlot.Path, run.ID), Checksum: result.ScriptChecksum, Engine: "R/metafor"}, EngineVersions: cloneVersions(result.Versions), Warnings: append([]string{}, result.Warnings...), ReportEmbedding: embeddings}
}

func WriteAnalysisArtifactManifest(path string, manifest AnalysisArtifactManifest) error {
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		return err
	}
	data, err := json.MarshalIndent(manifest, "", "  ")
	if err != nil {
		return err
	}
	return os.WriteFile(path, data, 0o644)
}

func defaultPlotSettings(kind string) map[string]string {
	settings := map[string]string{"format": "svg", "renderer": "researchforge-go", "coordinateSystem": "deterministic"}
	if kind == "forest" {
		settings["confidenceInterval"] = "95%"
	} else if kind == "funnel" {
		settings["yAxis"] = "standard-error"
	}
	return settings
}

func scriptPathFromPlot(plotPath, runID string) string {
	if plotPath == "" {
		return runID + "-script.R"
	}
	return filepath.Join(filepath.Dir(plotPath), runID+"-script.R")
}

func cloneVersions(in map[string]string) map[string]string {
	out := map[string]string{}
	for k, v := range in {
		out[k] = v
	}
	return out
}
