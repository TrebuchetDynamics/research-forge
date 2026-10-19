package claims_test

import (
	"encoding/json"
	"reflect"
	"testing"
	"time"

	"github.com/TrebuchetDynamics/research-forge/internal/claims"
)

// Dropping a field (including an unchecked facet) must not silently lose data.
func TestClaimJSONRoundTrip(t *testing.T) {
	for _, input := range []string{
		`{
			"schemaVersion":"0", "id":"claim-1",
			"text":"Artificial photosynthesis efficiency increased.",
			"workflowStatus":"suggested",
			"evidence":[{
				"identifier":{"scheme":"doi","value":"10.1234/example"},
				"passageId":"p1", "relation":"supports", "exactText":"Efficiency increased.",
				"contentHash":"sha256:0123456789abcdef0123456789abcdef0123456789abcdef0123456789abcdef",
				"offsets":{"start":12,"end":33,"parserName":"grobid","parserVersion":"0.8.2"}
			}],
			"facets":{
				"sourceIdentity":{"state":"passed","checker":{"kind":"engine","id":"rforge","version":"dev"},"timestamp":"2026-10-19T12:00:00Z"},
				"contentIntegrity":{"state":"failed","checker":{"kind":"engine","id":"rforge","version":"dev"},"timestamp":"2026-10-19T12:00:00Z"},
				"passageMatch":{"state":"inconclusive","checker":{"kind":"engine","id":"rforge","version":"dev"},"timestamp":"2026-10-19T12:00:00Z"},
				"semanticSupport":{"state":"passed","checker":{"kind":"agent","id":"pi","version":"1","model":"model-1"},"timestamp":"2026-10-19T12:00:00Z"},
				"humanApproval":{"state":"failed","checker":{"kind":"human","id":"ada"},"timestamp":"2026-10-19T12:00:00Z"}
			}
		}`,
		`{"schemaVersion":"0","id":"claim-2","text":"Unassessed assertion.","evidence":[],"facets":{"sourceIdentity":null,"contentIntegrity":null,"passageMatch":null,"semanticSupport":null,"humanApproval":null}}`,
	} {
		var record claims.Claim
		if err := json.Unmarshal([]byte(input), &record); err != nil {
			t.Fatal(err)
		}
		encoded, err := json.Marshal(record)
		if err != nil {
			t.Fatal(err)
		}
		var want, got any
		if err := json.Unmarshal([]byte(input), &want); err != nil {
			t.Fatal(err)
		}
		if err := json.Unmarshal(encoded, &got); err != nil {
			t.Fatal(err)
		}
		if !reflect.DeepEqual(got, want) {
			t.Fatalf("round trip lost data:\n got %s\nwant %s", encoded, input)
		}
	}
}

// Removing the agent/model guard must fail at both construction and JSON input.
func TestSemanticSupportCheckerPolicy(t *testing.T) {
	for _, tc := range []struct {
		name    string
		checker claims.Checker
		wantErr bool
	}{
		{"agent", claims.Checker{Kind: claims.CheckerAgent, ID: "pi", Model: "model-1"}, false},
		{"engine", claims.Checker{Kind: claims.CheckerEngine, ID: "rforge", Model: "model-1"}, true},
		{"human", claims.Checker{Kind: claims.CheckerHuman, ID: "ada", Model: "model-1"}, true},
		{"unnamed agent", claims.Checker{Kind: claims.CheckerAgent, ID: " ", Model: "model-1"}, true},
		{"missing model", claims.Checker{Kind: claims.CheckerAgent, ID: "pi"}, true},
	} {
		t.Run(tc.name, func(t *testing.T) {
			assessment := claims.Assessment{State: claims.StatePassed, Checker: tc.checker, Timestamp: time.Date(2026, 10, 19, 12, 0, 0, 0, time.UTC)}
			_, err := claims.NewSemanticSupport(assessment)
			if (err != nil) != tc.wantErr {
				t.Fatalf("constructor error = %v, wantErr %v", err, tc.wantErr)
			}
			payload, err := json.Marshal(assessment)
			if err != nil {
				t.Fatal(err)
			}
			var decoded claims.SemanticSupport
			if err := json.Unmarshal(payload, &decoded); (err != nil) != tc.wantErr {
				t.Fatalf("JSON error = %v, wantErr %v", err, tc.wantErr)
			}
		})
	}
}

// A recorded assessment needs an outcome and time; zero values cannot bypass this.
func TestAssessmentMetadata(t *testing.T) {
	at := time.Date(2026, 10, 19, 12, 0, 0, 0, time.UTC)
	for _, tc := range []struct {
		state claims.State
		at    time.Time
	}{
		{claims.StatePassed, time.Time{}},
		{claims.State("not_checked"), at},
		{claims.State("unknown"), at},
	} {
		if _, err := claims.NewSemanticSupport(claims.Assessment{State: tc.state, Checker: claims.Checker{Kind: claims.CheckerAgent, ID: "pi", Model: "model-1"}, Timestamp: tc.at}); err == nil {
			t.Fatalf("accepted incomplete assessment: %+v", tc)
		}
	}
	for _, zero := range []any{claims.SemanticSupport{}, claims.HumanApproval{}} {
		if _, err := json.Marshal(zero); err == nil {
			t.Fatalf("serialized uninitialized %T", zero)
		}
	}
}

// Approval must remain attributable to a named human, not an automated checker.
func TestHumanApprovalCheckerPolicy(t *testing.T) {
	for _, tc := range []struct {
		checker claims.Checker
		wantErr bool
	}{
		{claims.Checker{Kind: claims.CheckerHuman, ID: "ada"}, false},
		{claims.Checker{Kind: claims.CheckerHuman, ID: " "}, true},
		{claims.Checker{Kind: claims.CheckerAgent, ID: "pi"}, true},
		{claims.Checker{Kind: claims.CheckerEngine, ID: "rforge"}, true},
	} {
		assessment := claims.Assessment{State: claims.StatePassed, Checker: tc.checker, Timestamp: time.Date(2026, 10, 19, 12, 0, 0, 0, time.UTC)}
		_, err := claims.NewHumanApproval(assessment)
		if (err != nil) != tc.wantErr {
			t.Fatalf("checker %+v: constructor error = %v, wantErr %v", tc.checker, err, tc.wantErr)
		}
		payload, err := json.Marshal(assessment)
		if err != nil {
			t.Fatal(err)
		}
		var decoded claims.HumanApproval
		if err := json.Unmarshal(payload, &decoded); (err != nil) != tc.wantErr {
			t.Fatalf("checker %+v: JSON error = %v, wantErr %v", tc.checker, err, tc.wantErr)
		}
	}
}
