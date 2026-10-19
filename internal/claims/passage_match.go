package claims

import (
	"strings"
	"unicode"
	"unicode/utf8"
)

// PassageMatch is a half-open UTF-8 byte range in the original passage, not
// normalized text. Exact matching is preferred; normalized means canonical
// Unicode equivalence (NFD, Unicode 15.0) and collapsed Unicode whitespace.
// Neither matching mode checks semantic support.
type PassageMatch struct {
	State   State        `json:"state"`
	Method  string       `json:"method,omitempty"`
	Start   int          `json:"start"`
	End     int          `json:"end"`
	Offsets *TextOffsets `json:"offsets,omitempty"`
}

func MatchPassage(text, quote string) PassageMatch {
	if strings.TrimSpace(quote) == "" || !utf8.ValidString(text) || !utf8.ValidString(quote) {
		return PassageMatch{State: StateInconclusive}
	}
	if start := strings.Index(text, quote); start >= 0 {
		return PassageMatch{State: StatePassed, Method: "exact", Start: start, End: start + len(quote)}
	}
	haystack, needle := normalizePassage(text), normalizePassage(quote)
	if len(needle) == 0 {
		return PassageMatch{State: StateInconclusive}
	}
	for i := 0; i+len(needle) <= len(haystack); i++ {
		end := i + len(needle)
		// Do not accept half of a decomposed character/combining sequence.
		if combiningClasses[haystack[i].value] != 0 || (end < len(haystack) && combiningClasses[haystack[end].value] != 0) {
			continue
		}
		matched := true
		for j := range needle {
			if haystack[i+j].value != needle[j].value {
				matched = false
				break
			}
		}
		if matched {
			startByte, endByte := len(text), 0
			for _, unit := range haystack[i:end] {
				if unit.start < startByte {
					startByte = unit.start
				}
				if unit.end > endByte {
					endByte = unit.end
				}
			}
			// A match cannot split a single original rune's decomposition (e.g.
			// matching one Jamo inside a precomposed Hangul syllable).
			if normalizedText(text[startByte:endByte]) != normalizedText(quote) {
				continue
			}
			return PassageMatch{State: StatePassed, Method: "normalized", Start: startByte, End: endByte}
		}
	}
	return PassageMatch{State: StateFailed}
}

type normalizedUnit struct {
	value      rune
	start, end int
}

func normalizePassage(text string) []normalizedUnit {
	units := []normalizedUnit{}
	for start, value := range text {
		end := start + utf8.RuneLen(value)
		decomposition := canonicalDecompositions[value]
		// Unicode's algorithmic Hangul canonical decomposition.
		if value >= 0xac00 && value < 0xac00+11172 {
			index := value - 0xac00
			runes := []rune{0x1100 + index/588, 0x1161 + (index%588)/28}
			if tail := index % 28; tail != 0 {
				runes = append(runes, 0x11a7+tail)
			}
			decomposition = string(runes)
		}
		if decomposition == "" {
			decomposition = string(value)
		}
		for _, r := range decomposition {
			units = append(units, normalizedUnit{r, start, end})
			// Stable canonical ordering within each combining sequence.
			for i := len(units) - 1; i > 0 && combiningClasses[units[i].value] != 0 && combiningClasses[units[i-1].value] > combiningClasses[units[i].value]; i-- {
				units[i], units[i-1] = units[i-1], units[i]
			}
		}
	}
	out := make([]normalizedUnit, 0, len(units))
	for _, unit := range units {
		if unicode.IsSpace(unit.value) {
			if len(out) == 0 {
				continue
			}
			unit.value = ' '
			if out[len(out)-1].value == ' ' {
				out[len(out)-1].end = unit.end
				continue
			}
		}
		out = append(out, unit)
	}
	if len(out) > 0 && out[len(out)-1].value == ' ' {
		out = out[:len(out)-1]
	}
	return out
}
