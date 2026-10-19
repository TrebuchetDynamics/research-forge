# Research-quality benchmark specification

Status: implementation contract, version 1. This specifies V-B3 (engine harness),
V-B4 (candidate gold sets), and the shared scoring contract for V-B6 (agent tier).
It does not assert that these runners or datasets already exist.

The **Research-quality benchmark** measures how well rforge, or an agent equipped
with rforge, finds and cites evidence against a checksum-locked **Held-out question
set**. It is distinct from **Scientific benchmarking meta-analysis**; use these
terms as defined in [CONTEXT.md](../../CONTEXT.md#research-quality-benchmark).

Start here: dataset builders use sections 2–3; harness builders use sections 4–7;
agent-runner builders additionally use section 8. MUST, MUST NOT and SHOULD are
normative requirements, recommendations being explicitly identified as such.

## 1. Authority, boundaries and tiers

Sources of requirements:

- [Accepted plan](../verifiable-engine-plan.md#verifiable-research-engine-plan),
  especially **Planner's calls** (two tiers and independent gold) and **Owner
  answers** (leakage controls and the paired agent pilot).
- [Task contracts](../../TODO.md#12-verifiable-research-engine-cycle): V-B3,
  V-B4, V-B4-lock, V-B5, V-B6, V-B6-run and V-D3. Dataset approval and live spend
  remain owner-gated; this specification does not authorize either.
- [Fixture policy](../fixture-policy.md) and
  [ADR 0009](../adr/0009-verification-facets-and-provenance-levels.md).

| Tier | Execution | What comparisons mean |
| --- | --- | --- |
| Engine | Default offline replay through real connectors, normalization and dedupe; CI-safe | Retrieval and dedupe deltas on identical recorded responses and query plans |
| Agent | Explicit opt-in live runs, Pi primary and Codex secondary | Within-host, same-model paired differences, with uncertainty over questions |

Neither tier runs an LLM inside rforge. Never present fixture/demo constants as a
measured baseline. V-B3 MUST remove the hardcoded default cross-tool numbers or
label their output `demo` and exclude it from Research-quality benchmark reports.
The current scaffold is [cross_tool.go](../../internal/benchmarks/cross_tool.go);
the existing connector seam is `SourceConnector.Search` / `SourceQuery` in
[sources.go](../../internal/sources/sources.go). This specification defines new
artifacts rather than changing those interfaces in documentation.

## 2. Question-set package and schema

### 2.1 Layout and lifecycle

A set is a directory, not one directory per question:

```text
benchmarks/sets/candidates/<set_id>/set.json   # V-B4: metadata only
benchmarks/sets/candidates/<set_id>/review.md
benchmarks/sets/<set_id>/set.json              # V-B4-lock: approved then locked
benchmarks/sets/<set_id>/review.md
benchmarks/sets/<set_id>/snapshots/index.json
benchmarks/sets/<set_id>/snapshots/bodies/<sha256>.bin
```

`set.json` is the complete input manifest. `review.md` records human rationale,
source-list locators, rights, unresolved concerns and leakage risk per question;
it is not an agent input. The manifest locks its bytes via `review_sheet_sha256`.
A published review's included **paper records**, not effect-size rows, are the
gold unit. Distinct papers from one study remain distinct unless the source list
explicitly identifies them as the same paper. Multiple database identifiers for
one paper belong to one `paper_id`; do not count them as separate hits.

Lifecycle:

1. V-B4 creates `candidate` packages with `approval: null`,
   `snapshots_sha256: null` and a computed `set_checksum`. Do not record HTTP or
   choose approval identities. Draft at least 8 `held_out` and 4 separate `dev`
   questions spanning at least 3 domains, including `artificial-photosynthesis`;
   the held-out subset itself MUST include those 3 domains. Every question uses
   the included-paper list of a published systematic review. Synthetic artificial
   photosynthesis data is permitted only in a separately named harness fixture,
   never as a substitute for the pilot's independent gold.
2. The owner resolves the review sheet, approves the set and split, and moves it
   out of `candidates/`. State becomes `approved`, with named approver/date and a
   recomputed checksum; snapshots are still null. Approval authorizes recording,
   not live agent spending.
3. V-B4-lock invokes opt-in `--record` on that approved package. A successful
   capture writes the snapshot bundle, sets its digest, changes state to `locked`
   and recomputes `set_checksum`. On failure, leave the approved manifest intact;
   discard the incomplete bundle. Never overwrite a locked set in place.
4. Any change to a locked input (including gold, split, queries, rights, labels,
   snapshots or scoring settings) requires a new `set_id`, review and approval,
   and a new checksum/baseline. Old artifacts remain addressable. A schema change
   additionally increments `schema_version`.

Set IDs and question IDs are opaque, stable slugs without review titles, dates,
DOIs or relevance hints. Question IDs are unique within the set; paper IDs are
unique within their question. Each list of gold papers is nonempty and complete:
never silently drop an included paper because an identifier is difficult to find.
Resolve such identifiers before submitting a conforming candidate, or describe
an explicitly nonconforming draft in the review sheet.

### 2.2 Encoding and checksums

All structured artifacts are UTF-8 JSON without duplicate keys or non-finite
numbers. Unknown object fields are rejected, except inside the explicitly open
`filters`, `headers` and label maps. Dates are real Gregorian `YYYY-MM-DD` dates;
timestamps are UTC RFC 3339 with `Z`. Relative paths MUST remain inside the set,
without absolute paths, `..`, or symlink escapes.

`sha256` means 64 lowercase hexadecimal characters. Structured hashes use SHA-256
of [RFC 8785 JSON Canonicalization Scheme](https://www.rfc-editor.org/rfc/rfc8785)
(JCS) bytes without a newline. `set_checksum` hashes all of `set.json` with only
the root `set_checksum` member removed. It therefore covers questions, gold,
review identifiers, dates, splits, query plans, budgets, review-sheet bytes via
its hash, and the snapshot digest. Snapshot hashes are defined in section 4;
the snapshot index MUST NOT contain `set_checksum` (no checksum cycle).
File/body hashes use exact bytes, not canonical JSON.

Canonical stored JSON and reports use JCS plus one final LF. The checksum excludes
that LF. Reject digest mismatches before executing or scoring. Reformatting a
manifest alone does not change its semantic checksum; changing any value does.

### 2.3 Manifest JSON Schema

This embedded JSON Schema (2020-12) is normative, together with the semantic rules
below it. It is embedded to keep the v1 contract in one document; V-B3 may
materialize it as a validator. Examples and schemas here are specifications, not
committed candidate data or executable schema migrations.

```json
{
  "$schema": "https://json-schema.org/draft/2020-12/schema",
  "title": "Research-quality benchmark question set v1",
  "type": "object",
  "additionalProperties": false,
  "required": ["schema_version", "set_id", "state", "approval", "review_sheet_sha256", "snapshots_sha256", "set_checksum", "k", "http_policy", "questions"],
  "properties": {
    "schema_version": {"const": 1},
    "set_id": {"$ref": "#/$defs/slug"},
    "state": {"enum": ["candidate", "approved", "locked"]},
    "approval": {"oneOf": [{"type": "null"}, {"$ref": "#/$defs/approval"}]},
    "review_sheet_sha256": {"$ref": "#/$defs/sha256"},
    "snapshots_sha256": {"oneOf": [{"type": "null"}, {"$ref": "#/$defs/sha256"}]},
    "set_checksum": {"$ref": "#/$defs/sha256"},
    "k": {"type": "array", "minItems": 1, "uniqueItems": true, "items": {"type": "integer", "minimum": 1}},
    "http_policy": {"$ref": "#/$defs/http_policy"},
    "questions": {"type": "array", "minItems": 1, "items": {"$ref": "#/$defs/question"}}
  },
  "$defs": {
    "slug": {"type": "string", "pattern": "^[a-z][a-z0-9-]*$"},
    "text": {"type": "string", "minLength": 1},
    "sha256": {"type": "string", "pattern": "^[0-9a-f]{64}$"},
    "identifiers": {
      "type": "array", "minItems": 1, "uniqueItems": true,
      "items": {
        "type": "object", "additionalProperties": false,
        "required": ["scheme", "value"],
        "properties": {
          "scheme": {"enum": ["doi", "pmid", "pmcid", "arxiv", "openalex", "semantic_scholar", "ads_bibcode", "url"]},
          "value": {"$ref": "#/$defs/text"}
        }
      }
    },
    "approval": {
      "type": "object", "additionalProperties": false,
      "required": ["approver", "approved_at"],
      "properties": {
        "approver": {"$ref": "#/$defs/text"},
        "approved_at": {"type": "string", "format": "date-time"}
      }
    },
    "review": {
      "type": "object", "additionalProperties": false,
      "required": ["identifiers", "title", "search_date", "search_date_locator", "included_list_locator", "rights"],
      "properties": {
        "identifiers": {"$ref": "#/$defs/identifiers"},
        "title": {"$ref": "#/$defs/text"},
        "search_date": {"type": "string", "format": "date"},
        "search_date_locator": {"$ref": "#/$defs/text"},
        "included_list_locator": {"$ref": "#/$defs/text"},
        "rights": {"$ref": "#/$defs/text"}
      }
    },
    "paper": {
      "type": "object", "additionalProperties": false,
      "required": ["paper_id", "identifiers", "included_list_entry", "publication_date", "publication_date_locator"],
      "properties": {
        "paper_id": {"$ref": "#/$defs/slug"},
        "identifiers": {"$ref": "#/$defs/identifiers"},
        "included_list_entry": {"$ref": "#/$defs/text"},
        "publication_date": {"type": "string", "pattern": "^[0-9]{4}(-[0-9]{2}(-[0-9]{2})?)?$"},
        "publication_date_locator": {"$ref": "#/$defs/text"}
      }
    },
    "http_policy": {
      "type": "object", "additionalProperties": false,
      "required": ["algorithm", "max_retries", "timeout_ms", "request_delay_ms", "max_retry_after_ms", "max_response_bytes", "max_redirects"],
      "properties": {
        "algorithm": {"const": "source-http-v1"},
        "max_retries": {"type": "integer", "minimum": 0},
        "timeout_ms": {"type": "integer", "minimum": 1},
        "request_delay_ms": {"type": "integer", "minimum": 0},
        "max_retry_after_ms": {"type": "integer", "minimum": 1},
        "max_response_bytes": {"type": "integer", "minimum": 1},
        "max_redirects": {"type": "integer", "minimum": 0}
      }
    },
    "call": {
      "type": "object", "additionalProperties": false,
      "required": ["source", "terms", "limit", "page_cursor", "filters"],
      "properties": {
        "source": {"$ref": "#/$defs/slug"},
        "terms": {"$ref": "#/$defs/text"},
        "limit": {"type": "integer", "minimum": 1},
        "page_cursor": {"type": "string"},
        "filters": {"type": "object", "additionalProperties": {"type": "string"}}
      }
    },
    "question": {
      "type": "object", "additionalProperties": false,
      "required": ["question_id", "question", "domain", "split", "gold_source_review", "gold", "search_plan", "budgets"],
      "properties": {
        "question_id": {"$ref": "#/$defs/slug"},
        "question": {"$ref": "#/$defs/text"},
        "domain": {"$ref": "#/$defs/slug"},
        "split": {"enum": ["dev", "held_out"]},
        "gold_source_review": {"$ref": "#/$defs/review"},
        "gold": {"type": "array", "minItems": 1, "items": {"$ref": "#/$defs/paper"}},
        "search_plan": {"type": "array", "minItems": 1, "items": {"$ref": "#/$defs/call"}},
        "budgets": {
          "type": "object", "additionalProperties": false,
          "required": ["max_http_requests", "max_wall_clock_ms"],
          "properties": {
            "max_http_requests": {"type": "integer", "minimum": 1},
            "max_wall_clock_ms": {"type": "integer", "minimum": 1}
          }
        }
      }
    }
  }
}
```

Additional validation, mandatory even if the schema library ignores `format`:

- `candidate`: approval and snapshot digest null; `approved`: approval non-null,
  snapshot digest null; `locked`: both non-null. Whitespace-only text is invalid.
- `k` is ascending, default authoring choice `[10, 20, 50]`; never silently alter
  it during scoring. Questions are stored in question-ID order, gold in paper-ID
  order and identifier lists in `(scheme, value)` order. Search-plan order is
  meaningful. Call `filters` MUST specify the provider's publication-date cap;
  section 3 defines the independent date guard if the provider cap is coarse.
- Each gold paper's `publication_date` records its earliest documented public
  publication date, at the available precision (year, month or day), and
  `publication_date_locator` cites the metadata source and date field/section.
  Validate real date components and the section 3 upper-bound rule against the
  review's cap offline, including gold papers absent from snapshots. Do not use
  gold dates to rescue retrieved records whose own date metadata is missing.
- `search_date` is the review's last completed literature-search date, not its
  publication date. With several databases/dates, use the latest final search
  date and document the others in `review.md`. Month-only/year-only/absent dates
  are not conforming; do not invent a day. Locators name a public URL/identifier
  and section/table/page allowing a curator to verify the date and included list.
- Identifier aliases cannot map to two different gold papers or to both gold and
  a reference review. Collect known aliases of the review, including alternate
  editions, preprints and updates, in its identifier list. The review sheet
  explains alias resolution and the boundary of the review family.
- Store identifier values canonically: DOI lowercase without resolver/`doi:`
  prefix; PMID decimal without leading zeros; PMCID uppercase `PMC` plus decimal;
  arXiv identifier without resolver/`arXiv:` or terminal `vN`; OpenAlex uppercase
  `W` plus digits without resolver; Semantic Scholar lowercase 40-hex paper ID;
  ADS bibcode case-preserving; URL absolute HTTPS, lowercase host, no fragment
  or default port, preserving path and query. Trim surrounding whitespace first.
  Apply the same normalization to retrieved aliases. Do not equate unrelated
  schemes, fuzzy-match titles for gold, or guess missing identifiers. A curator
  may add independently verified aliases before locking, never after seeing
  held-out results. URL aliases must explicitly include distinct landing URLs.
- A tiny synthetic harness package may have fewer questions/domains; its set ID
  MUST start `fixture-` and reports MUST set `purpose: "fixture"`. Pilot sets
  have `purpose: "evaluation"` in reports and meet the counts in section 2.1.

## 3. Leakage controls and access separation

1. Curators/scorers may read the full package. Agents and query planners receive
   only `question_id`, `question`, `domain`, the date cap and their budgets. Never
   mount `set.json`, `review.md`, snapshot labels, gold, review identifiers,
   exclusion lists or scoring reports into an agent's filesystem/context. A
   separate trusted gateway holds exclusions; do not tell the agent what review
   to avoid. Engine execution receives the locked search plan, not gold-driven
   queries; gold is read only by scoring and the leakage gate.
2. Author the question independently from the research problem, not by copying
   the review title. A curator MUST confirm that question wording and query
   terms do not quote the review title, identify the review/authors, or include
   gold identifiers. Case-folded, punctuation/whitespace-normalized full-title
   substring matches are a mandatory automatic rejection; the review-sheet
   check also catches distinctive partial quotations. Record the check in the
   sheet before approval. Curator searches to resolve identifiers are not
   benchmark searches and MUST NOT be reused as the engine plan.
3. The reference review and all its identifiers are forbidden in retrieved
   records, final results, citations and citation-expansion seeds. Check raw
   connector records **before dedupe**, and check merged outputs and seeds again.
   Match normalized identifiers and canonical URLs; exact normalized review
   title is an additional fail-closed check when aliases are absent. Block before
   exposing the response to an agent. If a reference review appears, invalidate
   the run with `reference_review_leak`; do not quietly remove it and score the
   remaining output. Agent prompts/transcripts MUST NOT contain the review or
   its content. This also applies to native-search results in the without-rforge
   condition; a host that cannot interpose a gate is not qualified for this tier.
4. Every search and expansion is publication-date-capped inclusively at
   `search_date` (UTC end of day). The trusted gateway sets/validates the provider
   filter; no uncapped fallback is allowed. Where the API supports only a year
   or ignores a finer filter, also filter individual records using their raw
   publication-date metadata before delivering them to the engine/agent. Choose
   the earliest documented public publication date across online/print dates.
   A partial date is represented by its latest possible day (month end or
   December 31); unknown dates are excluded. Exclude records whose resulting
   upper bound exceeds the cap, recording the reason/count. If the connector
   cannot expose enough metadata to enforce this, fail `date_cap_unenforceable`.
   Apply the same rule to gold validation: included papers with unavailable
   dates require curator-resolved metadata in the manifest before locking. A modern capture is
   not a historical database reconstruction: disclose retrospective indexing
   and metadata changes as a limitation.
5. Partition by review family, not just question ID. No reference-review family
   or gold paper may occur in both splits. Reject cross-split normalized-identifier
   overlap; a curator must also check semantic question overlap, review updates
   and near-duplicate topics. Related questions from the same review stay in one
   split. Require review-family independence between the pilot's 8 held-out
   questions so question-level intervals do not treat one review as eight units.
6. Tune queries, budgets, dedupe thresholds and code using `dev` only. Lock
   questions, plans, metrics and analysis choices before held-out execution;
   never choose K, exclusions or repetitions after seeing held-out outcomes.
   A maintainer/CI may replay held-out snapshots for regression measurement,
   but MUST NOT feed those results back into tuning. Once detailed held-out
   outcomes guide a change, mark that version exposed and use a newly approved
   held-out set for fresh improvement claims. A checksum detects drift, not
   confidentiality; repository access alone cannot enforce a hidden split.

## 4. HTTP snapshot format and replay

### 4.1 Index and response bodies

`snapshots/index.json` has exactly these fields. The typed notation below is
normative: all fields required, nullable only where stated, no extra fields.
Arrays preserve capture order unless an ordering rule is specified.

| Field | Type and rule |
| --- | --- |
| `schema_version` | Integer `1` |
| `set_id` | Manifest's set ID; no set checksum here |
| `capture` | Object: `captured_at` (UTC timestamp), `rforge_version` (nonempty), `revision` (nonempty Git SHA/build revision), `rights` (nonempty rights/attribution and permitted metadata projection explanation) |
| `questions` | Array of the question objects below, sorted by question ID; includes both splits |

Each question object has:

| Field | Type and rule |
| --- | --- |
| `question_id` | Existing question ID |
| `started_at` | UTC timestamp at question clock zero, before the first planned call |
| `wall_clock_ms` | Nonnegative integer: recorded elapsed time from clock zero to completion/stop, including local normalization/dedupe, retries and waits |
| `timing` | Ordered clock checkpoints as defined in section 4.4; includes unsuccessful budget checks, not just completed HTTP |
| `stop_reason` | `completed`, `budget_exhausted`, `coverage_saturated`, `interrupted`, or `error_threshold` |
| `date_exclusions` | Array of `{record_id, reason}`; reason `after_cap` or `unknown_date`, sorted by record ID |
| `identities` | Object mapping stable raw `record_id` to independently curated entity-ID slug; absent keys mean unjudged, not distinct |
| `identity_evidence` | Object mapping each used entity ID to nonempty metadata-only curator rationale/identifier evidence; gold entities use their `paper_id`, non-gold use `other-` prefixed slugs |
| `exchanges` | Array of the exchange objects below in request-start order |

Each exchange has:

| Field | Type and rule |
| --- | --- |
| `sequence` | One-based contiguous integer within the question |
| `plan_call` | One-based search-plan index; pagination/retries remain attached to their call |
| `request` | Object: `method` (uppercase), `url` (absolute), `headers` (string-to-string-array map), `body_sha256` (hash or null for no body) |
| `request_key` | SHA-256 of JCS of the sanitized canonical `request` object |
| `response` | Null on transport failure; otherwise object with `status` (integer 100–599), `headers` (string-to-string-array map), `body_sha256` (hash, including hash of empty bytes) |
| `transport_error` | Null on response; otherwise `timeout`, `connection_reset`, or `connection_refused` |
| `elapsed_ms` | Nonnegative integer measured request duration |
| `retry_after_ms` | Nonnegative integer wait performed after this exchange; zero if none |
| `redactions` | Array of nonempty descriptions of omitted credential fields or legal metadata projection; empty when none |

Exactly one of response and transport error is non-null. Both request bodies
(when present) and response bodies live at `bodies/<sha256>.bin`. Verify every
referenced body hash and reject missing bodies. `snapshots_sha256` is the hash
of JCS of the index; it transitively covers body bytes. The index, labels and
body projections are finalized and checksummed before locking. Curators can
label identities after capture but before this final lock; labels MUST NOT be
inferred from the deduper being evaluated. Unjudged pairs remain explicit.

A stable `record_id` is SHA-256 of JCS `[source, source_id]` from the raw connector
record. Require a nonempty source-native ID; if unavailable, use the sorted
canonical identifier array as the second element instead. Repeated occurrences
of the same raw identity count once for pairwise dedupe evaluation. The harness
retains raw-to-output-cluster membership even when merged Paper records drop an
identifier. Entity labels are private scorer data, not connector inputs.

### 4.2 Request normalization, privacy and legality

- Uppercase method; lowercase URL scheme/host, remove default port and fragment,
  keep escaped path, sort decoded query `(key, value)` pairs bytewise and encode
  with RFC 3986 percent encoding (space `%20`, not `+`). Repeated pairs remain.
  Do not normalize query text, cursors, date caps or other behavioral values.
- Replace credential/contact query values (`api_key`, `key`, `token`, `email`,
  `mailto`) with literal `REDACTED`; redact the corresponding JSON body fields.
  Use dummy credentials in replay. Never store original credentials or their
  hashes. Sanitize echoed URLs/credentials in response bodies too, recording the
  transformation. Hash bodies **after** sanitation.
- Request header allowlist: `accept`, `content-type`, `range`. Response header
  allowlist: `content-type`, `content-encoding`, `retry-after`, `link`, `location`.
  Lowercase names, trim value-edge whitespace, preserve value order, sanitize
  credentials in links, omit every other header (including cookies, auth and
  request IDs). Capture transport-decoded bodies and omit `content-encoding`
  after decompression; replay must not decompress twice.
- Bodies are API payloads consumed by the real connector, not serialized
  `PaperRecord` outputs. Prefer API field selection for minimal legal metadata.
  If response projection is necessary, document which fields were removed;
  retain identifiers, titles, publication dates, pagination and errors required
  by the connector and gate. Do not fabricate successful API responses for a
  real evaluation. Synthetic responses are restricted to `fixture-` packages.
- Follow [fixture policy](../fixture-policy.md): metadata only, no copyrighted
  PDFs, private notes or unlicensed abstracts/full text. Record attribution and
  redistribution permission in `capture.rights` and the review sheet. Rights
  that prevent committing the necessary payload make that source unsuitable;
  do not make an offline report depend on a private download.

### 4.3 Locked HTTP policy

The manifest's `http_policy` applies to every connector and is injected in both
record and replay, overriding environment-dependent timing/retry settings (not
credentials). Authoring defaults: `algorithm: "source-http-v1"`, `max_retries: 2`,
`timeout_ms: 10000`, `request_delay_ms: 0`, `max_retry_after_ms: 30000`,
`max_response_bytes: 10485760`, `max_redirects: 10`. All values are stored, never
implicitly supplied during scoring. The v1 algorithm follows the shared client
in [http.go](../../internal/sources/http.go), with an injectable clock:

- Each HTTP operation has at most `1 + max_retries` attempts. Wait
  `request_delay_ms` before each attempt. Apply `timeout_ms` per attempt and the
  question's remaining deadline; whichever expires first wins. Every redirect
  counts as an HTTP request, follows the standard Go HTTP redirect semantics,
  and is limited by `max_redirects` (zero disables following).
- Retry transport failures and statuses 429 or 500–599 only. Other statuses end
  the operation. Parse `Retry-After` as positive integer seconds or HTTP-date
  relative to the injected current UTC time; absent/invalid/past means zero.
  If the delay is zero for 429, use `1000 * 2^attempt` ms with zero-based attempt.
  Other retryable statuses and transport failures default to zero backoff.
- If a retry delay exceeds `max_retry_after_ms`, stop that operation with a
  visible source error, rather than silently capping or sleeping. Otherwise
  wait the delay before the next attempt; never wait after the final attempt.
  Request/clock budgets take precedence and can prevent a retry or its wait.
  Record the actual backoff in `retry_after_ms`; do not count per-attempt
  `request_delay_ms` in that field. Reject oversized bodies at
  `max_response_bytes`. No additional adapter-level retries are allowed.

### 4.4 Replay algorithm and clock

Inject a replay HTTP transport into the production connectors; never substitute
pre-normalized results. Calls execute sequentially in stored plan order, and
records retain connector response order. Request matching uses
`(question_id, request_key, occurrence_number)`; repeated identical requests
consume successive exchanges. Pagination is **not** automatic: each page is an
explicit plan entry with its locked cursor. Retries consume exchanges under the
same entry. Budgets count every HTTP attempt, including retries/errors.

Replay MUST disable all network access, bypass external caches, use an empty
workspace, verify every matched exchange, and fail on missing, extra or unconsumed
exchanges. No fallback to live HTTP is allowed. Replay redirects and retries
through the same recorded transport; all outgoing requests, including redirects,
are recorded and counted. Replay wall time is `started_at` plus the virtual
elapsed clock. Response statuses and transport failures go through the real
connector error path. Sleeps advance virtual time rather than sleeping. Request
and clock budgets are checked before every planned call, HTTP attempt and wait;
record uses a monotonic clock and replay uses these stored decision checkpoints.

Each `timing` entry is exactly `{stage, plan_call, sequence, elapsed_ms}`.
`elapsed_ms` is a nonnegative absolute offset from `started_at`, in nondecreasing
order. Stages are `before_call`, `after_call`, `before_http`, `after_http`,
`before_wait`, `after_wait`, `after_dedupe`, `stop`. `plan_call` is the one-based
plan index for call/HTTP/wait stages, otherwise null. `sequence` is the exchange
index for HTTP stages, otherwise null. Capture `before_*` **before** checking a
budget, including a denied next operation; a denied operation has no matching
`after_*` or exchange. `after_call` follows the call's response processing, date
gate and normalization, even on source error. `after_dedupe` follows final dedupe.
`stop` is the single final checkpoint, with offset exactly `wall_clock_ms`.

At each corresponding decision/stage in replay, consume the next matching
checkpoint and advance the virtual clock to its absolute offset **before** the
budget check. Reject a stage/index mismatch or time regression. Do not add local
processing time only at the end: these checkpoints preserve local CPU gaps at
all budget decisions. `before_http`/`after_http` offsets must differ by the
exchange's `elapsed_ms`. Completed wait offsets must cover the requested wait
and agree with the locked policy; their excess represents recorded local work.
All selected timing entries must be consumed, just like exchanges. A stopped
call may have a denied `before_http` whose sequence is the next unused index,
without a corresponding request; this is not a missing-exchange failure.

For example, a 1000 ms budget can contain a 100 ms HTTP request followed by
950 ms normalization. The next `before_call` checkpoint is 1050 ms: replay
stops there without attempting a nonexistent second request. Require
`wall_clock_ms >= sum(elapsed_ms + retry_after_ms)` over exchanges. Budget-
exhausted waits may be interrupted at the remaining deadline; their partial
elapsed time and final stop are recorded, without an `after_wait` checkpoint.

The replay report's `wall_clock_ms` is the captured elapsed value, not today's
execution duration. It measures the recorded workload and is byte-reproducible;
it MUST NOT be used to claim a new implementation runs faster. Actual local
replay duration may be printed to stderr, never placed in the canonical report.
Agent/live and record latency use a monotonic clock and are not deterministic.
Do not serialize today's timestamp, hostname, temp paths, random IDs, cache state
or nondeterministic error strings into replay reports.

## 5. Engine execution contract (V-B3)

```text
rforge benchmark run --set <dir> [--replay|--record] [--split dev|held_out] [--out <report.json>]
```

- Default mode is `--replay`; flags are mutually exclusive. Default split is
  `held_out`. Replay accepts only `locked` sets; record accepts only `approved`
  sets and requires explicit network opt-in by the flag. Record captures **both**
  splits so the resulting package is complete; `--split` selects report rows.
  Replay consumes exactly the selected split's exchanges, ignoring the other
  split by design. A candidate is validation-only input, not runnable gold.
- No hidden queries, source defaults, credential-dependent source additions or
  expansion are part of the v1 engine plan. Each manifest call maps to
  `SourceQuery{Terms, Limit, PageCursor, Filters}`. Use the actual connector,
  paper normalization and production deduper. Automatic expansion is off in
  engine v1; enforce the seed gate whenever later plans/agent tools expand.
- Enforce the leakage gate before outputs reach dedupe, then validate membership
  afterwards. Preserve raw records and dedupe clusters for scorer audit in a
  private sidecar; do not give these artifacts to agents. Assign output ranks by
  first raw appearance `(plan index, response position)`, with a merged cluster
  taking its earliest member's position. Do not gold-sort or relevance-rerank.
- Follow the stored caps and retry policy of the recorded workload. A partial
  valid run (budget exhaustion, interruption or connector error) is scored on
  its actual output, including empty output; no silent question deletion.
  `coverage_saturated` is reserved for a recorded engine stop signal, not a
  scorer decision. Emit source failures visibly in the report.
- Emit the section 7 report to stdout (only JSON), or to `--out` with a concise
  path message on stderr. Exit 0 for a valid scored run, including low scores or
  budget stops; exit 1 for invalid/checksum/leakage/replay failures; exit 2 for
  usage errors. Invalid runs have `status: "invalid"`, deterministic error codes,
  null metric values and no usable aggregate. Never publish a partial success
  report as a baseline. Record publication is atomic: no locked state on failure.
- V-A4 can attach engine provenance events and output hashes alongside the report;
  the report itself cannot declare its own `engine_verified` status. Do not delay
  V-B3 on later claim facets. Keep manual annotations `agent_reported`.

## 6. Scoring definitions shared by both tiers

### 6.1 Retrieval and ranking

For a question let `G` be its distinct gold paper IDs and `R` its complete ordered
returned list after the system's dedupe. Agent runs use the ordered bibliography
in the final answer as `R`, not hidden tool hits; also retain the raw tool trace
for requests/dedupe audit. Require an explicitly ordered bibliography in both
conditions. Each returned cluster occupies one rank. A repeated gold identity
can earn credit at its first rank only; later duplicates occupy ranks but add no
hits. Do not remove duplicates or rerank inside the scorer.

Match by canonical identifiers to locked gold aliases, not titles or an LLM.
A cluster containing independently labeled different entities is a false merge
and earns **zero** retrieval hits, even if it contains gold aliases. Where more
than one gold identity matches a cluster, it likewise earns zero. Unrecognized
aliases do not create a hit or extra gold; a cluster matching exactly one gold
identity with no known conflict earns one hit. Report unjudged merge counts so
missing labels cannot masquerade as perfect dedupe.

Let `hK` be unique gold hits in the first `min(K, |R|)` ranks, and `h` hits in all
ranks. For every locked K:

| Metric | Numerator / denominator | Edge cases |
| --- | --- | --- |
| `recall_at_k` | `hK / |G|` | Missing ranks earn no hits |
| `recall` (overall recall) | `h / |G|` | Empty retrieval gives 0; empty gold is invalid |
| `precision_at_k` | `hK / K` | Fixed K denominator: missing ranks count as nonrelevant |

Precision here is **gold-set precision**, not a claim that every non-gold paper
is irrelevant: review inclusion lists can omit genuinely relevant work. Always
state that limitation. No post-hoc relevance expansion of held-out gold.

### 6.2 Dedupe, requests and latency

Dedupe is pairwise over unique raw record identities. For every unordered pair
placed in the same output cluster, count one `dedupe_true_merges` if the locked
identity labels agree, one `dedupe_false_merges` if both exist and differ, or one
`dedupe_unjudged_merges` if either is missing. Thus a cluster of n distinct raw
records proposes n(n−1)/2 pairs, not n−1 procedural merge operations. Also count
`dedupe_missed_merges`: labeled equal-entity pairs left in separate clusters.
Report counts, not a guessed zero when raw membership is unavailable. For an
agent/native tool without raw membership, all four counts are null with
`dedupe_status: "unavailable"`; engine runs MUST expose membership. An engine
with zero proposed pairs has observed counts of zero, not unavailable.

`http_requests` counts every attempted HTTP request at the gateway, including
errors, retries, redirects, resolution and citation fetches. Offline replay
reports those same logical attempts, not actual network traffic (which is zero).
`search_calls` counts calls returning a search-results page (including pagination);
`tool_calls` counts all host tool invocations, with nested rforge work counting
one invocation but all underlying HTTP attempts. Engine `tool_calls` is zero.
A cache starts empty per run; later cache hits count tool calls but not new HTTP.
Record input/output tokens and billed USD separately for agents; null for the
engine. These are measured usage, never budget estimates. `wall_clock_ms` is
elapsed monotonic time including model/tool waits and retries until final answer
or stop; section 4 defines replay's captured equivalent.

### 6.3 Citation correctness (enabled by V-D3)

Before V-D3, emit `citation_status: "not_implemented"`, all citation counts and
`citation_correctness` null; do not claim zero errors or perfect citations.
After V-D3, evaluate **every claim-to-source citation occurrence** in the final
answer, not just unique papers. Correctness is the fraction whose source identity,
content integrity and passage match all pass the deterministic checker. A missing
quote/passage, unresolved source or failed/missing facet stays in the denominator
and earns no numerator. Store each facet separately in the private scoring audit;
this conjunction defines a metric, not a collapsed verification facet.

Emit `citation_status: "measured"` with numerator/denominator; no citations gives
`citation_status: "no_citations"`, counts zero and value null, never 1. Semantic
support remains `agent_reported` with model identity, and human approval requires
a named reviewer. Neither is inferred from deterministic correctness. A citation
can pass mechanical checks and still fail to support the claim scientifically.

## 7. Report JSON schema, aggregation and reproducibility

The following schema defines the scored report. Nullable values carry the statuses
specified above; numeric ratios are unrounded IEEE-754 finite numbers serialized
by JCS, not percentages or strings. No NaN or Infinity. Every selected question
has one row per condition/repetition, sorted by `(question_id, condition,
repetition)`; engine condition is `engine`, repetition 1. Error arrays sort by
`(question_id, code)`. The version/revision identify the engine under evaluation;
`protocol_sha256` locks the scorer/protocol configuration (section 8 for agents,
or SHA-256 of the exact bytes of this specification for engine v1).

```json
{
  "$schema": "https://json-schema.org/draft/2020-12/schema",
  "title": "Research-quality benchmark report v1",
  "type": "object", "additionalProperties": false,
  "required": ["schema_version", "purpose", "tier", "mode", "status", "set_id", "set_checksum", "snapshots_sha256", "split", "rforge_version", "revision", "protocol_sha256", "k", "runs", "aggregates", "paired", "errors"],
  "properties": {
    "schema_version": {"const": 1},
    "purpose": {"enum": ["fixture", "evaluation"]},
    "tier": {"enum": ["engine", "agent"]},
    "mode": {"enum": ["replay", "record", "live"]},
    "status": {"enum": ["complete", "invalid"]},
    "set_id": {"type": "string", "minLength": 1},
    "set_checksum": {"$ref": "#/$defs/hash"},
    "snapshots_sha256": {"oneOf": [{"$ref": "#/$defs/hash"}, {"type": "null"}]},
    "split": {"enum": ["dev", "held_out"]},
    "rforge_version": {"type": "string", "minLength": 1},
    "revision": {"type": "string", "minLength": 1},
    "protocol_sha256": {"$ref": "#/$defs/hash"},
    "k": {"type": "array", "minItems": 1, "uniqueItems": true, "items": {"type": "integer", "minimum": 1}},
    "runs": {"type": "array", "items": {"$ref": "#/$defs/run"}},
    "aggregates": {"oneOf": [{"type": "null"}, {"type": "array", "items": {"$ref": "#/$defs/aggregate"}}]},
    "paired": {"type": "array", "items": {"$ref": "#/$defs/paired"}},
    "errors": {"type": "array", "items": {"$ref": "#/$defs/error"}}
  },
  "$defs": {
    "hash": {"type": "string", "pattern": "^[0-9a-f]{64}$"},
    "count": {"type": ["integer", "null"], "minimum": 0},
    "number": {"type": ["number", "null"], "minimum": 0},
    "ratio": {
      "type": "object", "additionalProperties": false,
      "required": ["numerator", "denominator", "value"],
      "properties": {
        "numerator": {"$ref": "#/$defs/count"},
        "denominator": {"$ref": "#/$defs/count"},
        "value": {"type": ["number", "null"], "minimum": 0, "maximum": 1}
      }
    },
    "at_k": {
      "type": "object", "additionalProperties": false,
      "required": ["k", "recall_at_k", "precision_at_k"],
      "properties": {
        "k": {"type": "integer", "minimum": 1},
        "recall_at_k": {"$ref": "#/$defs/ratio"},
        "precision_at_k": {"$ref": "#/$defs/ratio"}
      }
    },
    "run": {
      "type": "object", "additionalProperties": false,
      "required": ["question_id", "domain", "condition", "repetition", "stop_reason", "returned", "gold_count", "at_k", "recall", "dedupe_status", "dedupe_true_merges", "dedupe_false_merges", "dedupe_unjudged_merges", "dedupe_missed_merges", "http_requests", "search_calls", "tool_calls", "wall_clock_ms", "input_tokens", "output_tokens", "cost_usd", "date_excluded", "citation_status", "citation_correctness"],
      "properties": {
        "question_id": {"type": "string", "minLength": 1},
        "domain": {"type": "string", "minLength": 1},
        "condition": {"enum": ["engine", "with_rforge", "without_rforge"]},
        "repetition": {"type": "integer", "minimum": 1},
        "stop_reason": {"enum": ["completed", "budget_exhausted", "coverage_saturated", "interrupted", "error_threshold"]},
        "returned": {"$ref": "#/$defs/count"},
        "gold_count": {"type": "integer", "minimum": 1},
        "at_k": {"type": "array", "items": {"$ref": "#/$defs/at_k"}},
        "recall": {"$ref": "#/$defs/ratio"},
        "dedupe_status": {"enum": ["measured", "unavailable", "invalid"]},
        "dedupe_true_merges": {"$ref": "#/$defs/count"},
        "dedupe_false_merges": {"$ref": "#/$defs/count"},
        "dedupe_unjudged_merges": {"$ref": "#/$defs/count"},
        "dedupe_missed_merges": {"$ref": "#/$defs/count"},
        "http_requests": {"$ref": "#/$defs/count"},
        "search_calls": {"$ref": "#/$defs/count"},
        "tool_calls": {"$ref": "#/$defs/count"},
        "wall_clock_ms": {"$ref": "#/$defs/count"},
        "input_tokens": {"$ref": "#/$defs/count"},
        "output_tokens": {"$ref": "#/$defs/count"},
        "cost_usd": {"$ref": "#/$defs/number"},
        "date_excluded": {"$ref": "#/$defs/count"},
        "citation_status": {"enum": ["not_implemented", "measured", "no_citations", "invalid"]},
        "citation_correctness": {"$ref": "#/$defs/ratio"}
      }
    },
    "aggregate": {
      "type": "object", "additionalProperties": false,
      "required": ["condition", "domain", "metric", "k", "question_count", "macro_mean", "micro"],
      "properties": {
        "condition": {"enum": ["engine", "with_rforge", "without_rforge"]},
        "domain": {"type": ["string", "null"]},
        "metric": {"$ref": "#/$defs/metric"},
        "k": {"type": ["integer", "null"], "minimum": 1},
        "question_count": {"type": "integer", "minimum": 0},
        "macro_mean": {"$ref": "#/$defs/number"},
        "micro": {"oneOf": [{"$ref": "#/$defs/ratio"}, {"type": "null"}]}
      }
    },
    "metric": {"enum": ["recall_at_k", "precision_at_k", "recall", "dedupe_true_merges", "dedupe_false_merges", "dedupe_unjudged_merges", "dedupe_missed_merges", "http_requests", "search_calls", "tool_calls", "wall_clock_ms", "input_tokens", "output_tokens", "cost_usd", "citation_correctness"]},
    "paired": {
      "type": "object", "additionalProperties": false,
      "required": ["metric", "k", "question_count", "repetitions", "mean_difference", "ci95_low", "ci95_high", "method"],
      "properties": {
        "metric": {"$ref": "#/$defs/metric"},
        "k": {"type": ["integer", "null"], "minimum": 1},
        "question_count": {"type": "integer", "minimum": 0},
        "repetitions": {"type": "integer", "minimum": 1},
        "mean_difference": {"type": ["number", "null"]},
        "ci95_low": {"type": ["number", "null"]},
        "ci95_high": {"type": ["number", "null"]},
        "method": {"const": "paired-question-student-t"}
      }
    },
    "error": {
      "type": "object", "additionalProperties": false,
      "required": ["question_id", "code"],
      "properties": {
        "question_id": {"type": ["string", "null"]},
        "code": {"enum": ["invalid_set", "checksum_mismatch", "reference_review_leak", "date_cap_unenforceable", "snapshot_mismatch", "source_error", "protocol_violation"]}
      }
    }
  }
}
```

Semantic constraints beyond the schema:

- Root `k`, question IDs/domains, gold counts and set/snapshot hashes MUST match
  the validated manifest. Engine mode is replay/record with a non-null snapshot
  hash; agent mode is live (snapshot hash identifies the engine fixture lock,
  not a claim that live requests replayed it). Rows contain exactly one `at_k`
  entry per K. Ratio value equals numerator/denominator; zero denominator means
  null value. Unmeasured citations use all-null ratios; `no_citations` uses 0/0.
- A complete report has a full row matrix and non-null scored retrieval values.
  Source errors may coexist with complete status if visible and output is scored.
  Infrastructure failure before execution produces an invalid report, not a
  zero-quality observation. Invalid reports have `aggregates: null`, `paired: []`
  and all row metric/count values null except `gold_count`; `at_k` retains Ks
  with null ratios, facet metric statuses become `invalid`. Set-wide failures
  before a manifest can be validated emit only a nonzero-exit JSON diagnostic
  `{ "error": "<code>" }`, not a misleading report with fabricated hashes.
- Aggregate by condition across questions, and separately by domain (`domain:
  null` means all domains). First average each metric's repetitions within a
  question, then macro-average over questions with measurements. Counts/costs
  are means per question, not sums disguised as means. `question_count` is the
  number contributing, so unavailable metrics remain visible. For ratio metrics
  also emit `micro` with integer numerators/denominators summed over all runs of
  contributing questions. Include a question only with all N measurements; every
  included question has the same N, so this equals the ratio of sums of their
  within-question mean counts. For non-ratio metrics `micro` is null. Never pool
  unequal N. Macro aggregates also require all N measurements for each included
  question, avoiding inconsistent treatment of unavailable repetitions.
- Engine `paired` is empty. Agent paired rows are section 8's contrasts. Emit
  one aggregate/paired row per metric (and K where applicable), even if null;
  null aggregate mean with question count zero means no observations. `k` is
  null for non-at-K metrics. Sort aggregates by `(condition, domain, metric, k)`
  and paired rows by `(metric, k)`, with null first. A changed engine baseline
  comparison MUST cite both set checksums, snapshot hashes, versions/revisions
  and protocol hashes; do not compare different inputs as an implementation win.

## 8. Agent-tier paired protocol (V-B6)

### 8.1 Pre-registration and execution

Before a live run, commit a protocol JSON under `benchmarks/agent-tier/` and hash
its JCS bytes into `protocol_sha256`. It MUST pin: host/version, model provider
and exact model/version, sampling settings, shared system/task prompt bytes,
condition-specific tool/skill instructions, rforge version/revision, set checksum,
ordered question IDs, N, all caps, run-order seed, scorer version, enabled citation
metric version, and the list of primary/secondary metrics. Missing model identity
or opaque tool accounting disqualifies a host rather than making an unmatched
comparison. Live execution requires separate owner authorization and spend cap.

Pilot: **Pi, 8 held-out questions × 2 conditions × N=3 repetitions = 48 runs**.
Run Codex second as a separate experiment/report; do not pool hosts or models.
The expanded release experiment is pre-registered separately using pilot effect
sizes; the 8-question pilot alone does not establish the 2.0 release gate.

For each `(question, repetition)` pair:

- `with_rforge` has the ResearchForge skill/tools; `without_rforge` has the host's
  native search, no rforge binary, plugin, skill, saved outputs or shared cache.
  Both otherwise receive the same research question, date cap, response-format
  instructions and host policy. Tool descriptions may differ only as required
  by the condition. Pin **the same model**, sampling settings, token/search/tool
  caps and question in each pair. Do not compare different model families.
- Default pilot caps per run: 32,000 total model tokens (input including cached
  tokens plus output), 20 search-page calls, 60 total host tool calls, 100 HTTP
  attempts and 600,000 ms wall time. These are specification defaults, not
  measured usage. Changes are allowed only in the pre-registered protocol and
  must be identical per pair. Nested rforge queries/pages consume the same
  search/HTTP budget as native calls; batching is not a budget escape. Check
  available budget before each invocation; terminate at the cap, recording
  actual overrun from an indivisible in-flight response instead of hiding it.
- Use isolated fresh workspaces and sessions for every run. No previous answer,
  gold, review, scored feedback or cross-condition memory may enter the prompt.
  Apply section 3's trusted gate to **both** conditions, including native search
  and fetched pages. A seeded balanced random order assigns with/without first;
  archive the exact schedule with the protocol. Interleave pairs in time to
  limit service/model drift. Pair repetition seeds where the provider supports
  them; retain differing realized randomness, do not cherry-pick runs.
- Keep the actual final/partial answer at budget stops. Valid run failures
  (model refuses, tool returns errors, empty answer) remain in the analysis as
  observed output; empty bibliography has recall/precision zero. A host crash
  or missing recording before a run starts is infrastructure failure: archive
  it, rerun **both** members of that pair under the same protocol and retain the
  superseded attempt in the private audit. A leakage/protocol failure invalidates
  the experiment until resolved, never just the worse condition's row.

### 8.2 Blind scoring

A separate custodian stores the condition-to-run mapping, raw transcripts,
provenance, request/token counters and workspace artifacts. Before scoring,
convert answers to a common structure containing only ordered bibliography,
claims, quotes and source locators. Strip rforge commands, tool names, provenance
files, run IDs, host/condition markers and formatting signatures; do not edit
substantive claims, identifiers, quotes or bibliography order. Give artifacts
random opaque labels with randomized presentation order. Keep the mapping secret
from scorers until scores and adjudications are frozen.

The scorer may see held-out gold and legal source passages, never condition labels
or operational counters. Run deterministic retrieval/facet scoring; blind human
adjudication may resolve extraction ambiguity against locked aliases, but MUST
NOT add new relevance labels or invent semantic correctness. Freeze score files
before the custodian joins condition labels and measured operational counters.
Retain both blinded artifacts and checksum-linked private audit artifacts.
Blindness concerns output attribution, not hiding gold from the trusted scorer.

### 8.3 Paired differences and 95% confidence intervals

Primary outcome: overall `recall`. Secondary outcomes: each recall@K and
precision@K, dedupe counts where measurable in **both** conditions, requests,
search/tool calls, wall time, tokens, billed USD and (after V-D3) citation
correctness. Report all outcomes; no composite quality score or post-hoc choice
of the most favorable K. Secondary intervals are descriptive, not multiplicity-
adjusted significance claims.

For each metric m and question q, first form the N repetition differences
`d(q,r) = m(with_rforge,q,r) - m(without_rforge,q,r)`. Then compute the
question-level mean `d(q) = sum_r d(q,r) / N`. With Q questions, report:

```text
mean_difference = sum_q d(q) / Q
s2 = sum_q (d(q) - mean_difference)^2 / (Q - 1)
ci95 = mean_difference +/- t_quantile(0.975, df=Q-1) * sqrt(s2 / Q)
```

This is a paired Student-t interval **over questions**, never an interval treating
24 repetitions or 48 runs as independent questions. For Q=8, the critical value
is approximately 2.364624; use the quantile, not a hardcoded 1.96. Positive deltas
favor retrieval/citation quality; negative deltas favor lower false merges,
requests, cost and latency. Do not clip intervals to [0,1] or suppress negative
results. With Q<2, emit mean if available and null interval bounds; zero variance
with Q>=2 yields a point interval. Report Q and N with every contrast.

Unavailable metrics (for example native dedupe membership or no-citation ratios)
are never imputed as zero. Include a question in that metric's paired contrast
only if **both** conditions have all N measurements; report the reduced Q.
When Q=0 all three estimates are null. Retrieval and usage should have the full
8-question matrix; a reduced matrix must be disclosed and cannot silently stand
in for the pre-registered primary analysis. The assumption of independent review
families is checked at curation (section 3); disclose small-Q and live-service
variability in pilot conclusions. Scientific semantic support, when scored by
humans, is a separate blinded rubric/report, not this deterministic metric.

## 9. Builder acceptance and update triggers

V-B3 MUST demonstrate offline, at the public command seam:

- A tiny legal `fixture-` set produces byte-identical reports on two replays
  with the same version/revision; no external network is possible.
- Real normalization/dedupe consumes HTTP bytes, and counts/ranks agree with a
  hand-calculated gold example including empty/short results and a false merge.
- A reference review encountered before dedupe or as an expansion seed fails;
  neither dropping the review nor granting recall from its reference list passes.
- Tampered manifest/body/labels, uncapped dates, unmatched requests or changed
  budgets cannot masquerade as the locked baseline. Before V-D3, citation
  correctness is explicitly unavailable; after V-D3 its facets stay separate.

These are behavioral acceptance cases, not a requirement to duplicate existing
unit tests; follow the repository's focused-test policy when implementing them.

V-B4's review sheet MUST include one row per question: ID, domain, split, review
identifiers, search-date evidence, included-list evidence and count, links to
manifest gold publication dates and their evidence, unresolved identifier/date/
rights issues, title-wording check, cross-split/review-family
check and leakage risk. Every candidate validates under section 2; the owner
performs approval and snapshot locking separately. The sheet clearly states
that metadata curation is not benchmark execution or a baseline result.

Revisit this specification when the question-set/report version, connector date
or pagination handling, dedupe identity policy, V-D3 facets, host accounting,
agent protocol or legal snapshot policy changes. Any resulting held-out input
change follows section 2's new-version/approval rule. V-B5 publishes the dated
baseline with its report and digests; later improvement claims cite measured
deltas against that exact lock, not the old cross-tool demo.
