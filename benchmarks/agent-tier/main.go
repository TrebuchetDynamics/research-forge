// Agent-tier is a custodian/scoring utility, not an LLM runner. No command
// launches a host agent, performs a search, or authorizes live spending.
package main

import (
	"flag"
	"fmt"
	"os"
	"path/filepath"
	"strconv"
	"strings"

	b "github.com/TrebuchetDynamics/research-forge/internal/benchmarks"
)

func makeNewDir(path string) error {
	if path == "" {
		return fmt.Errorf("missing output directory")
	}
	return os.Mkdir(path, 0700)
}

type AgentInput struct {
	QuestionID     string         `json:"question_id"`
	Question       string         `json:"question"`
	Domain         string         `json:"domain"`
	DateCap        string         `json:"date_cap"`
	Caps           Caps           `json:"caps"`
	System         string         `json:"system"`
	Task           string         `json:"task"`
	Tools          string         `json:"tool_instructions"`
	Sampling       Sampling       `json:"sampling"`
	Model          Model          `json:"model"`
	Seed           *int64         `json:"seed"`
	ProviderPolicy ProviderPolicy `json:"provider_policy"`
}

func prepare(setDir, protocolFile, out string) error {
	s, err := b.LoadAgentSet(setDir)
	if err != nil {
		return err
	}
	var p Protocol
	if err = b.ReadAgentArtifact(protocolFile, &p); err != nil {
		return err
	}
	if err = validateProtocol(p, s); err != nil {
		return err
	}
	return prepareFiles(s, p, out)
}
func prepareFiles(s b.Set, p Protocol, out string) error {
	if err := makeNewDir(out); err != nil {
		return err
	}
	if err := save(filepath.Join(out, "protocol.json"), p); err != nil {
		return err
	}
	slots := schedule(p)
	if err := save(filepath.Join(out, "schedule.json"), slots); err != nil {
		return err
	}
	inputs := filepath.Join(out, "inputs")
	if err := makeNewDir(inputs); err != nil {
		return err
	}
	questions := map[string]b.Question{}
	for _, q := range s.Questions {
		questions[q.QuestionID] = q
	}
	for i, slot := range slots {
		q := questions[slot.QuestionID]
		tools := p.Prompts.With
		if slot.Condition == "without_rforge" {
			tools = p.Prompts.Without
		}
		var seed *int64
		if p.Sampling.PairedSeeds {
			v := p.RunOrderSeed + int64(slot.Repetition)
			seed = &v
		}
		input := AgentInput{q.QuestionID, q.Question, q.Domain, q.Review.SearchDate, p.Caps, p.Prompts.System, p.Prompts.Task, tools, p.Sampling, p.Model, seed, p.ProviderPolicy}
		if err := save(filepath.Join(inputs, fmt.Sprintf("%04d.json", i)), input); err != nil {
			return err
		}
	}
	return nil
}
func stub(setDir, out, rateLimitProvider string) error {
	s, err := b.LoadAgentSet(setDir)
	if err != nil {
		return err
	}
	if s.SetID != "fixture-tiny" {
		return fmt.Errorf("stub requires fixture-tiny, never evaluation data")
	}
	secondary := []string{}
	for _, m := range metrics {
		if m != "recall" {
			secondary = append(secondary, m)
		}
	}
	ids := []string{}
	for _, q := range s.Questions {
		if q.Split == "held_out" {
			ids = append(ids, q.QuestionID)
		}
	}
	p := Protocol{Version: 2, Host: Host{"pi", "stub-no-host-execution"}, Model: Model{"stub", "canned-v1"}, Sampling: Sampling{0, 1, true}, Prompts: Prompts{
		System: "Research using only the qualified, date-capped gateway. Do not access other files or previous runs. Log provider failures, HTTP status, Retry-After and stop reasons. After a 429 skip that provider for the remainder of this observation; do not retry or wait. Continue with remaining qualified providers within the same budget.",
		Task:   "Answer the supplied question. Return JSON with an ordered bibliography of identifier arrays and claims containing text, quote and locator. Preserve duplicate bibliography ranks; return partial output at budget stops.",
		With:   "Use the ResearchForge skill and tools through the trusted gateway. Use search batch --sources crossref,semantic-scholar,arxiv --continue-on-error. The gateway must skip throttled providers across all queries and nested HTTP calls; never resume failed pairs during this observation. Retain failures.jsonl, manifest.json and batch-state.json stop reasons. Do not use OpenAlex unless explicitly pre-registered.", Without: "Use only the host native search through the trusted gateway; no ResearchForge binary, plugin, skill or saved outputs. The gateway logs native provider failures and skips any provider after its first 429 for the remainder of this observation.",
	}, RforgeVersion: "fixture", Revision: "canned-v1", SetChecksum: s.SetChecksum, QuestionIDs: ids, Repetitions: 3, Caps: Caps{32000, 20, 60, 100, 600000}, RunOrderSeed: 20261019, ScorerVersion: scorerVersion, CitationVersion: "not_implemented", Primary: []string{"recall"}, Secondary: secondary, ProviderPolicy: ProviderPolicy{[]string{"crossref", "semantic-scholar", "arxiv"}, true, "skip_for_run"}}
	if rateLimitProvider != "" && !containsProvider(p.ProviderPolicy.WithSources, rateLimitProvider) {
		return fmt.Errorf("protocol_violation: stub rate-limit provider is not in the registered mix")
	}
	if err = validateProtocol(p, s); err != nil {
		return err
	}
	if err = prepareFiles(s, p, out); err != nil {
		return err
	}
	e := Experiment{ProtocolSHA256: hash(p), Schedule: schedule(p), Runs: []RawRun{}}
	for i, slot := range e.Schedule {
		// These are canned identifiers, independent of the scorer/gold. A repeated
		// p1 occupies a rank without gaining another hit in either condition.
		papers := []Paper{{[]b.Identifier{{Scheme: "doi", Value: "https://doi.org/10.9999/P1"}}}, {[]b.Identifier{{Scheme: "doi", Value: "10.9999/p1"}}}}
		if slot.Condition == "with_rforge" {
			papers = append(papers, Paper{[]b.Identifier{{Scheme: "doi", Value: "10.9999/p2"}}})
		}
		r := RawRun{Slot: slot, Model: p.Model, Caps: p.Caps, RunID: "private-run-" + strconv.Itoa(i), Transcript: "rforge search; native_search; provenance/events.jsonl; condition=" + slot.Condition, Artifacts: []string{"provenance.json"}, StopReason: "completed", Usage: Usage{100, 50, .001, 1, 2, 2, 100}, Gate: Gate{false, true, true, true, 0}, Answer: Answer{papers, []Claim{{"Synthetic energy conversion has several reported routes.", "synthetic excerpt", "https://example.org/source"}}}}
		r.ProviderEvents = stubProviderEvents(p.ProviderPolicy.WithSources, rateLimitProvider)
		r.Usage.SearchCalls, r.Usage.HTTPRequests = 0, 0
		for _, event := range r.ProviderEvents {
			if event.Action != "skipped" {
				r.Usage.SearchCalls++
				r.Usage.HTTPRequests++
			}
		}
		e.Runs = append(e.Runs, r)
	}
	return save(filepath.Join(out, "experiment.json"), e)
}
func run(args []string) error {
	if len(args) == 0 {
		return fmt.Errorf("usage: agent-tier prepare|stub|blind|score|report [flags]")
	}
	flags := flag.NewFlagSet(args[0], flag.ContinueOnError)
	set := flags.String("set", "", "locked set directory (custodian/scorer only)")
	protocol := flags.String("protocol", "", "pre-registered protocol JSON")
	private := flags.String("private", "", "custodian directory (never mount for scorer)")
	blinded := flags.String("blind", "", "scorer-only answer directory")
	scores := flags.String("scores", "", "frozen scores JSON")
	out := flags.String("out", "", "new output directory/file")
	rateLimit := flags.String("rate-limit-provider", "", "stub only: simulate one provider returning HTTP 429")
	if err := flags.Parse(args[1:]); err != nil {
		return err
	}
	if flags.NArg() != 0 || *out == "" {
		return fmt.Errorf("usage: --out required, no positional arguments")
	}
	if *rateLimit != "" && args[0] != "stub" {
		return fmt.Errorf("--rate-limit-provider is restricted to the offline stub")
	}
	switch args[0] {
	case "stub":
		return stub(*set, *out, *rateLimit)
	case "prepare":
		return prepare(*set, *protocol, *out)
	case "blind":
		return blind(*private, *out)
	case "score":
		if *private != "" || *protocol != "" || *scores != "" {
			return fmt.Errorf("scorer cannot receive private inputs")
		}
		return scoreAnswers(*set, *blinded, *out)
	case "report":
		return report(*set, *private, *blinded, *scores, *out)
	default:
		return fmt.Errorf("unknown command: %s", args[0])
	}
}
func errorCode(err error) string {
	for _, code := range []string{"invalid_set", "checksum_mismatch", "reference_review_leak", "date_cap_unenforceable", "snapshot_mismatch", "source_error"} {
		if strings.Contains(err.Error(), code) {
			return code
		}
	}
	return "protocol_violation"
}
func main() {
	if err := run(os.Args[1:]); err != nil {
		diagnostic, _ := b.CanonicalJSON(map[string]string{"error": errorCode(err)})
		fmt.Fprint(os.Stdout, string(diagnostic))
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}
