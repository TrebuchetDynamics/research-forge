package cli

import (
	"errors"
	"io"

	"github.com/TrebuchetDynamics/research-forge/internal/research"
)

func researchService() research.Service {
	return research.Service{
		Build:     research.BuildInfo{Version: Version, Commit: Commit, Date: Date},
		Connector: func(name string) (research.SearchConnector, bool) { return searchConnector(name) },
	}
}

func writeResearchError(stdout, stderr io.Writer, opts globalOptions, err error) int {
	var operation *research.Error
	if errors.As(err, &operation) {
		return writeError(stdout, stderr, opts, operation.ExitCode, operation.Code, operation.Error())
	}
	return writeError(stdout, stderr, opts, 1, "research_failed", err.Error())
}
