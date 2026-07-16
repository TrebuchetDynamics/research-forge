package oss

import (
	"encoding/json"
	"fmt"
	"net/http"
	"net/url"
	"strings"
	"time"
)

// GitHubMetadataOptions configures inventory metadata refresh from a GitHub-compatible API.
type GitHubMetadataOptions struct {
	BaseURL string
	Client  *http.Client
	Now     func() time.Time
}

// InventoryRefreshResult reports an inventory metadata refresh outcome.
type InventoryRefreshResult struct {
	Refreshed int      `json:"refreshed"`
	Skipped   int      `json:"skipped"`
	Issues    []string `json:"issues"`
}

// RefreshInventoryGitHubMetadata refreshes GitHub repository metadata in-place for manifest entries with owner/repo repositories.
func RefreshInventoryGitHubMetadata(path string, opts GitHubMetadataOptions) (InventoryRefreshResult, error) {
	manifest, err := LoadInventoryManifest(path)
	if err != nil {
		return InventoryRefreshResult{}, err
	}
	baseURL := strings.TrimRight(opts.BaseURL, "/")
	if baseURL == "" {
		baseURL = "https://api.github.com"
	}
	client := opts.Client
	if client == nil {
		client = &http.Client{Timeout: 15 * time.Second}
	}
	now := opts.Now
	if now == nil {
		now = time.Now
	}
	result := InventoryRefreshResult{}
	for i, entry := range manifest.Entries {
		repo := strings.TrimSpace(entry.Repository)
		if repo == "" || strings.Count(repo, "/") != 1 {
			result.Skipped++
			continue
		}
		metadata, err := fetchGitHubRepositoryMetadata(client, baseURL, repo)
		if err != nil {
			result.Issues = append(result.Issues, repo+": "+err.Error())
			continue
		}
		manifest.Entries[i].Stars = metadata.StargazersCount
		manifest.Entries[i].Forks = metadata.ForksCount
		manifest.Entries[i].Archived = metadata.Archived
		manifest.Entries[i].PushedAt = strings.TrimSpace(metadata.PushedAt)
		manifest.Entries[i].LicenseSPDX = strings.TrimSpace(metadata.License.SPDXID)
		manifest.Entries[i].MetadataRefreshedAt = now().UTC().Format(time.RFC3339)
		result.Refreshed++
	}
	if len(result.Issues) > 0 {
		return result, nil
	}
	return result, SaveInventoryManifest(path, manifest)
}

// SaveInventoryManifest writes a machine-readable OSS inventory manifest.
func SaveInventoryManifest(path string, manifest InventoryManifest) error {
	data, err := json.MarshalIndent(manifest, "", "  ")
	if err != nil {
		return err
	}
	data = append(data, '\n')
	return writeOutput(path, data)
}

type githubRepositoryMetadata struct {
	StargazersCount int    `json:"stargazers_count"`
	ForksCount      int    `json:"forks_count"`
	Archived        bool   `json:"archived"`
	PushedAt        string `json:"pushed_at"`
	License         struct {
		SPDXID string `json:"spdx_id"`
	} `json:"license"`
}

func fetchGitHubRepositoryMetadata(client *http.Client, baseURL, repo string) (githubRepositoryMetadata, error) {
	endpoint, err := url.Parse(baseURL + "/repos/" + repo)
	if err != nil {
		return githubRepositoryMetadata{}, err
	}
	request, err := http.NewRequest(http.MethodGet, endpoint.String(), nil)
	if err != nil {
		return githubRepositoryMetadata{}, err
	}
	request.Header.Set("User-Agent", "ResearchForge-inventory/dev")
	response, err := client.Do(request)
	if err != nil {
		return githubRepositoryMetadata{}, err
	}
	defer response.Body.Close()
	if response.StatusCode < 200 || response.StatusCode >= 300 {
		return githubRepositoryMetadata{}, fmt.Errorf("GitHub status %d", response.StatusCode)
	}
	var metadata githubRepositoryMetadata
	if err := json.NewDecoder(response.Body).Decode(&metadata); err != nil {
		return githubRepositoryMetadata{}, err
	}
	return metadata, nil
}
