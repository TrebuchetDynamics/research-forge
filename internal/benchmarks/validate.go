package benchmarks

import (
	"fmt"
	"net/url"
	"os"
	"path/filepath"
	"regexp"
	"sort"
	"strconv"
	"strings"
	"time"
	"unicode"
)

var slugRE = regexp.MustCompile(`^[a-z][a-z0-9-]*$`)
var hashRE = regexp.MustCompile(`^[0-9a-f]{64}$`)
var arxivVersion = regexp.MustCompile(`v[0-9]+$`)
var europePMCCapRE = regexp.MustCompile(`^FIRST_PDATE:\[\* TO ([0-9]{4}-[0-9]{2}-[0-9]{2})\]$`)

func invalid(code string) error { return fmt.Errorf("%s", code) }

// Read through os.Root so even a symlinked body cannot escape the package.
func readSetFile(dir, path string) ([]byte, error) {
	if filepath.IsAbs(path) || strings.Contains(path, "..") {
		return nil, invalid("invalid_set")
	}
	root, e := os.OpenRoot(dir)
	if e != nil {
		return nil, e
	}
	defer root.Close()
	return root.ReadFile(path)
}
func normalizeTitle(s string) string {
	return strings.Join(strings.FieldsFunc(strings.ToLower(s), func(r rune) bool { return !unicode.IsLetter(r) && !unicode.IsDigit(r) }), " ")
}
func canonicalIdentifier(id Identifier) string {
	v := strings.TrimSpace(id.Value)
	switch id.Scheme {
	case "doi":
		v = strings.ToLower(v)
		for _, prefix := range []string{"https://doi.org/", "http://doi.org/", "doi:"} {
			v = strings.TrimPrefix(v, prefix)
		}
		if !strings.HasPrefix(v, "10.") || !strings.Contains(v, "/") {
			return ""
		}
	case "pmid":
		v = strings.TrimLeft(v, "0")
		if !regexp.MustCompile(`^[1-9][0-9]*$`).MatchString(v) {
			return ""
		}
	case "pmcid":
		v = strings.ToUpper(v)
		v = "PMC" + strings.TrimLeft(strings.TrimPrefix(v, "PMC"), "0")
		if !regexp.MustCompile(`^PMC[1-9][0-9]*$`).MatchString(v) {
			return ""
		}
	case "arxiv":
		for _, p := range []string{"https://arxiv.org/abs/", "http://arxiv.org/abs/", "arXiv:"} {
			v = strings.TrimPrefix(v, p)
		}
		v = arxivVersion.ReplaceAllString(v, "")
	case "openalex":
		v = strings.TrimPrefix(strings.TrimPrefix(v, "https://openalex.org/"), "http://openalex.org/")
		v = strings.ToUpper(v)
		if !regexp.MustCompile(`^W[0-9]+$`).MatchString(v) {
			return ""
		}
	case "semantic_scholar":
		v = strings.ToLower(v)
		if !regexp.MustCompile(`^[0-9a-f]{40}$`).MatchString(v) {
			return ""
		}
	case "ads_bibcode":
	case "url":
		u, e := url.Parse(v)
		if e != nil || strings.ToLower(u.Scheme) != "https" || u.Host == "" || u.User != nil {
			return ""
		}
		u.Scheme = "https"
		u.Host = strings.ToLower(u.Host)
		u.Fragment = ""
		if u.Port() == "443" {
			u.Host = u.Hostname()
		}
		v = u.String()
	default:
		return ""
	}
	if v == "" {
		return ""
	}
	return id.Scheme + ":" + v
}
func aliases(ids []Identifier) map[string]bool {
	m := map[string]bool{}
	for _, id := range ids {
		if s := canonicalIdentifier(id); s != "" {
			m[s] = true
		}
	}
	return m
}
func validateIdentifiers(ids []Identifier) bool {
	if len(ids) == 0 {
		return false
	}
	last := ""
	for _, id := range ids {
		s := canonicalIdentifier(id)
		if s == "" || s != id.Scheme+":"+id.Value || s <= last {
			return false
		}
		last = s
	}
	return true
}
func dateUpper(s string) (time.Time, error) {
	layout := "2006-01-02"
	switch len(s) {
	case 4:
		layout = "2006"
	case 7:
		layout = "2006-01"
	case 10:
	default:
		return time.Time{}, invalid("invalid_set")
	}
	d, e := time.Parse(layout, s)
	if e != nil || d.Year() < 1 {
		return time.Time{}, invalid("invalid_set")
	}
	if len(s) == 4 {
		d = d.AddDate(1, 0, -1)
	} else if len(s) == 7 {
		d = d.AddDate(0, 1, -1)
	}
	return d, nil
}
func utc(s string) bool {
	_, e := time.Parse(time.RFC3339, s)
	return e == nil && strings.HasSuffix(s, "Z")
}

// ValidateCandidateSet checks unapproved, snapshot-free pilot input without
// executing it or changing its lifecycle. allowAPFallback is an explicit curator
// opt-in for the owner's documented AP exception; it relaxes only AP presence.
// Callers must separately establish the source-list evidence for that fallback.
func ValidateCandidateSet(dir string, allowAPFallback bool) error {
	_, err := loadSetWithPolicy(dir, setValidationPolicy{candidate: true, allowAPFallback: allowAPFallback})
	return err
}

type setValidationPolicy struct {
	record, candidate, allowAPFallback bool
}

func loadSet(dir string, record bool) (Set, error) {
	return loadSetWithPolicy(dir, setValidationPolicy{record: record})
}

func loadSetWithPolicy(dir string, policy setValidationPolicy) (Set, error) {
	var s Set
	b, e := readSetFile(dir, "set.json")
	if e != nil {
		return s, invalid("invalid_set")
	}
	v, e := decodeStrict(b, &s)
	if e != nil {
		return s, invalid("invalid_set")
	}
	m := v.(map[string]any)
	delete(m, "set_checksum")
	c, e := canonical(m)
	if e != nil || digest(c) != s.SetChecksum {
		return s, invalid("checksum_mismatch")
	}
	sheet, e := readSetFile(dir, "review.md")
	if e != nil || digest(sheet) != s.ReviewSheetSHA256 {
		return s, invalid("checksum_mismatch")
	}
	if s.SchemaVersion != 1 || !slugRE.MatchString(s.SetID) || !hashRE.MatchString(s.SetChecksum) || !hashRE.MatchString(s.ReviewSheetSHA256) || len(s.K) == 0 || len(s.Questions) == 0 {
		return s, invalid("invalid_set")
	}
	if policy.candidate {
		if s.State != "candidate" || s.Approval != nil || s.SnapshotsSHA256 != nil {
			return s, invalid("invalid_set")
		}
	} else if (policy.record && (s.State != "approved" || s.SnapshotsSHA256 != nil)) || (!policy.record && (s.State != "locked" || s.SnapshotsSHA256 == nil)) || s.Approval == nil || strings.TrimSpace(s.Approval.Approver) == "" || !utc(s.Approval.ApprovedAt) {
		return s, invalid("invalid_set")
	}
	if s.SnapshotsSHA256 != nil && !hashRE.MatchString(*s.SnapshotsSHA256) {
		return s, invalid("invalid_set")
	}
	for i, k := range s.K {
		if k < 1 || (i > 0 && k <= s.K[i-1]) {
			return s, invalid("invalid_set")
		}
	}
	p := s.HTTPPolicy
	if p.Algorithm != "source-http-v1" || p.MaxRetries < 0 || p.TimeoutMS < 1 || p.RequestDelayMS < 0 || p.MaxRetryAfterMS < 1 || p.MaxResponseBytes < 1 || p.MaxRedirects < 0 {
		return s, invalid("invalid_set")
	}
	// All aliases are checked across splits, not just paper IDs.
	splitIDs := map[string]string{}
	reviewIDs := map[string]bool{}
	goldIDs := map[string]bool{}
	heldReviews := map[string]bool{}
	domains := map[string]bool{}
	held, dev := 0, 0
	last := ""
	for _, q := range s.Questions {
		if !slugRE.MatchString(q.QuestionID) || q.QuestionID <= last || !slugRE.MatchString(q.Domain) || strings.TrimSpace(q.Question) == "" || (q.Split != "dev" && q.Split != "held_out") || len(q.Gold) == 0 || len(q.SearchPlan) == 0 || q.Budgets.MaxHTTPRequests < 1 || q.Budgets.MaxWallClockMS < 1 {
			return s, invalid("invalid_set")
		}
		last = q.QuestionID
		r := q.Review
		cap, e := time.Parse("2006-01-02", r.SearchDate)
		// Month precision: the review reports only the month of its final search, so the
		// cap is conservatively that month's first day; gold published later in the month
		// then fails the gold date check below and must be dropped by the curator.
		precisionOK := r.SearchDatePrecision == "" || r.SearchDatePrecision == "day" || (r.SearchDatePrecision == "month" && e == nil && cap.Day() == 1)
		if e != nil || cap.Year() < 1 || !precisionOK || !validateIdentifiers(r.Identifiers) || strings.TrimSpace(r.Title) == "" || strings.TrimSpace(r.SearchDateLocator) == "" || strings.TrimSpace(r.IncludedListLocator) == "" || strings.TrimSpace(r.Rights) == "" {
			return s, invalid("invalid_set")
		}
		title := normalizeTitle(r.Title)
		if title == "" || strings.Contains(normalizeTitle(q.Question), title) {
			return s, invalid("reference_review_leak")
		}
		if q.Split == "held_out" {
			held++
			domains[q.Domain] = true
		} else {
			dev++
		}
		ids := append([]Identifier{}, r.Identifiers...)
		for _, id := range r.Identifiers {
			key := canonicalIdentifier(id)
			reviewIDs[key] = true
			if q.Split == "held_out" {
				if heldReviews[key] {
					return s, invalid("invalid_set")
				}
				heldReviews[key] = true
			}
		}
		prev := ""
		paperAliases := map[string]string{}
		for _, g := range q.Gold {
			d, e := dateUpper(g.PublicationDate)
			if e != nil || d.After(cap) || !slugRE.MatchString(g.PaperID) || g.PaperID <= prev || !validateIdentifiers(g.Identifiers) || strings.TrimSpace(g.IncludedListEntry) == "" || strings.TrimSpace(g.PublicationDateLocator) == "" {
				return s, invalid("invalid_set")
			}
			prev = g.PaperID
			for _, id := range g.Identifiers {
				key := canonicalIdentifier(id)
				if paperAliases[key] != "" {
					return s, invalid("invalid_set")
				}
				paperAliases[key] = g.PaperID
				goldIDs[key] = true
			}
			ids = append(ids, g.Identifiers...)
		}
		for _, id := range ids {
			if strings.Contains(strings.ToLower(q.Question), strings.ToLower(id.Value)) {
				return s, invalid("reference_review_leak")
			}
			key := canonicalIdentifier(id)
			if prior := splitIDs[key]; prior != "" && prior != q.Split {
				return s, invalid("invalid_set")
			}
			splitIDs[key] = q.Split
		}
		for _, call := range q.SearchPlan {
			if call.Limit < 1 || strings.TrimSpace(call.Terms) == "" || !slugRE.MatchString(call.Source) {
				return s, invalid("invalid_set")
			}
			inputs := []string{call.Terms, call.PageCursor}
			for _, v := range call.Filters {
				inputs = append(inputs, v)
			}
			for _, input := range inputs {
				if strings.Contains(normalizeTitle(input), title) {
					return s, invalid("reference_review_leak")
				}
				for _, id := range ids {
					if strings.Contains(strings.ToLower(input), strings.ToLower(id.Value)) {
						return s, invalid("reference_review_leak")
					}
				}
			}
			if !dateCapped(call, q.Review.SearchDate) {
				return s, invalid("date_cap_unenforceable")
			}
		}
	}
	for key := range goldIDs {
		if reviewIDs[key] {
			return s, invalid("invalid_set")
		}
	}
	waivedAP := false
	if s.Approval != nil {
		seen := map[string]bool{}
		for _, w := range s.Approval.Waivers {
			if w != WaiverAPDomainAbsent || seen[w] {
				return s, invalid("invalid_set")
			}
			seen[w] = true
		}
		waivedAP = seen[WaiverAPDomainAbsent]
	}
	if policy.candidate || !strings.HasPrefix(s.SetID, "fixture-") {
		if held < 8 || dev < 4 || len(domains) < 3 || (!domains["artificial-photosynthesis"] && !(policy.candidate && policy.allowAPFallback) && !waivedAP) {
			return s, invalid("invalid_set")
		}
	}
	return s, nil
}

// Only adapters that preserve a provider cap and raw date metadata qualify.
// Unknown adapters fail closed rather than silently executing uncapped queries.
func dateCapped(c PlanCall, cap string) bool {
	var prefix string
	switch c.Source {
	case "openalex":
		prefix = "to_publication_date:"
	case "crossref":
		prefix = "until-pub-date:"
	case "europepmc":
		match := europePMCCapRE.FindStringSubmatch(c.Filters["filter"])
		if match == nil {
			return false
		}
		upper, e := dateUpper(match[1])
		ce, _ := dateUpper(cap)
		return e == nil && !upper.After(ce)
	default:
		return false
	}
	found := false
	for _, v := range strings.Split(c.Filters["filter"], ",") {
		if strings.HasPrefix(v, prefix) {
			if found {
				return false
			}
			found = true
			d := strings.TrimPrefix(v, prefix)
			upper, e := dateUpper(d)
			ce, _ := dateUpper(cap)
			if e != nil || upper.After(ce) {
				return false
			}
		}
	}
	return found
}
func loadIndex(dir string, s Set) (SnapshotIndex, map[string][]byte, error) {
	var idx SnapshotIndex
	b, e := readSetFile(dir, "snapshots/index.json")
	if e != nil {
		return idx, nil, invalid("snapshot_mismatch")
	}
	v, e := decodeStrict(b, &idx)
	if e != nil {
		return idx, nil, invalid("snapshot_mismatch")
	}
	j, e := canonical(v)
	if e != nil || digest(j) != *s.SnapshotsSHA256 {
		return idx, nil, invalid("checksum_mismatch")
	}
	if idx.SchemaVersion != 1 || idx.SetID != s.SetID || !utc(idx.Capture.CapturedAt) || strings.TrimSpace(idx.Capture.RforgeVersion) == "" || strings.TrimSpace(idx.Capture.Revision) == "" || strings.TrimSpace(idx.Capture.Rights) == "" || len(idx.Questions) != len(s.Questions) {
		return idx, nil, invalid("snapshot_mismatch")
	}
	bodies := map[string][]byte{}
	fetch := func(hash string) error {
		if !hashRE.MatchString(hash) {
			return invalid("snapshot_mismatch")
		}
		if _, ok := bodies[hash]; ok {
			return nil
		}
		b, e := readSetFile(dir, "snapshots/bodies/"+hash+".bin")
		if e != nil || digest(b) != hash {
			return invalid("checksum_mismatch")
		}
		bodies[hash] = b
		return nil
	}
	for i, q := range idx.Questions {
		if q.QuestionID != s.Questions[i].QuestionID || !utc(q.StartedAt) || q.WallClockMS < 0 || len(q.Timing) == 0 || q.Timing[len(q.Timing)-1].Stage != "stop" || q.Timing[len(q.Timing)-1].ElapsedMS != q.WallClockMS {
			return idx, nil, invalid("snapshot_mismatch")
		}
		switch q.StopReason {
		case "completed", "budget_exhausted", "coverage_saturated", "interrupted", "error_threshold":
		default:
			return idx, nil, invalid("snapshot_mismatch")
		}
		elapsed := int64(0)
		for _, t := range q.Timing {
			if t.ElapsedMS < elapsed || t.ElapsedMS > q.WallClockMS {
				return idx, nil, invalid("snapshot_mismatch")
			}
			elapsed = t.ElapsedMS
		}
		var total int64
		for n, ex := range q.Exchanges {
			if ex.Sequence != n+1 || ex.PlanCall < 1 || ex.PlanCall > len(s.Questions[i].SearchPlan) || ex.ElapsedMS < 0 || ex.RetryAfterMS < 0 || ex.RequestKey != structuredHash(ex.Request) || (ex.Response == nil) == (ex.TransportError == nil) {
				return idx, nil, invalid("snapshot_mismatch")
			}
			if ex.Request.BodySHA256 != nil {
				if e := fetch(*ex.Request.BodySHA256); e != nil {
					return idx, nil, e
				}
			}
			if ex.Response != nil {
				if ex.Response.Status < 100 || ex.Response.Status > 599 {
					return idx, nil, invalid("snapshot_mismatch")
				}
				if e := fetch(ex.Response.BodySHA256); e != nil {
					return idx, nil, e
				}
			} else {
				switch *ex.TransportError {
				case "timeout", "connection_reset", "connection_refused":
				default:
					return idx, nil, invalid("snapshot_mismatch")
				}
			}
			total += ex.ElapsedMS + ex.RetryAfterMS
		}
		if total > q.WallClockMS {
			return idx, nil, invalid("snapshot_mismatch")
		}
		for id, entity := range q.Identities {
			if !hashRE.MatchString(id) || !slugRE.MatchString(entity) || strings.TrimSpace(q.IdentityEvidence[entity]) == "" {
				return idx, nil, invalid("snapshot_mismatch")
			}
			gold := false
			for _, g := range s.Questions[i].Gold {
				gold = gold || g.PaperID == entity
			}
			if !gold && !strings.HasPrefix(entity, "other-") {
				return idx, nil, invalid("snapshot_mismatch")
			}
		}
		if !sort.SliceIsSorted(q.DateExclusions, func(i, j int) bool { return q.DateExclusions[i].RecordID < q.DateExclusions[j].RecordID }) {
			return idx, nil, invalid("snapshot_mismatch")
		}
	}
	return idx, bodies, nil
}

func intString(n int) string { return strconv.Itoa(n) }
