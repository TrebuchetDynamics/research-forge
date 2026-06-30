package cli

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"
)

func executeObsidian(args []string, stdout, stderr io.Writer, opts globalOptions) int {
	if len(args) == 0 {
		return writeError(stdout, stderr, opts, 2, "usage",
			"usage: rforge obsidian install [--dry-run]\n       rforge obsidian open --vault <dir> [--dry-run]")
	}
	switch args[0] {
	case "install":
		return executeObsidianInstall(args[1:], stdout, stderr, opts)
	case "open":
		return executeObsidianOpen(args[1:], stdout, stderr, opts)
	default:
		return writeError(stdout, stderr, opts, 2, "unknown_obsidian_subcommand",
			fmt.Sprintf("unknown obsidian subcommand %q — try: install, open", args[0]))
	}
}

// ── install ───────────────────────────────────────────────────────────────────

type obsidianRelease struct {
	version string
	url     string
	dest    string
}

// obsidianLatestVersion fetches the latest release tag from GitHub.
// Falls back to a known-good version on error.
func obsidianLatestVersion() string {
	const fallback = "1.8.10"
	resp, err := http.Get("https://api.github.com/repos/obsidianmd/obsidian-releases/releases/latest") //nolint:gosec
	if err != nil {
		return fallback
	}
	defer resp.Body.Close()
	if resp.StatusCode != 200 {
		return fallback
	}
	var payload struct {
		TagName string `json:"tag_name"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&payload); err != nil {
		return fallback
	}
	v := strings.TrimPrefix(payload.TagName, "v")
	if v == "" {
		return fallback
	}
	return v
}

func obsidianReleaseInfo() obsidianRelease {
	version := obsidianLatestVersion()
	goos := runtime.GOOS
	goarch := runtime.GOARCH

	switch goos {
	case "linux":
		_ = goarch
		url := fmt.Sprintf("https://github.com/obsidianmd/obsidian-releases/releases/download/v%s/Obsidian-%s.AppImage", version, version)
		dest := filepath.Join(obsidianLocalBin(), "obsidian")
		return obsidianRelease{version: version, url: url, dest: dest}

	case "darwin":
		url := fmt.Sprintf("https://github.com/obsidianmd/obsidian-releases/releases/download/v%s/Obsidian-%s-universal.dmg", version, version)
		return obsidianRelease{version: version, url: url, dest: "/Applications/Obsidian.app"}

	case "windows":
		url := fmt.Sprintf("https://github.com/obsidianmd/obsidian-releases/releases/download/v%s/Obsidian.%s.exe", version, version)
		return obsidianRelease{version: version, url: url, dest: ""}

	default:
		return obsidianRelease{version: version, url: "https://obsidian.md/download", dest: ""}
	}
}

func executeObsidianInstall(args []string, stdout, stderr io.Writer, opts globalOptions) int {
	dryRun := false
	for _, a := range args {
		if a == "--dry-run" {
			dryRun = true
		}
	}

	rel := obsidianReleaseInfo()

	if dryRun {
		if opts.JSON {
			return writeJSON(stdout, 0, map[string]any{
				"version":  rel.version,
				"url":      rel.url,
				"dest":     rel.dest,
				"platform": runtime.GOOS + "/" + runtime.GOARCH,
				"dry_run":  true,
			})
		}
		fmt.Fprintf(stdout, "Obsidian v%s for %s/%s\n", rel.version, runtime.GOOS, runtime.GOARCH)
		fmt.Fprintf(stdout, "  url:  %s\n", rel.url)
		if rel.dest != "" {
			fmt.Fprintf(stdout, "  dest: %s\n", rel.dest)
		}
		fmt.Fprintln(stdout, "(dry-run — nothing downloaded)")
		return 0
	}

	switch runtime.GOOS {
	case "linux":
		return obsidianInstallLinux(rel, stdout, stderr, opts)
	case "darwin":
		fmt.Fprintf(stdout, "Download Obsidian v%s for macOS:\n  %s\n", rel.version, rel.url)
		fmt.Fprintln(stdout, "Open the .dmg and drag Obsidian to Applications.")
		return 0
	case "windows":
		fmt.Fprintf(stdout, "Download Obsidian v%s for Windows:\n  %s\n", rel.version, rel.url)
		fmt.Fprintln(stdout, "Run the installer and follow the prompts.")
		return 0
	default:
		fmt.Fprintf(stdout, "Download Obsidian from: %s\n", rel.url)
		return 0
	}
}

func obsidianInstallLinux(rel obsidianRelease, stdout, stderr io.Writer, opts globalOptions) int {
	binDir := obsidianLocalBin()
	if err := os.MkdirAll(binDir, 0o755); err != nil {
		return writeError(stdout, stderr, opts, 1, "mkdir_failed", err.Error())
	}

	fmt.Fprintf(stdout, "Downloading Obsidian v%s...\n", rel.version)
	fmt.Fprintf(stdout, "  from: %s\n", rel.url)
	fmt.Fprintf(stdout, "  to:   %s\n", rel.dest)

	if err := downloadFile(rel.url, rel.dest); err != nil {
		return writeError(stdout, stderr, opts, 1, "download_failed", err.Error())
	}
	if err := os.Chmod(rel.dest, 0o755); err != nil {
		return writeError(stdout, stderr, opts, 1, "chmod_failed", err.Error())
	}

	fmt.Fprintln(stdout, "Obsidian installed.")
	if !strings.Contains(obsidianLocalBin(), os.Getenv("PATH")) {
		fmt.Fprintf(stdout, "Add %s to your PATH if it isn't already:\n", binDir)
		fmt.Fprintf(stdout, "  echo 'export PATH=\"%s:$PATH\"' >> ~/.bashrc\n", binDir)
	}
	fmt.Fprintf(stdout, "\nOpen a vault:\n  rforge obsidian open --vault <vault-dir>\n")
	return 0
}

func obsidianLocalBin() string {
	home, _ := os.UserHomeDir()
	return filepath.Join(home, ".local", "bin")
}

// ── open ──────────────────────────────────────────────────────────────────────

func executeObsidianOpen(args []string, stdout, stderr io.Writer, opts globalOptions) int {
	vaultDir, dryRun := "", false
	for i := 0; i < len(args); i++ {
		switch args[i] {
		case "--vault":
			if i+1 < len(args) {
				vaultDir = args[i+1]
				i++
			}
		case "--dry-run":
			dryRun = true
		}
	}
	if vaultDir == "" {
		return writeError(stdout, stderr, opts, 2, "usage",
			"usage: rforge obsidian open --vault <dir> [--dry-run]")
	}

	// Verify vault dir exists
	if _, err := os.Stat(vaultDir); err != nil {
		return writeError(stdout, stderr, opts, 1, "vault_not_found",
			fmt.Sprintf("vault dir does not exist: %s", vaultDir))
	}

	absVault, err := filepath.Abs(vaultDir)
	if err != nil {
		absVault = vaultDir
	}

	if dryRun {
		if opts.JSON {
			return writeJSON(stdout, 0, map[string]any{
				"vault":   absVault,
				"dry_run": true,
			})
		}
		fmt.Fprintf(stdout, "would open vault: %s\n", absVault)
		return 0
	}

	binary, err := obsidianFindBinary()
	if err != nil {
		msg := fmt.Sprintf("Obsidian not found. Install it with:\n  rforge obsidian install\n\nOr download from https://obsidian.md/download")
		return writeError(stdout, stderr, opts, 1, "obsidian_not_found", msg)
	}

	fmt.Fprintf(stdout, "opening vault: %s\n", absVault)
	cmd := exec.Command(binary, absVault)
	cmd.Stdout = nil
	cmd.Stderr = nil
	if err := cmd.Start(); err != nil {
		return writeError(stdout, stderr, opts, 1, "obsidian_launch_failed", err.Error())
	}
	// Detach — don't wait for the GUI process to exit
	go func() { _ = cmd.Wait() }()
	return 0
}

func downloadFile(url, dest string) error {
	resp, err := http.Get(url) //nolint:gosec // URL is a hardcoded GitHub releases URL
	if err != nil {
		return err
	}
	defer resp.Body.Close()
	if resp.StatusCode != 200 {
		return fmt.Errorf("HTTP %d from %s", resp.StatusCode, url)
	}
	f, err := os.Create(dest)
	if err != nil {
		return err
	}
	defer f.Close()
	_, err = io.Copy(f, resp.Body)
	return err
}

func obsidianFindBinary() (string, error) {
	// Check PATH first
	if path, err := exec.LookPath("obsidian"); err == nil {
		return path, nil
	}

	// Common install locations
	home, _ := os.UserHomeDir()
	candidates := []string{
		filepath.Join(home, ".local", "bin", "obsidian"),
		"/usr/bin/obsidian",
		"/usr/local/bin/obsidian",
		"/snap/bin/obsidian",
	}
	// macOS
	if runtime.GOOS == "darwin" {
		candidates = append(candidates, "/Applications/Obsidian.app/Contents/MacOS/Obsidian")
	}

	for _, c := range candidates {
		if _, err := os.Stat(c); err == nil {
			return c, nil
		}
	}
	return "", fmt.Errorf("obsidian binary not found")
}
