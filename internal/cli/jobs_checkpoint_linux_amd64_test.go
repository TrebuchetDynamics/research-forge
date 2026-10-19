package cli

import (
	"bufio"
	"bytes"
	"context"
	"fmt"
	"net/http"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strconv"
	"strings"
	"syscall"
	"testing"
	"time"

	"github.com/TrebuchetDynamics/research-forge/internal/jobs"
	"github.com/TrebuchetDynamics/research-forge/internal/search"
)

// Kill at the actual publication syscall, not an arbitrary sleep: moving the
// old checkpoint/results away first can destroy the only resumable state.
func TestJobsResumeKilledDuringPublication(t *testing.T) {
	if dir := os.Getenv("RFORGE_CHECKPOINT_KILL_DIR"); dir != "" {
		runtime.LockOSThread()
		if _, _, errno := syscall.RawSyscall(syscall.SYS_PTRACE, syscall.PTRACE_TRACEME, 0, 0); errno != 0 {
			fmt.Println("ptrace unavailable:", errno)
			return
		}
		fmt.Println(syscall.Gettid())
		if err := syscall.Tgkill(os.Getpid(), syscall.Gettid(), syscall.SIGSTOP); err != nil {
			t.Fatal(err)
		}
		if code := Execute([]string{"jobs", "resume", os.Getenv("RFORGE_CHECKPOINT_KILL_ID"), "--dir", dir}, os.Stdout, os.Stderr); code != 0 {
			t.Fatalf("worker exit %d", code)
		}
		return
	}
	server := newJobsTestServer(t, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		q := r.URL.Query().Get("search")
		fmt.Fprintf(w, `{"results":[{"id":"https://openalex.org/W-%s","title":"%s","publication_year":2026}]}`, q, q)
	}))
	defer server.Close()
	t.Setenv("RFORGE_OPENALEX_URL", server.URL)
	for _, boundary := range []string{"job", "batch-state.json", "results.jsonl"} {
		t.Run(boundary, func(t *testing.T) {
			dir := t.TempDir()
			runCLI := func(args ...string) {
				t.Helper()
				var out, stderr bytes.Buffer
				if code := Execute(args, &out, &stderr); code != 0 {
					t.Fatalf("%v: code=%d out=%s err=%s", args, code, &out, &stderr)
				}
			}
			runCLI("search", "batch", "--query", "alpha", "--query", "beta", "--sources", "openalex", "--limit", "2", "--out", dir)
			control, err := os.ReadFile(filepath.Join(dir, "results-deduped.jsonl"))
			if err != nil {
				t.Fatal(err)
			}
			// Fixture: alpha is committed and beta has not yet run. Keep the
			// normal artifacts so the real adapter and receipt writer run too.
			store := jobs.NewStore(dir)
			job, err := store.Create(search.BatchJobKind, search.BatchJobInput{Queries: []string{"alpha", "beta"}, Sources: []string{"openalex"}, Limit: 2})
			if err != nil {
				t.Fatal(err)
			}
			state := readBatchState(t, dir)
			state.JobID, state.Completed, state.RequestsUsed = job.ID, state.Completed[:1], 1
			*state.ResultCount = 1
			if err := search.WriteBatchState(dir, state); err != nil {
				t.Fatal(err)
			}
			results := bytes.SplitAfter(control, []byte("\n"))[0]
			if err := os.WriteFile(filepath.Join(dir, "results.jsonl"), results, 0o644); err != nil {
				t.Fatal(err)
			}
			target := filepath.Join(dir, boundary)
			if boundary == "job" {
				target = filepath.Join(dir, "jobs", job.ID+".json")
			}
			killJobAtPublication(t, dir, job.ID, target)
			interrupted, err := store.Status(job.ID)
			if err != nil || interrupted.Status != jobs.Interrupted {
				t.Fatalf("publication kill lost job: %+v, %v", interrupted, err)
			}
			runCLI("jobs", "resume", job.ID, "--dir", dir)
			final, err := store.Status(job.ID)
			if err != nil || final.Status != jobs.Completed || final.Progress.Completed != 2 {
				t.Fatalf("resume after publication kill: %+v, %v", final, err)
			}
			got, err := os.ReadFile(filepath.Join(dir, "results-deduped.jsonl"))
			if err != nil || !bytes.Equal(got, control) {
				t.Fatalf("resumed records differ: %s / %s (%v)", got, control, err)
			}
		})
	}
}

func killJobAtPublication(t *testing.T, dir, id, target string) {
	t.Helper()
	runtime.LockOSThread()
	defer runtime.UnlockOSThread()
	ctx, cancel := context.WithTimeout(context.Background(), 15*time.Second)
	defer cancel()
	cmd := exec.CommandContext(ctx, os.Args[0], "-test.run=^TestJobsResumeKilledDuringPublication$")
	cmd.Env = append(os.Environ(), "RFORGE_CHECKPOINT_KILL_DIR="+dir, "RFORGE_CHECKPOINT_KILL_ID="+id)
	pipe, err := cmd.StdoutPipe()
	if err != nil {
		t.Fatal(err)
	}
	cmd.Stderr = os.Stderr
	if err := cmd.Start(); err != nil {
		t.Fatal(err)
	}
	defer func() { _ = cmd.Process.Kill(); _ = cmd.Wait() }()
	scanner := bufio.NewScanner(pipe)
	if !scanner.Scan() {
		t.Fatal("worker did not report its tracing thread")
	}
	if strings.HasPrefix(scanner.Text(), "ptrace unavailable:") {
		t.Skip(scanner.Text()) // Sandboxes may prohibit tracing even child processes.
	}
	tid, err := strconv.Atoi(scanner.Text())
	if err != nil {
		t.Fatal(err)
	}
	var status syscall.WaitStatus
	if _, err := syscall.Wait4(tid, &status, syscall.WALL, nil); err != nil {
		t.Fatal(err)
	}
	if !status.Stopped() {
		t.Fatalf("worker did not stop: %v", status)
	}
	if err := syscall.PtraceSetOptions(tid, syscall.PTRACE_O_TRACESYSGOOD); err != nil {
		t.Fatal(err)
	}
	pathAt := func(addr uint64) string {
		var path []byte
		var b [1]byte
		for len(path) < 4096 {
			if _, err := syscall.PtracePeekData(tid, uintptr(addr)+uintptr(len(path)), b[:]); err != nil {
				t.Fatal(err)
			}
			if b[0] == 0 {
				break
			}
			path = append(path, b[0])
		}
		return string(path)
	}
	entering, replacing := true, false
	for {
		if err := syscall.PtraceSyscall(tid, 0); err != nil {
			t.Fatal(err)
		}
		if _, err := syscall.Wait4(tid, &status, syscall.WALL, nil); err != nil {
			t.Fatal(err)
		}
		if !status.Stopped() {
			t.Fatalf("worker exited before publishing %s: %v", target, status)
		}
		if status.StopSignal() != syscall.SIGTRAP|0x80 {
			continue
		}
		var regs syscall.PtraceRegs
		if err := syscall.PtraceGetRegs(tid, &regs); err != nil {
			t.Fatal(err)
		}
		if entering {
			replacing = false
			switch regs.Orig_rax {
			case syscall.SYS_RENAME:
				replacing = pathAt(regs.Rdi) == target || pathAt(regs.Rsi) == target
			case syscall.SYS_RENAMEAT:
				replacing = pathAt(regs.Rsi) == target || pathAt(regs.R10) == target
			case syscall.SYS_UNLINK:
				replacing = pathAt(regs.Rdi) == target
			case syscall.SYS_UNLINKAT:
				replacing = pathAt(regs.Rsi) == target
			}
		} else if replacing && regs.Rax == 0 {
			break // Kill after the publication syscall, before the next user instruction.
		}
		entering = !entering
	}
	if err := cmd.Process.Kill(); err != nil {
		t.Fatal(err)
	}
	_ = cmd.Wait()
}
