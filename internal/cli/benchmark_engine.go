package cli

import (
	"context"
	"fmt"
	"io"
	"os"

	"github.com/TrebuchetDynamics/research-forge/internal/benchmarks"
	"github.com/TrebuchetDynamics/research-forge/internal/filetxn"
)

func executeBenchmarkEngine(args []string, stdout, stderr io.Writer) int {
	o := benchmarks.EngineOptions{Split: "held_out", Version: Version, Revision: Commit}
	out := ""
	mode := ""
	seen := map[string]bool{}
	usage := func() int {
		fmt.Fprintln(stderr, "usage: rforge benchmark run --set <dir> [--replay|--record] [--split dev|held_out] [--out <report.json>]")
		return 2
	}
	for i := 0; i < len(args); i++ {
		flag := args[i]
		if seen[flag] {
			return usage()
		}
		seen[flag] = true
		switch flag {
		case "--replay", "--record":
			if mode != "" {
				return usage()
			}
			mode = flag
			o.Record = flag == "--record"
		case "--set", "--split", "--out":
			if i+1 >= len(args) || args[i+1] == "" {
				return usage()
			}
			i++
			switch flag {
			case "--set":
				o.SetDir = args[i]
			case "--split":
				o.Split = args[i]
			case "--out":
				out = args[i]
			}
		default:
			return usage()
		}
	}
	if o.SetDir == "" || (o.Split != "dev" && o.Split != "held_out") {
		return usage()
	}
	result, err := benchmarks.RunEngine(context.Background(), o)
	code := 0
	if err != nil {
		code = 1
	}
	// Audit artifacts contain scorer-private labels/membership. Never hand them
	// to an agent. Stdout remains exclusively canonical report JSON.
	if out != "" {
		outputs := []filetxn.Output{{Path: out, Data: result.Report, Mode: 0600}}
		if len(result.Audit) > 0 {
			outputs = append(outputs, filetxn.Output{Path: out + ".audit.json", Data: result.Audit, Mode: 0600})
		}
		// ReplaceAll preserves existing output permissions. Tighten the private
		// sidecar even when a previous artifact was accidentally world-readable.
		if e := filetxn.ReplaceAllThen(outputs, func() error {
			if len(result.Audit) > 0 {
				return os.Chmod(out+".audit.json", 0600)
			}
			return nil
		}); e != nil {
			fmt.Fprintln(stderr, "benchmark output:", e)
			return 1
		}
		fmt.Fprintln(stderr, "wrote benchmark report to", out)
	} else {
		if len(result.Audit) > 0 {
			f, e := os.CreateTemp("", "rforge-benchmark-audit-*.json")
			if e != nil {
				fmt.Fprintln(stderr, "benchmark audit:", e)
				return 1
			}
			path := f.Name()
			_, e = f.Write(result.Audit)
			closeErr := f.Close()
			if e == nil {
				e = closeErr
			}
			if e != nil {
				_ = os.Remove(path)
				fmt.Fprintln(stderr, "benchmark audit:", e)
				return 1
			}
			fmt.Fprintln(stderr, "private benchmark audit:", path)
		}
		if _, e := stdout.Write(result.Report); e != nil {
			return 1
		}
	}
	return code
}
