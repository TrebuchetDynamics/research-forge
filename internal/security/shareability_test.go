package security

import "testing"

func TestDetectShareabilityLeakRecognizesPrefixedCredentialName(t *testing.T) {
	kind, found := DetectShareabilityLeak("OPENAI_API_KEY=live-provider-credential-123456")
	if !found || kind != ShareabilityLeakCredential {
		t.Fatalf("DetectShareabilityLeak found=%t kind=%q, want credential", found, kind)
	}
}

func TestDetectShareabilityLeakDoesNotTreatArbitraryBracketedValueAsRedacted(t *testing.T) {
	kind, found := DetectShareabilityLeak("apiKey=[actual-secret-value]")
	if !found || kind != ShareabilityLeakCredential {
		t.Fatalf("DetectShareabilityLeak found=%t kind=%q, want credential", found, kind)
	}
}

func TestDetectShareabilityLeakClassifiesHighConfidencePatterns(t *testing.T) {
	for _, tt := range []struct {
		name string
		text string
		kind string
	}{
		{name: "bearer", text: "Authorization: Bearer live-token-value-123456", kind: ShareabilityLeakCredential},
		{name: "private key", text: "-----BEGIN PRIVATE KEY-----", kind: ShareabilityLeakCredential},
		{name: "unix home", text: "/Users/reviewer/research/paper.pdf", kind: ShareabilityLeakLocalPath},
		{name: "windows home", text: `C:\Users\reviewer\research\paper.pdf`, kind: ShareabilityLeakLocalPath},
		{name: "file URL", text: "file:///home/reviewer/research/paper.pdf", kind: ShareabilityLeakLocalPath},
	} {
		t.Run(tt.name, func(t *testing.T) {
			kind, found := DetectShareabilityLeak(tt.text)
			if !found || kind != tt.kind {
				t.Fatalf("DetectShareabilityLeak found=%t kind=%q, want %q", found, kind, tt.kind)
			}
		})
	}
}

func TestDetectShareabilityLeakAllowsPlaceholdersAndScientificProse(t *testing.T) {
	for _, text := range []string{
		`{"apiKey":"[redacted]"}`,
		"PASSWORD=${REPORT_PASSWORD}",
		"token=<provided-at-runtime>",
		"The paper studies secreted proteins and private-sector research.",
		"documents/open-access/paper.pdf",
		"https://example.org/users/alice/profile",
	} {
		if kind, found := DetectShareabilityLeak(text); found {
			t.Fatalf("DetectShareabilityLeak(%q) found unexpected %q", text, kind)
		}
	}
}
