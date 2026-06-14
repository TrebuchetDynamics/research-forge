package analysis

import (
	"bytes"
	"context"
	"os"
	"os/exec"
	"strings"
	"time"
)

// RscriptRunner executes generated metafor scripts with a real Rscript binary.
// It implements Runner for opt-in R/metafor integration runs; the deterministic
// FakeRunner remains the default for normal, network-free tests.
type RscriptRunner struct {
	// Path is the Rscript binary to invoke; it defaults to "Rscript" (resolved
	// from PATH) when empty.
	Path string
	// Timeout bounds a single Rscript execution; it defaults to 5 minutes.
	Timeout time.Duration
}

func (r RscriptRunner) binary() string {
	if strings.TrimSpace(r.Path) != "" {
		return r.Path
	}
	return "Rscript"
}

func (r RscriptRunner) timeout() time.Duration {
	if r.Timeout > 0 {
		return r.Timeout
	}
	return 5 * time.Minute
}

// Run writes the metafor script to a temporary file and executes it with
// Rscript, capturing stdout and stderr. Arguments are passed directly (no
// shell), so the script content cannot inject extra commands.
func (r RscriptRunner) Run(script string) (RunOutput, error) {
	file, err := os.CreateTemp("", "rforge-metafor-*.R")
	if err != nil {
		return RunOutput{}, err
	}
	defer os.Remove(file.Name())
	if _, err := file.WriteString(script); err != nil {
		_ = file.Close()
		return RunOutput{}, err
	}
	if err := file.Close(); err != nil {
		return RunOutput{}, err
	}
	ctx, cancel := context.WithTimeout(context.Background(), r.timeout())
	defer cancel()
	cmd := exec.CommandContext(ctx, r.binary(), file.Name())
	var stdout, stderr bytes.Buffer
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr
	runErr := cmd.Run()
	return RunOutput{Stdout: stdout.String(), Stderr: stderr.String()}, runErr
}

// ToolVersions reports the detected R version on a best-effort basis; it returns
// an empty map when the Rscript binary is unavailable.
func (r RscriptRunner) ToolVersions() map[string]string {
	versions := map[string]string{}
	out, err := exec.Command(r.binary(), "--version").CombinedOutput()
	if err != nil {
		return versions
	}
	if line := strings.TrimSpace(string(out)); line != "" {
		versions["R"] = line
	}
	return versions
}
