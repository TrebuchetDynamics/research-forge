package cli

import (
	"encoding/json"
	"fmt"
	"io"
	"net/url"
	"os"
	"runtime"

	"github.com/TrebuchetDynamics/research-forge/internal/project"
)

var (
	Version = "dev"
	Commit  = "unknown"
	Date    = "unknown"
)

type globalOptions struct {
	JSON     bool
	Project  string
	Config   string
	LogLevel string
}

// Execute runs the rforge CLI and returns a process-style exit code.
func Execute(args []string, stdout, stderr io.Writer) int {
	opts, remaining, ok := parseGlobalOptions(args)
	if !ok {
		return writeError(stdout, stderr, opts, 2, "invalid_global_flag", "invalid global flag usage")
	}
	if len(remaining) == 0 || remaining[0] == "--help" || remaining[0] == "-h" || remaining[0] == "help" {
		printHelp(stdout)
		return 0
	}

	switch remaining[0] {
	case "version":
		data := map[string]any{"name": "rforge", "version": Version, "commit": Commit, "date": Date}
		if opts.JSON {
			return writeJSON(stdout, 0, data)
		}
		fmt.Fprintf(stdout, "rforge %s (%s, %s)\n", Version, Commit, Date)
		return 0
	case "project":
		return executeProject(remaining[1:], stdout, stderr, opts)
	case "doctor":
		return executeDoctor(stdout, stderr, opts)
	default:
		return writeError(stdout, stderr, opts, 2, "unknown_command", fmt.Sprintf("unknown command %q", remaining[0]))
	}
}

func executeDoctor(stdout, stderr io.Writer, opts globalOptions) int {
	checks := []project.HealthCheck{
		{Name: "go_runtime", OK: runtime.Version() != "", Message: runtime.Version(), Action: "Use the reported Go runtime for local builds and CI."},
	}
	if opts.Project != "" {
		checks = append(checks, project.CheckHealth(opts.Project).Checks...)
	}
	if endpoint := os.Getenv("RFORGE_GROBID_URL"); endpoint != "" {
		checks = append(checks, optionalHTTPEndpointCheck("grobid_endpoint", endpoint, "Set RFORGE_GROBID_URL to a valid GROBID HTTP endpoint, or unset it to skip this optional check."))
	}
	if opts.JSON {
		return writeJSON(stdout, 0, map[string]any{"checks": checks})
	}
	for _, check := range checks {
		status := "fail"
		if check.OK {
			status = "pass"
		}
		fmt.Fprintf(stdout, "%s: %s (%s) action: %s\n", check.Name, status, check.Message, check.Action)
	}
	return 0
}

func optionalHTTPEndpointCheck(name, endpoint, failureAction string) project.HealthCheck {
	parsed, err := url.Parse(endpoint)
	if err != nil || parsed.Scheme == "" || parsed.Host == "" || (parsed.Scheme != "http" && parsed.Scheme != "https") {
		return project.HealthCheck{Name: name, OK: false, Message: endpoint, Action: failureAction}
	}
	return project.HealthCheck{Name: name, OK: true, Message: endpoint, Action: "No action needed."}
}

func executeProject(args []string, stdout, stderr io.Writer, opts globalOptions) int {
	if len(args) == 0 {
		return writeError(stdout, stderr, opts, 2, "missing_project_subcommand", "missing project subcommand")
	}
	switch args[0] {
	case "create":
		path, title, ok := parseProjectCreate(args[1:])
		if !ok {
			return writeError(stdout, stderr, opts, 2, "usage", "usage: rforge project create <path> --title <title>")
		}
		created, err := project.Create(path, project.CreateOptions{Title: title})
		if err != nil {
			return writeError(stdout, stderr, opts, 1, "project_create_failed", fmt.Sprintf("create project: %v", err))
		}
		if opts.JSON {
			return writeJSON(stdout, 0, projectData(created))
		}
		fmt.Fprintf(stdout, "created project %s\n", created.Path)
		return 0
	case "inspect":
		if len(args) != 2 {
			return writeError(stdout, stderr, opts, 2, "usage", "usage: rforge project inspect <path>")
		}
		inspected, err := project.Inspect(args[1])
		if err != nil {
			return writeError(stdout, stderr, opts, 1, "project_inspect_failed", fmt.Sprintf("inspect project: %v", err))
		}
		if opts.JSON {
			return writeJSON(stdout, 0, projectData(inspected))
		}
		fmt.Fprintf(stdout, "path: %s\ntitle: %s\nstorage: %s\n", inspected.Path, inspected.Title, inspected.StorageMode)
		return 0
	case "list":
		if len(args) != 2 {
			return writeError(stdout, stderr, opts, 2, "usage", "usage: rforge project list <root>")
		}
		projects, err := project.List(args[1])
		if err != nil {
			return writeError(stdout, stderr, opts, 1, "project_list_failed", fmt.Sprintf("list projects: %v", err))
		}
		if opts.JSON {
			items := make([]map[string]any, 0, len(projects))
			for _, proj := range projects {
				items = append(items, projectData(proj))
			}
			return writeJSON(stdout, 0, map[string]any{"projects": items})
		}
		for _, proj := range projects {
			fmt.Fprintf(stdout, "%s\t%s\n", proj.Path, proj.Title)
		}
		return 0
	default:
		return writeError(stdout, stderr, opts, 2, "unknown_project_subcommand", fmt.Sprintf("unknown project subcommand %q", args[0]))
	}
}

func projectData(proj project.Project) map[string]any {
	return map[string]any{"path": proj.Path, "title": proj.Title, "storageMode": proj.StorageMode}
}

func parseGlobalOptions(args []string) (globalOptions, []string, bool) {
	var opts globalOptions
	for i := 0; i < len(args); i++ {
		switch args[i] {
		case "--json":
			opts.JSON = true
		case "--project":
			if i+1 >= len(args) {
				return opts, nil, false
			}
			opts.Project = args[i+1]
			i++
		case "--config":
			if i+1 >= len(args) {
				return opts, nil, false
			}
			opts.Config = args[i+1]
			i++
		case "--log-level":
			if i+1 >= len(args) {
				return opts, nil, false
			}
			opts.LogLevel = args[i+1]
			i++
		default:
			return opts, args[i:], true
		}
	}
	return opts, nil, true
}

func writeJSON(stdout io.Writer, code int, data any) int {
	_ = json.NewEncoder(stdout).Encode(map[string]any{"ok": true, "data": data})
	return code
}

func writeError(stdout, stderr io.Writer, opts globalOptions, code int, errorCode, message string) int {
	if opts.JSON {
		_ = json.NewEncoder(stdout).Encode(map[string]any{
			"ok": false,
			"error": map[string]any{
				"code":    errorCode,
				"message": message,
			},
		})
		return code
	}
	fmt.Fprintln(stderr, message)
	return code
}

func parseProjectCreate(args []string) (string, string, bool) {
	var path string
	var title string
	for i := 0; i < len(args); i++ {
		arg := args[i]
		switch arg {
		case "--title":
			if i+1 >= len(args) {
				return "", "", false
			}
			title = args[i+1]
			i++
		default:
			if path != "" {
				return "", "", false
			}
			path = arg
		}
	}
	return path, title, path != "" && title != ""
}

func printHelp(w io.Writer) {
	fmt.Fprintln(w, "rforge - ResearchForge command-line tool")
	fmt.Fprintln(w)
	fmt.Fprintln(w, "Usage:")
	fmt.Fprintln(w, "  rforge version")
	fmt.Fprintln(w, "  rforge doctor")
	fmt.Fprintln(w, "  rforge project create <path> --title <title>")
	fmt.Fprintln(w, "  rforge project inspect <path>")
	fmt.Fprintln(w, "  rforge project list <root>")
}
