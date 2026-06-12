package analysis

import (
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"os"
	"path/filepath"
	"strconv"
	"strings"

	"github.com/TrebuchetDynamics/research-forge/internal/evidence"
)

type AnalysisRun struct {
	SchemaVersion string
	ID            string
	InputRows     []InputRow
}
type InputRow struct {
	PaperID    string
	EffectSize float64
	Variance   float64
}
type EffectSizeCalculator interface {
	Calculate(map[string]string) (float64, float64, error)
}
type StandardizedMeanDifference struct{}

func (StandardizedMeanDifference) Calculate(values map[string]string) (float64, float64, error) {
	mt, _ := strconv.ParseFloat(values["mean_treatment"], 64)
	mc, _ := strconv.ParseFloat(values["mean_control"], 64)
	sd, _ := strconv.ParseFloat(values["sd_pooled"], 64)
	nt, _ := strconv.ParseFloat(values["n_treatment"], 64)
	nc, _ := strconv.ParseFloat(values["n_control"], 64)
	if sd == 0 || nt == 0 || nc == 0 {
		return 0, 0, fmt.Errorf("effect size inputs are incomplete")
	}
	return (mt - mc) / sd, 1/nt + 1/nc, nil
}

func Prepare(id string, items []evidence.EvidenceItem) (AnalysisRun, error) {
	calc := StandardizedMeanDifference{}
	run := AnalysisRun{SchemaVersion: "1", ID: id}
	for _, item := range items {
		if item.Status != evidence.StatusAccepted {
			continue
		}
		es, v, err := calc.Calculate(item.Values)
		if err != nil {
			return AnalysisRun{}, err
		}
		run.InputRows = append(run.InputRows, InputRow{PaperID: item.PaperID, EffectSize: es, Variance: v})
	}
	return run, nil
}
func GenerateMetaforScript(run AnalysisRun) string {
	var b strings.Builder
	b.WriteString("library(metafor)\n")
	b.WriteString("data <- data.frame(yi=c(")
	for i, row := range run.InputRows {
		if i > 0 {
			b.WriteString(",")
		}
		fmt.Fprintf(&b, "%g", row.EffectSize)
	}
	b.WriteString("), vi=c(")
	for i, row := range run.InputRows {
		if i > 0 {
			b.WriteString(",")
		}
		fmt.Fprintf(&b, "%g", row.Variance)
	}
	b.WriteString("))\nmodel <- rma(yi = yi, vi = vi, data=data)\nprint(model)\n")
	return b.String()
}

type HeterogeneityMetrics struct {
	I2   float64
	Tau2 float64
	Q    float64
}

func ParseHeterogeneity(output string) (HeterogeneityMetrics, error) {
	m := HeterogeneityMetrics{}
	for _, line := range strings.Split(output, "\n") {
		parts := strings.SplitN(line, "=", 2)
		if len(parts) != 2 {
			continue
		}
		value, _ := strconv.ParseFloat(strings.TrimSpace(parts[1]), 64)
		switch strings.TrimSpace(parts[0]) {
		case "I2":
			m.I2 = value
		case "tau2":
			m.Tau2 = value
		case "Q":
			m.Q = value
		}
	}
	return m, nil
}

type Runner interface {
	Run(script string) (RunOutput, error)
	ToolVersions() map[string]string
}
type RunOutput struct {
	Stdout string
	Stderr string
}
type FakeRunner struct {
	Stdout   string
	Stderr   string
	Versions map[string]string
}

func (f FakeRunner) Run(script string) (RunOutput, error) {
	return RunOutput{Stdout: f.Stdout, Stderr: f.Stderr}, nil
}
func (f FakeRunner) ToolVersions() map[string]string { return f.Versions }

type Artifact struct {
	Path     string
	Checksum string
}
type Scaffold struct{ Available bool }
type AnalysisResult struct {
	Versions            map[string]string
	ScriptChecksum      string
	OutputChecksum      string
	Warnings            []string
	ForestPlot          Artifact
	FunnelPlot          Artifact
	Metrics             HeterogeneityMetrics
	MetaRegression      Scaffold
	SubgroupAnalysis    Scaffold
	PublicationBias     Scaffold
	SensitivityAnalysis Scaffold
}

func RunMetafor(dir string, run AnalysisRun, runner Runner) (AnalysisResult, error) {
	if err := os.MkdirAll(dir, 0o755); err != nil {
		return AnalysisResult{}, err
	}
	script := GenerateMetaforScript(run)
	scriptPath := filepath.Join(dir, run.ID+"-script.R")
	if err := os.WriteFile(scriptPath, []byte(script), 0o644); err != nil {
		return AnalysisResult{}, err
	}
	out, err := runner.Run(script)
	if err != nil {
		return AnalysisResult{}, err
	}
	outputPath := filepath.Join(dir, run.ID+"-output.txt")
	if err := os.WriteFile(outputPath, []byte(out.Stdout), 0o644); err != nil {
		return AnalysisResult{}, err
	}
	forest := Artifact{Path: filepath.Join(dir, run.ID+"-forest.png")}
	funnel := Artifact{Path: filepath.Join(dir, run.ID+"-funnel.png")}
	warnings := []string{}
	if strings.TrimSpace(out.Stderr) != "" {
		warnings = append(warnings, out.Stderr)
	}
	metrics, _ := ParseHeterogeneity(out.Stdout)
	return AnalysisResult{Versions: runner.ToolVersions(), ScriptChecksum: checksum([]byte(script)), OutputChecksum: checksum([]byte(out.Stdout)), Warnings: warnings, ForestPlot: forest, FunnelPlot: funnel, Metrics: metrics}, nil
}
func checksum(data []byte) string { sum := sha256.Sum256(data); return hex.EncodeToString(sum[:]) }
