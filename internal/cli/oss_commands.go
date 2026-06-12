package cli

import (
	"context"
	"fmt"
	"io"
	"path/filepath"

	"github.com/TrebuchetDynamics/research-forge/internal/oss"
)

func executeOSS(args []string, stdout, stderr io.Writer, opts globalOptions) int {
	if len(args) == 0 {
		return writeError(stdout, stderr, opts, 2, "usage", "usage: rforge --project <path> oss <add|list|license-check>")
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
