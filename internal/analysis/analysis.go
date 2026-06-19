package analysis

import (
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"math"
	"os"
	"path/filepath"
	"sort"
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
	ViSource   string            // "ci", "se", "floor", or "" for arm-pair calculators
	Moderators map[string]string // benchmarking moderator fields keyed by field name
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

// RawContinuousResult holds the output of a single raw-continuous effect calculation.
type RawContinuousResult struct {
	Yi       float64
	Vi       float64
	ViSource string // "ci", "se", or "floor"
}

// RawContinuousOutcome pools a single continuous metric per study (e.g., STH efficiency %).
// It implements EffectSizeCalculator for use with PrepareWithCalculator and also exposes
// CalculateRaw for PrepareRawContinuous, which also records ViSource and Moderators.
type RawContinuousOutcome struct {
	VarianceFloor float64 // floor vi when no CI or SE is reported; defaults to 0.0025
}

func (r RawContinuousOutcome) floor() float64 {
	if r.VarianceFloor > 0 {
		return r.VarianceFloor
	}
	return 0.0025
}

// CalculateRaw returns the full result including ViSource provenance tag.
func (r RawContinuousOutcome) CalculateRaw(values map[string]string) (RawContinuousResult, error) {
	yiStr := strings.TrimSpace(values["value_pct"])
	if yiStr == "" {
		return RawContinuousResult{}, fmt.Errorf("value_pct is required for raw-continuous effect measure")
	}
	yi, err := strconv.ParseFloat(yiStr, 64)
	if err != nil {
		return RawContinuousResult{}, fmt.Errorf("value_pct is not numeric: %s", yiStr)
	}
	ciLow, errLow := strconv.ParseFloat(strings.TrimSpace(values["ci_lower"]), 64)
	ciHigh, errHigh := strconv.ParseFloat(strings.TrimSpace(values["ci_upper"]), 64)
	if errLow == nil && errHigh == nil && ciHigh > ciLow {
		se := (ciHigh - ciLow) / (2 * 1.96)
		return RawContinuousResult{Yi: yi, Vi: se * se, ViSource: "ci"}, nil
	}
	se, errSE := strconv.ParseFloat(strings.TrimSpace(values["se"]), 64)
	if errSE == nil && se > 0 {
		return RawContinuousResult{Yi: yi, Vi: se * se, ViSource: "se"}, nil
	}
	return RawContinuousResult{Yi: yi, Vi: r.floor(), ViSource: "floor"}, nil
}

// Calculate implements EffectSizeCalculator.
func (r RawContinuousOutcome) Calculate(values map[string]string) (float64, float64, error) {
	res, err := r.CalculateRaw(values)
	return res.Yi, res.Vi, err
}

// PrepareRawContinuous builds an AnalysisRun for scientific benchmarking meta-analysis.
// It records ViSource provenance per row and copies moderatorFields from evidence values.
func PrepareRawContinuous(id string, items []evidence.EvidenceItem, varianceFloor float64, moderatorFields []string) (AnalysisRun, error) {
	calc := RawContinuousOutcome{VarianceFloor: varianceFloor}
	run := AnalysisRun{SchemaVersion: "1", ID: id}
	for _, item := range items {
		if item.Status != evidence.StatusAccepted {
			continue
		}
		res, err := calc.CalculateRaw(item.Values)
		if err != nil {
			return AnalysisRun{}, err
		}
		row := InputRow{PaperID: item.PaperID, EffectSize: res.Yi, Variance: res.Vi, ViSource: res.ViSource}
		if len(moderatorFields) > 0 {
			row.Moderators = make(map[string]string, len(moderatorFields))
			for _, f := range moderatorFields {
				row.Moderators[f] = item.Values[f]
			}
		}
		run.InputRows = append(run.InputRows, row)
	}
	return run, nil
}

// ExcludeByViSource returns a copy of run omitting rows whose ViSource equals the given tag.
func ExcludeByViSource(run AnalysisRun, viSource string) AnalysisRun {
	out := AnalysisRun{SchemaVersion: run.SchemaVersion, ID: run.ID + "-excl-" + viSource}
	for _, row := range run.InputRows {
		if row.ViSource != viSource {
			out.InputRows = append(out.InputRows, row)
		}
	}
	return out
}

// ReadinessIssue records a missing required field for one accepted evidence item.
type ReadinessIssue struct {
	PaperID      string `json:"paperId"`
	MissingField string `json:"missingField"`
}

// ReadinessReport summarises whether all accepted evidence items carry the required fields.
type ReadinessReport struct {
	RunID      string           `json:"runId"`
	Ready      bool             `json:"ready"`
	TotalItems int              `json:"totalItems"`
	ReadyItems int              `json:"readyItems"`
	Issues     []ReadinessIssue `json:"issues"`
}

// BenchmarkingReadiness checks that every accepted evidence item has all requiredFields populated.
// Pass nil requiredFields to use the default STH% set.
func BenchmarkingReadiness(runID string, items []evidence.EvidenceItem, requiredFields []string) ReadinessReport {
	if len(requiredFields) == 0 {
		requiredFields = []string{"value_pct", "device_type", "auxiliary_bias", "measurement_standard"}
	}
	report := ReadinessReport{RunID: runID}
	for _, item := range items {
		if item.Status != evidence.StatusAccepted {
			continue
		}
		report.TotalItems++
		ready := true
		for _, field := range requiredFields {
			if strings.TrimSpace(item.Values[field]) == "" {
				report.Issues = append(report.Issues, ReadinessIssue{PaperID: item.PaperID, MissingField: field})
				ready = false
			}
		}
		if ready {
			report.ReadyItems++
		}
	}
	report.Ready = len(report.Issues) == 0 && report.TotalItems > 0
	return report
}

func GenerateMetaforScript(run AnalysisRun) string {
	modKeys := collectModeratorKeys(run)
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
	b.WriteString(")")
	for _, key := range modKeys {
		safe := rSafeColumnName(key)
		fmt.Fprintf(&b, ", %s=c(", safe)
		for i, row := range run.InputRows {
			if i > 0 {
				b.WriteString(",")
			}
			val := ""
			if row.Moderators != nil {
				val = row.Moderators[key]
			}
			if val == "" {
				b.WriteString("NA")
			} else {
				fmt.Fprintf(&b, "%q", val)
			}
		}
		b.WriteString(")")
	}
	b.WriteString(")\n")
	if len(modKeys) > 0 {
		safeKeys := make([]string, len(modKeys))
		for i, k := range modKeys {
			safeKeys[i] = rSafeColumnName(k)
		}
		fmt.Fprintf(&b, "model <- rma(yi = yi, vi = vi, mods = ~%s, data=data)\n", strings.Join(safeKeys, "+"))
	} else {
		b.WriteString("model <- rma(yi = yi, vi = vi, data=data)\n")
	}
	b.WriteString("print(model)\n")
	return b.String()
}

func collectModeratorKeys(run AnalysisRun) []string {
	seen := map[string]bool{}
	var keys []string
	for _, row := range run.InputRows {
		for k := range row.Moderators {
			if !seen[k] {
				seen[k] = true
				keys = append(keys, k)
			}
		}
	}
	sort.Strings(keys)
	return keys
}

func rSafeColumnName(name string) string {
	var b strings.Builder
	for _, r := range name {
		if (r >= 'a' && r <= 'z') || (r >= 'A' && r <= 'Z') || (r >= '0' && r <= '9') || r == '_' {
			b.WriteRune(r)
		} else {
			b.WriteRune('_')
		}
	}
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

func publicationPlotStyle() string {
	return `<desc>ResearchForge publication-ready plot styling</desc><style>text{font-family:Inter,Arial,sans-serif;fill:#172033}.axis{stroke:#5b6472;stroke-width:1.5}.effect{stroke:#2457a6;stroke-width:2.5}.point{fill:#2457a6}.funnel-point{fill:#b85c1d}</style>`
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
	b.WriteString(`<title>Forest plot</title>` + publicationPlotStyle() + `<line class="axis" x1="320" y1="30" x2="320" y2="` + fmt.Sprint(height-30) + `"/>`)
	for i, row := range run.InputRows {
		y := 55 + i*34
		x := scaleEffect(row.EffectSize, min, max, 80, 580)
		se := math.Sqrt(row.Variance)
		left := scaleEffect(row.EffectSize-1.96*se, min, max, 80, 580)
		right := scaleEffect(row.EffectSize+1.96*se, min, max, 80, 580)
		label := xmlEscape(row.PaperID)
		fmt.Fprintf(&b, `<line class="effect" x1="%d" y1="%d" x2="%d" y2="%d"/><circle class="point" cx="%d" cy="%d" r="5"/><text x="12" y="%d" font-size="12">%s</text>`, left, y, right, y, x, y, y+4, label)
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
	b.WriteString(`<title>Funnel plot</title>` + publicationPlotStyle() + `<line class="axis" x1="60" y1="360" x2="600" y2="360"/><line class="axis" x1="60" y1="40" x2="60" y2="360"/>`)
	for _, row := range run.InputRows {
		x := scaleEffect(row.EffectSize, min, max, 80, 580)
		y := 60 + int((math.Sqrt(row.Variance)/maxSE)*280)
		fmt.Fprintf(&b, `<circle class="funnel-point" cx="%d" cy="%d" r="5"><title>%s</title></circle>`, x, y, xmlEscape(row.PaperID))
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
