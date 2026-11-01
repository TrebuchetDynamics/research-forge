// Package fields holds the small normalization helpers that source connectors
// share. Each helper has at least two connector call sites.
package fields

import (
	"encoding/json"
	"regexp"
	"strings"
)

// htmlTagRe matches HTML tags for stripping.
var htmlTagRe = regexp.MustCompile(`<[^>]+>`)

// NonEmptyStrings returns the trimmed values that are not empty.
func NonEmptyStrings(values ...string) []string {
	out := []string{}
	for _, value := range values {
		value = strings.TrimSpace(value)
		if value != "" {
			out = append(out, value)
		}
	}
	return out
}

// NormalizeDOI lowercases a DOI and removes a URL or "doi:" prefix.
func NormalizeDOI(value string) string {
	value = strings.ToLower(strings.TrimSpace(value))
	value = strings.TrimPrefix(value, "https://doi.org/")
	value = strings.TrimPrefix(value, "http://doi.org/")
	value = strings.TrimPrefix(value, "doi:")
	return strings.TrimSpace(value)
}

// CompactSpace collapses runs of whitespace into one space.
func CompactSpace(value string) string {
	return strings.Join(strings.Fields(value), " ")
}

// StripHTMLTags removes HTML tags from s and collapses whitespace.
func StripHTMLTags(s string) string {
	result := htmlTagRe.ReplaceAllString(s, " ")
	return strings.Join(strings.Fields(result), " ")
}

// FirstNonEmpty returns the first value that is not blank.
func FirstNonEmpty(values ...string) string {
	for _, value := range values {
		if strings.TrimSpace(value) != "" {
			return value
		}
	}
	return ""
}

// JSONScalarString decodes a JSON string or number into a trimmed string.
func JSONScalarString(raw json.RawMessage) string {
	var text string
	if err := json.Unmarshal(raw, &text); err == nil {
		return strings.TrimSpace(text)
	}
	var number json.Number
	if err := json.Unmarshal(raw, &number); err == nil {
		return number.String()
	}
	return ""
}

// DOIURL returns the doi.org URL for a DOI.
func DOIURL(doi string) string {
	if doi == "" {
		return ""
	}
	return "https://doi.org/" + doi
}

// NormalizeArXivID trims whitespace, an abs URL prefix and a version suffix.
func NormalizeArXivID(value string) string {
	value = strings.TrimSpace(value)
	value = strings.TrimPrefix(value, "http://arxiv.org/abs/")
	value = strings.TrimPrefix(value, "https://arxiv.org/abs/")
	if idx := strings.Index(value, "v"); idx > 0 {
		value = value[:idx]
	}
	return value
}

// HTMLTagRe matches HTML tags. Connectors that need custom replacement text
// use it directly; otherwise use StripHTMLTags.
var HTMLTagRe = htmlTagRe
