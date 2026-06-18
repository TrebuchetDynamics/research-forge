package forge

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/TrebuchetDynamics/research-forge/internal/project"
	"github.com/TrebuchetDynamics/research-forge/internal/provenance"
	"github.com/TrebuchetDynamics/research-forge/internal/reviewpkg"
)

const schemaVersion = "1"

type StateID string

const (
	StateQuestionDraft       StateID = "question_draft"
	StateProtocolPlan        StateID = "protocol_plan"
	StateSourcePlan          StateID = "source_plan"
	StateImportPlan          StateID = "import_plan"
	StateDedupeReview        StateID = "dedupe_review"
	StateFullTextAcquisition StateID = "full_text_acquisition"
	StateParserArbitration   StateID = "parser_arbitration"
	StateIndexing            StateID = "indexing"
	StateScreening           StateID = "screening"
	StateExtraction          StateID = "extraction"
	StateAnalysis            StateID = "analysis"
	StateReportBuild         StateID = "report_build"
	StatePackageExport       StateID = "package_export"
	StateDone                StateID = "done"
)

type State struct {
	SchemaVersion       string        `json:"schemaVersion"`
	ProjectPath         string        `json:"projectPath"`
	Manifest            string        `json:"manifest"`
	Question            string        `json:"question"`
	SourceChoices       []string      `json:"sourceChoices,omitempty"`
	ToolChoices         []string      `json:"toolChoices,omitempty"`
	PrivacyLegalPreview []string      `json:"privacyLegalPreview,omitempty"`
	CurrentState        StateID       `json:"currentState"`
	Approvals           []Approval    `json:"approvals"`
	BlockedReviewGates  []BlockedGate `json:"blockedReviewGates"`
	NextSafeActions     []NextAction  `json:"nextSafeActions"`
	ValidationReceipts  []string      `json:"validationReceipts"`
	UpdatedAt           string        `json:"updatedAt"`
}

type Approval struct {
	Gate      string `json:"gate"`
	Note      string `json:"note,omitempty"`
	Actor     string `json:"actor"`
	Timestamp string `json:"timestamp"`
}

type BlockedGate struct {
	Gate             string `json:"gate"`
	AppliesAt        string `json:"appliesAt"`
	RequiredDecision string `json:"requiredDecision"`
}

type NextAction struct {
	Label string `json:"label"`
	CLI   string `json:"cli"`
}
type InitOptions struct {
	Question      string
	Actor         string
	SourceChoices []string
	ToolChoices   []string
}
type ApprovalInput struct{ Gate, Note, Actor string }

type PackageCompletion struct {
	State        State                 `json:"state"`
	Package      reviewpkg.Package     `json:"package"`
	AuditReport  reviewpkg.AuditReport `json:"auditReport"`
	ReplayReport reviewpkg.AuditReport `json:"replayReport"`
	PackagePath  string                `json:"packagePath"`
}

type GateSpec struct {
	Gate             string `json:"gate"`
	AppliesAt        string `json:"appliesAt"`
	RequiredDecision string `json:"requiredDecision"`
}

func ReviewGates() map[string]GateSpec {
	return map[string]GateSpec{
		"question approval":           {"question approval", string(StateQuestionDraft), "owner approves canonical question"},
		"protocol approval":           {"protocol approval", string(StateProtocolPlan), "question, criteria, extraction schema seed are acceptable"},
		"network/API approval":        {"network/API approval", string(StateSourcePlan), "connector plan, credentials, rate limits, privacy risk accepted"},
		"identity approval":           {"identity approval", string(StateDedupeReview), "merges/splits accepted or conflicts explicitly deferred"},
		"legal acquisition approval":  {"legal acquisition approval", string(StateFullTextAcquisition), "download/archive/shareability decision recorded"},
		"parser arbitration approval": {"parser arbitration approval", string(StateParserArbitration), "selected fields/passages/references accepted or corrected"},
		"screening approval":          {"screening approval", string(StateScreening), "required stages complete, conflicts adjudicated"},
		"evidence approval":           {"evidence approval", string(StateExtraction), "accepted evidence has source support and correction history"},
		"analysis approval":           {"analysis approval", string(StateAnalysis), "warnings/method choices acknowledged"},
		"claim approval":              {"claim approval", string(StateReportBuild), "claims trace to accepted evidence or are blocked"},
		"package approval":            {"package approval", string(StatePackageExport), "manifest, checksums, redaction, replay/audit pass"},
	}
}

var orderedStates = []StateID{StateQuestionDraft, StateProtocolPlan, StateSourcePlan, StateImportPlan, StateDedupeReview, StateFullTextAcquisition, StateParserArbitration, StateIndexing, StateScreening, StateExtraction, StateAnalysis, StateReportBuild, StatePackageExport, StateDone}
var gateForState = map[StateID]string{StateQuestionDraft: "question approval", StateProtocolPlan: "protocol approval", StateSourcePlan: "network/API approval", StateDedupeReview: "identity approval", StateFullTextAcquisition: "legal acquisition approval", StateParserArbitration: "parser arbitration approval", StateScreening: "screening approval", StateExtraction: "evidence approval", StateAnalysis: "analysis approval", StateReportBuild: "claim approval", StatePackageExport: "package approval"}

func Init(projectPath string, opts InitOptions) (State, error) {
	question := strings.TrimSpace(opts.Question)
	if question == "" {
		return State{}, fmt.Errorf("research question is required")
	}
	if _, err := project.Inspect(projectPath); err != nil {
		if _, err := project.Create(projectPath, project.CreateOptions{Title: "ResearchForge guided review"}); err != nil {
			return State{}, err
		}
	}
	state := State{SchemaVersion: schemaVersion, ProjectPath: projectPath, Manifest: "rforge.project.toml", Question: question, SourceChoices: cleanList(opts.SourceChoices), ToolChoices: cleanList(opts.ToolChoices), PrivacyLegalPreview: privacyLegalPreview(opts.SourceChoices, opts.ToolChoices), CurrentState: StateQuestionDraft}
	state.refresh()
	if err := save(projectPath, state); err != nil {
		return State{}, err
	}
	if err := appendTransition(projectPath, "", state.CurrentState, actor(opts.Actor), map[string]any{"question": question}, state); err != nil {
		return State{}, err
	}
	return state, nil
}

func Status(projectPath string) (State, error) {
	state, err := load(projectPath)
	if err != nil {
		return State{}, err
	}
	state.refresh()
	return state, nil
}

func Next(projectPath, actorName string) (State, error) {
	state, err := Status(projectPath)
	if err != nil {
		return State{}, err
	}
	if len(state.BlockedReviewGates) > 0 {
		return state, fmt.Errorf("blocked review gate: %s", state.BlockedReviewGates[0].Gate)
	}
	return advance(projectPath, state, actor(actorName), map[string]any{"command": "next"})
}

func Approve(projectPath string, input ApprovalInput) (State, error) {
	state, err := Status(projectPath)
	if err != nil {
		return State{}, err
	}
	gate := strings.TrimSpace(input.Gate)
	if gate == "" {
		return State{}, fmt.Errorf("review gate is required")
	}
	required := gateForState[state.CurrentState]
	if required == "" {
		return State{}, fmt.Errorf("state %s has no approvable gate", state.CurrentState)
	}
	if !strings.EqualFold(gate, required) {
		return state, fmt.Errorf("gate %q cannot advance state %s; required gate is %q", gate, state.CurrentState, required)
	}
	state.Approvals = append(state.Approvals, Approval{Gate: required, Note: input.Note, Actor: actor(input.Actor), Timestamp: time.Now().UTC().Format(time.RFC3339)})
	return advance(projectPath, state, actor(input.Actor), map[string]any{"gate": required, "note": input.Note})
}

func CompleteFixtureSourceImport(projectPath, actorName string) (State, error) {
	state, err := Status(projectPath)
	if err != nil {
		return State{}, err
	}
	if state.CurrentState != StateImportPlan {
		return State{}, fmt.Errorf("fixture source import requires state %s; current state is %s", StateImportPlan, state.CurrentState)
	}
	if err := reviewpkg.WriteArtificialPhotosynthesisFixtureSourceImport(projectPath); err != nil {
		return State{}, err
	}
	state.ValidationReceipts = append(state.ValidationReceipts, "offline artificial photosynthesis source plan and imports prepared")
	return advance(projectPath, state, actor(actorName), map[string]any{"sourcePlan": "data/source-plans/artificial-photosynthesis.json", "library": "data/library.json", "importReceipts": "data/import-receipts/fake-sources.json"})
}

func CompleteFixtureReferenceManager(projectPath, actorName string) (State, error) {
	state, err := Status(projectPath)
	if err != nil {
		return State{}, err
	}
	if state.CurrentState != StateDedupeReview {
		return State{}, fmt.Errorf("fixture reference-manager import requires state %s; current state is %s", StateDedupeReview, state.CurrentState)
	}
	if err := reviewpkg.WriteArtificialPhotosynthesisReferenceManagerFixture(projectPath); err != nil {
		return State{}, err
	}
	state.ValidationReceipts = append(state.ValidationReceipts, "offline Zotero/JabRef reference-manager fidelity artifacts prepared")
	state.refresh()
	if err := save(projectPath, state); err != nil {
		return State{}, err
	}
	if err := appendTransition(projectPath, state.CurrentState, state.CurrentState, actor(actorName), map[string]any{"library": "data/library.json", "referenceManagerReports": "data/reference-manager/"}, state); err != nil {
		return State{}, err
	}
	return state, nil
}

func CompleteFixtureAcquisition(projectPath, actorName string) (State, error) {
	state, err := Status(projectPath)
	if err != nil {
		return State{}, err
	}
	if state.CurrentState != StateFullTextAcquisition {
		return State{}, fmt.Errorf("fixture acquisition requires state %s; current state is %s", StateFullTextAcquisition, state.CurrentState)
	}
	if err := reviewpkg.WriteArtificialPhotosynthesisAcquisitionFixture(projectPath); err != nil {
		return State{}, err
	}
	state.ValidationReceipts = append(state.ValidationReceipts, "offline legal acquisition and document asset artifacts prepared")
	return advance(projectPath, state, actor(actorName), map[string]any{"legalAcquisition": "data/legal-acquisition-queue.json", "documentAssets": "data/document-assets.json"})
}

func CompleteFixturePackage(projectPath, packagePath, actorName string) (PackageCompletion, error) {
	state, err := Status(projectPath)
	if err != nil {
		return PackageCompletion{}, err
	}
	if state.CurrentState != StatePackageExport {
		return PackageCompletion{}, fmt.Errorf("fixture package completion requires state %s; current state is %s", StatePackageExport, state.CurrentState)
	}
	if strings.TrimSpace(packagePath) == "" {
		return PackageCompletion{}, fmt.Errorf("package path is required")
	}
	if err := reviewpkg.WriteArtificialPhotosynthesisFixtureProject(projectPath); err != nil {
		return PackageCompletion{}, err
	}
	state.ValidationReceipts = append(state.ValidationReceipts, "offline artificial photosynthesis fixture artifacts prepared")
	state.refresh()
	if err := save(projectPath, state); err != nil {
		return PackageCompletion{}, err
	}
	pkg, err := reviewpkg.Create(projectPath, packagePath, reviewpkg.Options{CreatedBy: actor(actorName), Question: state.Question})
	if err != nil {
		return PackageCompletion{}, err
	}
	audit, err := reviewpkg.Audit(packagePath)
	if err != nil {
		return PackageCompletion{}, err
	}
	if !audit.OK {
		return PackageCompletion{}, fmt.Errorf("package audit failed")
	}
	replay, err := reviewpkg.Replay(packagePath)
	if err != nil {
		return PackageCompletion{}, err
	}
	if !replay.OK {
		return PackageCompletion{}, fmt.Errorf("package replay failed")
	}
	prev := state.CurrentState
	state.CurrentState = StateDone
	state.Approvals = append(state.Approvals, Approval{Gate: "package approval", Note: "fixture package audit and replay passed", Actor: actor(actorName), Timestamp: time.Now().UTC().Format(time.RFC3339)})
	state.ValidationReceipts = append(state.ValidationReceipts, "package audit passed: "+packagePath, "package replay passed: "+packagePath)
	state.refresh()
	if err := save(projectPath, state); err != nil {
		return PackageCompletion{}, err
	}
	if err := appendTransition(projectPath, prev, StateDone, actor(actorName), map[string]any{"package": packagePath, "auditOK": audit.OK, "replayOK": replay.OK}, state); err != nil {
		return PackageCompletion{}, err
	}
	return PackageCompletion{State: state, Package: pkg, AuditReport: audit, ReplayReport: replay, PackagePath: packagePath}, nil
}

func Reopen(projectPath string, target StateID, reason, actorName string) (State, error) {
	if strings.TrimSpace(reason) == "" {
		return State{}, fmt.Errorf("reopen reason is required")
	}
	if !validState(target) {
		return State{}, fmt.Errorf("unknown forge state %q", target)
	}
	state, err := Status(projectPath)
	if err != nil {
		return State{}, err
	}
	prev := state.CurrentState
	state.CurrentState = target
	state.ValidationReceipts = append(state.ValidationReceipts, "reopened: "+reason)
	state.refresh()
	if err := save(projectPath, state); err != nil {
		return State{}, err
	}
	if err := appendTransition(projectPath, prev, target, actor(actorName), map[string]any{"reason": reason}, state); err != nil {
		return State{}, err
	}
	return state, nil
}

func ProvenanceEvents(projectPath string) ([]provenance.Event, error) {
	return provenance.Read(projectPath)
}

func advance(projectPath string, state State, actorName string, inputs map[string]any) (State, error) {
	prev := state.CurrentState
	next, ok := nextState(prev)
	if !ok {
		return state, fmt.Errorf("state %s cannot advance", prev)
	}
	state.CurrentState = next
	state.refresh()
	if err := save(projectPath, state); err != nil {
		return State{}, err
	}
	if err := appendTransition(projectPath, prev, next, actorName, inputs, state); err != nil {
		return State{}, err
	}
	return state, nil
}

func (s State) BlockedBy(gate string) bool {
	for _, blocked := range s.BlockedReviewGates {
		if strings.EqualFold(blocked.Gate, gate) {
			return true
		}
	}
	return false
}

func (s *State) refresh() {
	s.UpdatedAt = time.Now().UTC().Format(time.RFC3339)
	s.BlockedReviewGates = blockedFor(*s)
	s.NextSafeActions = actionsFor(s.CurrentState)
}
func blockedFor(s State) []BlockedGate {
	gate := gateForState[s.CurrentState]
	if gate == "" || approved(s.Approvals, gate) {
		return nil
	}
	spec := ReviewGates()[gate]
	return []BlockedGate{{Gate: spec.Gate, AppliesAt: spec.AppliesAt, RequiredDecision: spec.RequiredDecision}}
}
func approved(approvals []Approval, gate string) bool {
	for _, approval := range approvals {
		if strings.EqualFold(approval.Gate, gate) {
			return true
		}
	}
	return false
}
func nextState(state StateID) (StateID, bool) {
	for i, s := range orderedStates {
		if s == state && i+1 < len(orderedStates) {
			return orderedStates[i+1], true
		}
	}
	return "", false
}
func validState(state StateID) bool {
	for _, s := range orderedStates {
		if s == state {
			return true
		}
	}
	return false
}

func actionsFor(state StateID) []NextAction {
	switch state {
	case StateQuestionDraft:
		return []NextAction{{"Compile protocol", "rforge protocol compile --type pico --question <question>"}, {"Approve canonical question", "rforge forge approve --gate 'question approval' --note <note>"}}
	case StateProtocolPlan:
		return []NextAction{{"Review protocol skeleton", "rforge protocol compile --type pico --question <question>"}, {"Approve protocol", "rforge forge approve --gate 'protocol approval' --note <note>"}}
	case StateSourcePlan:
		return []NextAction{{"Preview source plan", "rforge protocol plan-sources --question <question>"}, {"Approve network/API plan", "rforge forge approve --gate 'network/API approval' --note <note>"}}
	case StateImportPlan:
		return []NextAction{{"Import approved references", "rforge search import --source openalex --query <query>"}}
	case StateDedupeReview:
		return []NextAction{{"Resolve identity decisions", "rforge library identity-resolve"}}
	case StateFullTextAcquisition:
		return []NextAction{{"Review legal acquisition queue", "rforge oa acquisition-queue"}}
	case StateParserArbitration:
		return []NextAction{{"Compare parser outputs", "rforge parse compare --left <a> --right <b>"}}
	case StateIndexing:
		return []NextAction{{"Benchmark retrieval", "rforge retrieve benchmark --out data/retrieval-benchmark.json"}}
	case StateScreening:
		return []NextAction{{"Run screening cockpit", "rforge screen active-run --stage title_abstract --out data/screening-run.json"}}
	case StateExtraction:
		return []NextAction{{"Build evidence grid", "rforge evidence grid --out data/evidence-grid.json"}}
	case StateAnalysis:
		return []NextAction{{"Run analysis", "rforge analysis prepare --effect standardized-mean-difference"}}
	case StateReportBuild:
		return []NextAction{{"Trace report claims", "rforge report trace --out data/report-trace.json"}}
	case StatePackageExport:
		return []NextAction{{"Create and audit package", "rforge package create --out review.rforgepkg && rforge package audit review.rforgepkg"}}
	default:
		return nil
	}
}

func statePath(projectPath string) string {
	return filepath.Join(projectPath, "data", "forge-state.json")
}
func save(projectPath string, state State) error {
	if err := os.MkdirAll(filepath.Dir(statePath(projectPath)), 0o755); err != nil {
		return err
	}
	data, err := json.MarshalIndent(state, "", "  ")
	if err != nil {
		return err
	}
	return os.WriteFile(statePath(projectPath), append(data, '\n'), 0o644)
}
func load(projectPath string) (State, error) {
	var s State
	data, err := os.ReadFile(statePath(projectPath))
	if err != nil {
		return s, err
	}
	err = json.Unmarshal(data, &s)
	return s, err
}
func cleanList(values []string) []string {
	out := []string{}
	seen := map[string]bool{}
	for _, value := range values {
		trimmed := strings.TrimSpace(value)
		if trimmed != "" && !seen[strings.ToLower(trimmed)] {
			seen[strings.ToLower(trimmed)] = true
			out = append(out, trimmed)
		}
	}
	return out
}

func privacyLegalPreview(sources, tools []string) []string {
	preview := []string{"Network/API sources require rate-limit, credential, and cache review before live calls", "Full-text acquisition and package sharing require reviewer approval for OA/license and privacy gates"}
	for _, tool := range cleanList(tools) {
		lower := strings.ToLower(tool)
		if strings.Contains(lower, "qdrant") || strings.Contains(lower, "embedding") {
			preview = append(preview, "Embedding/vector tools require text-egress consent and model/version locks")
		}
		if strings.Contains(lower, "grobid") || strings.Contains(lower, "parser") {
			preview = append(preview, "External parser outputs require parser manifests and arbitration approval")
		}
	}
	return preview
}

func actor(a string) string {
	if strings.TrimSpace(a) == "" {
		return "rforge"
	}
	return strings.TrimSpace(a)
}

func appendTransition(projectPath string, prev, next StateID, actorName string, inputs map[string]any, state State) error {
	now := time.Now().UTC()
	return provenance.Append(projectPath, provenance.Event{SchemaVersion: schemaVersion, ID: "evt_" + now.Format("20060102T150405Z") + "_forge", Timestamp: now.Format(time.RFC3339), Actor: actorName, Action: "forge.state.transition", Target: projectPath, Inputs: inputs, Outputs: map[string]any{"previousState": prev, "nextState": next, "blockedReviewGates": state.BlockedReviewGates, "nextSafeActions": state.NextSafeActions}, Warnings: []string{}})
}
