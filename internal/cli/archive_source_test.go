package cli

import (
	"bytes"
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/TrebuchetDynamics/research-forge/internal/project"
)

func TestArchiveSourceCLI(t *testing.T) {
	for _, command := range []struct {
		name, action, errorCode string
	}{
		{"archive", "create", "archive_failed"},
		{"package", "archive", "package_archive_failed"},
	} {
		for _, mode := range []string{"text", "json"} {
			for _, sourceKind := range []string{"symlink", "file", "directory"} {
				for _, destination := range []string{"absent", "existing"} {
					t.Run(command.name+"/"+mode+"/"+sourceKind+"/"+destination, func(t *testing.T) {
						dir := t.TempDir()
						payload := filepath.Join(dir, "payload.txt")
						if err := os.WriteFile(payload, []byte("source bytes"), 0o640); err != nil {
							t.Fatal(err)
						}
						source := dir
						switch sourceKind {
						case "symlink":
							source = filepath.Join(t.TempDir(), "link")
							if err := os.Symlink(dir, source); err != nil {
								t.Skipf("symlink unsupported: %v", err)
							}
						case "file":
							source = payload
						}
						outputDir := t.TempDir()
						out := filepath.Join(outputDir, "backup.tar")
						if destination == "existing" {
							if err := os.WriteFile(out, []byte("prior archive"), 0o600); err != nil {
								t.Fatal(err)
							}
						}
						args := []string{command.name, command.action, source, out}
						if mode == "json" {
							args = append([]string{"--json"}, args...)
						}
						var stdout, stderr bytes.Buffer
						code := Execute(args, &stdout, &stderr)
						valid := sourceKind == "directory"
						wantCode := 1
						if valid {
							wantCode = 0
						}
						if code != wantCode {
							t.Errorf("exit = %d, want %d: stdout=%s stderr=%s", code, wantCode, &stdout, &stderr)
						}
						message := stderr.String()
						if mode == "json" {
							var envelope struct {
								OK    *bool                          `json:"ok"`
								Data  json.RawMessage                `json:"data"`
								Error struct{ Code, Message string } `json:"error"`
							}
							if err := json.Unmarshal(stdout.Bytes(), &envelope); err != nil || envelope.OK == nil || *envelope.OK != valid || stderr.Len() != 0 {
								t.Errorf("JSON envelope = %s, stderr=%s, %v", &stdout, &stderr, err)
							}
							if !valid && (envelope.Error.Code != command.errorCode || len(envelope.Data) != 0) {
								t.Errorf("failure envelope = %s, want %s without success data", &stdout, command.errorCode)
							}
							message = envelope.Error.Message
						} else if !valid && stdout.Len() != 0 {
							t.Errorf("failure emitted success output: %s", &stdout)
						} else if valid && (stdout.Len() == 0 || stderr.Len() != 0) {
							t.Errorf("success stdout=%s stderr=%s", &stdout, &stderr)
						}
						wantEntries := 1
						if !valid {
							if !strings.Contains(message, "directory") && !strings.Contains(message, source) {
								t.Errorf("error lacks source/directory context: %q", message)
							}
							if destination == "existing" {
								if data, err := os.ReadFile(out); err != nil || string(data) != "prior archive" {
									t.Errorf("prior archive changed: %q, %v", data, err)
								}
								if info, err := os.Stat(out); err != nil || info.Mode().Perm() != 0o600 {
									t.Errorf("prior archive permissions changed: %v, %v", info, err)
								}
							} else {
								wantEntries = 0
								if _, err := os.Lstat(out); !os.IsNotExist(err) {
									t.Errorf("absent output created: %v", err)
								}
							}
						} else {
							restored := t.TempDir()
							if err := project.Restore(out, restored); err != nil {
								t.Fatal(err)
							}
							if data, err := os.ReadFile(filepath.Join(restored, "payload.txt")); err != nil || string(data) != "source bytes" {
								t.Errorf("restored payload = %q, %v", data, err)
							}
						}
						if entries, err := os.ReadDir(outputDir); err != nil || len(entries) != wantEntries {
							t.Errorf("output inventory = %v, %v; want %d entries", entries, err, wantEntries)
						}
						if data, err := os.ReadFile(payload); err != nil || string(data) != "source bytes" {
							t.Errorf("source payload changed: %q, %v", data, err)
						}
					})
				}
			}
		}
	}
}
