package forge

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/TrebuchetDynamics/research-forge/internal/filetxn"
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
	if err := saveTransition(projectPath, "", state.CurrentState, actor(opts.Actor), map[string]any{"question": question}, state); err != nil {
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
	artifacts, err := captureForgeArtifacts(projectPath, reviewpkg.ArtificialPhotosynthesisFixtureSourceImportPaths())
	if err != nil {
		return State{}, err
	}
	rollback := func(cause error) (State, error) {
		if restoreErr := artifacts.restore(); restoreErr != nil {
			return State{}, fmt.Errorf("%w; roll back fixture source import: %v", cause, restoreErr)
		}
		return State{}, cause
	}
	if err := reviewpkg.WriteArtificialPhotosynthesisFixtureSourceImport(projectPath); err != nil {
		return rollback(err)
	}
	state.ValidationReceipts = append(state.ValidationReceipts, "offline artificial photosynthesis source plan and imports prepared")
	state, err = advance(projectPath, state, actor(actorName), map[string]any{"sourcePlan": "data/source-plans/artificial-photosynthesis.json", "library": "data/library.json", "importReceipts": "data/import-receipts/fake-sources.json"})
	if err != nil {
		return rollback(err)
	}
	return state, nil
}

func CompleteFixtureReferenceManager(projectPath, actorName string) (State, error) {
	state, err := Status(projectPath)
	if err != nil {
		return State{}, err
	}
	if state.CurrentState != StateDedupeReview {
		return State{}, fmt.Errorf("fixture reference-manager import requires state %s; current state is %s", StateDedupeReview, state.CurrentState)
	}
	artifacts, err := captureForgeArtifacts(projectPath, reviewpkg.ArtificialPhotosynthesisReferenceManagerFixturePaths())
	if err != nil {
		return State{}, err
	}
	rollback := func(cause error) (State, error) {
		if restoreErr := artifacts.restore(); restoreErr != nil {
			return State{}, fmt.Errorf("%w; roll back fixture reference-manager import: %v", cause, restoreErr)
		}
		return State{}, cause
	}
	if err := reviewpkg.WriteArtificialPhotosynthesisReferenceManagerFixture(projectPath); err != nil {
		return rollback(err)
	}
	state.ValidationReceipts = append(state.ValidationReceipts, "offline Zotero/JabRef reference-manager fidelity artifacts prepared")
	state.refresh()
	if err := saveTransition(projectPath, state.CurrentState, state.CurrentState, actor(actorName), map[string]any{"library": "data/library.json", "referenceManagerReports": "data/reference-manager/"}, state); err != nil {
		return rollback(err)
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
	artifacts, err := captureForgeArtifacts(projectPath, reviewpkg.ArtificialPhotosynthesisAcquisitionFixturePaths())
	if err != nil {
		return State{}, err
	}
	rollback := func(cause error) (State, error) {
		if restoreErr := artifacts.restore(); restoreErr != nil {
			return State{}, fmt.Errorf("%w; roll back fixture acquisition: %v", cause, restoreErr)
		}
		return State{}, cause
	}
	if err := reviewpkg.WriteArtificialPhotosynthesisAcquisitionFixture(projectPath); err != nil {
		return rollback(err)
	}
	state.ValidationReceipts = append(state.ValidationReceipts, "offline legal acquisition and document asset artifacts prepared")
	state, err = advance(projectPath, state, actor(actorName), map[string]any{"legalAcquisition": "data/legal-acquisition-queue.json", "documentAssets": "data/document-assets.json"})
	if err != nil {
		return rollback(err)
	}
	return state, nil
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
	artifacts, err := captureForgeArtifacts(projectPath, reviewpkg.ArtificialPhotosynthesisFixtureProjectPaths())
	if err != nil {
		return PackageCompletion{}, err
	}
	packageOutput, err := captureForgePackageOutput(projectPath, packagePath)
	if err != nil {
		return PackageCompletion{}, err
	}
	defer packageOutput.discard()
	rollback := func(cause error) (PackageCompletion, error) {
		failures := make([]string, 0, 2)
		if restoreErr := packageOutput.restore(); restoreErr != nil {
			failures = append(failures, fmt.Sprintf("package output: %v", restoreErr))
		}
		if restoreErr := artifacts.restore(); restoreErr != nil {
			failures = append(failures, fmt.Sprintf("project artifacts: %v", restoreErr))
		}
		if len(failures) > 0 {
			return PackageCompletion{}, fmt.Errorf("%w; roll back fixture package: %s", cause, strings.Join(failures, "; "))
		}
		return PackageCompletion{}, cause
	}
	if err := reviewpkg.WriteArtificialPhotosynthesisFixtureProject(projectPath); err != nil {
		return rollback(err)
	}
	state.ValidationReceipts = append(state.ValidationReceipts, "offline artificial photosynthesis fixture artifacts prepared")
	state.refresh()
	if err := save(projectPath, state); err != nil {
		return rollback(err)
	}
	pkg, err := reviewpkg.Create(projectPath, packagePath, reviewpkg.Options{CreatedBy: actor(actorName), Question: state.Question})
	if err != nil {
		return rollback(err)
	}
	audit, err := reviewpkg.Audit(packagePath)
	if err != nil {
		return rollback(err)
	}
	if !audit.OK {
		failedChecks := make([]string, 0)
		for _, check := range audit.Checks {
			if !check.OK {
				failedChecks = append(failedChecks, check.Code+": "+check.Message)
			}
		}
		return rollback(fmt.Errorf("package audit failed: %s", strings.Join(failedChecks, "; ")))
	}
	replay, err := reviewpkg.Replay(packagePath)
	if err != nil {
		return rollback(err)
	}
	if !replay.OK {
		return rollback(fmt.Errorf("package replay failed"))
	}
	prev := state.CurrentState
	state.CurrentState = StateDone
	state.Approvals = append(state.Approvals, Approval{Gate: "package approval", Note: "fixture package audit and replay passed", Actor: actor(actorName), Timestamp: time.Now().UTC().Format(time.RFC3339)})
	state.ValidationReceipts = append(state.ValidationReceipts, "package audit passed: "+packagePath, "package replay passed: "+packagePath)
	state.refresh()
	if err := saveTransition(projectPath, prev, StateDone, actor(actorName), map[string]any{"package": packagePath, "auditOK": audit.OK, "replayOK": replay.OK}, state); err != nil {
		return rollback(err)
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
	if err := saveTransition(projectPath, prev, target, actor(actorName), map[string]any{"reason": reason}, state); err != nil {
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
	if err := saveTransition(projectPath, prev, next, actorName, inputs, state); err != nil {
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

type forgeArtifactFileSnapshot struct {
	path    string
	data    []byte
	mode    os.FileMode
	existed bool
}

type forgeArtifactSnapshot struct {
	files   []forgeArtifactFileSnapshot
	newDirs []string
}

type forgePackageOutputSnapshot struct {
	path       string
	backupPath string
	newDirs    []string
}

func captureForgePackageOutput(projectPath, packagePath string) (forgePackageOutputSnapshot, error) {
	if err := reviewpkg.ValidatePackageOutputPath(projectPath, packagePath); err != nil {
		return forgePackageOutputSnapshot{}, err
	}
	path, err := filepath.Abs(packagePath)
	if err != nil {
		return forgePackageOutputSnapshot{}, err
	}
	snapshot := forgePackageOutputSnapshot{path: path}
	for dir := filepath.Dir(path); ; dir = filepath.Dir(dir) {
		info, err := os.Lstat(dir)
		if err == nil {
			if !info.IsDir() {
				return forgePackageOutputSnapshot{}, fmt.Errorf("package output parent is not a directory: %s", dir)
			}
			break
		}
		if !os.IsNotExist(err) {
			return forgePackageOutputSnapshot{}, err
		}
		snapshot.newDirs = append(snapshot.newDirs, dir)
		if parent := filepath.Dir(dir); parent == dir {
			return forgePackageOutputSnapshot{}, fmt.Errorf("package output has no existing parent directory: %s", path)
		}
	}
	if _, err := os.Lstat(path); err != nil {
		if os.IsNotExist(err) {
			return snapshot, nil
		}
		return forgePackageOutputSnapshot{}, err
	}
	backupPath, err := os.MkdirTemp(filepath.Dir(path), "."+filepath.Base(path)+".rforge-rollback-*")
	if err != nil {
		return forgePackageOutputSnapshot{}, err
	}
	if err := os.Remove(backupPath); err != nil {
		return forgePackageOutputSnapshot{}, err
	}
	if err := os.Rename(path, backupPath); err != nil {
		return forgePackageOutputSnapshot{}, err
	}
	snapshot.backupPath = backupPath
	return snapshot, nil
}

func (snapshot forgePackageOutputSnapshot) restore() error {
	if err := os.RemoveAll(snapshot.path); err != nil {
		return err
	}
	if snapshot.backupPath != "" {
		if err := os.Rename(snapshot.backupPath, snapshot.path); err != nil {
			return fmt.Errorf("restore %s from %s: %w", snapshot.path, snapshot.backupPath, err)
		}
	}
	for _, dir := range snapshot.newDirs {
		if err := os.Remove(dir); err != nil && !os.IsNotExist(err) {
			return fmt.Errorf("remove created package directory %s: %w", dir, err)
		}
	}
	return nil
}

func (snapshot forgePackageOutputSnapshot) discard() {
	if snapshot.backupPath != "" {
		_ = os.RemoveAll(snapshot.backupPath)
	}
}

func captureForgeArtifacts(projectPath string, relativePaths []string) (forgeArtifactSnapshot, error) {
	root := filepath.Clean(projectPath)
	snapshot := forgeArtifactSnapshot{files: make([]forgeArtifactFileSnapshot, 0, len(relativePaths))}
	paths := make([]string, 0, len(relativePaths))
	dirs := make([]string, 0)
	seenDirs := map[string]bool{}
	for _, relativePath := range relativePaths {
		relativePath = filepath.Clean(filepath.FromSlash(relativePath))
		if relativePath == "." || filepath.IsAbs(relativePath) || relativePath == ".." || strings.HasPrefix(relativePath, ".."+string(filepath.Separator)) {
			return forgeArtifactSnapshot{}, fmt.Errorf("unsafe fixture artifact path %q", relativePath)
		}
		path := filepath.Join(root, relativePath)
		paths = append(paths, path)
		chain := make([]string, 0)
		for dir := filepath.Dir(path); dir != root; dir = filepath.Dir(dir) {
			if dir == "." || dir == string(filepath.Separator) {
				return forgeArtifactSnapshot{}, fmt.Errorf("fixture artifact escapes project: %s", path)
			}
			chain = append(chain, dir)
		}
		for i := len(chain) - 1; i >= 0; i-- {
			if !seenDirs[chain[i]] {
				seenDirs[chain[i]] = true
				dirs = append(dirs, chain[i])
			}
		}
	}
	for _, dir := range dirs {
		info, err := os.Lstat(dir)
		if err != nil {
			if os.IsNotExist(err) {
				snapshot.newDirs = append(snapshot.newDirs, dir)
				continue
			}
			return forgeArtifactSnapshot{}, err
		}
		if !info.IsDir() {
			return forgeArtifactSnapshot{}, fmt.Errorf("fixture artifact parent is not a directory: %s", dir)
		}
	}
	for _, path := range paths {
		info, err := os.Lstat(path)
		if err != nil {
			if os.IsNotExist(err) {
				snapshot.files = append(snapshot.files, forgeArtifactFileSnapshot{path: path})
				continue
			}
			return forgeArtifactSnapshot{}, err
		}
		if !info.Mode().IsRegular() {
			return forgeArtifactSnapshot{}, fmt.Errorf("fixture artifact path is not a regular file: %s", path)
		}
		data, err := os.ReadFile(path)
		if err != nil {
			return forgeArtifactSnapshot{}, err
		}
		snapshot.files = append(snapshot.files, forgeArtifactFileSnapshot{path: path, data: data, mode: info.Mode(), existed: true})
	}
	return snapshot, nil
}

func (snapshot forgeArtifactSnapshot) restore() error {
	failures := make([]string, 0)
	for i := len(snapshot.files) - 1; i >= 0; i-- {
		file := snapshot.files[i]
		var err error
		if file.existed {
			err = filetxn.Replace(file.path, file.data, file.mode)
		} else {
			err = os.Remove(file.path)
			if os.IsNotExist(err) {
				err = nil
			}
		}
		if err != nil {
			failures = append(failures, fmt.Sprintf("restore %s: %v", file.path, err))
		}
	}
	for i := len(snapshot.newDirs) - 1; i >= 0; i-- {
		if err := os.Remove(snapshot.newDirs[i]); err != nil && !os.IsNotExist(err) {
			failures = append(failures, fmt.Sprintf("remove created directory %s: %v", snapshot.newDirs[i], err))
		}
	}
	if len(failures) > 0 {
		return fmt.Errorf("restore fixture artifacts: %s", strings.Join(failures, "; "))
	}
	return nil
}

type forgeStateSnapshot struct {
	path           string
	data           []byte
	mode           os.FileMode
	existed        bool
	createdDataDir bool
}

func captureForgeState(projectPath string) (forgeStateSnapshot, error) {
	path := statePath(projectPath)
	snapshot := forgeStateSnapshot{path: path}
	dataDir := filepath.Dir(path)
	dirInfo, err := os.Lstat(dataDir)
	if err != nil {
		if !os.IsNotExist(err) {
			return forgeStateSnapshot{}, err
		}
		snapshot.createdDataDir = true
	} else if !dirInfo.IsDir() {
		return forgeStateSnapshot{}, fmt.Errorf("forge data path is not a directory: %s", dataDir)
	}
	info, err := os.Lstat(path)
	if err != nil {
		if os.IsNotExist(err) {
			return snapshot, nil
		}
		return forgeStateSnapshot{}, err
	}
	if !info.Mode().IsRegular() {
		return forgeStateSnapshot{}, fmt.Errorf("forge state path is not a regular file: %s", path)
	}
	data, err := os.ReadFile(path)
	if err != nil {
		return forgeStateSnapshot{}, err
	}
	snapshot.data = data
	snapshot.mode = info.Mode()
	snapshot.existed = true
	return snapshot, nil
}

func (snapshot forgeStateSnapshot) restore() error {
	if snapshot.existed {
		return filetxn.Replace(snapshot.path, snapshot.data, snapshot.mode)
	}
	if err := os.Remove(snapshot.path); err != nil && !os.IsNotExist(err) {
		return err
	}
	if snapshot.createdDataDir {
		if err := os.Remove(filepath.Dir(snapshot.path)); err != nil && !os.IsNotExist(err) {
			return err
		}
	}
	return nil
}

func saveTransition(projectPath string, prev, next StateID, actorName string, inputs map[string]any, state State) error {
	snapshot, err := captureForgeState(projectPath)
	if err != nil {
		return err
	}
	rollback := func(cause error) error {
		if restoreErr := snapshot.restore(); restoreErr != nil {
			return fmt.Errorf("%w; roll back forge state: %v", cause, restoreErr)
		}
		return cause
	}
	if err := save(projectPath, state); err != nil {
		return rollback(err)
	}
	if err := appendTransition(projectPath, prev, next, actorName, inputs, state); err != nil {
		return rollback(err)
	}
	return nil
}

func save(projectPath string, state State) error {
	path := statePath(projectPath)
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		return err
	}
	mode := os.FileMode(0o644)
	if info, err := os.Lstat(path); err == nil {
		if !info.Mode().IsRegular() {
			return fmt.Errorf("forge state path is not a regular file: %s", path)
		}
		mode = info.Mode().Perm()
	} else if !os.IsNotExist(err) {
		return err
	}
	data, err := json.MarshalIndent(state, "", "  ")
	if err != nil {
		return err
	}
	return filetxn.Replace(path, append(data, '\n'), mode)
}
func load(projectPath string) (State, error) {
	var s State
	data, err := os.ReadFile(statePath(projectPath))
	if err != nil {
		return s, err
	}
	if err := json.Unmarshal(data, &s); err != nil {
		return State{}, err
	}
	if s.SchemaVersion != schemaVersion {
		return State{}, fmt.Errorf("unsupported forge schema version %q", s.SchemaVersion)
	}
	if !validState(s.CurrentState) {
		return State{}, fmt.Errorf("invalid forge state %q", s.CurrentState)
	}
	return s, nil
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
