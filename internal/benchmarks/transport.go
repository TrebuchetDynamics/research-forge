package benchmarks

import (
	"bytes"
	"context"
	"errors"
	"io"
	"net"
	"net/http"
	"net/url"
	"os"
	"regexp"
	"sort"
	"strings"
	"syscall"
	"time"
)

var errBudget = errors.New("budget_exhausted")
var secretKeys = map[string]bool{"api_key": true, "x-api-key": true, "key": true, "token": true, "email": true, "mailto": true}

func normalizedURL(s string) (string, error) {
	u, e := url.Parse(s)
	if e != nil || u.Host == "" || (u.Scheme != "https" && u.Scheme != "http") || u.User != nil {
		return "", invalid("snapshot_mismatch")
	}
	u.Scheme = strings.ToLower(u.Scheme)
	u.Host = strings.ToLower(u.Host)
	u.Fragment = ""
	if u.Port() == "443" && u.Scheme == "https" || u.Port() == "80" && u.Scheme == "http" {
		u.Host = u.Hostname()
	}
	values, e := url.ParseQuery(u.RawQuery)
	if e != nil {
		return "", e
	}
	keys := []string{}
	for k := range values {
		keys = append(keys, k)
	}
	sort.Strings(keys)
	pairs := []string{}
	esc := func(s string) string { return strings.ReplaceAll(url.QueryEscape(s), "+", "%20") }
	for _, k := range keys {
		vs := values[k]
		if secretKeys[strings.ToLower(k)] {
			for i := range vs {
				vs[i] = "REDACTED"
			}
		}
		sort.Strings(vs)
		for _, v := range vs {
			pairs = append(pairs, esc(k)+"="+esc(v))
		}
	}
	u.RawQuery = strings.Join(pairs, "&")
	return u.String(), nil
}
func cleanHeaders(h http.Header, response bool) map[string][]string {
	allow := map[string]bool{"accept": true, "content-type": true, "range": true}
	if response {
		allow = map[string]bool{"content-type": true, "retry-after": true, "link": true, "location": true}
	}
	out := map[string][]string{}
	for k, vs := range h {
		k = strings.ToLower(k)
		if !allow[k] {
			continue
		}
		for _, v := range vs {
			v = sanitizeText(strings.TrimSpace(v))
			if k == "location" {
				v = sanitizeLinkURL(v)
			}
			if k == "link" {
				v = linkTarget.ReplaceAllStringFunc(v, func(target string) string { return "<" + sanitizeLinkURL(target[1:len(target)-1]) + ">" })
			}
			out[k] = append(out[k], v)
		}
	}
	return out
}

var linkTarget = regexp.MustCompile(`<[^>]+>`)
var embeddedURL = regexp.MustCompile(`https?://[^\s<>"']+`)

func sanitizeText(value string) string {
	// APIs may echo bearer credentials as plain text, not just api_key URLs.
	for _, name := range []string{"RFORGE_OPENALEX_API_KEY", "RFORGE_SEMANTIC_SCHOLAR_API_KEY"} {
		if key := strings.TrimSpace(os.Getenv(name)); key != "" {
			for _, secret := range []string{url.QueryEscape(key), url.PathEscape(key), key} {
				value = strings.ReplaceAll(value, secret, "REDACTED")
			}
		}
	}
	return embeddedURL.ReplaceAllStringFunc(value, sanitizeLinkURL)
}

// Response URLs are aliases, not request keys. Preserve path/query bytes and
// ordering; only replace sensitive values. Sorting here would change identity.
func sanitizeLinkURL(value string) string {
	u, e := url.Parse(value)
	if e != nil {
		return "REDACTED"
	}
	u.User = nil
	pairs := strings.Split(u.RawQuery, "&")
	for i, pair := range pairs {
		key, _, _ := strings.Cut(pair, "=")
		decoded, e := url.QueryUnescape(key)
		if e != nil {
			return "REDACTED"
		}
		if secretKeys[strings.ToLower(decoded)] {
			pairs[i] = key + "=REDACTED"
		}
	}
	u.RawQuery = strings.Join(pairs, "&")
	return u.String()
}

type session struct {
	record        bool
	q             Question
	snap          SnapshotQuestion
	policy        HTTPPolicy
	bodies        map[string][]byte
	start         time.Time
	elapsed       int64
	tickIndex     int
	exchangeIndex int
	call          int
	requests      int
	calls         int
	stopped       bool
	failure       error
	live          http.RoundTripper
	exclusions    []DateExclusion
	attemptStart  int64
	newAttempt    bool
	backoffs      map[int]int64
}

func intPtr(i int) *int { return &i }
func (s *session) fail(e error) error {
	if s.failure == nil {
		s.failure = e
	}
	return e
}
func (s *session) tick(stage string, seq int) error {
	t := Timing{Stage: stage}
	switch stage {
	case "before_call", "after_call", "before_http", "after_http", "before_wait", "after_wait":
		t.PlanCall = intPtr(s.call)
	}
	if seq > 0 {
		t.Sequence = intPtr(seq)
	}
	if s.record {
		s.elapsed = time.Since(s.start).Milliseconds()
		t.ElapsedMS = s.elapsed
		s.snap.Timing = append(s.snap.Timing, t)
		return nil
	}
	if s.tickIndex >= len(s.snap.Timing) {
		return s.fail(invalid("snapshot_mismatch"))
	}
	want := s.snap.Timing[s.tickIndex]
	s.tickIndex++
	eq := func(a, b *int) bool { return a == nil && b == nil || a != nil && b != nil && *a == *b }
	if t.Stage != want.Stage || !eq(t.PlanCall, want.PlanCall) || !eq(t.Sequence, want.Sequence) || want.ElapsedMS < s.elapsed {
		return s.fail(invalid("snapshot_mismatch"))
	}
	s.elapsed = want.ElapsedMS
	return nil
}
func (s *session) budget(ctx context.Context, request bool) error {
	if ctx.Err() != nil {
		s.stopped = true
		s.snap.StopReason = "interrupted"
		return ctx.Err()
	}
	if s.elapsed >= s.q.Budgets.MaxWallClockMS || request && s.requests >= s.q.Budgets.MaxHTTPRequests {
		s.stopped = true
		return errBudget
	}
	return nil
}
func (s *session) now() time.Time { return s.start.Add(time.Duration(s.elapsed) * time.Millisecond) }
func (s *session) wait(ctx context.Context, d time.Duration, retry bool) error {
	if !retry {
		s.newAttempt = true
	}
	if d <= 0 {
		return ctx.Err()
	}
	if e := s.tick("before_wait", 0); e != nil {
		return e
	}
	if e := s.budget(ctx, true); e != nil {
		return e
	}
	before := s.elapsed
	remaining := s.q.Budgets.MaxWallClockMS - before
	ms := d.Milliseconds()
	actual := ms
	if actual > remaining {
		actual = remaining
	}
	if retry && s.exchangeIndex > 0 {
		if s.backoffs == nil {
			s.backoffs = map[int]int64{}
		}
		s.backoffs[s.exchangeIndex] = actual
		if s.record {
			s.snap.Exchanges[s.exchangeIndex-1].RetryAfterMS = actual
		} else if s.snap.Exchanges[s.exchangeIndex-1].RetryAfterMS != actual {
			return s.fail(invalid("snapshot_mismatch"))
		}
	}
	if s.record {
		timer := time.NewTimer(time.Duration(actual) * time.Millisecond)
		defer timer.Stop()
		select {
		case <-ctx.Done():
			s.stopped = true
			s.snap.StopReason = "interrupted"
			return ctx.Err()
		case <-timer.C:
		}
	}
	if ms >= remaining {
		s.elapsed = before + actual
		s.stopped = true
		return errBudget
	}
	if e := s.tick("after_wait", 0); e != nil {
		return e
	}
	if s.elapsed-before < ms {
		return s.fail(invalid("snapshot_mismatch"))
	}
	return nil
}
func (s *session) RoundTrip(r *http.Request) (*http.Response, error) {
	sequence := s.exchangeIndex + 1
	if e := s.tick("before_http", sequence); e != nil {
		return nil, e
	}
	if e := s.budget(r.Context(), true); e != nil {
		return nil, e
	}
	before := s.elapsed
	if s.newAttempt {
		s.attemptStart = before
		s.newAttempt = false
	}
	s.requests++
	u, e := normalizedURL(sanitizeText(r.URL.String()))
	if e != nil {
		return nil, s.fail(e)
	}
	req := Request{Method: strings.ToUpper(r.Method), URL: u, Headers: cleanHeaders(r.Header, false)}
	if r.Body != nil {
		b, e := io.ReadAll(r.Body)
		if e != nil {
			return nil, s.fail(e)
		}
		r.Body = io.NopCloser(bytes.NewReader(b))
		if len(b) > 0 {
			safe, e := sanitizeJSON(b)
			if e != nil {
				return nil, s.fail(e)
			}
			h := digest(safe)
			req.BodySHA256 = &h
			if s.record {
				s.bodies[h] = safe
			}
		}
	}
	key := structuredHash(req)
	var ex Exchange
	var liveHeaders http.Header
	if !s.record {
		if s.exchangeIndex >= len(s.snap.Exchanges) {
			return nil, s.fail(invalid("snapshot_mismatch"))
		}
		ex = s.snap.Exchanges[s.exchangeIndex]
		if ex.Sequence != sequence || ex.PlanCall != s.call || ex.RequestKey != key {
			return nil, s.fail(invalid("snapshot_mismatch"))
		}
	} else {
		ex = Exchange{Sequence: sequence, PlanCall: s.call, Request: req, RequestKey: key, Redactions: []string{}}
		remaining := time.Duration(s.q.Budgets.MaxWallClockMS-before) * time.Millisecond
		ctx, cancel := context.WithTimeout(r.Context(), remaining)
		defer cancel()
		started := time.Now()
		resp, err := s.live.RoundTrip(r.Clone(ctx))
		if err != nil {
			code := "connection_reset"
			var ne net.Error
			if errors.As(err, &ne) && ne.Timeout() {
				code = "timeout"
			} else if errors.Is(err, syscall.ECONNREFUSED) {
				code = "connection_refused"
			}
			ex.TransportError = &code
		} else {
			liveHeaders = resp.Header.Clone()
			b, err := io.ReadAll(io.LimitReader(resp.Body, s.policy.MaxResponseBytes+1))
			_ = resp.Body.Close()
			if err != nil {
				return nil, s.fail(invalid("snapshot_mismatch"))
			}
			if int64(len(b)) > s.policy.MaxResponseBytes {
				return nil, s.fail(invalid("snapshot_mismatch"))
			}
			safe, err := projectBody(b)
			if err != nil {
				return nil, s.fail(err)
			}
			h := digest(safe)
			s.bodies[h] = safe
			ex.Response = &Response{Status: resp.StatusCode, Headers: cleanHeaders(resp.Header, true), BodySHA256: h}
			ex.Redactions = append(ex.Redactions, "Metadata projection: identifiers, titles, public dates, pagination and API errors only; credential fields redacted.")
		}
		ex.ElapsedMS = time.Since(started).Milliseconds()
		s.snap.Exchanges = append(s.snap.Exchanges, ex)
	}
	s.exchangeIndex++
	if e := s.tick("after_http", sequence); e != nil {
		return nil, e
	}
	if !s.record && (s.elapsed-before != ex.ElapsedMS || ((s.elapsed-s.attemptStart > s.policy.TimeoutMS || s.elapsed > s.q.Budgets.MaxWallClockMS) && (ex.TransportError == nil || *ex.TransportError != "timeout"))) {
		return nil, s.fail(invalid("snapshot_mismatch"))
	}
	if s.record {
		s.snap.Exchanges[sequence-1].ElapsedMS = s.elapsed - before
	}
	if ex.TransportError != nil {
		switch *ex.TransportError {
		case "timeout":
			return nil, context.DeadlineExceeded
		case "connection_refused":
			return nil, syscall.ECONNREFUSED
		default:
			return nil, syscall.ECONNRESET
		}
	}
	headers := http.Header(headerTitle(ex.Response.Headers))
	if s.record {
		headers = liveHeaders
	}
	return &http.Response{StatusCode: ex.Response.Status, Header: headers, Body: io.NopCloser(bytes.NewReader(s.bodies[ex.Response.BodySHA256])), ContentLength: int64(len(s.bodies[ex.Response.BodySHA256])), Request: r}, nil
}
func headerTitle(h map[string][]string) map[string][]string {
	out := map[string][]string{}
	for k, v := range h {
		out[http.CanonicalHeaderKey(k)] = v
	}
	return out
}

func sanitizeJSON(b []byte) ([]byte, error) {
	v, e := parseJSON(b)
	if e != nil {
		return nil, invalid("snapshot_mismatch")
	}
	var walk func(any)
	walk = func(x any) {
		switch m := x.(type) {
		case map[string]any:
			for k, v := range m {
				if secretKeys[strings.ToLower(k)] {
					m[k] = "REDACTED"
				} else {
					walk(v)
				}
			}
		case []any:
			for _, v := range m {
				walk(v)
			}
		}
	}
	walk(v)
	return canonical(v)
}

// Capture only the legal metadata consumed by the supported real adapters.
// Dropping abstracts is a projection, never a synthetic successful response.
func projectBody(b []byte) ([]byte, error) {
	v, e := parseJSON(b)
	if e != nil {
		// Redirect/error bodies are not consumed by these JSON adapters. Omit
		// unlicensed HTML/text, preserving status and headers through their real
		// retry/redirect path. A 2xx non-JSON body remains a visible parse error.
		return []byte{}, nil
	}
	allowed := map[string]bool{"results": true, "meta": true, "next_cursor": true, "id": true, "doi": true, "title": true, "publication_year": true, "publication_date": true, "ids": true, "pmid": true, "pmcid": true, "primary_location": true, "landing_page_url": true, "license": true, "open_access": true, "is_oa": true, "oa_status": true, "type": true, "message": true, "items": true, "DOI": true, "published-print": true, "published-online": true, "issued": true, "date-parts": true, "URL": true, "container-title": true, "publisher": true, "next-cursor": true, "status": true, "error": true,
		"resultList": true, "result": true, "nextCursorMark": true, "pubYear": true,
		"firstPublicationDate": true, "electronicPublicationDate": true, "journalInfo": true, "printPublicationDate": true,
		"data": true, "next": true, "offset": true, "total": true, "paperId": true, "year": true, "publicationDate": true,
		"externalIds": true, "ArXiv": true, "PubMed": true, "venue": true, "url": true, "isOpenAccess": true, "openAccessPdf": true,
	}
	var walk func(any) any
	walk = func(x any) any {
		switch m := x.(type) {
		case map[string]any:
			out := map[string]any{}
			for k, v := range m {
				if secretKeys[strings.ToLower(k)] {
					continue
				}
				if allowed[k] {
					out[k] = walk(v)
				}
			}
			return out
		case []any:
			out := []any{}
			for _, v := range m {
				out = append(out, walk(v))
			}
			return out
		case string:
			return sanitizeText(m)
		default:
			return x
		}
	}
	return canonical(walk(v))
}
