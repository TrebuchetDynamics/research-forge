package cli

import (
	"cmp"
	"encoding/json"
	"fmt"
	"io"
	"regexp"
	"slices"
	"strconv"
	"strings"
)

// Contract versions are minimum-compatible, monotonically increasing integers.
// Only implemented families belong here; mcp and claims are not yet available.
func commandCapabilities() map[string]int {
	return map[string]int{"search": 1, "paper": 1, "brief": 1, "compare": 1, "provenance": 1}
}

type capabilityRequirement struct {
	name    string
	version int
}

const capabilitiesUsage = "usage: rforge capabilities [--json] [--min-rforge-version <version>] [--require <family>=<contract-version>]..."

func executeCapabilities(args []string, stdout, stderr io.Writer, opts globalOptions) int {
	// Accept --json after the command as well as the global prefix, including
	// on failures, so adapters always receive the same error envelope.
	filtered := make([]string, 0, len(args))
	for _, arg := range args {
		if arg == "--json" {
			opts.JSON = true
		} else {
			filtered = append(filtered, arg)
		}
	}
	minimum := ""
	var requirements []capabilityRequirement
	for i := 0; i < len(filtered); i++ {
		if i+1 >= len(filtered) {
			return writeError(stdout, stderr, opts, 2, "usage", capabilitiesUsage)
		}
		value := filtered[i+1]
		switch filtered[i] {
		case "--min-rforge-version":
			if _, ok := parseCapabilityVersion(value); !ok || minimum != "" {
				return writeError(stdout, stderr, opts, 2, "usage", capabilitiesUsage+"; minimum must be a semantic version")
			}
			minimum = value
		case "--require":
			name, contract, ok := strings.Cut(value, "=")
			version, err := strconv.Atoi(contract)
			if !ok || strings.TrimSpace(name) != name || name == "" || err != nil || version < 1 {
				return writeError(stdout, stderr, opts, 2, "usage", capabilitiesUsage+"; contract versions must be positive integers")
			}
			requirements = append(requirements, capabilityRequirement{name, version})
		default:
			return writeError(stdout, stderr, opts, 2, "usage", capabilitiesUsage)
		}
		i++
	}

	if minimum != "" && Version != "dev" {
		actual, ok := parseCapabilityVersion(Version)
		if !ok {
			return writeCapabilityGap(stdout, stderr, opts, "version_invalid", "", Version, minimum)
		}
		required, _ := parseCapabilityVersion(minimum)
		if compareCapabilityVersions(actual, required) < 0 {
			return writeCapabilityGap(stdout, stderr, opts, "version_too_old", "", Version, minimum)
		}
	}
	capabilities := commandCapabilities()
	for _, requirement := range requirements {
		actual, exists := capabilities[requirement.name]
		if !exists {
			return writeCapabilityGap(stdout, stderr, opts, "capability_missing", requirement.name, "missing", strconv.Itoa(requirement.version))
		}
		if actual < requirement.version {
			return writeCapabilityGap(stdout, stderr, opts, "capability_too_old", requirement.name, strconv.Itoa(actual), strconv.Itoa(requirement.version))
		}
	}

	errorCodes := []string{"usage", "version_invalid", "version_too_old", "capability_missing", "capability_too_old"}
	if opts.JSON {
		return writeJSON(stdout, 0, map[string]any{
			"name": "rforge", "version": Version, "commit": Commit, "date": Date,
			"capabilities": capabilities, "error_codes": errorCodes,
		})
	}
	fmt.Fprintf(stdout, "rforge %s (%s, %s)\n", Version, Commit, Date)
	names := make([]string, 0, len(capabilities))
	for name := range capabilities {
		names = append(names, name)
	}
	slices.Sort(names)
	for _, name := range names {
		fmt.Fprintf(stdout, "%s\t%d\n", name, capabilities[name])
	}
	fmt.Fprintf(stdout, "error_codes: %s\n", strings.Join(errorCodes, ", "))
	return 0
}

func writeCapabilityGap(stdout, stderr io.Writer, opts globalOptions, code, capability, actual, required string) int {
	subject := "rforge version"
	if capability != "" {
		subject = "capability " + capability
	}
	message := fmt.Sprintf("%s: found %s, require >= %s; install or upgrade rforge with install.sh", subject, actual, required)
	if !opts.JSON {
		return writeError(stdout, stderr, opts, 1, code, message)
	}
	detail := map[string]string{
		"code": code, "message": message, "actual": actual, "required": required, "install": "install.sh",
	}
	if capability != "" {
		detail["capability"] = capability
	}
	_ = json.NewEncoder(stdout).Encode(map[string]any{"ok": false, "error": detail})
	return 1
}

// Use SemVer precedence (optional v prefix), not lexical ordering. Keeping the
// numeric components as strings avoids imposing machine integer limits on build
// versions. Build metadata is validated but does not affect precedence.
var capabilityVersionPattern = regexp.MustCompile(`^v?(0|[1-9][0-9]*)\.(0|[1-9][0-9]*)\.(0|[1-9][0-9]*)(?:-([0-9A-Za-z-]+(?:\.[0-9A-Za-z-]+)*))?(?:\+[0-9A-Za-z-]+(?:\.[0-9A-Za-z-]+)*)?$`)

type capabilityVersion struct {
	core       [3]string
	prerelease []string
}

func parseCapabilityVersion(value string) (capabilityVersion, bool) {
	parts := capabilityVersionPattern.FindStringSubmatch(value)
	if parts == nil {
		return capabilityVersion{}, false
	}
	version := capabilityVersion{core: [3]string{parts[1], parts[2], parts[3]}}
	if parts[4] != "" {
		version.prerelease = strings.Split(parts[4], ".")
		for _, identifier := range version.prerelease {
			if capabilityNumeric(identifier) && len(identifier) > 1 && identifier[0] == '0' {
				return capabilityVersion{}, false
			}
		}
	}
	return version, true
}

func compareCapabilityVersions(a, b capabilityVersion) int {
	for i := range a.core {
		if order := compareCapabilityNumbers(a.core[i], b.core[i]); order != 0 {
			return order
		}
	}
	if len(a.prerelease) == 0 && len(b.prerelease) != 0 {
		return 1
	}
	if len(a.prerelease) != 0 && len(b.prerelease) == 0 {
		return -1
	}
	for i := 0; i < len(a.prerelease) && i < len(b.prerelease); i++ {
		x, y := a.prerelease[i], b.prerelease[i]
		order := strings.Compare(x, y)
		xNumeric, yNumeric := capabilityNumeric(x), capabilityNumeric(y)
		switch {
		case xNumeric && yNumeric:
			order = compareCapabilityNumbers(x, y)
		case xNumeric && !yNumeric:
			order = -1
		case !xNumeric && yNumeric:
			order = 1
		}
		if order != 0 {
			return order
		}
	}
	return cmp.Compare(len(a.prerelease), len(b.prerelease))
}

func compareCapabilityNumbers(a, b string) int {
	if order := cmp.Compare(len(a), len(b)); order != 0 {
		return order
	}
	return strings.Compare(a, b)
}

func capabilityNumeric(value string) bool {
	return strings.Trim(value, "0123456789") == ""
}
