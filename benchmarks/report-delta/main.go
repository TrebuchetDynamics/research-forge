// Report-delta compares existing engine replay reports offline. It never runs
// retrieval, writes input files, or authorizes research-quality claims.
package main

import (
	"flag"
	"fmt"
	"io"
	"os"

	b "github.com/TrebuchetDynamics/research-forge/internal/benchmarks"
)

func run(args []string, stdout, stderr io.Writer) int {
	flags := flag.NewFlagSet("report-delta", flag.ContinueOnError)
	flags.SetOutput(stderr)
	baseline := flags.String("baseline", "", "baseline engine replay report JSON")
	candidate := flags.String("candidate", "", "candidate engine replay report JSON")
	if err := flags.Parse(args); err != nil {
		return 2
	}
	if *baseline == "" || *candidate == "" || flags.NArg() != 0 {
		fmt.Fprintln(stderr, "usage: report-delta --baseline OLD.json --candidate NEW.json")
		return 2
	}
	var old, new b.EngineReport
	for _, input := range []struct {
		path   string
		report *b.EngineReport
	}{{*baseline, &old}, {*candidate, &new}} {
		// Despite its historical name, this is the shared strict benchmark decoder:
		// missing/unknown fields, duplicate keys and malformed JSON all fail closed.
		if err := b.ReadAgentArtifact(input.path, input.report); err != nil {
			fmt.Fprintf(stderr, "report-delta: %s: %v\n", input.path, err)
			return 1
		}
	}
	comparison, err := b.CompareEngineReports(old, new)
	if err != nil {
		fmt.Fprintf(stderr, "report-delta: %v\n", err)
		return 1
	}
	data, err := b.CanonicalJSON(comparison)
	if err != nil {
		fmt.Fprintf(stderr, "report-delta: %v\n", err)
		return 1
	}
	if _, err = stdout.Write(data); err != nil {
		fmt.Fprintf(stderr, "report-delta: write: %v\n", err)
		return 1
	}
	return 0
}

func main() { os.Exit(run(os.Args[1:], os.Stdout, os.Stderr)) }
