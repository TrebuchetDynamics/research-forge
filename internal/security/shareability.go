package security

import (
	"regexp"
	"strings"
)

const (
	ShareabilityLeakCredential = "credential"
	ShareabilityLeakLocalPath  = "private local path"
)

var (
	namedCredentialPattern  = regexp.MustCompile(`(?i)\b(?:[a-z0-9]+[_-])*(?:api[_-]?key|access[_-]?token|auth[_-]?token|token|password|passwd|client[_-]?secret|secret[_-]?key|secret|private[_-]?key)\b\s*["']?\s*(?::|=)\s*["']?([^\s"',}\]]+)`)
	bearerCredentialPattern = regexp.MustCompile(`(?i)\bauthorization\b\s*(?::|=)\s*["']?bearer\s+([a-z0-9._~+/=-]{8,})`)
	privateKeyPattern       = regexp.MustCompile(`-----BEGIN (?:RSA |EC |OPENSSH )?PRIVATE KEY-----`)
	unixHomePathPattern     = regexp.MustCompile(`(?i)(?:^|[\s"'=(:,])(?:/home|/users)/[a-z0-9._-]+(?:/|\b)`)
	windowsHomePathPattern  = regexp.MustCompile(`(?i)[a-z]:[\\/]+users[\\/]+[^\\/\s"'<>]+[\\/]`)
	fileURLPattern          = regexp.MustCompile(`(?i)\bfile:(?://)?(?:/|[a-z]:[\\/])`)
)

// DetectShareabilityLeak reports high-confidence credential or private-path
// evidence without returning the matched value.
func DetectShareabilityLeak(text string) (string, bool) {
	if privateKeyPattern.MatchString(text) || bearerCredentialPattern.MatchString(text) {
		return ShareabilityLeakCredential, true
	}
	for _, match := range namedCredentialPattern.FindAllStringSubmatch(text, -1) {
		if len(match) > 1 && !credentialPlaceholder(match[1]) {
			return ShareabilityLeakCredential, true
		}
	}
	if unixHomePathPattern.MatchString(text) || windowsHomePathPattern.MatchString(text) || fileURLPattern.MatchString(text) {
		return ShareabilityLeakLocalPath, true
	}
	return "", false
}

func credentialPlaceholder(value string) bool {
	value = strings.TrimSpace(strings.Trim(value, `"'`))
	lower := strings.ToLower(value)
	if lower == "" || strings.HasPrefix(lower, "$") {
		return true
	}
	lower = strings.Trim(lower, "[]<>")
	switch lower {
	case "redacted", "masked", "provided-at-runtime", "test", "fixture", "example", "placeholder", "changeme", "none", "null", "unset", "not-set":
		return true
	}
	return strings.Trim(value, "*xX-") == ""
}
