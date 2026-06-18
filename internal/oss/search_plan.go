package oss

import (
	"fmt"
	"sort"
	"strings"
)

// SearchProviderPlan describes one provider/query an agent can use to discover
// open-source projects without committing to cloning or integrating them.
type SearchProviderPlan struct {
	Provider  string   `json:"provider"`
	Kind      string   `json:"kind"`
	Query     string   `json:"query"`
	URL       string   `json:"url"`
	Signals   []string `json:"signals"`
	HumanGate string   `json:"humanGate"`
	Notes     string   `json:"notes"`
}

// SearchPlan is a deterministic multi-provider OSS discovery plan.
type SearchPlan struct {
	SchemaVersion string               `json:"schemaVersion"`
	Query         string               `json:"query"`
	Ecosystem     string               `json:"ecosystem"`
	Providers     []SearchProviderPlan `json:"providers"`
}

// BuildSearchPlan builds a source-coverage-first plan across code forges,
// package registries, archival indexes, and dependency/security databases.
func BuildSearchPlan(query, ecosystem string) (SearchPlan, error) {
	query = strings.TrimSpace(query)
	if query == "" {
		return SearchPlan{}, fmt.Errorf("OSS search query is required")
	}
	ecosystem = strings.ToLower(strings.TrimSpace(ecosystem))
	if ecosystem == "" {
		ecosystem = "all"
	}
	providers := baseSearchProviders(query)
	providers = append(providers, ecosystemProviders(query, ecosystem)...)
	sort.SliceStable(providers, func(i, j int) bool { return providers[i].Provider < providers[j].Provider })
	return SearchPlan{SchemaVersion: "1", Query: query, Ecosystem: ecosystem, Providers: providers}, nil
}

func baseSearchProviders(query string) []SearchProviderPlan {
	escaped := strings.ReplaceAll(query, " ", "+")
	return []SearchProviderPlan{
		provider("GitHub", "forge", query, "https://github.com/search?q="+escaped+"&type=repositories", []string{"stars", "forks", "recent commits", "issues", "license", "topics"}, "pattern-reference before dependency/integration", "Use GitHub API/search for broad discovery; avoid star-only ranking."),
		provider("GitLab", "forge", query, "https://gitlab.com/search?search="+escaped+"&scope=projects", []string{"last activity", "stars", "forks", "license", "CI config"}, "pattern-reference before dependency/integration", "Covers projects not mirrored to GitHub."),
		provider("Codeberg", "forge", query, "https://codeberg.org/explore/repos?q="+escaped, []string{"stars", "forks", "activity", "license"}, "pattern-reference before dependency/integration", "Forgejo/Gitea ecosystem coverage."),
		provider("SourceHut", "forge", query, "https://sr.ht/projects?search="+escaped, []string{"mailing-list activity", "repository activity", "license"}, "pattern-reference before dependency/integration", "Useful for projects that avoid social-code metrics."),
		provider("Software Heritage", "archive", query, "https://archive.softwareheritage.org/browse/search/?q="+escaped, []string{"archive presence", "origin URLs", "visit status"}, "archive/reference only", "Use to detect archived origins and mirrors; not a quality signal alone."),
		provider("OpenSSF Scorecard", "security", query, "scorecard --repo <candidate-url>", []string{"branch protection", "dependency update policy", "CI tests", "security policy"}, "security review before integration", "Run after candidates are selected, not as initial search."),
	}
}

func ecosystemProviders(query, ecosystem string) []SearchProviderPlan {
	all := []SearchProviderPlan{}
	add := func(p SearchProviderPlan) { all = append(all, p) }
	if ecosystem == "all" || ecosystem == "go" {
		add(provider("pkg.go.dev", "package-registry", query, "https://pkg.go.dev/search?q="+strings.ReplaceAll(query, " ", "+"), []string{"import path", "versions", "licenses", "dependents"}, "dependency approval before import", "Best for Go library discovery and module health."))
	}
	if ecosystem == "all" || ecosystem == "python" {
		add(provider("PyPI", "package-registry", query, "https://pypi.org/search/?q="+strings.ReplaceAll(query, " ", "+"), []string{"release cadence", "project links", "license", "downloads"}, "dependency approval before import", "Pair with repository links because PyPI search quality is limited."))
	}
	if ecosystem == "all" || ecosystem == "javascript" || ecosystem == "js" || ecosystem == "node" {
		add(provider("npm", "package-registry", query, "https://www.npmjs.com/search?q="+strings.ReplaceAll(query, " ", "%20"), []string{"weekly downloads", "versions", "maintainers", "repository link"}, "dependency approval before import", "Use package metadata plus repository inspection to avoid popularity-only selection."))
	}
	if ecosystem == "all" || ecosystem == "rust" {
		add(provider("crates.io", "package-registry", query, "https://crates.io/search?q="+strings.ReplaceAll(query, " ", "+"), []string{"downloads", "recent versions", "repository", "license"}, "dependency approval before import", "Best for Rust ecosystem candidates."))
	}
	if ecosystem == "all" || ecosystem == "data" {
		add(provider("Zenodo/GitHub links", "research-archive", query, "https://zenodo.org/search?q="+strings.ReplaceAll(query, " ", "+"), []string{"DOI", "software release archive", "license"}, "citation/reference review", "Find citable releases for research software."))
	}
	return all
}

func provider(name, kind, query, url string, signals []string, gate, notes string) SearchProviderPlan {
	return SearchProviderPlan{Provider: name, Kind: kind, Query: query, URL: url, Signals: signals, HumanGate: gate, Notes: notes}
}
