package cli

import (
	"context"
	"fmt"
	"io"
	"path/filepath"
	"strconv"
	"strings"
	"time"

	"github.com/TrebuchetDynamics/research-forge/internal/oss"
)

func executeOSS(args []string, stdout, stderr io.Writer, opts globalOptions) int {
	if len(args) == 0 {
		return writeError(stdout, stderr, opts, 2, "usage", "usage: rforge oss <inventory-check|add|list|license-check>")
	}
	if args[0] == "search-plan" {
		query, ecosystem, ok := parseOSSSearchPlan(args[1:])
		if !ok {
			return writeError(stdout, stderr, opts, 2, "usage", "usage: rforge oss search-plan --query <text> [--ecosystem all|go|python|js|rust|data]")
		}
		plan, err := oss.BuildSearchPlan(query, ecosystem)
		if err != nil {
			return writeError(stdout, stderr, opts, 2, "oss_search_plan_invalid", err.Error())
		}
		if opts.JSON {
			return writeJSON(stdout, 0, map[string]any{"searchPlan": plan})
		}
		fmt.Fprintf(stdout, "# OSS search plan: %s\n\n", plan.Query)
		for _, provider := range plan.Providers {
			fmt.Fprintf(stdout, "- %s (%s): %s\n  Signals: %s\n  Gate: %s\n", provider.Provider, provider.Kind, provider.URL, strings.Join(provider.Signals, ", "), provider.HumanGate)
		}
		return 0
	}
	if args[0] == "inventory-check" {
		if len(args) != 2 {
			return writeError(stdout, stderr, opts, 2, "usage", "usage: rforge oss inventory-check <manifest.json>")
		}
		result, err := oss.ValidateInventoryManifest(args[1])
		if err != nil {
			return writeError(stdout, stderr, opts, 1, "oss_inventory_check_failed", fmt.Sprintf("check inventory: %v", err))
		}
		if opts.JSON {
			return writeJSON(stdout, 0, result)
		}
		if len(result.Issues) == 0 {
			fmt.Fprintf(stdout, "inventory ok: %d entries\n", result.EntryCount)
			return 0
		}
		for _, issue := range result.Issues {
			fmt.Fprintln(stdout, issue)
		}
		return 1
	}
	if args[0] == "inventory-report" {
		manifestPath, area, ok := parseOSSInventoryReport(args[1:])
		if !ok {
			return writeError(stdout, stderr, opts, 2, "usage", "usage: rforge oss inventory-report <manifest.json> [--area <area>]")
		}
		manifest, err := oss.LoadInventoryManifest(manifestPath)
		if err != nil {
			return writeError(stdout, stderr, opts, 1, "oss_inventory_report_failed", fmt.Sprintf("read inventory: %v", err))
		}
		report := oss.BuildInventoryReport(manifest, oss.InventoryReportOptions{Area: area})
		if opts.JSON {
			return writeJSON(stdout, 0, report)
		}
		fmt.Fprint(stdout, report.Markdown)
		return 0
	}
	if args[0] == "inventory-roadmap" {
		manifestPath, todoPath, ok := parseOSSInventoryRoadmap(args[1:])
		if !ok {
			return writeError(stdout, stderr, opts, 2, "usage", "usage: rforge oss inventory-roadmap <manifest.json> --todo <TODO.md>")
		}
		report, err := oss.BuildInventoryRoadmapReport(manifestPath, todoPath)
		if err != nil {
			return writeError(stdout, stderr, opts, 1, "oss_inventory_roadmap_failed", fmt.Sprintf("build inventory roadmap: %v", err))
		}
		if opts.JSON {
			return writeJSON(stdout, 0, report)
		}
		fmt.Fprint(stdout, report.Markdown)
		return 0
	}
	if args[0] == "inventory-drift" {
		if len(args) != 2 {
			return writeError(stdout, stderr, opts, 2, "usage", "usage: rforge oss inventory-drift <manifest.json>")
		}
		result, err := oss.CheckInventoryDrift(args[1])
		if err != nil {
			return writeError(stdout, stderr, opts, 1, "oss_inventory_drift_failed", fmt.Sprintf("check inventory drift: %v", err))
		}
		if opts.JSON {
			return writeJSON(stdout, 0, result)
		}
		fmt.Fprint(stdout, result.Markdown)
		return 0
	}
	if args[0] == "inventory-refresh" {
		manifestPath, source, baseURL, ok := parseOSSInventoryRefresh(args[1:])
		if !ok || source != "github" {
			return writeError(stdout, stderr, opts, 2, "usage", "usage: rforge oss inventory-refresh <manifest.json> --source github [--base-url <url>]")
		}
		result, err := oss.RefreshInventoryGitHubMetadata(manifestPath, oss.GitHubMetadataOptions{BaseURL: baseURL})
		if err != nil {
			return writeError(stdout, stderr, opts, 1, "oss_inventory_refresh_failed", fmt.Sprintf("refresh inventory: %v", err))
		}
		if opts.JSON {
			return writeJSON(stdout, 0, result)
		}
		fmt.Fprintf(stdout, "refreshed %d inventory entries, skipped %d\n", result.Refreshed, result.Skipped)
		return 0
	}
	if args[0] == "inventory-policy" {
		manifestPath, staleMonths, now, ok := parseOSSInventoryPolicy(args[1:])
		if !ok {
			return writeError(stdout, stderr, opts, 2, "usage", "usage: rforge oss inventory-policy <manifest.json> [--stale-after 18mo] [--now <rfc3339>]")
		}
		manifest, err := oss.LoadInventoryManifest(manifestPath)
		if err != nil {
			return writeError(stdout, stderr, opts, 1, "oss_inventory_policy_failed", fmt.Sprintf("read inventory: %v", err))
		}
		result := oss.CheckInventoryPolicy(manifest, oss.InventoryPolicyOptions{StaleAfterMonths: staleMonths, Now: now})
		if opts.JSON {
			return writeJSON(stdout, 0, result)
		}
		fmt.Fprint(stdout, result.Markdown)
		return 0
	}
	if opts.Project == "" {
		return writeError(stdout, stderr, opts, 2, "missing_project", "--project is required for oss commands")
	}
	registry, err := oss.OpenRegistry(filepath.Join(opts.Project, "data", "oss.json"))
	if err != nil {
		return writeError(stdout, stderr, opts, 1, "oss_registry_open_failed", fmt.Sprintf("open oss registry: %v", err))
	}
	switch args[0] {
	case "add":
		name, area, ok := parseOSSAdd(args[1:])
		if !ok {
			return writeError(stdout, stderr, opts, 2, "usage", "usage: rforge --project <path> oss add <owner/repo> [--area <area>]")
		}
		study, err := oss.NewRepositoryStudy(oss.RepositoryStudyInput{Name: name, Area: area})
		if err != nil {
			return writeError(stdout, stderr, opts, 2, "invalid_repository", err.Error())
		}
		if err := registry.Add(study); err != nil {
			return writeError(stdout, stderr, opts, 1, "oss_add_failed", fmt.Sprintf("add repository: %v", err))
		}
		if opts.JSON {
			return writeJSON(stdout, 0, map[string]any{"repository": study})
		}
		fmt.Fprintf(stdout, "added OSS repository %s\n", study.Name)
		return 0
	case "list":
		if len(args) != 1 {
			return writeError(stdout, stderr, opts, 2, "usage", "usage: rforge --project <path> oss list")
		}
		items, err := registry.List()
		if err != nil {
			return writeError(stdout, stderr, opts, 1, "oss_list_failed", fmt.Sprintf("list repositories: %v", err))
		}
		if opts.JSON {
			return writeJSON(stdout, 0, map[string]any{"repositories": items})
		}
		for _, item := range items {
			fmt.Fprintf(stdout, "%s\t%s\n", item.Name, item.Area)
		}
		return 0
	case "clone":
		name, remoteURL, ok := parseOSSClone(args[1:])
		if !ok {
			return writeError(stdout, stderr, opts, 2, "usage", "usage: rforge --project <path> oss clone <owner/repo> [--url <url>]")
		}
		if remoteURL == "" {
			remoteURL = "https://github.com/" + name + ".git"
		}
		result, err := oss.CloneRepository(context.Background(), opts.Project, name, remoteURL, oss.GitCloneRunner{})
		if err != nil {
			return writeError(stdout, stderr, opts, 1, "oss_clone_failed", fmt.Sprintf("clone repository: %v", err))
		}
		if opts.JSON {
			return writeJSON(stdout, 0, map[string]any{"clone": result})
		}
		fmt.Fprintf(stdout, "cloned OSS repository %s to %s\n", result.Name, result.Path)
		return 0
	case "refresh":
		name, metadata, ok := parseOSSRefresh(args[1:])
		if !ok {
			return writeError(stdout, stderr, opts, 2, "usage", "usage: rforge --project <path> oss refresh <owner/repo> [--interval <interval>] [--stale] [--archived]")
		}
		if err := registry.RefreshMetadata(name, metadata); err != nil {
			return writeError(stdout, stderr, opts, 1, "oss_refresh_failed", fmt.Sprintf("refresh metadata: %v", err))
		}
		if opts.JSON {
			return writeJSON(stdout, 0, map[string]any{"refreshed": name})
		}
		fmt.Fprintf(stdout, "refreshed OSS repository %s\n", name)
		return 0
	case "scan":
		name, topic, ok := parseOSSScan(args[1:])
		if !ok {
			return writeError(stdout, stderr, opts, 2, "usage", "usage: rforge --project <path> oss scan <owner/repo> --topic <topic>")
		}
		scan, err := oss.WriteTopicScan(opts.Project, name, topic)
		if err != nil {
			return writeError(stdout, stderr, opts, 1, "oss_scan_failed", fmt.Sprintf("write scan: %v", err))
		}
		if opts.JSON {
			return writeJSON(stdout, 0, map[string]any{"scan": scan})
		}
		fmt.Fprintf(stdout, "wrote OSS scan %s\n", scan.Path)
		return 0
	case "report":
		area, ok := parseOSSReport(args[1:])
		if !ok {
			return writeError(stdout, stderr, opts, 2, "usage", "usage: rforge --project <path> oss report --area <area>")
		}
		report, err := oss.BuildAreaReport(opts.Project, registry, area)
		if err != nil {
			return writeError(stdout, stderr, opts, 1, "oss_report_failed", fmt.Sprintf("build report: %v", err))
		}
		if opts.JSON {
			return writeJSON(stdout, 0, map[string]any{"report": report.Markdown})
		}
		fmt.Fprint(stdout, report.Markdown)
		return 0
	case "note":
		name, area, ok := parseOSSAdd(args[1:])
		if !ok {
			return writeError(stdout, stderr, opts, 2, "usage", "usage: rforge --project <path> oss note <owner/repo> [--area <area>]")
		}
		path, err := oss.WriteStudyNote(opts.Project, name, area)
		if err != nil {
			return writeError(stdout, stderr, opts, 1, "oss_note_failed", fmt.Sprintf("write note: %v", err))
		}
		if opts.JSON {
			return writeJSON(stdout, 0, map[string]any{"path": path})
		}
		fmt.Fprintf(stdout, "wrote OSS note %s\n", path)
		return 0
	case "license-check":
		if len(args) != 2 {
			return writeError(stdout, stderr, opts, 2, "usage", "usage: rforge --project <path> oss license-check <owner/repo>")
		}
		clonePath, err := oss.ResolveClonePath(opts.Project, args[1])
		if err != nil {
			return writeError(stdout, stderr, opts, 2, "invalid_repository", err.Error())
		}
		license, err := oss.DetectLicenseFile(clonePath)
		if err != nil {
			return writeError(stdout, stderr, opts, 1, "oss_license_check_failed", fmt.Sprintf("license check: %v", err))
		}
		if opts.JSON {
			return writeJSON(stdout, 0, map[string]any{"license": license})
		}
		fmt.Fprintf(stdout, "%s\t%t\t%s\n", args[1], license.Found, license.Kind)
		return 0
	default:
		return writeError(stdout, stderr, opts, 2, "usage", "usage: rforge --project <path> oss <add|list|license-check>")
	}
}

func parseOSSInventoryPolicy(args []string) (string, int, time.Time, bool) {
	if len(args) == 0 {
		return "", 0, time.Time{}, false
	}
	manifestPath := args[0]
	staleMonths := 18
	now := time.Time{}
	for i := 1; i < len(args); i++ {
		switch args[i] {
		case "--stale-after":
			if i+1 >= len(args) {
				return "", 0, time.Time{}, false
			}
			months, ok := parseMonths(args[i+1])
			if !ok {
				return "", 0, time.Time{}, false
			}
			staleMonths = months
			i++
		case "--now":
			if i+1 >= len(args) {
				return "", 0, time.Time{}, false
			}
			parsed, err := time.Parse(time.RFC3339, args[i+1])
			if err != nil {
				return "", 0, time.Time{}, false
			}
			now = parsed
			i++
		default:
			return "", 0, time.Time{}, false
		}
	}
	return manifestPath, staleMonths, now, manifestPath != ""
}

func parseMonths(value string) (int, bool) {
	value = strings.TrimSpace(value)
	value = strings.TrimSuffix(value, "months")
	value = strings.TrimSuffix(value, "month")
	value = strings.TrimSuffix(value, "mo")
	months, err := strconv.Atoi(value)
	return months, err == nil && months > 0
}

func parseOSSInventoryRefresh(args []string) (string, string, string, bool) {
	if len(args) < 3 {
		return "", "", "", false
	}
	manifestPath := args[0]
	values := map[string]string{}
	for i := 1; i < len(args); i++ {
		switch args[i] {
		case "--source", "--base-url":
			if i+1 >= len(args) {
				return "", "", "", false
			}
			values[args[i]] = args[i+1]
			i++
		default:
			return "", "", "", false
		}
	}
	return manifestPath, values["--source"], values["--base-url"], manifestPath != "" && values["--source"] != ""
}

func parseOSSInventoryRoadmap(args []string) (string, string, bool) {
	if len(args) != 3 || args[1] != "--todo" {
		return "", "", false
	}
	return args[0], args[2], strings.TrimSpace(args[0]) != "" && strings.TrimSpace(args[2]) != ""
}

func parseOSSSearchPlan(args []string) (string, string, bool) {
	query := ""
	ecosystem := "all"
	for i := 0; i < len(args); i++ {
		switch args[i] {
		case "--query":
			if i+1 >= len(args) {
				return "", "", false
			}
			query = args[i+1]
			i++
		case "--ecosystem":
			if i+1 >= len(args) {
				return "", "", false
			}
			ecosystem = args[i+1]
			i++
		default:
			return "", "", false
		}
	}
	return query, ecosystem, strings.TrimSpace(query) != ""
}

func parseOSSInventoryReport(args []string) (string, string, bool) {
	if len(args) != 1 && len(args) != 3 {
		return "", "", false
	}
	if len(args) == 3 {
		if args[1] != "--area" || args[2] == "" {
			return "", "", false
		}
		return args[0], args[2], args[0] != ""
	}
	return args[0], "", args[0] != ""
}

func parseOSSAdd(args []string) (string, string, bool) {
	if len(args) != 1 && len(args) != 3 {
		return "", "", false
	}
	name := args[0]
	area := ""
	if len(args) == 3 {
		if args[1] != "--area" {
			return "", "", false
		}
		area = args[2]
	}
	return name, area, true
}

func parseOSSClone(args []string) (string, string, bool) {
	if len(args) != 1 && len(args) != 3 {
		return "", "", false
	}
	name := args[0]
	remoteURL := ""
	if len(args) == 3 {
		if args[1] != "--url" {
			return "", "", false
		}
		remoteURL = args[2]
	}
	return name, remoteURL, true
}

func parseOSSScan(args []string) (string, string, bool) {
	if len(args) != 3 || args[1] != "--topic" {
		return "", "", false
	}
	return args[0], args[2], true
}

func parseOSSReport(args []string) (string, bool) {
	if len(args) != 2 || args[0] != "--area" {
		return "", false
	}
	return args[1], true
}

func parseOSSRefresh(args []string) (string, oss.RefreshMetadata, bool) {
	if len(args) == 0 {
		return "", oss.RefreshMetadata{}, false
	}
	metadata := oss.RefreshMetadata{}
	for i := 1; i < len(args); i++ {
		switch args[i] {
		case "--interval":
			if i+1 >= len(args) {
				return "", oss.RefreshMetadata{}, false
			}
			metadata.RefreshInterval = args[i+1]
			i++
		case "--stale":
			metadata.Stale = true
		case "--archived":
			metadata.Archived = true
		default:
			return "", oss.RefreshMetadata{}, false
		}
	}
	return args[0], metadata, true
}
