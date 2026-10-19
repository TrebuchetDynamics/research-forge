package claims

import (
	"crypto/sha256"
	"fmt"
	"strings"
	"time"

	"github.com/TrebuchetDynamics/research-forge/internal/library"
	"github.com/TrebuchetDynamics/research-forge/internal/parsing"
)

// CheckInput supplies stored, offline evidence. Timestamp is the check time;
// callers supply it explicitly for deterministic replay. No source is fetched live.
type CheckInput struct {
	LibraryRecords  []library.PaperRecord
	ParsedDocuments []parsing.ParsedDocument
	Timestamp       time.Time
}

type EvidenceCheck struct {
	SourceIdentity   State        `json:"sourceIdentity"`
	ContentIntegrity State        `json:"contentIntegrity"`
	PassageMatch     PassageMatch `json:"passageMatch"`
}

type CheckResult struct {
	Claim Claim `json:"claim"`
	// Evidence is in the same order as Claim.Evidence.
	Evidence []EvidenceCheck `json:"evidence"`
}

// Check replaces only the three deterministic facets. Failed dominates
// inconclusive, which dominates passed; an empty evidence set is inconclusive.
// Semantic support and human approval are preserved, never inferred.
func Check(record Claim, input CheckInput) (CheckResult, error) {
	at := input.Timestamp
	if at.IsZero() {
		return CheckResult{}, fmt.Errorf("claim check requires an explicit timestamp")
	}
	result := CheckResult{Claim: record, Evidence: []EvidenceCheck{}}
	identity, integrity, match := StatePassed, StatePassed, StatePassed
	if len(record.Evidence) == 0 {
		identity, integrity, match = StateInconclusive, StateInconclusive, StateInconclusive
	}
	for _, ref := range record.Evidence {
		checked := checkReference(ref, input)
		result.Evidence = append(result.Evidence, checked)
		identity = combineState(identity, checked.SourceIdentity)
		integrity = combineState(integrity, checked.ContentIntegrity)
		match = combineState(match, checked.PassageMatch.State)
	}
	assessment := func(state State) *Assessment {
		return &Assessment{State: state, Checker: Checker{Kind: CheckerEngine, ID: "rforge.claim.check", Version: "1"}, Timestamp: at}
	}
	result.Claim.Facets.SourceIdentity = assessment(identity)
	result.Claim.Facets.ContentIntegrity = assessment(integrity)
	result.Claim.Facets.PassageMatch = assessment(match)
	return result, nil
}

func combineState(a, b State) State {
	if a == StateFailed || b == StateFailed {
		return StateFailed
	}
	if a == StateInconclusive || b == StateInconclusive {
		return StateInconclusive
	}
	return StatePassed
}

func checkReference(ref EvidenceReference, input CheckInput) EvidenceCheck {
	out := EvidenceCheck{SourceIdentity: StateFailed, ContentIntegrity: StateInconclusive, PassageMatch: PassageMatch{State: StateInconclusive}}
	var source *library.PaperRecord
	for i := range input.LibraryRecords {
		candidate := &input.LibraryRecords[i]
		if identifierMatches(*candidate, ref.Identifier) {
			if source != nil {
				out.SourceIdentity = StateInconclusive
				return out
			}
			source = candidate
		}
	}
	if source == nil {
		return out
	}
	out.SourceIdentity = StateInconclusive
	compared := false
	observe := func(state State) {
		if !compared {
			out.SourceIdentity = state
			compared = true
		} else {
			out.SourceIdentity = combineState(out.SourceIdentity, state)
		}
	}
	compareTitle := func(title string) {
		expected, actual := normalizedText(title), normalizedText(source.Title)
		if expected == "" {
			return
		}
		switch {
		case actual == "":
			observe(StateInconclusive)
		case expected != actual:
			observe(StateFailed)
		default:
			observe(StatePassed)
		}
	}
	if meta := ref.SourceMetadata; meta != nil {
		compareTitle(meta.Title)
		if meta.Year != 0 {
			switch {
			case source.Year == 0:
				observe(StateInconclusive)
			case meta.Year != source.Year:
				observe(StateFailed)
			default:
				observe(StatePassed)
			}
		}
	}
	var found *parsing.Passage
	var parserName, parserVersion string
	for _, doc := range input.ParsedDocuments {
		if !paperMatches(*source, doc.PaperID) {
			continue
		}
		compareTitle(doc.Title)
		for _, section := range doc.Sections {
			for i := range section.Passages {
				passage := &section.Passages[i]
				if passage.ID != ref.PassageID || ref.PassageID == "" {
					continue
				}
				if passage.PaperID != "" && !paperMatches(*source, passage.PaperID) {
					continue
				}
				if found != nil {
					return out
				} // Ambiguous parser output cannot be selected arbitrarily.
				found = passage
				parserName, parserVersion = doc.ParserName, doc.ParserVersion
			}
		}
	}
	if found == nil {
		return out
	}
	actual := fmt.Sprintf("sha256:%x", sha256.Sum256([]byte(found.Text)))
	if ref.ContentHash != "" || found.ContentHash != "" {
		switch {
		case ref.ContentHash != "" && ref.ContentHash != actual, found.ContentHash != "" && found.ContentHash != actual:
			out.ContentIntegrity = StateFailed
		case ref.ContentHash != "" && found.ContentHash != "":
			out.ContentIntegrity = StatePassed
		}
	}
	out.PassageMatch = MatchPassage(found.Text, ref.ExactText)
	if out.PassageMatch.State == StatePassed {
		m := &out.PassageMatch
		if found.Offset.Start >= 0 && found.Offset.End-found.Offset.Start == len(found.Text) && parserName != "" && parserVersion != "" {
			m.Offsets = &TextOffsets{Start: found.Offset.Start + m.Start, End: found.Offset.Start + m.End, ParserName: parserName, ParserVersion: parserVersion}
		}
		passageOffsets := TextOffsets{Start: found.Offset.Start, End: found.Offset.End, ParserName: parserName, ParserVersion: parserVersion}
		if ref.Offsets != nil && (m.Offsets == nil || *ref.Offsets != passageOffsets) {
			m.State = StateFailed
		}
	}
	return out
}

func normalizedText(text string) string {
	var b strings.Builder
	for _, unit := range normalizePassage(text) {
		b.WriteRune(unit.value)
	}
	return b.String()
}

func identifierMatches(record library.PaperRecord, id Identifier) bool {
	if strings.TrimSpace(id.Value) == "" {
		return false
	}
	value := strings.TrimSpace(id.Value)
	ids := record.Identifiers
	switch strings.ToLower(strings.TrimSpace(id.Scheme)) {
	case "local":
		return value == record.RecordID
	case "doi", "crossref":
		clean := func(s string) string {
			s = strings.ToLower(strings.TrimSpace(s))
			for _, prefix := range []string{"https://doi.org/", "http://doi.org/", "doi:"} {
				s = strings.TrimPrefix(s, prefix)
			}
			return strings.TrimSpace(s)
		}
		return clean(value) != "" && (clean(value) == clean(ids.DOI) || clean(value) == clean(ids.CrossrefID))
	case "arxiv":
		return strings.TrimPrefix(value, "arXiv:") != "" && strings.TrimPrefix(value, "arXiv:") == strings.TrimPrefix(ids.ArXivID, "arXiv:")
	case "pmid":
		return value == ids.PMID
	case "pmcid":
		return strings.TrimPrefix(strings.ToUpper(value), "PMC") != "" && strings.TrimPrefix(strings.ToUpper(value), "PMC") == strings.TrimPrefix(strings.ToUpper(ids.PMCID), "PMC")
	case "openalex":
		return strings.TrimPrefix(value, "https://openalex.org/") != "" && strings.TrimPrefix(value, "https://openalex.org/") == strings.TrimPrefix(ids.OpenAlexID, "https://openalex.org/")
	case "semantic-scholar":
		return value == ids.SemanticScholarID
	case "zotero":
		return value == ids.ZoteroItemKey
	case "ads":
		return value == ids.ADSBibcode
	}
	return false
}

func paperMatches(record library.PaperRecord, id string) bool {
	if id == "" {
		return false
	}
	if record.RecordID != "" && id == record.RecordID {
		return true
	}
	for _, scheme := range []string{"doi", "arxiv", "pmid", "pmcid", "openalex", "semantic-scholar", "zotero", "ads"} {
		if identifierMatches(record, Identifier{Scheme: scheme, Value: id}) {
			return true
		}
	}
	return false
}
