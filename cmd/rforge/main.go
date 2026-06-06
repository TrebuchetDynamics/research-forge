package main

import (
	"os"

	"github.com/TrebuchetDynamics/research-forge/internal/cli"
)

var (
	version = "dev"
	commit  = "unknown"
	date    = "unknown"
)

func main() {
	cli.Version = version
	cli.Commit = commit
	cli.Date = date
	os.Exit(cli.Execute(os.Args[1:], os.Stdout, os.Stderr))
}
