package main

import (
	"fmt"
	"strings"

	b "github.com/TrebuchetDynamics/research-forge/internal/benchmarks"
)

func containsProvider(providers []string, name string) bool {
	for _, provider := range providers {
		if provider == name {
			return true
		}
	}
	return false
}

func validateProviderPolicy(p ProviderPolicy) error {
	if len(p.WithSources) < 2 || !p.ContinueOnError || p.RateLimitAction != "skip_for_run" {
		return fmt.Errorf("protocol_violation: require a provider mix, continue-on-error and skip_for_run")
	}
	seen := map[string]bool{}
	for _, provider := range p.WithSources {
		if strings.TrimSpace(provider) == "" || provider != strings.ToLower(strings.TrimSpace(provider)) || seen[provider] {
			return fmt.Errorf("protocol_violation: invalid/duplicate provider")
		}
		seen[provider] = true
	}
	return nil
}

func validateProviderEvents(run RawRun, policy ProviderPolicy) error {
	if len(run.ProviderEvents) == 0 && (run.Usage.SearchCalls > 0 || run.Usage.HTTPRequests > 0) {
		return fmt.Errorf("protocol_violation: missing provider receipts")
	}
	blocked := map[string]bool{}
	var attempts int64
	for _, e := range run.ProviderEvents {
		if strings.TrimSpace(e.Provider) == "" || e.Provider != strings.ToLower(strings.TrimSpace(e.Provider)) || strings.TrimSpace(e.Query) == "" || !validStop(e.StopReason) {
			return fmt.Errorf("protocol_violation: invalid provider receipt")
		}
		if run.Slot.Condition == "with_rforge" && !containsProvider(policy.WithSources, e.Provider) {
			return fmt.Errorf("protocol_violation: unregistered provider")
		}
		if e.Action == "skipped" {
			if !blocked[e.Provider] || e.HTTPStatus != 0 || e.RetryAfter != "" || strings.TrimSpace(e.Error) == "" || e.StopReason != "error_threshold" {
				return fmt.Errorf("protocol_violation: invalid provider skip")
			}
			continue
		}
		if blocked[e.Provider] {
			return fmt.Errorf("protocol_violation: provider retried after HTTP 429")
		}
		attempts++
		switch e.Action {
		case "completed":
			if e.HTTPStatus < 200 || e.HTTPStatus >= 400 || e.Error != "" {
				return fmt.Errorf("protocol_violation: hidden provider failure")
			}
		case "failed":
			if (e.HTTPStatus != 0 && (e.HTTPStatus < 100 || e.HTTPStatus > 599)) || strings.TrimSpace(e.Error) == "" {
				return fmt.Errorf("protocol_violation: invalid provider failure")
			}
		default:
			return fmt.Errorf("protocol_violation: invalid provider action")
		}
		if e.HTTPStatus == 429 {
			blocked[e.Provider] = true
		}
	}
	if attempts != run.Usage.HTTPRequests {
		return fmt.Errorf("protocol_violation: provider attempts do not reconcile with HTTP usage")
	}
	return nil
}

func validStop(reason string) bool {
	switch reason {
	case "completed", "budget_exhausted", "coverage_saturated", "interrupted", "error_threshold":
		return true
	}
	return false
}

// The stub is canned fixture data, not a live gateway. Native search uses the
// same synthetic providers here solely to exercise receipts in both conditions.
func stubProviderEvents(providers []string, throttled string) []ProviderEvent {
	events := []ProviderEvent{}
	blocked := map[string]bool{}
	for _, query := range []string{"synthetic energy", "synthetic energy conversion"} {
		for _, provider := range providers {
			e := ProviderEvent{Provider: provider, Query: query, Action: "completed", HTTPStatus: 200, StopReason: "completed"}
			if blocked[provider] {
				e.Action, e.HTTPStatus, e.Error, e.StopReason = "skipped", 0, "provider quarantined after HTTP 429", "error_threshold"
			} else if provider == throttled {
				e.Action, e.HTTPStatus, e.RetryAfter, e.Error, e.StopReason = "failed", 429, "3600", "rate limited", "error_threshold"
				blocked[provider] = true
			}
			events = append(events, e)
		}
	}
	return events
}

type ProviderAvailability struct {
	Slot             Slot            `json:"slot"`
	ProviderLost     bool            `json:"provider_lost"`
	PairProviderLost bool            `json:"pair_provider_lost"`
	Events           []ProviderEvent `json:"events"`
}

type AgentReport struct {
	b.EngineReport
	ProviderAvailability []ProviderAvailability `json:"provider_availability"`
}

func availability(runs []RawRun) []ProviderAvailability {
	type pair struct {
		question   string
		repetition int
	}
	lost := map[pair]bool{}
	rows := []ProviderAvailability{}
	for _, run := range runs {
		row := ProviderAvailability{Slot: run.Slot, Events: run.ProviderEvents}
		for _, e := range run.ProviderEvents {
			if e.Action == "failed" {
				row.ProviderLost = true
				lost[pair{run.Slot.QuestionID, run.Slot.Repetition}] = true
			}
		}
		rows = append(rows, row)
	}
	for i := range rows {
		rows[i].PairProviderLost = lost[pair{rows[i].Slot.QuestionID, rows[i].Slot.Repetition}]
	}
	return rows
}
