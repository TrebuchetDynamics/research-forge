package provenance

import (
	"fmt"
	"strings"
)

func validateRecord(record map[string]any) []string {
	var problems []string
	for _, field := range []string{"schema_version", "question", "depth", "sources", "timestamp", "rforge_version", "outputs", "errors"} {
		if _, ok := record[field]; !ok {
			problems = append(problems, fmt.Sprintf("missing required field %q", field))
		}
	}
	schema, ok := record["schema_version"].(string)
	if !ok || schema != "1" && schema != "2" {
		problems = append(problems, `schema_version must be "1" or "2"`)
	}
	for _, field := range []string{"question", "timestamp"} {
		if value, ok := record[field].(string); !ok || strings.TrimSpace(value) == "" {
			problems = append(problems, field+" must be a non-empty string")
		}
	}
	if depth, ok := record["depth"].(string); !ok || !(depth == "quick" || depth == "standard" || depth == "comprehensive") {
		problems = append(problems, "depth must be one of quick, standard, comprehensive")
	}
	version, ok := record["rforge_version"].(map[string]any)
	if !ok {
		problems = append(problems, "rforge_version must be an object copied from `rforge --json version` data")
	} else {
		for _, field := range []string{"version", "commit", "date"} {
			if value, ok := version[field].(string); !ok || strings.TrimSpace(value) == "" {
				problems = append(problems, fmt.Sprintf("rforge_version.%s must be a non-empty string", field))
			}
		}
	}
	fields := []string{"sources", "errors"}
	if schema == "1" {
		fields = append(fields, "outputs")
	}
	if _, exists := record["queries"]; exists {
		fields = append(fields, "queries")
	}
	for _, field := range fields {
		values, ok := record[field].([]any)
		if !ok {
			problems = append(problems, field+" must be an array of strings")
			continue
		}
		for i, value := range values {
			if _, ok := value.(string); !ok {
				problems = append(problems, fmt.Sprintf("%s[%d] must be a string, got %T", field, i, value))
			}
		}
	}
	return problems
}
