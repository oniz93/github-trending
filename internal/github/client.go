package github

import (
	"encoding/json"
	"fmt"
	"io/ioutil"
	"log"
	"net/http"
	"time"
)

import models "github.com/teomiscia/github-trending/internal/models"

const (
	githubAPIURL = "https://api.github.com"
	// You might want to make this configurable or more dynamic
	defaultSearchQuery = "stars:>50"
)

// GitHubClient provides methods for interacting with the GitHub API.
type GitHubClient struct {
	httpClient *http.Client
	token      string
}

// NewGitHubClient creates a new GitHubClient.
func NewGitHubClient(token string) *GitHubClient {
	return &GitHubClient{
		httpClient: &http.Client{Timeout: 10 * time.Second},
		token:      token,
	}
}

// SearchRepositoriesResponse represents the structure of the GitHub API search response.
type SearchRepositoriesResponse struct {
	TotalCount        int                 `json:"total_count"`
	IncompleteResults bool                `json:"incomplete_results"`
	Items             []models.Repository `json:"items"`
}

// SearchRepositories searches for repositories on GitHub based on a query.
func (c *GitHubClient) SearchRepositories(query string, page int) (*SearchRepositoriesResponse, error) {
	backoffTime := 5 * time.Second
	for {
		req, err := http.NewRequest("GET", fmt.Sprintf("%s/search/repositories?q=%s&page=%d&per_page=100", githubAPIURL, query, page), nil)
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

		if resp.StatusCode == http.StatusForbidden {
			log.Printf("Rate limit hit. Waiting for %v before retrying...", backoffTime)
			time.Sleep(backoffTime)
			backoffTime *= 2 // Double the backoff time for the next potential failure
			resp.Body.Close()
			continue
		}

		if resp.StatusCode != http.StatusOK {
			bodyBytes, _ := ioutil.ReadAll(resp.Body)
			resp.Body.Close()
			return nil, fmt.Errorf("GitHub API returned non-200 status: %d - %s", resp.StatusCode, string(bodyBytes))
		}

		backoffTime = 5 * time.Second // Reset backoff time on success

		bodyBytes, err := ioutil.ReadAll(resp.Body)
		if err != nil {
			resp.Body.Close()
			return nil, fmt.Errorf("failed to read response body: %w", err)
		}
		resp.Body.Close()

		var searchResp SearchRepositoriesResponse
		if err := json.Unmarshal(bodyBytes, &searchResp); err != nil {
			return nil, fmt.Errorf("failed to unmarshal response: %w", err)
		}

		time.Sleep(2 * time.Second) // Wait 2 seconds between successful requests
		return &searchResp, nil
	}
}

// FetchRepoData is a placeholder for fetching detailed repository data.
// This function will be implemented by the crawler service.
func FetchRepoData(repoURL string, token string) ([]byte, error) {
	return nil, fmt.Errorf("FetchRepoData not implemented in discovery service")
}

// GetTags fetches the tags for a repository.
func (c *GitHubClient) GetTags(repoFullName string) ([]string, error) {
	req, err := http.NewRequest("GET", fmt.Sprintf("%s/repos/%s/tags", githubAPIURL, repoFullName), nil)
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
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("GitHub API returned non-200 status: %d", resp.StatusCode)
	}

	bodyBytes, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("failed to read response body: %w", err)
	}

	var tags []struct {
		Name string `json:"name"`
	}
	if err := json.Unmarshal(bodyBytes, &tags); err != nil {
		return nil, fmt.Errorf("failed to unmarshal response: %w", err)
	}

	var tagNames []string
	for _, tag := range tags {
		tagNames = append(tagNames, tag.Name)
	}

	return tagNames, nil
}

// GetLanguages fetches the languages for a repository.
func (c *GitHubClient) GetLanguages(repoFullName string) (map[string]int, error) {
	req, err := http.NewRequest("GET", fmt.Sprintf("%s/repos/%s/languages", githubAPIURL, repoFullName), nil)
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
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("GitHub API returned non-200 status: %d", resp.StatusCode)
	}

	bodyBytes, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("failed to read response body: %w", err)
	}

	var languages map[string]int
	if err := json.Unmarshal(bodyBytes, &languages); err != nil {
		return nil, fmt.Errorf("failed to unmarshal response: %w", err)
	}

	return languages, nil
}
