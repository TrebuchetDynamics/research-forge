// Package text holds the small string helpers that two or more parser
// adapters or reconciliation packages share.
package text

import "strings"

// CompactSpace collapses runs of whitespace into one space.
func CompactSpace(value string) string {
	return strings.Join(strings.Fields(value), " ")
}

// CompactParsingSpace is the historical name used by the PaperMage adapter.
func CompactParsingSpace(value string) string { return CompactSpace(value) }

// NormalizeReferenceDOI lowercases a reference DOI and removes a URL or
// "doi:" prefix.
func NormalizeReferenceDOI(value string) string {
	value = strings.TrimSpace(strings.ToLower(value))
	value = strings.TrimPrefix(value, "https://doi.org/")
	value = strings.TrimPrefix(value, "http://doi.org/")
	value = strings.TrimPrefix(value, "doi:")
	return strings.TrimSpace(value)
}
