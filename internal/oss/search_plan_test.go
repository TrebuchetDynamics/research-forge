package oss

import "testing"

func TestBuildSearchPlanCoversForgesArchivesSecurityAndRegistries(t *testing.T) {
	plan, err := BuildSearchPlan("meta analysis tools", "all")
	if err != nil {
		t.Fatalf("BuildSearchPlan: %v", err)
	}
	if plan.SchemaVersion != "1" || plan.Query != "meta analysis tools" || plan.Ecosystem != "all" {
		t.Fatalf("plan header = %#v", plan)
	}
	for _, want := range []string{"GitHub", "GitLab", "Codeberg", "SourceHut", "Software Heritage", "OpenSSF Scorecard", "pkg.go.dev", "PyPI", "npm", "crates.io", "Zenodo/GitHub links"} {
		if !searchPlanHasProvider(plan, want) {
			t.Fatalf("missing provider %q in %#v", want, plan.Providers)
		}
	}
	for _, provider := range plan.Providers {
		if provider.HumanGate == "" || len(provider.Signals) == 0 {
			t.Fatalf("provider missing gate/signals: %#v", provider)
		}
	}
}

func TestBuildSearchPlanFiltersEcosystemRegistries(t *testing.T) {
	plan, err := BuildSearchPlan("gradient boosting", "rust")
	if err != nil {
		t.Fatalf("BuildSearchPlan: %v", err)
	}
	if !searchPlanHasProvider(plan, "crates.io") {
		t.Fatalf("rust plan missing crates.io: %#v", plan.Providers)
	}
	if searchPlanHasProvider(plan, "npm") || searchPlanHasProvider(plan, "PyPI") {
		t.Fatalf("rust plan should not include npm/PyPI registries: %#v", plan.Providers)
	}
}

func TestBuildSearchPlanRejectsBlankQuery(t *testing.T) {
	if _, err := BuildSearchPlan("  ", "all"); err == nil {
		t.Fatal("expected blank query to fail")
	}
}

func searchPlanHasProvider(plan SearchPlan, name string) bool {
	for _, provider := range plan.Providers {
		if provider.Provider == name {
			return true
		}
	}
	return false
}
