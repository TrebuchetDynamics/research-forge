package benchmarks

import (
	"context"
	"fmt"
	"net/http"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"time"

	"github.com/TrebuchetDynamics/research-forge/internal/filetxn"
	"github.com/TrebuchetDynamics/research-forge/internal/library"
	"github.com/TrebuchetDynamics/research-forge/internal/search"
	"github.com/TrebuchetDynamics/research-forge/internal/sources"
)

// ProtocolSHA256 is the exact-byte digest of docs/benchmark/spec.md v1.
// Changes to that contract require regenerating this fingerprint.
const ProtocolSHA256 = "a14f0e137a310243c19461c3a8d4e5af60505bd58ed334afd8f7d288053b6020"

type EngineOptions struct {
	SetDir, Split, Version, Revision string
	Record                           bool
	// Optional live gateway for embedding applications; ignored in replay.
	RecordTransport http.RoundTripper
}
type EngineResult struct {
	Report []byte
	Audit  []byte
}
type QuestionAudit struct {
	QuestionID string                 `json:"question_id"`
	Raw        []sources.SourceRecord `json:"raw"`
	Clusters   [][]string             `json:"clusters"`
}

// RunEngine runs only locked plans through actual connectors and the production
// batch deduper. Replay has no live transport, source cache, workspace or LLM.
func RunEngine(ctx context.Context, o EngineOptions) (EngineResult, error) {
	diagnostic := func(code string) (EngineResult, error) {
		b, _ := CanonicalJSON(map[string]string{"error": code})
		return EngineResult{Report: b}, invalid(code)
	}
	if o.Split == "" {
		o.Split = "held_out"
	}
	if o.Split != "dev" && o.Split != "held_out" {
		return diagnostic("invalid_set")
	}
	if o.Version == "" {
		o.Version = "dev"
	}
	if o.Revision == "" {
		o.Revision = "unknown"
	}
	s, e := loadSet(o.SetDir, o.Record)
	if e != nil {
		return diagnostic(e.Error())
	}
	var idx SnapshotIndex
	bodies := map[string][]byte{}
	if !o.Record {
		idx, bodies, e = loadIndex(o.SetDir, s)
		if e != nil {
			return diagnostic(e.Error())
		}
	} else {
		idx = SnapshotIndex{SchemaVersion: 1, SetID: s.SetID, Capture: Capture{CapturedAt: time.Now().UTC().Format(time.RFC3339Nano), RforgeVersion: o.Version, Revision: o.Revision, Rights: "Metadata-only capture; redistribution rights and attribution: see checksum-locked review.md. Abstracts and full text omitted."}, Questions: []SnapshotQuestion{}}
	}
	report := EngineReport{SchemaVersion: 1, Purpose: "evaluation", Tier: "engine", Mode: "replay", Status: "complete", SetID: s.SetID, SetChecksum: s.SetChecksum, SnapshotsSHA256: s.SnapshotsSHA256, Split: o.Split, RforgeVersion: o.Version, Revision: o.Revision, ProtocolSHA256: ProtocolSHA256, K: s.K, Runs: []Run{}, Paired: []any{}, Errors: []RunError{}}
	if strings.HasPrefix(s.SetID, "fixture-") {
		report.Purpose = "fixture"
	}
	if o.Record {
		report.Mode = "record"
	}
	audit := []QuestionAudit{}
	reviews := []Review{}
	for _, q := range s.Questions {
		reviews = append(reviews, q.Review)
	}
	for i, q := range s.Questions {
		if !o.Record && q.Split != o.Split {
			continue
		}
		sess := &session{record: o.Record, q: q, policy: s.HTTPPolicy, bodies: bodies, exclusions: []DateExclusion{}}
		if o.Record {
			sess.start = time.Now()
			sess.snap = SnapshotQuestion{QuestionID: q.QuestionID, StartedAt: sess.start.UTC().Format(time.RFC3339Nano), Timing: []Timing{}, StopReason: "completed", DateExclusions: []DateExclusion{}, Identities: map[string]string{}, IdentityEvidence: map[string]string{}, Exchanges: []Exchange{}}
			sess.live = o.RecordTransport
			if sess.live == nil {
				transport := &http.Transport{TLSHandshakeTimeout: 10 * time.Second, ResponseHeaderTimeout: time.Duration(s.HTTPPolicy.TimeoutMS) * time.Millisecond, DisableCompression: true}
				sess.live = transport
				defer transport.CloseIdleConnections()
			}
		} else {
			sess.snap = idx.Questions[i]
			sess.start, _ = time.Parse(time.RFC3339, sess.snap.StartedAt)
		}
		row, qa, codes := executeQuestion(ctx, sess, s.K, reviews)
		if o.Record {
			sess.snap.WallClockMS = sess.elapsed
			sess.snap.DateExclusions = sess.exclusions
			idx.Questions = append(idx.Questions, sess.snap)
		}
		for _, code := range codes {
			report.Errors = append(report.Errors, RunError{QuestionID: &q.QuestionID, Code: code})
			if code != "source_error" {
				report.Status = "invalid"
			}
		}
		if q.Split == o.Split {
			report.Runs = append(report.Runs, row)
			audit = append(audit, qa)
		}
	}
	if len(report.Runs) == 0 {
		return diagnostic("invalid_set")
	}
	if o.Record && report.Status == "complete" {
		idxHash := structuredHash(idx)
		s.SnapshotsSHA256 = &idxHash
		s.State = "locked"
		s.SetChecksum = ""
		b, _ := CanonicalJSON(s)
		v, _ := parseJSON(b)
		delete(v.(map[string]any), "set_checksum")
		c, _ := canonical(v)
		s.SetChecksum = digest(c)
		if e := publishCapture(o.SetDir, s, idx, bodies); e != nil {
			return diagnostic("snapshot_mismatch")
		}
		report.SetChecksum = s.SetChecksum
		report.SnapshotsSHA256 = s.SnapshotsSHA256
	}
	sort.Slice(report.Errors, func(i, j int) bool {
		a, b := report.Errors[i], report.Errors[j]
		if *a.QuestionID != *b.QuestionID {
			return *a.QuestionID < *b.QuestionID
		}
		return a.Code < b.Code
	})
	if report.Status == "invalid" {
		for i, r := range report.Runs {
			report.Runs[i] = invalidRow(r, s.K)
		}
		report.Aggregates = nil
	} else {
		report.Aggregates = aggregate(report.Runs, s.K)
	}
	b, e := CanonicalJSON(report)
	if e != nil {
		return diagnostic("protocol_violation")
	}
	a, _ := CanonicalJSON(audit)
	result := EngineResult{Report: b, Audit: a}
	if report.Status == "invalid" {
		return result, invalid("protocol_violation")
	}
	return result, nil
}

func executeQuestion(ctx context.Context, s *session, ks []int, reviews []Review) (Run, QuestionAudit, []string) {
	forbidden := func(raw sources.SourceRecord) bool {
		for _, review := range reviews {
			if leaks(raw, review) {
				return true
			}
		}
		return false
	}
	q := s.q
	r := Run{QuestionID: q.QuestionID, Domain: q.Domain, Condition: "engine", Repetition: 1, GoldCount: len(q.Gold), CitationStatus: "not_implemented", DedupeStatus: "measured", AtK: []AtK{}}
	qa := QuestionAudit{QuestionID: q.QuestionID, Raw: []sources.SourceRecord{}, Clusters: [][]string{}}
	codes := []string{}
	add := func(code string) {
		for _, v := range codes {
			if v == code {
				return
			}
		}
		codes = append(codes, code)
	}
	rawByID := map[string]sources.SourceRecord{}
	papers := []library.PaperRecord{}
	excluded := map[string]string{}
	for i, call := range q.SearchPlan {
		s.call = i + 1
		if e := s.tick("before_call", 0); e != nil {
			break
		}
		if e := s.budget(ctx, true); e != nil {
			break
		}
		if !s.record && s.tickIndex < len(s.snap.Timing) && s.snap.Timing[s.tickIndex].Stage == "after_dedupe" { // An explicit captured engine stop signal, never inferred from gold.
			if s.snap.StopReason != "coverage_saturated" && s.snap.StopReason != "interrupted" && s.snap.StopReason != "error_threshold" {
				s.fail(invalid("snapshot_mismatch"))
			}
			s.stopped = true
			break
		}
		opts := sources.HTTPClientOptions{MaxRetries: s.policy.MaxRetries, Timeout: time.Duration(s.policy.TimeoutMS) * time.Millisecond, RequestDelay: time.Duration(s.policy.RequestDelayMS) * time.Millisecond, MaxRetryAfter: time.Duration(s.policy.MaxRetryAfterMS) * time.Millisecond, MaxResponseBytes: s.policy.MaxResponseBytes, Transport: s, Now: s.now, Wait: s.wait, CheckRedirect: func(req *http.Request, via []*http.Request) error {
			if len(via) > s.policy.MaxRedirects {
				return fmt.Errorf("redirect limit")
			}
			return nil
		}}
		// Replay deadlines are enforced on the virtual clock, never by a real
		// timer whose expiry would depend on host scheduling/load.
		if !s.record {
			opts.Timeout = -1
		}
		var conn sources.SourceConnector
		switch call.Source {
		case "openalex":
			opts.BaseURL = "https://api.openalex.org"
			conn = sources.NewOpenAlexConnector(sources.NewHTTPClient(opts))
		case "crossref":
			opts.BaseURL = "https://api.crossref.org"
			conn = sources.NewCrossrefConnector(sources.NewHTTPClient(opts))
		default:
			s.fail(invalid("date_cap_unenforceable"))
		}
		if s.failure != nil {
			break
		}
		s.calls++
		response, e := conn.Search(ctx, sources.SourceQuery{Terms: call.Terms, Limit: call.Limit, PageCursor: call.PageCursor, Filters: call.Filters})
		if e != nil && !s.stopped && s.failure == nil {
			add("source_error")
		}
		for _, raw := range response.Records {
			qa.Raw = append(qa.Raw, raw)
			if forbidden(raw) {
				s.fail(invalid("reference_review_leak"))
				break
			}
			id := recordID(raw)
			if id == "" {
				s.fail(invalid("protocol_violation"))
				break
			}
			reason := dateReason(raw, q.Review.SearchDate)
			if reason != "" {
				excluded[id] = reason
				continue
			}
			if _, ok := rawByID[id]; ok {
				continue
			}
			rawByID[id] = raw
			normalized, _ := sources.PaperRecords(sources.SourceResponse{Records: []sources.SourceRecord{raw}, RawRef: response.RawRef})
			if len(normalized) != 1 {
				s.fail(invalid("protocol_violation"))
				break
			}
			// The production deduper preserves SourceRefs, so this private tag tracks
			// membership even when the merged PaperRecord drops a competing alias.
			normalized[0].SourceRefs[0].Metadata = map[string]string{"benchmark_record_id": id}
			papers = append(papers, normalized[0])
		}
		if s.failure != nil {
			break
		}
		if e := s.tick("after_call", 0); e != nil {
			break
		}
		if s.stopped {
			break
		}
	}
	merged := search.DedupePapers(papers)
	if s.failure == nil {
		_ = s.tick("after_dedupe", 0)
	}
	for _, p := range merged {
		raw := sources.SourceRecord{Title: p.Title, Identifiers: sources.Identifiers{DOI: p.Identifiers.DOI, OpenAlexID: p.Identifiers.OpenAlexID, ArXivID: p.Identifiers.ArXivID, PMID: p.Identifiers.PMID, PMCID: p.Identifiers.PMCID, SemanticScholarID: p.Identifiers.SemanticScholarID, ADSBibcode: p.Identifiers.ADSBibcode}, URLs: p.URLs}
		if forbidden(raw) {
			s.fail(invalid("reference_review_leak"))
		}
		cluster := []string{}
		for _, ref := range p.SourceRefs {
			if id := ref.Metadata["benchmark_record_id"]; id != "" {
				cluster = append(cluster, id)
			}
		}
		qa.Clusters = append(qa.Clusters, cluster)
	}
	if s.failure == nil {
		_ = s.tick("stop", 0)
	}
	for id, reason := range excluded {
		s.exclusions = append(s.exclusions, DateExclusion{RecordID: id, Reason: reason})
	}
	sort.Slice(s.exclusions, func(i, j int) bool { return s.exclusions[i].RecordID < s.exclusions[j].RecordID })
	reason := "completed"
	if s.stopped {
		reason = "budget_exhausted"
		if s.snap.StopReason == "interrupted" || s.snap.StopReason == "coverage_saturated" || s.snap.StopReason == "error_threshold" {
			reason = s.snap.StopReason
		}
	}
	if s.record {
		s.snap.StopReason = reason
	} else {
		for _, ex := range s.snap.Exchanges {
			if s.backoffs[ex.Sequence] != ex.RetryAfterMS {
				s.fail(invalid("snapshot_mismatch"))
			}
		}
		if reason != s.snap.StopReason || s.tickIndex != len(s.snap.Timing) || s.exchangeIndex != len(s.snap.Exchanges) || structuredHash(s.exclusions) != structuredHash(s.snap.DateExclusions) || s.elapsed != s.snap.WallClockMS {
			s.fail(invalid("snapshot_mismatch"))
		}
	}
	if s.failure != nil {
		add(s.failure.Error())
	}
	r.StopReason = reason
	r.Returned = count(len(merged))
	r.HTTPRequests = count(s.requests)
	r.SearchCalls = count(s.calls)
	r.ToolCalls = count(0)
	r.WallClockMS = &s.elapsed
	r.DateExcluded = count(len(s.exclusions))
	score(&r, q, ks, qa.Clusters, rawByID, s.snap.Identities)
	return r, qa, codes
}
func sourceAliases(r sources.SourceRecord) map[string]bool {
	ids := r.Identifiers
	out := []Identifier{{"doi", ids.DOI}, {"doi", ids.CrossrefID}, {"openalex", ids.OpenAlexID}, {"arxiv", ids.ArXivID}, {"pmid", ids.PMID}, {"pmcid", ids.PMCID}, {"semantic_scholar", ids.SemanticScholarID}, {"ads_bibcode", ids.ADSBibcode}}
	for _, u := range r.URLs {
		out = append(out, Identifier{"url", u})
	}
	return aliases(out)
}
func leaks(r sources.SourceRecord, review Review) bool {
	for key := range sourceAliases(r) {
		if aliases(review.Identifiers)[key] {
			return true
		}
	}
	return normalizeTitle(r.Title) != "" && normalizeTitle(r.Title) == normalizeTitle(review.Title)
}
func recordID(r sources.SourceRecord) string {
	if strings.TrimSpace(r.SourceID) != "" {
		return structuredHash([]any{r.Source, r.SourceID})
	}
	ids := sourceAliases(r)
	keys := []string{}
	for k := range ids {
		keys = append(keys, k)
	}
	sort.Strings(keys)
	a := []Identifier{}
	for _, k := range keys {
		scheme, v, _ := strings.Cut(k, ":")
		a = append(a, Identifier{scheme, v})
	}
	if len(a) == 0 {
		return ""
	}
	return structuredHash([]any{r.Source, a})
}
func dateReason(r sources.SourceRecord, cap string) string {
	values := r.Metadata["publication_dates"]
	if values == "" {
		values = r.Metadata["publication_date"]
	}
	if values == "" && r.Year > 0 {
		values = fmt.Sprintf("%04d", r.Year)
	}
	var earliest time.Time
	for _, v := range strings.Split(values, ";") {
		d, e := dateUpper(v)
		if e == nil && (earliest.IsZero() || d.Before(earliest)) {
			earliest = d
		}
	}
	if earliest.IsZero() {
		return "unknown_date"
	}
	c, _ := dateUpper(cap)
	if earliest.After(c) {
		return "after_cap"
	}
	return ""
}
func publishCapture(dir string, s Set, idx SnapshotIndex, bodies map[string][]byte) error {
	root, e := os.OpenRoot(dir)
	if e != nil {
		return e
	}
	defer root.Close()
	// mkdir is exclusive: do not overwrite an old/partial/locked snapshot bundle.
	if e = root.Mkdir("snapshots", 0700); e != nil {
		return e
	}
	ok := false
	defer func() {
		if !ok {
			_ = root.RemoveAll("snapshots")
		}
	}()
	if e = root.Mkdir("snapshots/bodies", 0700); e != nil {
		return e
	}
	for hash, b := range bodies {
		if e = root.WriteFile("snapshots/bodies/"+hash+".bin", b, 0600); e != nil {
			return e
		}
	}
	b, e := CanonicalJSON(idx)
	if e != nil {
		return e
	}
	if e = root.WriteFile("snapshots/index.json", b, 0600); e != nil {
		return e
	}
	// Manifest is the publication marker and is replaced only after the complete
	// bundle exists. Readers cannot execute an approved/incomplete capture.
	b, e = CanonicalJSON(s)
	if e != nil {
		return e
	}
	if e = filetxn.Replace(filepath.Join(dir, "set.json"), b, 0600); e != nil {
		return e
	}
	ok = true
	return nil
}
