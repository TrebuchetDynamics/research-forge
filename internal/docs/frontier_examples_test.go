package docs

import (
	"bytes"
	"crypto/sha256"
	"encoding/csv"
	"encoding/json"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"reflect"
	"regexp"
	"strings"
	"testing"
	"time"

	"github.com/TrebuchetDynamics/research-forge/internal/cli"
	"github.com/TrebuchetDynamics/research-forge/internal/filetxn"
	"github.com/TrebuchetDynamics/research-forge/internal/provenance"
)

const frontierRoot = "research/frontier-science"

// These guards establish discovery and mechanical consistency, NOT whether a
// source supports a scientific inference or whether a human approved a claim.
func TestFrontierExamplesDiscoverable(t *testing.T) {
	root := filepath.Join("..", "..")
	var manifest struct {
		SchemaVersion int               `json:"schema_version"`
		Examples      []frontierExample `json:"examples"`
	}
	if err := frontierJSON(root, frontierRoot+"/examples.json", &manifest); err != nil {
		t.Fatalf("missing scientific example discovery: %v", err)
	}
	selected, err := frontierLatest(root)
	if err != nil {
		t.Fatal(err)
	}
	if manifest.SchemaVersion != 1 || !reflect.DeepEqual(manifest.Examples, selected) {
		t.Fatalf("manifest must link the latest qualifying campaign: want %+v, got %+v", selected, manifest)
	}
	readme := readContractFile(t, root, "README.md")
	index := readContractFile(t, root, frontierRoot+"/README.md")
	if !strings.Contains(readme, "## Scientific research examples") || !strings.Contains(readme, "./"+frontierRoot+"/README.md") {
		t.Fatal("root README must expose the scientific examples index")
	}
	for i, example := range selected {
		label := []string{"Majorana", "Nickelates"}[i]
		depth := []string{"Comprehensive", "Standard"}[i]
		// Labels and links must belong to the same discovery entry.
		linked := false
		for _, line := range strings.Split(readme, "\n") {
			if strings.Contains(line, label) && strings.Contains(line, depth) && strings.Contains(line, "("+"./"+example.Report+")") {
				linked = true
			}
		}
		if !linked {
			t.Errorf("missing direct %s/%s report link", label, depth)
		}
		for _, path := range example.paths() {
			rel, err := filepath.Rel(frontierRoot, path)
			if err != nil {
				t.Fatal(err)
			}
			if !strings.Contains(index, "("+filepath.ToSlash(rel)+")") {
				t.Errorf("index does not link %s", path)
			}
		}
		for _, document := range []string{example.Report, example.Reproduce} {
			text := readContractFile(t, root, document)
			for _, match := range regexp.MustCompile(`\]\(([^ )]+)\)`).FindAllStringSubmatch(text, -1) {
				if strings.Contains(match[1], ":") || strings.HasPrefix(match[1], "#") {
					continue
				}
				target := strings.SplitN(match[1], "#", 2)[0]
				if _, err := frontierRead(root, filepath.ToSlash(filepath.Join(filepath.Dir(document), target))); err != nil {
					t.Errorf("%s: %v", document, err)
				}
			}
		}
	}
}

type frontierExample struct {
	ID            string `json:"id"`
	Depth         string `json:"depth"`
	CampaignDir   string `json:"campaign_dir"`
	Report        string `json:"report"`
	Evidence      string `json:"evidence"`
	Provenance    string `json:"provenance"`
	CitationAudit string `json:"citation_audit"`
	Reproduce     string `json:"reproduce"`
}

func (e frontierExample) paths() []string {
	return []string{e.Report, e.Evidence, e.Provenance, e.CitationAudit, e.Reproduce}
}

func frontierExamples(dir string) []frontierExample {
	var examples []frontierExample
	for i, id := range []string{"majorana", "nickelates"} {
		base := dir + "/" + id + "/"
		examples = append(examples, frontierExample{id, []string{"comprehensive", "standard"}[i], dir,
			base + "report.md", base + "evidence.csv", base + "provenance.json", base + "citation-audit.json", base + "reproduce.md"})
	}
	return examples
}

func frontierLatest(root string) ([]frontierExample, error) {
	entries, err := os.ReadDir(filepath.Join(root, frontierRoot))
	if err != nil {
		return nil, err
	}
	var latest time.Time
	var selected string
	var rejected []string
	for _, entry := range entries {
		if !entry.IsDir() {
			continue
		}
		dir := frontierRoot + "/" + entry.Name()
		started, err := frontierCampaign(root, dir)
		if err != nil {
			rejected = append(rejected, dir+": "+err.Error())
			continue
		} // incomplete campaigns never displace a complete one
		if started.After(latest) || started.Equal(latest) && dir > selected {
			latest, selected = started, dir
		}
	}
	if selected == "" {
		return nil, fmt.Errorf("no qualifying genuine two-topic campaign; needs V-FS2: %s", strings.Join(rejected, "; "))
	}
	return frontierExamples(selected), nil
}

func frontierCampaign(root, dir string) (time.Time, error) {
	var run struct {
		CampaignID string `json:"campaign_id"`
		StartedAt  string `json:"started_at"`
		Purpose    string `json:"purpose"`
		Synthetic  bool   `json:"synthetic"`
		Topics     map[string]struct {
			Status          string `json:"status"`
			SearchAttempted bool   `json:"search_attempted"`
		} `json:"topics"`
	}
	if err := frontierJSON(root, dir+"/run.json", &run); err != nil {
		return time.Time{}, err
	}
	started, err := time.Parse(time.RFC3339Nano, run.StartedAt)
	_, offset := started.Zone()
	if err != nil || offset != 0 || run.CampaignID != filepath.Base(dir) || run.Purpose != "exploratory_not_gold" || run.Synthetic {
		return time.Time{}, fmt.Errorf("invalid/synthetic campaign identity: %s", dir)
	}
	var findings []json.RawMessage
	if err := frontierJSON(root, dir+"/findings.json", &findings); err != nil {
		return time.Time{}, err
	}
	for _, e := range frontierExamples(dir) {
		state := run.Topics[e.ID]
		if state.Status != "complete" || !state.SearchAttempted {
			return time.Time{}, fmt.Errorf("%s: not a complete real search", e.ID)
		}
		if err := frontierTopic(root, e); err != nil {
			return time.Time{}, fmt.Errorf("%s: %w", e.ID, err)
		}
	}
	return started, nil
}

func frontierTopic(root string, e frontierExample) error {
	base := e.CampaignDir + "/" + e.ID + "/"
	for _, path := range e.paths() {
		if _, err := frontierRead(root, path); err != nil {
			return err
		}
	}
	var index struct {
		Schema  string              `json:"schema_version"`
		Level   string              `json:"provenance_level"`
		Depth   string              `json:"depth"`
		Runs    []string            `json:"runs"`
		Outputs []provenance.Output `json:"outputs"`
	}
	if err := frontierJSON(root, e.Provenance, &index); err != nil {
		return err
	}
	if index.Schema != "2" || index.Level != "agent_reported" || index.Depth != e.Depth || len(index.Runs) == 0 {
		return fmt.Errorf("missing real runs or wrong synthesis attribution/depth")
	}
	indexed := map[string]bool{}
	for _, output := range index.Outputs {
		data, err := frontierRead(filepath.Join(root, base), output.Path)
		if err != nil {
			return err
		}
		if fmt.Sprintf("%x", sha256.Sum256(data)) != output.SHA256 {
			return fmt.Errorf("changed output: %s", output.Path)
		}
		indexed[output.Path] = true
	}
	for _, name := range []string{"report.md", "evidence.csv", "citation-audit.json", "reproduce.md", "primary-checks.json", "manifest.json", "results-deduped.jsonl", "provenance/events.jsonl"} {
		if !indexed[name] {
			return fmt.Errorf("unindexed required artifact: %s", name)
		}
	}
	// Invoke the real engine validator, not a hand-written receipt approximation.
	var stdout, stderr bytes.Buffer
	if cli.Execute([]string{"--json", "provenance", "validate", filepath.Join(root, e.Provenance)}, &stdout, &stderr) != 0 {
		return fmt.Errorf("engine provenance validation: %s %s", &stdout, &stderr)
	}
	var events []provenance.Event
	eventData, err := frontierRead(root, base+"provenance/events.jsonl")
	if err != nil {
		return err
	}
	decoder := json.NewDecoder(bytes.NewReader(eventData))
	for decoder.More() {
		var event provenance.Event
		if err := decoder.Decode(&event); err != nil {
			return err
		}
		events = append(events, event)
	}
	searched := false
	for _, id := range index.Runs {
		name := "provenance/runs/" + id + ".json"
		if !indexed[name] {
			return fmt.Errorf("unindexed engine identity: %s", id)
		}
		var receipt struct {
			Interrupted bool                `json:"interrupted"`
			Errors      []string            `json:"errors"`
			Outputs     []provenance.Output `json:"outputs"`
		}
		data, err := frontierRead(root, base+name)
		if err != nil {
			return err
		}
		if err := json.Unmarshal(data, &receipt); err != nil {
			return err
		}
		digest := fmt.Sprintf("%x", sha256.Sum256(data))
		for _, event := range events {
			_, timestampErr := time.Parse(time.RFC3339, event.Timestamp)
			// The search action must belong to the checksum-bound engine event,
			// not an unrelated event borrowing a validated receipt's run ID.
			boundSearch := event.RunID == id && event.Action == "search.batch" && event.ActorKind == "engine" && event.Target == name && event.Outputs["record_sha256"] == digest && event.SchemaVersion == "1" && strings.TrimSpace(event.ID) != "" && strings.TrimSpace(event.Actor) != "" && event.Inputs != nil && event.Warnings != nil && timestampErr == nil
			if boundSearch && !receipt.Interrupted && len(receipt.Errors) == 0 {
				for _, output := range receipt.Outputs {
					if output.Path == "results-deduped.jsonl" {
						searched = true
					}
				}
			}
		}
	}
	if !searched {
		return fmt.Errorf("no successful engine search event/receipt")
	}
	var manifest struct {
		Budget struct {
			Depth string `json:"depth"`
		} `json:"budget"`
		StopReason   string `json:"stop_reason"`
		RequestsUsed int    `json:"requests_used"`
	}
	if err := frontierJSON(root, base+"manifest.json", &manifest); err != nil {
		return err
	}
	if manifest.Budget.Depth != e.Depth || manifest.RequestsUsed <= 0 || !frontierOneOf(manifest.StopReason, "completed", "budget_exhausted", "coverage_saturated") {
		return fmt.Errorf("missing terminal search/depth")
	}
	results, err := frontierRead(root, base+"results-deduped.jsonl")
	if err != nil {
		return err
	}
	if len(bytes.TrimSpace(results)) == 0 {
		return fmt.Errorf("empty engine results")
	}
	return frontierAudit(root, base, e)
}

func frontierAudit(root, base string, e frontierExample) error {
	report, err := frontierRead(root, e.Report)
	if err != nil {
		return err
	}
	evidence, err := frontierRead(root, e.Evidence)
	if err != nil {
		return err
	}
	rows, err := csv.NewReader(bytes.NewReader(evidence)).ReadAll()
	if err != nil {
		return err
	}
	if len(rows) < 2 {
		return fmt.Errorf("empty evidence")
	}
	columns := map[string]int{}
	for i, name := range rows[0] {
		columns[name] = i
	}
	for _, name := range []string{"source_id", "source_locator", "quote_location"} {
		if _, ok := columns[name]; !ok {
			return fmt.Errorf("missing evidence column: %s", name)
		}
	}
	var audit struct {
		Topic      string `json:"topic"`
		AssessedBy struct {
			Kind     string `json:"kind"`
			Identity string `json:"identity"`
		} `json:"assessed_by"`
		EngineRunIDs []string `json:"engine_run_ids"`
		Claims       []struct {
			ID          string   `json:"id"`
			Anchor      string   `json:"report_anchor"`
			Claim       string   `json:"claim"`
			Rows        []int    `json:"evidence_rows"`
			SourceID    string   `json:"source_id"`
			Locator     string   `json:"source_locator"`
			Location    string   `json:"source_location"`
			Status      string   `json:"source_status"`
			Support     string   `json:"support"`
			Rationale   string   `json:"rationale"`
			Limitations []string `json:"access_limitations"`
		} `json:"claims"`
	}
	if err := frontierJSON(root, e.CitationAudit, &audit); err != nil {
		return err
	}
	var index struct {
		Runs []string `json:"runs"`
	}
	if err := frontierJSON(root, e.Provenance, &index); err != nil {
		return err
	}
	if audit.Topic != e.ID || audit.AssessedBy.Kind != "host_agent" || audit.AssessedBy.Identity == "" || !reflect.DeepEqual(audit.EngineRunIDs, index.Runs) || len(audit.Claims) == 0 {
		return fmt.Errorf("missing/wrong-topic host audit or run identities")
	}
	var primary struct {
		Records []struct {
			ID       string   `json:"source_id"`
			URL      string   `json:"url"`
			Title    []string `json:"title"`
			Authors  []string `json:"authors"`
			Date     []string `json:"date"`
			DOI      []string `json:"doi"`
			ArxivID  []string `json:"arxiv_id"`
			Excerpt  string   `json:"short_excerpt"`
			Location string   `json:"excerpt_location"`
		} `json:"records"`
	}
	if err := frontierJSON(root, base+"primary-checks.json", &primary); err != nil {
		return err
	}
	sources := map[string]string{}
	for _, record := range primary.Records {
		if record.ID == "" || len(record.Title) == 0 || len(record.Authors) == 0 || len(record.Date) == 0 || len(record.DOI)+len(record.ArxivID) == 0 || record.Excerpt == "" || record.Location == "" || !strings.Contains(string(report), "**"+record.ID+"**") || !strings.Contains(string(report), record.URL) {
			return fmt.Errorf("missing primary identity/passage/reference: %s", record.ID)
		}
		sources[record.ID] = record.URL
	}
	anchors := map[string]bool{}
	for _, match := range regexp.MustCompile(`<a id="([^"]+)"`).FindAllSubmatch(report, -1) {
		id := string(match[1])
		if anchors[id] {
			return fmt.Errorf("duplicate anchor: %s", id)
		}
		anchors[id] = true
	}
	seen, used := map[string]bool{}, map[int]bool{}
	scientificSource := false
	for _, claim := range audit.Claims {
		if claim.ID == "" || seen[claim.ID] || !anchors[claim.Anchor] || strings.TrimSpace(claim.Claim) == "" || claim.SourceID == "" || claim.Locator == "" || claim.Location == "" || claim.Rationale == "" || len(claim.Rows) == 0 || len(claim.Limitations) == 0 || !frontierOneOf(claim.Support, "supported", "disputed", "insufficient") || !frontierOneOf(claim.Status, "published", "preprint", "corrected", "retracted", "unknown") {
			return fmt.Errorf("invalid audit claim/anchor: %s", claim.ID)
		}
		seen[claim.ID] = true
		if claim.SourceID == "workflow" {
			if claim.Locator != "artifact:"+e.ID+"/manifest.json#coverage" {
				return fmt.Errorf("unresolved workflow reference")
			}
		} else {
			if sources[claim.SourceID] != claim.Locator {
				return fmt.Errorf("unresolved primary source: %s", claim.SourceID)
			}
			scientificSource = true
		}
		for _, number := range claim.Rows {
			if number < 1 || number >= len(rows) {
				return fmt.Errorf("invalid evidence row: %d", number)
			}
			row := rows[number]
			if row[columns["source_id"]] != claim.SourceID || row[columns["source_locator"]] != claim.Locator || row[columns["quote_location"]] == "" {
				return fmt.Errorf("wrong evidence reference: %s", claim.ID)
			}
			used[number] = true
		}
	}
	if len(used) != len(rows)-1 {
		return fmt.Errorf("unaudited evidence rows")
	}
	if !scientificSource {
		return fmt.Errorf("no scientific-source evidence")
	}
	return nil
}

// All temporary dossiers below are SYNTHETIC VALIDATOR INPUT, never scientific
// examples. Even perfect metadata cannot establish that a run or claim is real.
func TestFrontierExamplesRejectIncompleteOrSynthetic(t *testing.T) {
	cases := []struct {
		name   string
		change func(*testing.T, string, string)
		want   string
	}{
		{"synthetic", func(t *testing.T, r, d string) {
			frontierMutate(t, r, d+"/run.json", func(m map[string]any) { m["synthetic"] = true })
		}, "synthetic"},
		{"partial", func(t *testing.T, r, d string) {
			frontierMutate(t, r, d+"/run.json", func(m map[string]any) {
				m["topics"].(map[string]any)["majorana"].(map[string]any)["status"] = "partial"
			})
		}, "not a complete"},
		{"unattempted", func(t *testing.T, r, d string) {
			frontierMutate(t, r, d+"/run.json", func(m map[string]any) {
				m["topics"].(map[string]any)["nickelates"].(map[string]any)["search_attempted"] = false
			})
		}, "not a complete"},
		{"missing-run", func(t *testing.T, r, d string) {
			frontierMutate(t, r, d+"/majorana/provenance.json", func(m map[string]any) { m["runs"] = []string{} })
		}, "missing real runs"},
		{"missing-receipt", func(t *testing.T, r, d string) {
			files, err := filepath.Glob(filepath.Join(r, d, "majorana/provenance/runs/*.json"))
			if err != nil {
				t.Fatal(err)
			}
			if err := os.Remove(files[0]); err != nil {
				t.Fatal(err)
			}
		}, "no such file"},
		{"missing-audit", func(t *testing.T, r, d string) {
			if err := os.Remove(filepath.Join(r, d, "majorana/citation-audit.json")); err != nil {
				t.Fatal(err)
			}
		}, "no such file"},
		{"changed-output", func(t *testing.T, r, d string) { frontierPut(t, r, d+"/majorana/results-deduped.jsonl", "changed") }, "changed output"},
		{"failed-engine", func(t *testing.T, r, d string) {
			base := d + "/majorana/"
			var index struct {
				Runs []string `json:"runs"`
			}
			if err := frontierJSON(r, base+"provenance.json", &index); err != nil {
				t.Fatal(err)
			}
			name := base + "provenance/runs/" + index.Runs[0] + ".json"
			if err := os.Chmod(filepath.Join(r, name), 0o644); err != nil {
				t.Fatal(err)
			}
			frontierMutate(t, r, name, func(m map[string]any) { m["interrupted"] = true })
			data, err := frontierRead(r, name)
			if err != nil {
				t.Fatal(err)
			}
			// Rebind this synthetic event so failure is due to run state, not a
			// stale hash. Never rewrite a real immutable receipt this way.
			frontierMutate(t, r, base+"provenance/events.jsonl", func(m map[string]any) {
				m["outputs"].(map[string]any)["record_sha256"] = fmt.Sprintf("%x", sha256.Sum256(data))
			})
			frontierSeal(t, r, d+"/majorana")
		}, "no successful engine search"},
		{"missing-event", func(t *testing.T, r, d string) {
			frontierPut(t, r, d+"/majorana/provenance/events.jsonl", "")
			frontierSeal(t, r, d+"/majorana")
		}, "engine provenance validation"},
		{"not-a-search", func(t *testing.T, r, d string) {
			frontierMutate(t, r, d+"/majorana/provenance/events.jsonl", func(m map[string]any) { m["action"] = "citations.expand" })
			frontierSeal(t, r, d+"/majorana")
		}, "no successful engine search"},
		{"unbound-search-event", func(t *testing.T, r, d string) {
			name := d + "/majorana/provenance/events.jsonl"
			var id string
			frontierMutate(t, r, name, func(m map[string]any) { id = m["run_id"].(string); m["action"] = "citations.expand" })
			data, err := frontierRead(r, name)
			if err != nil {
				t.Fatal(err)
			}
			fake, err := json.Marshal(map[string]string{"run_id": id, "action": "search.batch"})
			if err != nil {
				t.Fatal(err)
			}
			spoofs := string(data) + "\n" + string(fake) + "\n"
			// Also try fully bound but engine-invalid search events. Neither
			// whitespace-only identity field can borrow the non-search binding.
			for _, field := range []string{"id", "actor"} {
				var event map[string]any
				if err := json.Unmarshal(data, &event); err != nil {
					t.Fatal(err)
				}
				event["action"], event[field] = "search.batch", " "
				encoded, err := json.Marshal(event)
				if err != nil {
					t.Fatal(err)
				}
				spoofs += string(encoded) + "\n"
			}
			frontierPut(t, r, name, spoofs)
			frontierSeal(t, r, d+"/majorana")
		}, "no successful engine search"},
		{"workflow-only-evidence", func(t *testing.T, r, d string) {
			base := d + "/majorana/"
			frontierPutJSON(t, r, base+"primary-checks.json", map[string]any{"records": []any{}})
			frontierPut(t, r, base+"evidence.csv", "source_id,source_locator,quote_location\nworkflow,artifact:majorana/manifest.json#coverage,manifest\n")
			frontierMutate(t, r, base+"citation-audit.json", func(m map[string]any) {
				claim := m["claims"].([]any)[0].(map[string]any)
				claim["source_id"], claim["source_locator"] = "workflow", "artifact:majorana/manifest.json#coverage"
			})
			frontierSeal(t, r, d+"/majorana")
		}, "no scientific-source evidence"},
		{"wrong-topic", func(t *testing.T, r, d string) {
			frontierMutate(t, r, d+"/majorana/citation-audit.json", func(m map[string]any) { m["topic"] = "nickelates" })
			frontierSeal(t, r, d+"/majorana")
		}, "wrong-topic"},
		{"wrong-audit-run", func(t *testing.T, r, d string) {
			frontierMutate(t, r, d+"/majorana/citation-audit.json", func(m map[string]any) { m["engine_run_ids"] = []string{"absent"} })
			frontierSeal(t, r, d+"/majorana")
		}, "run identities"},
		{"empty-evidence", func(t *testing.T, r, d string) {
			frontierPut(t, r, d+"/majorana/evidence.csv", "source_id,source_locator,quote_location\n")
			frontierSeal(t, r, d+"/majorana")
		}, "empty evidence"},
		{"missing-evidence-row", func(t *testing.T, r, d string) { frontierMutateClaim(t, r, d, "evidence_rows", []int{2}) }, "invalid evidence row"},
		{"wrong-evidence-source", func(t *testing.T, r, d string) {
			frontierPut(t, r, d+"/majorana/evidence.csv", "source_id,source_locator,quote_location\nother,https://example.org/synthetic,abstract\n")
			frontierSeal(t, r, d+"/majorana")
		}, "wrong evidence reference"},
		{"unresolved-anchor", func(t *testing.T, r, d string) { frontierMutateClaim(t, r, d, "report_anchor", "absent") }, "claim/anchor"},
		{"unresolved-source", func(t *testing.T, r, d string) {
			frontierMutateClaim(t, r, d, "source_locator", "https://example.org/absent")
		}, "unresolved primary source"},
		{"traversal", func(t *testing.T, r, d string) {
			frontierMutate(t, r, d+"/majorana/provenance.json", func(m map[string]any) { m["outputs"].([]any)[0].(map[string]any)["path"] = "../../outside" })
		}, "unsafe path"},
		{"absolute-path", func(t *testing.T, r, d string) {
			frontierMutate(t, r, d+"/majorana/provenance.json", func(m map[string]any) { m["outputs"].([]any)[0].(map[string]any)["path"] = filepath.Join(r, "outside") })
		}, "unsafe path"},
		{"symlink-escape", func(t *testing.T, r, d string) {
			path := filepath.Join(r, d, "majorana/report.md")
			if err := os.Remove(path); err != nil {
				t.Fatal(err)
			}
			frontierPut(t, r, "outside.md", "not a source")
			if err := os.Symlink(filepath.Join(r, "outside.md"), path); err != nil {
				t.Fatal(err)
			}
		}, "report.md"},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			root := t.TempDir()
			dir := frontierFixture(t, root, "synthetic-validator-input", "2026-10-21T00:00:00Z")
			if _, err := frontierCampaign(root, dir); err != nil {
				t.Fatalf("valid synthetic validator input: %v", err)
			}
			tc.change(t, root, dir)
			if _, err := frontierCampaign(root, dir); err == nil || !strings.Contains(err.Error(), tc.want) {
				t.Fatalf("want rejection containing %q, got %v", tc.want, err)
			}
			if _, err := frontierLatest(root); err == nil {
				t.Fatal("invalid campaign selected for promotion")
			}
		})
	}
	t.Run("latest-qualifying-UTC-then-campaign-id", func(t *testing.T) {
		root := t.TempDir()
		frontierFixture(t, root, "a-newer", "2026-10-22T00:00:00+00:00")
		frontierFixture(t, root, "z-older", "2026-10-21T00:00:00Z")
		want := frontierFixture(t, root, "b-newer", "2026-10-22T00:00:00Z")
		bad := frontierFixture(t, root, "c-incomplete", "2026-10-23T00:00:00Z")
		frontierPut(t, root, bad+"/run.json", "{}")
		got, err := frontierLatest(root)
		if err != nil || !reflect.DeepEqual(got, frontierExamples(want)) {
			t.Fatalf("wrong latest qualifying campaign: %+v, %v", got, err)
		}
	})
}

func frontierFixture(t *testing.T, root, id, started string) string {
	t.Helper()
	dir := frontierRoot + "/" + id
	frontierPutJSON(t, root, dir+"/run.json", map[string]any{"campaign_id": id, "started_at": started, "purpose": "exploratory_not_gold", "topics": map[string]any{
		"majorana": map[string]any{"status": "complete", "search_attempted": true}, "nickelates": map[string]any{"status": "complete", "search_attempted": true},
	}})
	frontierPut(t, root, dir+"/findings.json", "[]")
	for _, example := range frontierExamples(dir) {
		base := dir + "/" + example.ID
		frontierPut(t, root, base+"/report.md", "SYNTHETIC VALIDATOR INPUT ONLY\n<a id=\"test\"></a>**test-source** https://example.org/synthetic\n")
		frontierPut(t, root, base+"/evidence.csv", "source_id,source_locator,quote_location\ntest-source,https://example.org/synthetic,abstract\n")
		frontierPut(t, root, base+"/reproduce.md", "SYNTHETIC; no scientific result or live command")
		frontierPut(t, root, base+"/results-deduped.jsonl", "{\"synthetic\":true}\n")
		frontierPutJSON(t, root, base+"/manifest.json", map[string]any{"budget": map[string]any{"depth": example.Depth}, "requests_used": 1, "stop_reason": "completed"})
		meta := map[string]any{"question": "Synthetic validator input, not research", "depth": example.Depth, "sources": []string{"synthetic"}, "errors": []string{}, "rforge_version": map[string]string{"version": "test", "commit": "test", "date": "test"}}
		run, err := provenance.WriteRun(filepath.Join(root, base), "search.batch", meta, []string{filepath.Join(root, base, "results-deduped.jsonl"), filepath.Join(root, base, "manifest.json")})
		if err != nil {
			t.Fatal(err)
		}
		frontierPutJSON(t, root, base+"/primary-checks.json", map[string]any{"records": []any{map[string]any{"source_id": "test-source", "url": "https://example.org/synthetic", "title": []string{"Synthetic input"}, "authors": []string{"Test Author"}, "date": []string{"2026"}, "doi": []string{"test-only"}, "short_excerpt": "Synthetic passage", "excerpt_location": "abstract"}}})
		frontierPutJSON(t, root, base+"/citation-audit.json", map[string]any{"topic": example.ID, "assessed_by": map[string]string{"kind": "host_agent", "identity": "synthetic test input"}, "engine_run_ids": []string{run.RunID}, "claims": []any{map[string]any{
			"id": "test-claim", "report_anchor": "test", "claim": "Synthetic input", "evidence_rows": []int{1}, "source_id": "test-source", "source_locator": "https://example.org/synthetic", "source_location": "abstract", "source_status": "unknown", "support": "insufficient", "rationale": "Validator test only", "access_limitations": []string{"No scientific source"},
		}}})
		meta["schema_version"], meta["provenance_level"], meta["timestamp"], meta["runs"] = "2", "agent_reported", started, []string{run.RunID}
		outputs := []provenance.Output{}
		for _, name := range []string{"report.md", "evidence.csv", "reproduce.md", "results-deduped.jsonl", "manifest.json", "primary-checks.json", "citation-audit.json", "provenance/events.jsonl", "provenance/runs/" + run.RunID + ".json"} {
			outputs = append(outputs, provenance.Output{Path: name})
		}
		meta["outputs"] = outputs
		frontierPutJSON(t, root, base+"/provenance.json", meta)
		frontierSeal(t, root, base)
	}
	return dir
}

func frontierMutateClaim(t *testing.T, root, dir, key string, value any) {
	t.Helper()
	frontierMutate(t, root, dir+"/majorana/citation-audit.json", func(m map[string]any) { m["claims"].([]any)[0].(map[string]any)[key] = value })
	frontierSeal(t, root, dir+"/majorana")
}

func frontierSeal(t *testing.T, root, base string) {
	t.Helper()
	frontierMutate(t, root, base+"/provenance.json", func(m map[string]any) {
		for _, item := range m["outputs"].([]any) {
			output := item.(map[string]any)
			data, err := frontierRead(root, base+"/"+output["path"].(string))
			if err != nil {
				t.Fatal(err)
			}
			output["sha256"] = fmt.Sprintf("%x", sha256.Sum256(data))
		}
	})
}

func frontierMutate(t *testing.T, root, name string, change func(map[string]any)) {
	t.Helper()
	var m map[string]any
	if err := frontierJSON(root, name, &m); err != nil {
		t.Fatal(err)
	}
	change(m)
	frontierPutJSON(t, root, name, m)
}

func frontierPutJSON(t *testing.T, root, name string, value any) {
	t.Helper()
	data, err := json.Marshal(value)
	if err != nil {
		t.Fatal(err)
	}
	frontierPut(t, root, name, string(data))
}

func frontierPut(t *testing.T, root, name, data string) {
	t.Helper()
	path := filepath.Join(root, name)
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(path, []byte(data), 0o644); err != nil {
		t.Fatal(err)
	}
}

func frontierOneOf(value string, choices ...string) bool {
	for _, choice := range choices {
		if value == choice {
			return true
		}
	}
	return false
}

func frontierRead(root, name string) ([]byte, error) {
	if !filepath.IsLocal(name) || strings.Contains(name, "\\") {
		return nil, fmt.Errorf("unsafe path: %s", name)
	}
	// Reject lexical traversal too, even when normalization would stay local.
	for _, part := range strings.Split(name, "/") {
		if part == ".." {
			return nil, fmt.Errorf("unsafe traversal: %s", name)
		}
	}
	f, err := filetxn.OpenRegularInRoot(root, name)
	if err != nil {
		return nil, err
	}
	defer f.Close()
	return io.ReadAll(f)
}

func frontierJSON(root, name string, value any) error {
	data, err := frontierRead(root, name)
	if err != nil {
		return err
	}
	if err := json.Unmarshal(data, value); err != nil {
		return fmt.Errorf("%s: %w", name, err)
	}
	return nil
}
