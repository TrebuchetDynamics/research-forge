package analysis

import (
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"math"
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

type LogOddsRatio struct{}

type RiskRatio struct{}

type MeanDifference struct{}

type RiskDifference struct{}

type FisherZCorrelation struct{}

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

func (MeanDifference) Calculate(values map[string]string) (float64, float64, error) {
	mt, _ := strconv.ParseFloat(values["mean_treatment"], 64)
	mc, _ := strconv.ParseFloat(values["mean_control"], 64)
	sdt, _ := strconv.ParseFloat(values["sd_treatment"], 64)
	sdc, _ := strconv.ParseFloat(values["sd_control"], 64)
	nt, _ := strconv.ParseFloat(values["n_treatment"], 64)
	nc, _ := strconv.ParseFloat(values["n_control"], 64)
	if nt == 0 || nc == 0 || sdt == 0 || sdc == 0 {
		return 0, 0, fmt.Errorf("mean difference inputs are incomplete")
	}
	return mt - mc, (sdt*sdt)/nt + (sdc*sdc)/nc, nil
}

func (LogOddsRatio) Calculate(values map[string]string) (float64, float64, error) {
	eventsT, totalT, eventsC, totalC, err := binaryOutcomeInputs(values, "log odds ratio")
	if err != nil {
		return 0, 0, err
	}
	nonEventsT := totalT - eventsT
	nonEventsC := totalC - eventsC
	// Haldane-Anscombe correction keeps zero-cell binary outcomes estimable.
	if eventsT == 0 || nonEventsT == 0 || eventsC == 0 || nonEventsC == 0 {
		eventsT += 0.5
		nonEventsT += 0.5
		eventsC += 0.5
		nonEventsC += 0.5
	}
	return math.Log((eventsT * nonEventsC) / (nonEventsT * eventsC)), 1/eventsT + 1/nonEventsT + 1/eventsC + 1/nonEventsC, nil
}

func (RiskRatio) Calculate(values map[string]string) (float64, float64, error) {
	eventsT, totalT, eventsC, totalC, err := binaryOutcomeInputs(values, "risk ratio")
	if err != nil {
		return 0, 0, err
	}
	if eventsT == 0 || eventsC == 0 {
		eventsT += 0.5
		eventsC += 0.5
		totalT += 0.5
		totalC += 0.5
	}
	return math.Log((eventsT / totalT) / (eventsC / totalC)), 1/eventsT - 1/totalT + 1/eventsC - 1/totalC, nil
}

func (RiskDifference) Calculate(values map[string]string) (float64, float64, error) {
	eventsT, totalT, eventsC, totalC, err := binaryOutcomeInputs(values, "risk difference")
	if err != nil {
		return 0, 0, err
	}
	pt := eventsT / totalT
	pc := eventsC / totalC
	return pt - pc, (pt*(1-pt))/totalT + (pc*(1-pc))/totalC, nil
}

func (FisherZCorrelation) Calculate(values map[string]string) (float64, float64, error) {
	r, _ := strconv.ParseFloat(values["correlation"], 64)
	n, _ := strconv.ParseFloat(values["n"], 64)
	if r <= -1 || r >= 1 || n <= 3 {
		return 0, 0, fmt.Errorf("correlation inputs are incomplete")
	}
	return math.Atanh(r), 1 / (n - 3), nil
}

func binaryOutcomeInputs(values map[string]string, name string) (float64, float64, float64, float64, error) {
	eventsT, _ := strconv.ParseFloat(values["events_treatment"], 64)
	totalT, _ := strconv.ParseFloat(values["n_treatment"], 64)
	eventsC, _ := strconv.ParseFloat(values["events_control"], 64)
	totalC, _ := strconv.ParseFloat(values["n_control"], 64)
	if totalT == 0 || totalC == 0 || eventsT > totalT || eventsC > totalC {
		return 0, 0, 0, 0, fmt.Errorf("%s inputs are incomplete", name)
	}
	return eventsT, totalT, eventsC, totalC, nil
}

func Prepare(id string, items []evidence.EvidenceItem) (AnalysisRun, error) {
	return PrepareWithCalculator(id, items, StandardizedMeanDifference{})
}

func PrepareWithCalculator(id string, items []evidence.EvidenceItem, calc EffectSizeCalculator) (AnalysisRun, error) {
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
	forest, err := writeForestPlotArtifact(dir, run)
	if err != nil {
		return AnalysisResult{}, err
	}
	funnel, err := writeFunnelPlotArtifact(dir, run)
	if err != nil {
		return AnalysisResult{}, err
	}
	warnings := []string{}
	if strings.TrimSpace(out.Stderr) != "" {
		warnings = append(warnings, out.Stderr)
	}
	metrics, _ := ParseHeterogeneity(out.Stdout)
	return AnalysisResult{Versions: runner.ToolVersions(), ScriptChecksum: checksum([]byte(script)), OutputChecksum: checksum([]byte(out.Stdout)), Warnings: warnings, ForestPlot: forest, FunnelPlot: funnel, Metrics: metrics}, nil
}
func writeForestPlotArtifact(dir string, run AnalysisRun) (Artifact, error) {
	path := filepath.Join(dir, run.ID+"-forest.svg")
	data := []byte(forestPlotSVG(run))
	if err := os.WriteFile(path, data, 0o644); err != nil {
		return Artifact{}, err
	}
	return Artifact{Path: path, Checksum: checksum(data)}, nil
}

func writeFunnelPlotArtifact(dir string, run AnalysisRun) (Artifact, error) {
	path := filepath.Join(dir, run.ID+"-funnel.svg")
	data := []byte(funnelPlotSVG(run))
	if err := os.WriteFile(path, data, 0o644); err != nil {
		return Artifact{}, err
	}
	return Artifact{Path: path, Checksum: checksum(data)}, nil
}

func forestPlotSVG(run AnalysisRun) string {
	width := 640
	height := 80 + len(run.InputRows)*34
	if height < 140 {
		height = 140
	}
	min, max := effectRange(run)
	var b strings.Builder
	fmt.Fprintf(&b, `<svg xmlns="http://www.w3.org/2000/svg" role="img" aria-label="Forest plot" viewBox="0 0 %d %d">`, width, height)
	b.WriteString(`<title>Forest plot</title><line x1="320" y1="30" x2="320" y2="` + fmt.Sprint(height-30) + `" stroke="#888"/>`)
	for i, row := range run.InputRows {
		y := 55 + i*34
		x := scaleEffect(row.EffectSize, min, max, 80, 580)
		se := math.Sqrt(row.Variance)
		left := scaleEffect(row.EffectSize-1.96*se, min, max, 80, 580)
		right := scaleEffect(row.EffectSize+1.96*se, min, max, 80, 580)
		label := xmlEscape(row.PaperID)
		fmt.Fprintf(&b, `<line x1="%d" y1="%d" x2="%d" y2="%d" stroke="#2d6cdf"/><circle cx="%d" cy="%d" r="5" fill="#2d6cdf"/><text x="12" y="%d" font-size="12">%s</text>`, left, y, right, y, x, y, y+4, label)
	}
	b.WriteString(`</svg>`)
	return b.String()
}

func funnelPlotSVG(run AnalysisRun) string {
	width := 640
	height := 420
	min, max := effectRange(run)
	maxSE := 0.0
	for _, row := range run.InputRows {
		if row.Variance > 0 && math.Sqrt(row.Variance) > maxSE {
			maxSE = math.Sqrt(row.Variance)
		}
	}
	if maxSE == 0 {
		maxSE = 1
	}
	var b strings.Builder
	fmt.Fprintf(&b, `<svg xmlns="http://www.w3.org/2000/svg" role="img" aria-label="Funnel plot" viewBox="0 0 %d %d">`, width, height)
	b.WriteString(`<title>Funnel plot</title><line x1="60" y1="360" x2="600" y2="360" stroke="#888"/><line x1="60" y1="40" x2="60" y2="360" stroke="#888"/>`)
	for _, row := range run.InputRows {
		x := scaleEffect(row.EffectSize, min, max, 80, 580)
		y := 60 + int((math.Sqrt(row.Variance)/maxSE)*280)
		fmt.Fprintf(&b, `<circle cx="%d" cy="%d" r="5" fill="#d66b2d"><title>%s</title></circle>`, x, y, xmlEscape(row.PaperID))
	}
	b.WriteString(`</svg>`)
	return b.String()
}

func effectRange(run AnalysisRun) (float64, float64) {
	if len(run.InputRows) == 0 {
		return -1, 1
	}
	min, max := run.InputRows[0].EffectSize, run.InputRows[0].EffectSize
	for _, row := range run.InputRows {
		se := math.Sqrt(row.Variance)
		for _, value := range []float64{row.EffectSize - 1.96*se, row.EffectSize + 1.96*se} {
			if value < min {
				min = value
			}
			if value > max {
				max = value
			}
		}
	}
	if min == max {
		return min - 1, max + 1
	}
	return min, max
}

func scaleEffect(value, min, max float64, left, right int) int {
	if max <= min {
		return (left + right) / 2
	}
	return left + int(((value-min)/(max-min))*float64(right-left))
}

func xmlEscape(value string) string {
	value = strings.ReplaceAll(value, "&", "&amp;")
	value = strings.ReplaceAll(value, "<", "&lt;")
	value = strings.ReplaceAll(value, ">", "&gt;")
	value = strings.ReplaceAll(value, `"`, "&quot;")
	return value
}

func checksum(data []byte) string { sum := sha256.Sum256(data); return hex.EncodeToString(sum[:]) }
