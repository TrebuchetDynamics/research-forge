package cli

import (
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/TrebuchetDynamics/research-forge/internal/evidence"
	"github.com/TrebuchetDynamics/research-forge/internal/library"
)

// TestE2ELightGBMPriceDataLeakageReviewThroughCLI exercises ResearchForge with
// a quant-ML literature review prompt rather than the default artificial
// photosynthesis fixture. The scenario is intentionally falsification-first:
// after a known higher-timeframe lookahead leak, the project must preserve a
// review question, curated literature, screening decisions, supported evidence,
// and a reproducible audit trail without treating hyperparameter tuning as the
// first-class objective.
func TestE2ELightGBMPriceDataLeakageReviewThroughCLI(t *testing.T) {
	proj := filepath.Join(t.TempDir(), "lightgbm-price-data-review")
	const title = "Leak-free LightGBM review for 5-minute crypto direction prediction"
	mustRunCLI(t, "--json", "project", "create", proj, "--title", title)

	manifestBytes, err := os.ReadFile(filepath.Join(proj, "rforge.project.toml"))
	if err != nil {
		t.Fatalf("read manifest: %v", err)
	}
	if !strings.Contains(string(manifestBytes), title) {
		t.Fatalf("manifest did not preserve review title:\n%s", manifestBytes)
	}

	libFile := filepath.Join(t.TempDir(), "quant_ml_library.json")
	fixture := `[
  {"Title":"LightGBM: A Highly Efficient Gradient Boosting Decision Tree","Identifiers":{"ArXivID":"1711.07487"},"Authors":[{"Family":"Ke","Given":"Guolin"}],"Year":2017,"Venue":"NeurIPS","Abstract":"Gradient boosting decision tree model family baseline for tabular prediction; not domain evidence by itself."},
  {"Title":"Empirical Asset Pricing via Machine Learning","Identifiers":{"DOI":"10.1093/rfs/hhaa009"},"Authors":[{"Family":"Gu","Given":"Shihao"},{"Family":"Kelly","Given":"Bryan"},{"Family":"Xiu","Given":"Dacheng"}],"Year":2020,"Venue":"Review of Financial Studies","Abstract":"Machine learning can find weak return-predictive signals only under careful out-of-sample validation and economic evaluation."},
  {"Title":"DeepLOB: Deep Convolutional Neural Networks for Limit Order Books","Identifiers":{"ArXivID":"1808.03668"},"Authors":[{"Family":"Zhang","Given":"Zihao"},{"Family":"Zohren","Given":"Stefan"},{"Family":"Roberts","Given":"Stephen"}],"Year":2018,"Venue":"IEEE Transactions on Signal Processing","Abstract":"Short-horizon limit order book prediction requires event-time data discipline and leakage-aware chronological validation."},
  {"Title":"The Econometrics of Financial Markets","Identifiers":{"DOI":"10.1515/9781400830213"},"Authors":[{"Family":"Campbell","Given":"John Y."},{"Family":"Lo","Given":"Andrew W."},{"Family":"MacKinlay","Given":"A. Craig"}],"Year":1997,"Venue":"Princeton University Press","Abstract":"Financial return predictability is typically weak, noisy, non-stationary, and sensitive to data-snooping bias."}
]
`
	if err := os.WriteFile(libFile, []byte(fixture), 0o644); err != nil {
		t.Fatalf("write library fixture: %v", err)
	}
	mustRunCLI(t, "--json", "--project", proj, "import", "json", libFile)

	var lib struct {
		Data struct {
			Papers []library.PaperRecord `json:"papers"`
		} `json:"data"`
	}
	if err := json.Unmarshal(mustRunCLI(t, "--json", "--project", proj, "library", "list"), &lib); err != nil {
		t.Fatalf("decode library list: %v", err)
	}
	if len(lib.Data.Papers) != 4 {
		t.Fatalf("library size = %d, want 4", len(lib.Data.Papers))
	}

	mustRunCLI(t, "--project", proj, "screen", "configure", "--reason", "tuning-only", "--reason", "not-short-horizon")
	for _, paperID := range []string{"1711.07487", "10.1093/rfs/hhaa009", "1808.03668", "10.1515/9781400830213"} {
		mustRunCLI(t, "--project", proj, "screen", "decide", "--paper", paperID, "--stage", "title_abstract", "--decision", "include", "--reviewer", "quant-reviewer")
	}

	mustRunCLI(t, "--project", proj, "extraction", "schema", "add", "quant-ml-review",
		"--field", "research_question:string",
		"--field", "finding_type:string",
		"--field", "model_family:string",
		"--field", "validation_standard:string",
		"--field", "leakage_risk:string",
		"--field", "performance_expectation:string",
		"--field", "metric:string")

	for _, ev := range []struct {
		paperID string
		values  []string
		support string
	}{
		{"1711.07487", []string{
			"research_question=Is LightGBM appropriate after leakage removal?",
			"finding_type=model_family",
			"model_family=gradient_boosted_decision_trees",
			"validation_standard=domain_validation_required",
			"leakage_risk=not_addressed_by_model_choice",
			"performance_expectation=do_not_infer_market_signal_from_benchmark_algorithm",
			"metric=AUC_logloss_calibration_economic_metrics",
		}, "citation:lightgbm-method"},
		{"10.1093/rfs/hhaa009", []string{
			"research_question=What evidence supports real financial ML signal?",
			"finding_type=expected_performance",
			"model_family=machine_learning_for_returns",
			"validation_standard=chronological_out_of_sample_and_economic_tests",
			"leakage_risk=data_snooping_and_overfit",
			"performance_expectation=weak_signal_can_be_economically_relevant_if_stable",
			"metric=out_of_sample_R2_AUC_Sharpe_turnover_costs",
		}, "citation:asset-pricing-ml"},
		{"1808.03668", []string{
			"research_question=How should short-horizon order-book prediction be validated?",
			"finding_type=microstructure_validation",
			"model_family=order_book_prediction",
			"validation_standard=chronological_or_purged_walk_forward_split",
			"leakage_risk=lookahead_in_aggregation_rolling_normalization_and_labels",
			"performance_expectation=apparent_high_accuracy_often_collapses_under_clean_splits",
			"metric=AUC_logloss_calibration_PnL_after_costs",
		}, "citation:deeplob-validation"},
		{"10.1515/9781400830213", []string{
			"research_question=Is AUC 0.51-0.53 normal for 5-minute crypto direction?",
			"finding_type=falsification_prior",
			"model_family=financial_forecasting",
			"validation_standard=out_of_sample_before_optimization",
			"leakage_risk=nonstationarity_data_snooping_overlap_bias",
			"performance_expectation=near_random_directional_metrics_are_normal_for_liquid_markets",
			"metric=calibration_Brier_Sharpe_PnL_per_trade_turnover",
		}, "citation:financial-econometrics"},
	} {
		args := []string{"--project", proj, "extract", "add", "--paper", ev.paperID, "--schema", "quant-ml-review", "--support", ev.support, "--status", "accepted"}
		for _, value := range ev.values {
			args = append(args, "--value", value)
		}
		mustRunCLI(t, args...)
	}

	var audit struct {
		Data struct {
			Issues []json.RawMessage `json:"issues"`
		} `json:"data"`
	}
	if err := json.Unmarshal(mustRunCLI(t, "--json", "--project", proj, "evidence", "audit"), &audit); err != nil {
		t.Fatalf("decode evidence audit: %v", err)
	}
	if len(audit.Data.Issues) != 0 {
		t.Fatalf("evidence audit reported issues for supported leakage-review evidence: %d", len(audit.Data.Issues))
	}

	var items []evidence.EvidenceItem
	if err := readJSONFile(evidenceItemsPath(proj), &items); err != nil {
		t.Fatalf("read stored evidence: %v", err)
	}
	joined := ""
	for _, item := range items {
		joined += strings.Join([]string{
			item.Values["research_question"], item.Values["finding_type"], item.Values["validation_standard"],
			item.Values["leakage_risk"], item.Values["performance_expectation"], item.Values["metric"],
		}, " ") + "\n"
	}
	for _, want := range []string{
		"not_addressed_by_model_choice",
		"chronological_or_purged_walk_forward_split",
		"lookahead_in_aggregation_rolling_normalization_and_labels",
		"AUC 0.51-0.53 normal",
		"PnL_after_costs",
	} {
		if !strings.Contains(joined, want) {
			t.Fatalf("stored evidence missing %q:\n%s", want, joined)
		}
	}

	reportPath := filepath.Join(proj, "exports", "lightgbm-price-data-review.md")
	mustRunCLI(t, "--json", "--project", proj, "report", "build", "--out", reportPath)
	reportBytes, err := os.ReadFile(reportPath)
	if err != nil {
		t.Fatalf("read report: %v", err)
	}
	if !strings.Contains(string(reportBytes), title) {
		t.Fatalf("report missing quant review title:\n%s", reportBytes)
	}
	mustRunCLI(t, "--json", "--project", proj, "report", "audit")
}
