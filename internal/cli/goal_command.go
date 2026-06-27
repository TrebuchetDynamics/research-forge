package cli

import (
	"bufio"
	"encoding/json"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"
	"time"
)

type goalDefinition struct {
	Name      string `json:"name"`
	Metric    string `json:"metric"`
	Min       int    `json:"min"`
	CreatedAt string `json:"createdAt"`
}

func executeGoal(args []string, stdout, stderr io.Writer, opts globalOptions) int {
	if len(args) == 0 {
		return writeError(stdout, stderr, opts, 2, "usage", "usage: rforge goal set|audit")
	}
	switch args[0] {
	case "set":
		return executeGoalSet(args[1:], stdout, stderr, opts)
	case "audit":
		return executeGoalAudit(args[1:], stdout, stderr, opts)
	default:
		return writeError(stdout, stderr, opts, 2, "unknown_goal_subcommand", fmt.Sprintf("unknown goal subcommand %q", args[0]))
	}
}

func executeGoalSet(args []string, stdout, stderr io.Writer, opts globalOptions) int {
	if opts.Project == "" {
		return writeError(stdout, stderr, opts, 2, "missing_project", "--project is required for goal set")
	}
	name, metric, min, ok := parseGoalSet(args)
	if !ok {
		return writeError(stdout, stderr, opts, 2, "usage", "usage: rforge goal set --metric <metric> --min <N> --name <name>")
	}
	if err := goalSet(opts.Project, name, metric, min); err != nil {
		return writeError(stdout, stderr, opts, 1, "goal_set_failed", err.Error())
	}
	if opts.JSON {
		return writeJSON(stdout, 0, map[string]any{"action": "goal.set", "name": name, "metric": metric, "min": min})
	}
	fmt.Fprintln(stdout, "goal set")
	return 0
}

func parseGoalSet(args []string) (name, metric string, min int, ok bool) {
	for i := 0; i < len(args)-1; i++ {
		switch args[i] {
		case "--name":
			name = args[i+1]
			i++
		case "--metric":
			metric = args[i+1]
			i++
		case "--min":
			fmt.Sscanf(args[i+1], "%d", &min)
			i++
		}
	}
	return name, metric, min, metric != "" && min > 0
}

func goalSet(projectPath, name, metric string, min int) error {
	goalsPath := filepath.Join(projectPath, "goals.json")
	var goals []goalDefinition
	if data, err := os.ReadFile(goalsPath); err == nil {
		_ = json.Unmarshal(data, &goals)
	}
	goals = append(goals, goalDefinition{
		Name:      name,
		Metric:    metric,
		Min:       min,
		CreatedAt: time.Now().UTC().Format(time.RFC3339),
	})
	data, err := json.MarshalIndent(goals, "", "  ")
	if err != nil {
		return err
	}
	return os.WriteFile(goalsPath, data, 0o644)
}

func executeGoalAudit(args []string, stdout, stderr io.Writer, opts globalOptions) int {
	if opts.Project == "" {
		return writeError(stdout, stderr, opts, 2, "missing_project", "--project is required for goal audit")
	}
	ledger := ""
	for i := 0; i < len(args)-1; i++ {
		if args[i] == "--ledger" {
			ledger = args[i+1]
			i++
		}
	}
	if ledger == "" {
		return writeError(stdout, stderr, opts, 2, "usage", "usage: rforge goal audit --ledger <file>")
	}
	goals, loadErr := goalLoad(opts.Project)
	if loadErr != nil || len(goals) == 0 {
		if opts.JSON {
			return writeJSON(stdout, 0, map[string]any{"goals": []any{}, "count": 0})
		}
		fmt.Fprintln(stdout, "no goals set")
		return 0
	}
	count, countErr := goalCountLedger(ledger)
	if countErr != nil {
		return writeError(stdout, stderr, opts, 1, "goal_audit_failed", countErr.Error())
	}
	results := make([]map[string]any, 0, len(goals))
	for _, g := range goals {
		met := count >= g.Min
		results = append(results, map[string]any{
			"name":   g.Name,
			"metric": g.Metric,
			"min":    g.Min,
			"count":  count,
			"met":    met,
		})
	}
	if opts.JSON {
		var met bool
		for _, r := range results {
			if r["met"] == true {
				met = true
			}
		}
		return writeJSON(stdout, 0, map[string]any{"goals": results, "count": count, "met": met})
	}
	for _, r := range results {
		status := "not met"
		if r["met"] == true {
			status = "met"
		}
		fmt.Fprintf(stdout, "%-32s %s (%d / %d)\n", r["name"], status, count, r["min"])
	}
	return 0
}

func goalLoad(projectPath string) ([]goalDefinition, error) {
	data, err := os.ReadFile(filepath.Join(projectPath, "goals.json"))
	if os.IsNotExist(err) {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	var goals []goalDefinition
	return goals, json.Unmarshal(data, &goals)
}

func goalCountLedger(path string) (int, error) {
	f, err := os.Open(path)
	if os.IsNotExist(err) {
		return 0, nil
	}
	if err != nil {
		return 0, err
	}
	defer f.Close()
	count := 0
	scanner := bufio.NewScanner(f)
	for scanner.Scan() {
		if strings.TrimSpace(scanner.Text()) != "" {
			count++
		}
	}
	return count, scanner.Err()
}
