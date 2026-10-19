package benchmarks

// The engine format follows docs/benchmark/spec.md v1. Gold and identity labels
// belong only to this scorer; they are never supplied to a connector.

type Identifier struct {
	Scheme string `json:"scheme"`
	Value  string `json:"value"`
}
type Approval struct {
	Approver   string   `json:"approver"`
	ApprovedAt string   `json:"approved_at"`
	Waivers    []string `json:"waivers,omitempty"`
}

// WaiverAPDomainAbsent is the owner's recorded acceptance that a pilot set has no
// artificial-photosynthesis held-out question (spec section 2; owner, 2026-10-19).
const WaiverAPDomainAbsent = "ap_domain_absent"

type Review struct {
	Identifiers         []Identifier `json:"identifiers"`
	Title               string       `json:"title"`
	SearchDate          string       `json:"search_date"`
	SearchDatePrecision string       `json:"search_date_precision,omitempty"`
	SearchDateLocator   string       `json:"search_date_locator"`
	IncludedListLocator string       `json:"included_list_locator"`
	Rights              string       `json:"rights"`
}
type GoldPaper struct {
	PaperID                string       `json:"paper_id"`
	Identifiers            []Identifier `json:"identifiers"`
	IncludedListEntry      string       `json:"included_list_entry"`
	PublicationDate        string       `json:"publication_date"`
	PublicationDateLocator string       `json:"publication_date_locator"`
}
type HTTPPolicy struct {
	Algorithm        string `json:"algorithm"`
	MaxRetries       int    `json:"max_retries"`
	TimeoutMS        int64  `json:"timeout_ms"`
	RequestDelayMS   int64  `json:"request_delay_ms"`
	MaxRetryAfterMS  int64  `json:"max_retry_after_ms"`
	MaxResponseBytes int64  `json:"max_response_bytes"`
	MaxRedirects     int    `json:"max_redirects"`
}
type PlanCall struct {
	Source     string            `json:"source"`
	Terms      string            `json:"terms"`
	Limit      int               `json:"limit"`
	PageCursor string            `json:"page_cursor"`
	Filters    map[string]string `json:"filters"`
}
type Budgets struct {
	MaxHTTPRequests int   `json:"max_http_requests"`
	MaxWallClockMS  int64 `json:"max_wall_clock_ms"`
}
type Question struct {
	QuestionID string      `json:"question_id"`
	Question   string      `json:"question"`
	Domain     string      `json:"domain"`
	Split      string      `json:"split"`
	Review     Review      `json:"gold_source_review"`
	Gold       []GoldPaper `json:"gold"`
	SearchPlan []PlanCall  `json:"search_plan"`
	Budgets    Budgets     `json:"budgets"`
}
type Set struct {
	SchemaVersion     int        `json:"schema_version"`
	SetID             string     `json:"set_id"`
	State             string     `json:"state"`
	Approval          *Approval  `json:"approval"`
	ReviewSheetSHA256 string     `json:"review_sheet_sha256"`
	SnapshotsSHA256   *string    `json:"snapshots_sha256"`
	SetChecksum       string     `json:"set_checksum"`
	K                 []int      `json:"k"`
	HTTPPolicy        HTTPPolicy `json:"http_policy"`
	Questions         []Question `json:"questions"`
}
type Capture struct {
	CapturedAt    string `json:"captured_at"`
	RforgeVersion string `json:"rforge_version"`
	Revision      string `json:"revision"`
	Rights        string `json:"rights"`
}
type SnapshotIndex struct {
	SchemaVersion int                `json:"schema_version"`
	SetID         string             `json:"set_id"`
	Capture       Capture            `json:"capture"`
	Questions     []SnapshotQuestion `json:"questions"`
}
type Timing struct {
	Stage     string `json:"stage"`
	PlanCall  *int   `json:"plan_call"`
	Sequence  *int   `json:"sequence"`
	ElapsedMS int64  `json:"elapsed_ms"`
}
type DateExclusion struct {
	RecordID string `json:"record_id"`
	Reason   string `json:"reason"`
}
type SnapshotQuestion struct {
	QuestionID       string            `json:"question_id"`
	StartedAt        string            `json:"started_at"`
	WallClockMS      int64             `json:"wall_clock_ms"`
	Timing           []Timing          `json:"timing"`
	StopReason       string            `json:"stop_reason"`
	DateExclusions   []DateExclusion   `json:"date_exclusions"`
	Identities       map[string]string `json:"identities"`
	IdentityEvidence map[string]string `json:"identity_evidence"`
	Exchanges        []Exchange        `json:"exchanges"`
}
type Request struct {
	Method     string              `json:"method"`
	URL        string              `json:"url"`
	Headers    map[string][]string `json:"headers"`
	BodySHA256 *string             `json:"body_sha256"`
}
type Response struct {
	Status     int                 `json:"status"`
	Headers    map[string][]string `json:"headers"`
	BodySHA256 string              `json:"body_sha256"`
}
type Exchange struct {
	Sequence       int       `json:"sequence"`
	PlanCall       int       `json:"plan_call"`
	Request        Request   `json:"request"`
	RequestKey     string    `json:"request_key"`
	Response       *Response `json:"response"`
	TransportError *string   `json:"transport_error"`
	ElapsedMS      int64     `json:"elapsed_ms"`
	RetryAfterMS   int64     `json:"retry_after_ms"`
	Redactions     []string  `json:"redactions"`
}
type Ratio struct {
	Numerator   *int64   `json:"numerator"`
	Denominator *int64   `json:"denominator"`
	Value       *float64 `json:"value"`
}
type AtK struct {
	K         int   `json:"k"`
	Recall    Ratio `json:"recall_at_k"`
	Precision Ratio `json:"precision_at_k"`
}
type Run struct {
	QuestionID          string   `json:"question_id"`
	Domain              string   `json:"domain"`
	Condition           string   `json:"condition"`
	Repetition          int      `json:"repetition"`
	StopReason          string   `json:"stop_reason"`
	Returned            *int64   `json:"returned"`
	GoldCount           int      `json:"gold_count"`
	AtK                 []AtK    `json:"at_k"`
	Recall              Ratio    `json:"recall"`
	DedupeStatus        string   `json:"dedupe_status"`
	TrueMerges          *int64   `json:"dedupe_true_merges"`
	FalseMerges         *int64   `json:"dedupe_false_merges"`
	UnjudgedMerges      *int64   `json:"dedupe_unjudged_merges"`
	MissedMerges        *int64   `json:"dedupe_missed_merges"`
	HTTPRequests        *int64   `json:"http_requests"`
	SearchCalls         *int64   `json:"search_calls"`
	ToolCalls           *int64   `json:"tool_calls"`
	WallClockMS         *int64   `json:"wall_clock_ms"`
	InputTokens         *int64   `json:"input_tokens"`
	OutputTokens        *int64   `json:"output_tokens"`
	CostUSD             *float64 `json:"cost_usd"`
	DateExcluded        *int64   `json:"date_excluded"`
	CitationStatus      string   `json:"citation_status"`
	CitationCorrectness Ratio    `json:"citation_correctness"`
}
type Aggregate struct {
	Condition     string   `json:"condition"`
	Domain        *string  `json:"domain"`
	Metric        string   `json:"metric"`
	K             *int     `json:"k"`
	QuestionCount int      `json:"question_count"`
	MacroMean     *float64 `json:"macro_mean"`
	Micro         *Ratio   `json:"micro"`
}
type RunError struct {
	QuestionID *string `json:"question_id"`
	Code       string  `json:"code"`
}
type EngineReport struct {
	SchemaVersion   int         `json:"schema_version"`
	Purpose         string      `json:"purpose"`
	Tier            string      `json:"tier"`
	Mode            string      `json:"mode"`
	Status          string      `json:"status"`
	SetID           string      `json:"set_id"`
	SetChecksum     string      `json:"set_checksum"`
	SnapshotsSHA256 *string     `json:"snapshots_sha256"`
	Split           string      `json:"split"`
	RforgeVersion   string      `json:"rforge_version"`
	Revision        string      `json:"revision"`
	ProtocolSHA256  string      `json:"protocol_sha256"`
	K               []int       `json:"k"`
	Runs            []Run       `json:"runs"`
	Aggregates      []Aggregate `json:"aggregates"`
	Paired          []any       `json:"paired"`
	Errors          []RunError  `json:"errors"`
}
