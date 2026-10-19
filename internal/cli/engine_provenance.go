package cli

import "github.com/TrebuchetDynamics/research-forge/internal/provenance"

// recordEngineRun keeps all command adapters on the same receipt schema. The
// optional sidecar is an alias for compatibility, not the immutable authority.
func recordEngineRun(root, action string, metadata map[string]any, outputs []string, sidecar string) (provenance.Validation, error) {
	if metadata == nil {
		metadata = map[string]any{}
	}
	if _, ok := metadata["question"]; !ok {
		metadata["question"] = action
	}
	if _, ok := metadata["depth"]; !ok {
		metadata["depth"] = "standard"
	}
	if _, ok := metadata["sources"]; !ok {
		metadata["sources"] = []string{}
	}
	if _, ok := metadata["errors"]; !ok {
		metadata["errors"] = []string{}
	}
	metadata["rforge_version"] = map[string]string{"version": Version, "commit": Commit, "date": Date}
	var aliases []string
	if sidecar != "" {
		aliases = append(aliases, sidecar)
	}
	return provenance.WriteRun(root, action, metadata, outputs, aliases...)
}
