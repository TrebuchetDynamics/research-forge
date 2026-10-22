"""Trusted, external Pi-pilot research gateway. No models, gold, or scoring here.

Native Search Hub uses native() as its injected fetch implementation. rforge
uses request() through its existing per-source endpoint environment overrides.
Only this process sees the review exclusions. Fail closed before returning bytes.
"""
import calendar
import datetime as dt
import hashlib
import html
from html.parser import HTMLParser
import ipaddress
import json
import re
import socket
import time
import urllib.error
import urllib.parse as urlparse
import urllib.request


class GatewayError(RuntimeError):
    pass


class NoRedirect(urllib.request.HTTPRedirectHandler):
    def redirect_request(self, req, fp, code, msg, headers, newurl):
        return None


def upper_date(value):
    """Conservatively close partial publication dates, never guess missing ones."""
    try:
        parts = str(value).split("T")[0].split("-")
        year = int(parts[0])
        month = int(parts[1]) if len(parts) > 1 else 12
        day = int(parts[2]) if len(parts) > 2 else calendar.monthrange(year, month)[1]
        return dt.date(year, month, day).isoformat()
    except (ValueError, TypeError, OverflowError):
        return None


def normalized(value):
    return re.sub(r"[^\w]+", " ", html.unescape(urlparse.unquote(str(value))).casefold()).strip()


class NativeResults(HTMLParser):
    def __init__(self):
        super().__init__(convert_charrefs=True)
        self.results = []
        self.current = None
        self.capture = None

    def handle_starttag(self, tag, attrs):
        attrs = dict(attrs)
        classes = attrs.get("class", "").split()
        if "result__a" in classes:
            target = attrs.get("href", "")
            if target.startswith("//"):
                target = "https:" + target
            target = urlparse.parse_qs(urlparse.urlsplit(target).query).get("uddg", [target])[0]
            self.current = {"url": target, "title": "", "snippet": ""}
            self.results.append(self.current)
            self.capture = "title"
        elif "result__snippet" in classes and self.current:
            self.capture = "snippet"

    def handle_data(self, data):
        if self.current is not None and self.capture:
            self.current[self.capture] += data

    def handle_endtag(self, tag):
        if tag == "a":
            self.capture = None


class PublishedDates(HTMLParser):
    def __init__(self, target):
        super().__init__(convert_charrefs=True)
        self.target = target.rstrip("/")
        self.dates = []
        self.script = False
        self.buffer = ""

    def handle_starttag(self, tag, attrs):
        a = dict(attrs)
        if tag == "meta" and a.get("name", a.get("property", "")).lower() in (
            "citation_publication_date", "citation_online_date", "dc.date.issued", "dc.date",
            "prism.publicationdate", "article:published_time", "datepublished",
        ):
            self.dates.append(a.get("content", "").replace("/", "-"))
        if tag == "script" and a.get("type") == "application/ld+json":
            self.script, self.buffer = True, ""

    def handle_data(self, data):
        if self.script:
            self.buffer += data

    def handle_endtag(self, tag):
        if tag != "script" or not self.script:
            return
        self.script = False
        try:
            def visit(obj):
                if isinstance(obj, dict):
                    kind = obj.get("@type", [])
                    kinds = [kind] if isinstance(kind, str) else kind
                    locator = obj.get("url", obj.get("mainEntityOfPage", ""))
                    if isinstance(locator, dict):
                        locator = locator.get("@id", "")
                    if isinstance(locator, str) and locator.rstrip("/") == self.target and any(k in ("ScholarlyArticle", "Article", "NewsArticle", "MedicalScholarlyArticle", "Report") for k in kinds) and isinstance(obj.get("datePublished"), str):
                        self.dates.append(obj["datePublished"])
                    for value in obj.values():
                        visit(value)
                elif isinstance(obj, list):
                    for value in obj:
                        visit(value)
            visit(json.loads(self.buffer))
        except (ValueError, TypeError):
            pass


class Gateway:
    BASES = {"crossref": "https://api.crossref.org", "semantic-scholar": "https://api.semanticscholar.org", "europepmc": "https://www.ebi.ac.uk/europepmc"}

    def __init__(self, date_cap, reviews, *, transport=None, max_http=100, max_search=20, wall_ms=600000, trace_dir=None):
        if upper_date(date_cap) != date_cap:
            raise GatewayError("date_cap_unenforceable")
        self.date_cap, self.reviews = date_cap, reviews
        self.transport = transport or self._network
        self.max_http, self.max_search, self.wall_ms = max_http, max_search, wall_ms
        self.start = time.monotonic()
        self.events, self.quarantine = [], set()
        self.http_requests, self.search_calls, self.date_excluded = 0, 0, 0
        self.reference_review_leak, self.budget_exhausted = False, False
        self.trace_dir = trace_dir
        self.opener = urllib.request.build_opener(NoRedirect)

    def _network(self, url):
        # No automatic redirects: each hop is checked and counted by _fetch.
        agent = "Mozilla/5.0 (compatible; pi-search-hub/1.0)" if urlparse.urlsplit(url).hostname == "html.duckduckgo.com" else "ResearchForge-Pi-Pilot/1.0"
        req = urllib.request.Request(url, headers={"User-Agent": agent, "Accept-Encoding": "identity"})
        try:
            response = self.opener.open(req, timeout=min(25, max(1, (self.wall_ms / 1000) - (time.monotonic() - self.start))))
        except urllib.error.HTTPError as error:
            response = error
        with response:
            body = response.read(10 * 1024 * 1024 + 1)
            if len(body) > 10 * 1024 * 1024:
                raise GatewayError("response_too_large")
            return response.status, {k.lower(): v for k, v in response.headers.items()}, body

    def check(self):
        if self.reference_review_leak:
            raise GatewayError("reference_review_leak")
        if self.http_requests >= self.max_http or (time.monotonic() - self.start) * 1000 >= self.wall_ms:
            self.budget_exhausted = True
            raise GatewayError("budget_exhausted")

    def leak(self, value):
        text = normalized(value)
        # Preserve structured source context for MED's native id, which is a
        # PMID even when the provider omits a separate pmid field.
        med_ids = set()
        try:
            def visit(obj):
                if isinstance(obj, dict):
                    if str(obj.get("source", "")).upper() == "MED" and obj.get("id") is not None:
                        med_ids.add(str(obj["id"]))
                    for item in obj.values():
                        visit(item)
                elif isinstance(obj, list):
                    for item in obj:
                        visit(item)
            decoded = json.loads(value)
            text += " " + normalized(json.dumps(decoded, ensure_ascii=False))
            visit(decoded)
        except (ValueError, TypeError):
            pass
        for review in self.reviews:
            if normalized(review["title"]) in text:
                self.reference_review_leak = True
            for ident in review["identifiers"]:
                scheme, raw = ident["scheme"], ident["value"]
                if scheme in ("doi", "arxiv", "pmcid", "url"):
                    marker = normalized(re.sub(r"^https?://(?:dx\.)?doi\.org/", "", raw, flags=re.I))
                    if marker and marker in text:
                        self.reference_review_leak = True
                elif scheme == "pmid":
                    if raw in med_ids or re.search(r'(?:pubmed ncbi nlm nih gov|pmid|article med)\s+' + re.escape(raw) + r'\b', text):
                        self.reference_review_leak = True
                elif scheme in ("semantic_scholar", "openalex", "ads_bibcode"):
                    alias = re.sub(r"^https?://(?:www\.)?(?:openalex\.org|semanticscholar\.org/paper)/", "", raw, flags=re.I)
                    marker = normalized(alias)
                    if marker and re.search(r'\b' + re.escape(marker) + r'\b', text):
                        self.reference_review_leak = True
        if self.reference_review_leak:
            raise GatewayError("reference_review_leak")

    def _fetch(self, provider, url, query, *, search=False, depth=0):
        self.leak(url)
        if provider in self.quarantine:
            self.events.append(dict(provider=provider, query=query, action="skipped", http_status=0, retry_after="", error="provider quarantined after HTTP 429", stop_reason="error_threshold"))
            raise GatewayError("provider_quarantined")
        self.check()
        if search:
            if self.search_calls >= self.max_search:
                self.budget_exhausted = True
                raise GatewayError("budget_exhausted")
            self.search_calls += 1
        self.http_requests += 1
        event = dict(provider=provider, query=query, action="failed", http_status=0, retry_after="", error="transport failure", stop_reason="error_threshold")
        self.events.append(event)
        try:
            status, headers, body = self.transport(url)
            event["http_status"] = status
            event["retry_after"] = re.sub(r"[^\x20-\x7e]", "", headers.get("retry-after", ""))[:200]
            if status == 429:
                self.quarantine.add(provider)
            if self.trace_dir:
                name = f"{self.http_requests:04d}"
                (self.trace_dir / (name + ".body")).write_bytes(body)
                (self.trace_dir / (name + ".json")).write_text(json.dumps({"url": url, "status": status, "body_sha256": hashlib.sha256(body).hexdigest(), "retry_after": event["retry_after"]}) + "\n")
            if status >= 400 or status < 200:
                event["error"] = f"HTTP {status}"
                raise GatewayError(event["error"])
            self.leak(body.decode("utf-8", errors="replace"))
            event.update(action="completed", error="", stop_reason="completed")
            if status in (301, 302, 303, 307, 308):
                if depth >= 5:
                    raise GatewayError("redirect_limit")
                target = urlparse.urljoin(url, headers.get("location", ""))
                self.leak(target)
                host = urlparse.urlsplit(target).hostname
                if host != urlparse.urlsplit(url).hostname:
                    raise GatewayError("cross_provider_redirect_not_qualified")
                return self._fetch(provider, target, query, depth=depth + 1)
            return body
        except Exception as error:
            event.update(action="failed", error=str(error) if isinstance(error, GatewayError) else type(error).__name__, stop_reason="error_threshold")
            raise GatewayError(event["error"]) from error

    def allowed(self, values):
        dates = [upper_date(value) for value in values]
        dates = [value for value in dates if value]
        keep = bool(dates) and min(dates) <= self.date_cap
        if not keep:
            self.date_excluded += 1
        return keep

    def request(self, provider, path):
        if provider not in self.BASES:
            raise GatewayError("unregistered_provider")
        parsed = urlparse.urlsplit(path)
        if parsed.scheme or parsed.netloc:
            raise GatewayError("absolute_source_path")
        params = urlparse.parse_qs(parsed.query)
        query = params.get("query", params.get("search_query", [path]))[0]
        if provider == "crossref":
            if parsed.path != "/works":
                raise GatewayError("unsupported_operation")
            filters = [f for f in params.get("filter", [""])[0].split(",") if f and not f.startswith("until-")]
            params["filter"] = [",".join(filters + ["until-pub-date:" + self.date_cap])]
        elif provider == "semantic-scholar":
            if parsed.path != "/graph/v1/paper/search":
                raise GatewayError("unsupported_operation")
            params["year"] = [":" + self.date_cap[:4]]
            fields = params.get("fields", [""])[0].split(",")
            params["fields"] = [",".join(dict.fromkeys(fields + ["publicationDate", "year"]))]
        elif provider == "europepmc":
            if parsed.path != "/webservices/rest/search":
                raise GatewayError("unsupported_operation")
            params["query"] = ["(" + query + ") AND FIRST_PDATE:[1000-01-01 TO " + self.date_cap + "]"]
        url = self.BASES[provider] + parsed.path + "?" + urlparse.urlencode(params, doseq=True)
        body = self._fetch(provider, url, query, search=True)
        event = self.events[-1]
        try:
            if provider == "crossref":
                payload = json.loads(body)
                def dates(record):
                    result = []
                    for key in ("published-online", "published-print", "issued"):
                        for parts in record.get(key, {}).get("date-parts", []):
                            result.append("-".join(str(v).zfill(4 if i == 0 else 2) for i, v in enumerate(parts)))
                    return result
                payload["message"]["items"] = [r for r in payload["message"]["items"] if self.allowed(dates(r))]
            elif provider == "semantic-scholar":
                payload = json.loads(body)
                payload["data"] = [r for r in payload["data"] if self.allowed([r.get("publicationDate") or r.get("year")])]
            elif provider == "europepmc":
                payload = json.loads(body)
                payload["resultList"]["result"] = [r for r in payload["resultList"]["result"] if self.allowed([r.get("firstPublicationDate"), r.get("electronicPublicationDate"), r.get("journalInfo", {}).get("printPublicationDate")])]
            return json.dumps(payload).encode()
        except Exception as error:
            event.update(action="failed", error="malformed provider response", stop_reason="error_threshold")
            raise GatewayError("malformed provider response") from error

    def native(self, url, *, limit=5):
        parsed = urlparse.urlsplit(url)
        if parsed.scheme != "https" or parsed.hostname != "html.duckduckgo.com" or parsed.path != "/html/":
            raise GatewayError("unqualified_native_endpoint")
        params = urlparse.parse_qs(parsed.query)
        query = params.get("q", [""])[0]
        params["df"] = ["1900-01-01.." + self.date_cap]
        body = self._fetch("duckduckgo", urlparse.urlunsplit(parsed._replace(query=urlparse.urlencode(params, doseq=True))), query, search=True)
        parser = NativeResults()
        parser.feed(body.decode("utf-8", errors="replace"))
        if not parser.results:
            self.events[-1].update(action="failed", error="native search returned no parseable results", stop_reason="error_threshold")
            raise GatewayError("native search returned no parseable results")
        # HTML markup can split a forbidden title in raw bytes. Check the
        # rendered native records again before any of them can reach the host.
        for result in parser.results:
            self.leak(json.dumps(result, ensure_ascii=False))
        output = []
        for result in parser.results[:max(1, min(20, int(limit)))]:
            if not self._native_date(result["url"]):
                continue
            # Preserve the native result's exact title, URL and snippet. Only
            # record exclusion is allowed; no bibliography relevance edits.
            output.append('<a class="result__a" href="' + html.escape(result["url"], quote=True) + '">' + html.escape(result["title"]) + '</a><a class="result__snippet">' + html.escape(result["snippet"]) + '</a>')
        return "\n".join(output).encode()

    def _native_date(self, url):
        parsed = urlparse.urlsplit(url)
        if parsed.scheme not in ("https", "http") or not parsed.hostname or parsed.username or parsed.password:
            self.date_excluded += 1
            return False
        # The gateway may enrich dates using public page metadata; it never uses
        # engine lookup, reference gold or guessed years from the snippet.
        if self.transport == self._network:
            try:
                addresses = socket.getaddrinfo(parsed.hostname, parsed.port or (443 if parsed.scheme == "https" else 80))
                if not addresses or any(not ipaddress.ip_address(a[4][0]).is_global for a in addresses):
                    raise GatewayError("nonpublic_native_url")
            except OSError:
                self.date_excluded += 1
                return False
        provider = re.sub(r"[^a-z0-9-]", "-", parsed.hostname.lower())
        try:
            body = self._fetch(provider, url, "publication metadata: " + url)
        except GatewayError as error:
            if str(error) in ("reference_review_leak", "budget_exhausted"):
                raise
            self.date_excluded += 1
            return False
        parser = PublishedDates(url)
        parser.feed(body.decode("utf-8", errors="replace"))
        return self.allowed(parser.dates)
