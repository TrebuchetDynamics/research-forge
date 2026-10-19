package cli

import (
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"strings"
	"time"

	"github.com/TrebuchetDynamics/research-forge/internal/claims"
	"github.com/TrebuchetDynamics/research-forge/internal/filetxn"
)

const claimUsage = "usage: rforge claim add --file <claim.json> | list | show --id <id> | check --id <id> --project <path> [--json]"

// claimView adds computed attribution without changing the shared Claim schema
// or accepting a caller-supplied provenance level. It is not a truth judgment.
type claimView struct {
	claims.Claim
	SemanticSupportLevel string `json:"semanticSupportLevel,omitempty"`
}

func viewClaim(record claims.Claim) claimView {
	view := claimView{Claim: record}
	if record.Facets.SemanticSupport != nil {
		view.SemanticSupportLevel = "agent_reported"
	}
	return view
}

func executeClaim(args []string, stdout, stderr io.Writer, opts globalOptions) int {
	if len(args) == 0 {
		return writeError(stdout, stderr, opts, 2, "usage", claimUsage)
	}
	subcommand := args[0]
	file, id := "", ""
	for i := 1; i < len(args); i++ {
		switch args[i] {
		case "--json":
			opts.JSON = true
		case "--project", "--file", "--id":
			if i+1 >= len(args) || strings.TrimSpace(args[i+1]) == "" {
				return writeError(stdout, stderr, opts, 2, "usage", claimUsage)
			}
			switch args[i] {
			case "--project":
				opts.Project = args[i+1]
			case "--file":
				file = args[i+1]
			case "--id":
				id = args[i+1]
			}
			i++
		default:
			return writeError(stdout, stderr, opts, 2, "usage", claimUsage)
		}
	}
	if strings.TrimSpace(opts.Project) == "" {
		return writeError(stdout, stderr, opts, 2, "missing_project", "--project is required for claim commands")
	}
	switch subcommand {
	case "add":
		if file == "" || id != "" {
			return writeError(stdout, stderr, opts, 2, "usage", claimUsage)
		}
		input, err := filetxn.OpenRegular(file)
		if err != nil {
			return writeError(stdout, stderr, opts, 1, "claim_input_failed", err.Error())
		}
		decoder := json.NewDecoder(input)
		decoder.DisallowUnknownFields()
		var record claims.Claim
		decodeErr := decoder.Decode(&record)
		if decodeErr == nil {
			var extra any
			if err := decoder.Decode(&extra); err != io.EOF {
				decodeErr = fmt.Errorf("claim input must contain exactly one JSON record")
			}
		}
		closeErr := input.Close()
		if decodeErr != nil {
			return writeError(stdout, stderr, opts, 2, "claim_input_invalid", decodeErr.Error())
		}
		if closeErr != nil {
			return writeError(stdout, stderr, opts, 1, "claim_input_failed", closeErr.Error())
		}
		if err := claims.Append(opts.Project, record); err != nil {
			return writeError(stdout, stderr, opts, 1, "claim_add_failed", err.Error())
		}
		if opts.JSON {
			return writeJSON(stdout, 0, map[string]any{"claim": viewClaim(record)})
		}
		fmt.Fprintf(stdout, "appended claim %s\n", record.ID)
		return 0
	case "list":
		if file != "" || id != "" {
			return writeError(stdout, stderr, opts, 2, "usage", claimUsage)
		}
		records, err := claims.List(opts.Project)
		if err != nil {
			return writeError(stdout, stderr, opts, 1, "claim_read_failed", err.Error())
		}
		views := make([]claimView, 0, len(records))
		for _, record := range records {
			views = append(views, viewClaim(record))
		}
		if opts.JSON {
			return writeJSON(stdout, 0, map[string]any{"claims": views, "count": len(views)})
		}
		for _, record := range records {
			fmt.Fprintf(stdout, "%s\t%s\n", record.ID, record.Text)
		}
		return 0
	case "show", "check":
		if id == "" || file != "" {
			return writeError(stdout, stderr, opts, 2, "usage", claimUsage)
		}
		record, err := claims.Show(opts.Project, id)
		if errors.Is(err, claims.ErrNotFound) {
			return writeError(stdout, stderr, opts, 1, "claim_not_found", err.Error())
		}
		if err != nil {
			return writeError(stdout, stderr, opts, 1, "claim_read_failed", err.Error())
		}
		if subcommand == "check" {
			records, err := loadOptionalLibraryRecords(opts.Project)
			if err != nil {
				return writeError(stdout, stderr, opts, 1, "claim_library_read_failed", err.Error())
			}
			docs, err := loadOptionalParsedDocuments(opts.Project)
			if err != nil {
				return writeError(stdout, stderr, opts, 1, "claim_parsed_read_failed", err.Error())
			}
			result, err := claims.Check(record, claims.CheckInput{LibraryRecords: records, ParsedDocuments: docs, Timestamp: time.Now().UTC()})
			if err != nil {
				return writeError(stdout, stderr, opts, 1, "claim_check_failed", err.Error())
			}
			if err := claims.Append(opts.Project, result.Claim); err != nil {
				return writeError(stdout, stderr, opts, 1, "claim_check_failed", err.Error())
			}
			output := map[string]any{"claim": viewClaim(result.Claim), "evidence": result.Evidence}
			if opts.JSON {
				return writeJSON(stdout, 0, output)
			}
			data, err := json.MarshalIndent(output, "", "  ")
			if err != nil {
				return writeError(stdout, stderr, opts, 1, "claim_check_failed", err.Error())
			}
			fmt.Fprintln(stdout, string(data))
			return 0
		}
		if opts.JSON {
			return writeJSON(stdout, 0, map[string]any{"claim": viewClaim(record)})
		}
		data, err := json.MarshalIndent(viewClaim(record), "", "  ")
		if err != nil {
			return writeError(stdout, stderr, opts, 1, "claim_read_failed", err.Error())
		}
		fmt.Fprintln(stdout, string(data))
		return 0
	default:
		return writeError(stdout, stderr, opts, 2, "usage", claimUsage)
	}
}
