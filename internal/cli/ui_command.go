package cli

import (
	"fmt"
	"io"
	"net/http"
	"os"
	"strings"

	"github.com/TrebuchetDynamics/research-forge/internal/webui"
)

// uiServe is the server entry point. It is a package var so tests can exercise
// flag parsing and the JSON dry-run without binding a port.
var uiServe = func(addr string, handler http.Handler) error {
	server := &http.Server{Addr: addr, Handler: handler}
	return server.ListenAndServe()
}

// executeUI serves the local Go + HTMX research cockpit. The CLI remains the
// authoritative automation path (ADR 0006); this command only visualizes a
// project folder. Multiple dashboards can run concurrently on different ports
// via --addr, one per research folder.
func executeUI(args []string, stdout, stderr io.Writer, opts globalOptions) int {
	addr, ok := parseUIAddr(args)
	if !ok {
		return writeError(stdout, stderr, opts, 2, "usage", "usage: rforge [--project <path>] ui [--addr :8080]")
	}

	if opts.JSON {
		return writeJSON(stdout, 0, map[string]any{
			"status":  "serving_ready",
			"stack":   "go+htmx",
			"addr":    addr,
			"project": opts.Project,
			"routes":  webui.Routes(),
		})
	}

	fmt.Fprintf(stdout, "ResearchForge dashboard serving on http://%s (project: %s)\n", displayAddr(addr), projectOrNone(opts.Project))
	handler := webui.NewRouter(webui.Config{ProjectPath: opts.Project})
	if err := uiServe(addr, handler); err != nil {
		return writeError(stdout, stderr, opts, 1, "ui_server_failed", err.Error())
	}
	return 0
}

// parseUIAddr resolves the listen address from --addr, the RFORGE_UI_ADDR
// environment variable, or the :8080 default, in that order of precedence.
func parseUIAddr(args []string) (string, bool) {
	addr := strings.TrimSpace(os.Getenv("RFORGE_UI_ADDR"))
	if addr == "" {
		addr = ":8080"
	}
	for i := 0; i < len(args); i++ {
		switch args[i] {
		case "--addr":
			if i+1 >= len(args) || strings.TrimSpace(args[i+1]) == "" {
				return "", false
			}
			addr = args[i+1]
			i++
		default:
			return "", false
		}
	}
	return addr, true
}

func displayAddr(addr string) string {
	if strings.HasPrefix(addr, ":") {
		return "localhost" + addr
	}
	return addr
}

func projectOrNone(path string) string {
	if strings.TrimSpace(path) == "" {
		return "none"
	}
	return path
}
