package github

import (
	"bytes"
	"database/sql"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
	"strconv"
	"time"

	models "github.com/teomiscia/github-trending/internal/models"
)

const (
	defaultGitHubAPIURL = "https://api.github.com"
	// You might want to make this configurable or more dynamic
	defaultSearchQuery = "stars:>50"
)

// GitHubClient provides methods for interacting with the GitHub API.
type GitHubClient struct {
	httpClient *http.Client
	token      string
	baseURL    string // Add this field
}

// NewGitHubClient creates a new GitHubClient.
func NewGitHubClient(token string, httpClient *http.Client) *GitHubClient {
	if httpClient == nil {
		httpClient = &http.Client{Timeout: 10 * time.Second}
	}
	return &GitHubClient{
		httpClient: httpClient,
		token:      token,
		baseURL:    defaultGitHubAPIURL, // Initialize with default
	}
}

// SetBaseURL sets the base URL for the GitHub API client. Useful for testing.
func (c *GitHubClient) SetBaseURL(url string) {
	c.baseURL = url
}

// GithubOwner represents the owner of a repository as returned by the GitHub API.
type GithubOwner struct {
	Login     string  `json:"login"`
	ID        int     `json:"id"`
	NodeID    *string `json:"node_id"`
	AvatarURL string  `json:"avatar_url"`
	HTMLURL   string  `json:"html_url"`
	Type      string  `json:"type"`
	SiteAdmin *bool   `json:"site_admin"`
}

// GithubLicense represents the license of a repository as returned by the GitHub API.
type GithubLicense struct {
	Key    *string `json:"key"`
	Name   *string `json:"name"`
	SpdxID *string `json:"spdx_id"`
	URL    *string `json:"url"`
	NodeID *string `json:"node_id"`
}

// GithubRepository represents a repository as returned by the GitHub API.
type GithubRepository struct {
	ID              int            `json:"id"`
	NodeID          *string        `json:"node_id"`
	Name            string         `json:"name"`
	FullName        string         `json:"full_name"`
	Owner           GithubOwner    `json:"owner"`
	Private         bool           `json:"private"`
	HTMLURL         string         `json:"html_url"`
	Description     *string        `json:"description"`
	Fork            bool           `json:"fork"`
	URL             string         `json:"url"`
	CreatedAt       time.Time      `json:"created_at"`
	UpdatedAt       time.Time      `json:"updated_at"`
	PushedAt        time.Time      `json:"pushed_at"`
	Homepage        *string        `json:"homepage"`
	Size            int            `json:"size"`
	StargazersCount int            `json:"stargazers_count"`
	WatchersCount   int            `json:"watchers_count"`
	Language        *string        `json:"language"`
	ForksCount      int            `json:"forks_count"`
	OpenIssuesCount int            `json:"open_issues_count"`
	DefaultBranch   string         `json:"default_branch"`
	Score           float64        `json:"score"`
	Topics          []string       `json:"topics"`
	HasIssues       bool           `json:"has_issues"`
	HasProjects     bool           `json:"has_projects"`
	HasPages        bool           `json:"has_pages"`
	HasWiki         bool           `json:"has_wiki"`
	HasDownloads    bool           `json:"has_downloads"`
	HasDiscussions  bool           `json:"has_discussions"`
	Archived        bool           `json:"archived"`
	Disabled        bool           `json:"disabled"`
	Visibility      string         `json:"visibility"`
	License         *GithubLicense `json:"license"`
	AllowForking    bool           `json:"allow_forking"`
	IsTemplate      bool           `json:"is_template"`
}

// SearchRepositoriesResponse represents the structure of the GitHub API search response.
type SearchRepositoriesResponse struct {
	TotalCount        int                 `json:"total_count"`
	IncompleteResults bool                `json:"incomplete_results"`
	Items             []models.Repository `json:"items"`
}

type GithubSearchRepositoriesResponse struct {
	TotalCount        int                `json:"total_count"`
	IncompleteResults bool               `json:"incomplete_results"`
	Items             []GithubRepository `json:"items"`
}

func toModelsRepository(repo GithubRepository) models.Repository {
	mRepo := models.Repository{
		ID:              repo.ID,
		Name:            repo.Name,
		FullName:        repo.FullName,
		Private:         repo.Private,
		HTMLURL:         repo.HTMLURL,
		Fork:            repo.Fork,
		URL:             repo.URL,
		CreatedAt:       repo.CreatedAt,
		UpdatedAt:       repo.UpdatedAt,
		PushedAt:        repo.PushedAt,
		Size:            repo.Size,
		StargazersCount: repo.StargazersCount,
		WatchersCount:   repo.WatchersCount,
		ForksCount:      repo.ForksCount,
		OpenIssuesCount: repo.OpenIssuesCount,
		DefaultBranch:   repo.DefaultBranch,
		Score:           repo.Score,
		Topics:          repo.Topics,
		HasIssues:       repo.HasIssues,
		HasProjects:     repo.HasProjects,
		HasPages:        repo.HasPages,
		HasWiki:         repo.HasWiki,
		HasDownloads:    repo.HasDownloads,
		HasDiscussions:  repo.HasDiscussions,
		Archived:        repo.Archived,
		Disabled:        repo.Disabled,
		Visibility:      repo.Visibility,
		AllowForking:    repo.AllowForking,
		IsTemplate:      repo.IsTemplate,
		Owner: models.Owner{
			Login:     repo.Owner.Login,
			ID:        repo.Owner.ID,
			AvatarURL: repo.Owner.AvatarURL,
			HTMLURL:   repo.Owner.HTMLURL,
			Type:      repo.Owner.Type,
		},
		License: models.License{},
	}

	if repo.NodeID != nil {
		mRepo.NodeID = sql.NullString{String: *repo.NodeID, Valid: true}
	}
	if repo.Description != nil {
		mRepo.Description = sql.NullString{String: *repo.Description, Valid: true}
	}
	if repo.Homepage != nil {
		mRepo.Homepage = sql.NullString{String: *repo.Homepage, Valid: true}
	}
	if repo.Language != nil {
		mRepo.Language = sql.NullString{String: *repo.Language, Valid: true}
	}
	if repo.Owner.NodeID != nil {
		mRepo.Owner.NodeID = sql.NullString{String: *repo.Owner.NodeID, Valid: true}
	}
	if repo.Owner.SiteAdmin != nil {
		mRepo.Owner.SiteAdmin = sql.NullBool{Bool: *repo.Owner.SiteAdmin, Valid: true}
	}
	if repo.License != nil {
		if repo.License.Key != nil {
			mRepo.License.Key = sql.NullString{String: *repo.License.Key, Valid: true}
		}
		if repo.License.Name != nil {
			mRepo.License.Name = sql.NullString{String: *repo.License.Name, Valid: true}
		}
		if repo.License.SpdxID != nil {
			mRepo.License.SpdxID = sql.NullString{String: *repo.License.SpdxID, Valid: true}
		}
		if repo.License.URL != nil {
			mRepo.License.URL = sql.NullString{String: *repo.License.URL, Valid: true}
		}
		if repo.License.NodeID != nil {
			mRepo.License.NodeID = sql.NullString{String: *repo.License.NodeID, Valid: true}
		}
	}

	return mRepo
}

// get performs an authenticated GET, waiting out rate limits the way GitHub's headers ask.
func (c *GitHubClient) get(url string) ([]byte, error) {
	for attempt := 0; ; attempt++ {
		req, err := http.NewRequest("GET", url, nil)
		if err != nil {
			return nil, fmt.Errorf("failed to create request: %w", err)
		}
		req.Header.Set("Accept", "application/vnd.github.v3+json")
		if c.token != "" {
			req.Header.Set("Authorization", fmt.Sprintf("token %s", c.token))
		}

		resp, err := c.httpClient.Do(req)
		if err != nil {
			return nil, fmt.Errorf("failed to perform request: %w", err)
		}
		body, err := io.ReadAll(resp.Body)
		resp.Body.Close()
		if err != nil {
			return nil, fmt.Errorf("failed to read response body for %s: %w", url, err)
		}

		if wait, limited := rateLimitWait(resp, body, attempt); limited {
			log.Printf("Rate limited (%d) on %s. Waiting %v before retrying...", resp.StatusCode, url, wait)
			time.Sleep(wait)
			continue
		}
		if resp.StatusCode != http.StatusOK {
			return nil, fmt.Errorf("GitHub API returned non-200 status for %s: %d - %s", url, resp.StatusCode, string(body))
		}

		time.Sleep(2 * time.Second) // Wait 2 seconds between successful requests
		return body, nil
	}
}

// rateLimitWait reports whether resp is a rate-limit rejection and how long to back off.
// Order follows GitHub's docs: Retry-After, then X-RateLimit-Reset, then >=1 minute growing.
func rateLimitWait(resp *http.Response, body []byte, attempt int) (time.Duration, bool) {
	limited := resp.StatusCode == http.StatusTooManyRequests ||
		(resp.StatusCode == http.StatusForbidden && bytes.Contains(bytes.ToLower(body), []byte("rate limit")))
	if !limited {
		return 0, false
	}
	if s, err := strconv.Atoi(resp.Header.Get("Retry-After")); err == nil {
		return time.Duration(s) * time.Second, true
	}
	if resp.Header.Get("X-RateLimit-Remaining") == "0" {
		if reset, err := strconv.ParseInt(resp.Header.Get("X-RateLimit-Reset"), 10, 64); err == nil {
			return max(time.Until(time.Unix(reset, 0)), 0) + 5*time.Second, true
		}
	}
	return time.Minute << min(attempt, 4), true // 1, 2, 4, 8, 16 minutes
}

// SearchRepositories searches for repositories on GitHub based on a query.
func (c *GitHubClient) SearchRepositories(query string, page int) (*SearchRepositoriesResponse, error) {
	bodyBytes, err := c.get(fmt.Sprintf("%s/search/repositories?q=%s&page=%d&per_page=100", c.baseURL, query, page))
	if err != nil {
		return nil, err
	}

	var githubSearchResp GithubSearchRepositoriesResponse
	if err := json.Unmarshal(bodyBytes, &githubSearchResp); err != nil {
		return nil, fmt.Errorf("failed to unmarshal response: %w", err)
	}

	var modelRepos []models.Repository
	for _, item := range githubSearchResp.Items {
		modelRepos = append(modelRepos, toModelsRepository(item))
	}

	return &SearchRepositoriesResponse{
		TotalCount:        githubSearchResp.TotalCount,
		IncompleteResults: githubSearchResp.IncompleteResults,
		Items:             modelRepos,
	}, nil
}

// FetchRepoData is a placeholder for fetching detailed repository data.
// This function will be implemented by the crawler service.
func FetchRepoData(repoURL string, token string) ([]byte, error) {
	return nil, fmt.Errorf("FetchRepoData not implemented in discovery service")
}

// GetTags fetches the tags for a repository.
func (c *GitHubClient) GetTags(repoFullName string) ([]string, error) {
	bodyBytes, err := c.get(fmt.Sprintf("%s/repos/%s/tags", c.baseURL, repoFullName))
	if err != nil {
		return nil, err
	}

	var tags []struct {
		Name string `json:"name"`
	}
	if err := json.Unmarshal(bodyBytes, &tags); err != nil {
		return nil, fmt.Errorf("failed to unmarshal response for %s: %w", repoFullName, err)
	}

	var tagNames []string
	for _, tag := range tags {
		tagNames = append(tagNames, tag.Name)
	}
	return tagNames, nil
}

// GetLanguages fetches the languages for a repository.
func (c *GitHubClient) GetLanguages(repoFullName string) (map[string]int, error) {
	bodyBytes, err := c.get(fmt.Sprintf("%s/repos/%s/languages", c.baseURL, repoFullName))
	if err != nil {
		return nil, err
	}

	var languages map[string]int
	if err := json.Unmarshal(bodyBytes, &languages); err != nil {
		return nil, fmt.Errorf("failed to unmarshal response for %s: %w", repoFullName, err)
	}
	return languages, nil
}

// GetReadme fetches the README for a repository.
func (c *GitHubClient) GetReadme(repoFullName string) (string, error) {
	bodyBytes, err := c.get(fmt.Sprintf("%s/repos/%s/readme", c.baseURL, repoFullName))
	if err != nil {
		return "", err
	}

	var readme struct {
		DownloadURL string `json:"download_url"`
	}
	if err := json.Unmarshal(bodyBytes, &readme); err != nil {
		return "", fmt.Errorf("failed to unmarshal response for %s: %w", repoFullName, err)
	}
	return readme.DownloadURL, nil
}
