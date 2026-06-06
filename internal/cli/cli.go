package cli

import (
	"fmt"
	"io"

	"github.com/TrebuchetDynamics/research-forge/internal/project"
)

var (
	Version = "dev"
	Commit  = "unknown"
	Date    = "unknown"
)

// Execute runs the rforge CLI and returns a process-style exit code.
func Execute(args []string, stdout, stderr io.Writer) int {
	if len(args) == 0 || args[0] == "--help" || args[0] == "-h" || args[0] == "help" {
		printHelp(stdout)
		return 0
	}

	switch args[0] {
	case "version":
		fmt.Fprintf(stdout, "rforge %s (%s, %s)\n", Version, Commit, Date)
		return 0
	case "project":
		return executeProject(args[1:], stdout, stderr)
	default:
		fmt.Fprintf(stderr, "unknown command %q\n", args[0])
		printHelp(stderr)
		return 2
	}
}

func executeProject(args []string, stdout, stderr io.Writer) int {
	if len(args) == 0 {
		fmt.Fprintln(stderr, "missing project subcommand")
		return 2
	}
	switch args[0] {
	case "create":
		path, title, ok := parseProjectCreate(args[1:])
		if !ok {
			fmt.Fprintln(stderr, "usage: rforge project create <path> --title <title>")
			return 2
		}
		created, err := project.Create(path, project.CreateOptions{Title: title})
		if err != nil {
			fmt.Fprintf(stderr, "create project: %v\n", err)
			return 1
		}
		fmt.Fprintf(stdout, "created project %s\n", created.Path)
		return 0
	case "inspect":
		if len(args) != 2 {
			fmt.Fprintln(stderr, "usage: rforge project inspect <path>")
			return 2
		}
		inspected, err := project.Inspect(args[1])
		if err != nil {
			fmt.Fprintf(stderr, "inspect project: %v\n", err)
			return 1
		}
		fmt.Fprintf(stdout, "path: %s\ntitle: %s\nstorage: %s\n", inspected.Path, inspected.Title, inspected.StorageMode)
		return 0
	default:
		fmt.Fprintf(stderr, "unknown project subcommand %q\n", args[0])
		return 2
	}
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
	fmt.Fprintln(w, "  rforge project create <path> --title <title>")
	fmt.Fprintln(w, "  rforge project inspect <path>")
}
