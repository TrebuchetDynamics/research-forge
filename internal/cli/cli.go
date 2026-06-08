package cli

import (
	"encoding/json"
	"fmt"
	"io"
	"net/url"
	"os"
	"path/filepath"
	"runtime"
	"strings"

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
	if endpoint := os.Getenv("RFORGE_OPENSEARCH_URL"); endpoint != "" {
		checks = append(checks, optionalHTTPEndpointCheck("opensearch_endpoint", endpoint, "Set RFORGE_OPENSEARCH_URL to a valid OpenSearch HTTP endpoint, or unset it to skip this optional check."))
	}
	if endpoint := os.Getenv("RFORGE_QDRANT_URL"); endpoint != "" {
		checks = append(checks, optionalHTTPEndpointCheck("qdrant_endpoint", endpoint, "Set RFORGE_QDRANT_URL to a valid Qdrant HTTP endpoint, or unset it to skip this optional check."))
	}
	if rscript := os.Getenv("RFORGE_RSCRIPT_PATH"); rscript != "" {
		checks = append(checks, optionalRMetaforCheck(rscript))
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

func optionalRMetaforCheck(rscript string) project.HealthCheck {
	info, err := os.Stat(rscript)
	if err != nil || info.IsDir() {
		return project.HealthCheck{Name: "r_metafor", OK: false, Message: rscript, Action: "Set RFORGE_RSCRIPT_PATH to an executable Rscript with metafor available, or unset it to skip this optional check."}
	}
	return project.HealthCheck{Name: "r_metafor", OK: true, Message: rscript, Action: "No action needed."}
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
			return writeError(stdout, stderr, opts, 2, "usage", "usage: rforge project create [path] --title <title>")
		}
		if path == "" {
			defaultPath, err := defaultRepoProjectPath()
			if err != nil {
				return writeError(stdout, stderr, opts, 1, "repo_project_defaults_failed", err.Error())
			}
			path = defaultPath
		}
		created, err := project.Create(path, project.CreateOptions{Title: title})
		if err != nil {
			return writeError(stdout, stderr, opts, 1, "project_create_failed", fmt.Sprintf("create project: %v", err))
		}
		if err := writeRepoConfigForProject(created); err != nil {
			return writeError(stdout, stderr, opts, 1, "repo_config_failed", fmt.Sprintf("write repo config: %v", err))
		}
		if opts.JSON {
			return writeJSON(stdout, 0, projectData(created))
		}
		fmt.Fprintf(stdout, "created project %s\n", created.Path)
		return 0
	case "discover-assets":
		if len(args) != 1 {
			return writeError(stdout, stderr, opts, 2, "usage", "usage: rforge project discover-assets")
		}
		repoRoot, err := findRepoRoot(".")
		if err != nil {
			return writeError(stdout, stderr, opts, 1, "repo_discovery_failed", err.Error())
		}
		projectPath, err := repoProjectPath(repoRoot)
		if err != nil {
			return writeError(stdout, stderr, opts, 1, "repo_project_defaults_failed", err.Error())
		}
		assets, err := project.DiscoverAssets(repoRoot, projectPath)
		if err != nil {
			return writeError(stdout, stderr, opts, 1, "asset_discovery_failed", fmt.Sprintf("discover assets: %v", err))
		}
		if opts.JSON {
			return writeJSON(stdout, 0, map[string]any{"assets": assets})
		}
		for _, asset := range assets {
			fmt.Fprintf(stdout, "%s\t%s\timported=%v\n", asset.Kind, asset.Path, asset.Imported)
		}
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
	return map[string]any{
		"path":           proj.Path,
		"title":          proj.Title,
		"storageMode":    proj.StorageMode,
		"schemaVersion":  proj.SchemaVersion,
		"manifestPath":   proj.ManifestPath,
		"lockfilePath":   proj.LockfilePath,
		"provenancePath": proj.ProvenancePath,
		"storagePath":    proj.StoragePath,
	}
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
	return path, title, title != ""
}

func defaultRepoProjectPath() (string, error) {
	cwd, err := os.Getwd()
	if err != nil {
		return "", err
	}
	repoRoot, err := findRepoRoot(cwd)
	if err != nil {
		return "", err
	}
	return repoProjectPath(repoRoot)
}

func repoProjectPath(repoRoot string) (string, error) {
	configured, err := readRepoProjectPath(repoRoot)
	if err != nil {
		return "", err
	}
	if configured != "" {
		return filepath.Join(repoRoot, configured), nil
	}
	return filepath.Join(repoRoot, "research-forge"), nil
}

func readRepoProjectPath(repoRoot string) (string, error) {
	configBytes, err := os.ReadFile(filepath.Join(repoRoot, ".researchforge"))
	if err != nil {
		if os.IsNotExist(err) {
			return "", nil
		}
		return "", err
	}
	for _, line := range strings.Split(string(configBytes), "\n") {
		key, value, ok := strings.Cut(line, "=")
		if !ok || strings.TrimSpace(key) != "default_project_path" {
			continue
		}
		value = strings.TrimSpace(value)
		value = strings.Trim(value, "\"")
		if strings.Contains(value, "..") || filepath.IsAbs(value) {
			return "", fmt.Errorf("invalid default_project_path in .researchforge")
		}
		return value, nil
	}
	return "", nil
}

func writeRepoConfigForProject(proj project.Project) error {
	repoRoot, err := findRepoRoot(proj.Path)
	if err != nil {
		return nil
	}
	content := fmt.Sprintf("default_project_path = %q\ne2e_topic = %q\n", filepath.Base(proj.Path), "artificial photosynthesis")
	return os.WriteFile(filepath.Join(repoRoot, ".researchforge"), []byte(content), 0o644)
}

func findRepoRoot(start string) (string, error) {
	dir, err := filepath.Abs(start)
	if err != nil {
		return "", err
	}
	if info, err := os.Stat(dir); err == nil && !info.IsDir() {
		dir = filepath.Dir(dir)
	}
	for {
		if _, err := os.Stat(filepath.Join(dir, ".git")); err == nil {
			return dir, nil
		}
		parent := filepath.Dir(dir)
		if parent == dir {
			return "", fmt.Errorf("not inside a repository; pass an explicit project path")
		}
		dir = parent
	}
}

func printHelp(w io.Writer) {
	fmt.Fprintln(w, "rforge - ResearchForge command-line tool")
	fmt.Fprintln(w)
	fmt.Fprintln(w, "Usage:")
	fmt.Fprintln(w, "  rforge version")
	fmt.Fprintln(w, "  rforge doctor")
	fmt.Fprintln(w, "  rforge project create [path] --title <title>")
	fmt.Fprintln(w, "  rforge project discover-assets")
	fmt.Fprintln(w, "  rforge project inspect <path>")
	fmt.Fprintln(w, "  rforge project list <root>")
}
