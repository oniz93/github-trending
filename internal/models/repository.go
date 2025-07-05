package models

import "time"

// DiscoveryMessage is the message that the discovery service sends to the crawler.
// It contains the repository data and the date of the discovery.

type DiscoveryMessage struct {
	Repository Repository `json:"repository"`
	DiscoveredAt time.Time `json:"discovered_at"`
}

// CrawlResult is the message that the crawler service sends to the processor.
// It contains the repository data and the date of the discovery.

type CrawlResult struct {
	Repository Repository `json:"repository"`
	DiscoveredAt time.Time `json:"discovered_at"`
	CrawledAt    time.Time `json:"crawled_at"`
}

// ReadmeEmbedMessage is the message that triggers README embedding.
type ReadmeEmbedMessage struct {
	RepositoryID int64  `json:"repository_id"`
	MinioPath    string `json:"minio_path"`
}

type Repository struct {
	ID                int       `json:"id"`
	NodeID            string    `json:"node_id"`
	Name              string    `json:"name"`
	FullName          string    `json:"full_name"`
	Owner             Owner     `json:"owner"`
	Private           bool      `json:"private"`
	HTMLURL           string    `json:"html_url"`
	Description       string    `json:"description"`
	Fork              bool      `json:"fork"`
	URL               string    `json:"url"`
	CreatedAt         time.Time `json:"created_at"`
	UpdatedAt         time.Time `json:"updated_at"`
	PushedAt          time.Time `json:"pushed_at"`
	Homepage          string    `json:"homepage"`
	Size              int       `json:"size"`
	StargazersCount   int       `json:"stargazers_count"`
	WatchersCount     int       `json:"watchers_count"`
	Language          string    `json:"language"`
	ForksCount        int       `json:"forks_count"`
	OpenIssuesCount   int       `json:"open_issues_count"`
	DefaultBranch     string    `json:"default_branch"`
	Score             float64   `json:"score"`
	Topics            []string  `json:"topics"`
	HasIssues         bool      `json:"has_issues"`
	HasProjects       bool      `json:"has_projects"`
	HasPages          bool      `json:"has_pages"`
	HasWiki           bool      `json:"has_wiki"`
	HasDownloads      bool      `json:"has_downloads"`
	HasDiscussions    bool      `json:"has_discussions"`
	Archived          bool      `json:"archived"`
	Disabled          bool      `json:"disabled"`
	Visibility        string    `json:"visibility"`
	License           License   `json:"license"`
	AllowForking      bool      `json:"allow_forking"`
	IsTemplate        bool      `json:"is_template"`
	ReadmeURL         string    `json:"readme_url"`
	Tags              []string  `json:"tags"`
	Languages         map[string]int `json:"languages"`
	LastCrawledAt     time.Time `json:"last_crawled_at"`
}

type Owner struct {
	Login     string `json:"login"`
	ID        int    `json:"id"`
	NodeID    string `json:"node_id"`
	AvatarURL string `json:"avatar_url"`
	HTMLURL   string `json:"html_url"`
	Type      string `json:"type"`
	SiteAdmin bool   `json:"site_admin"`
}

type License struct {
	Key    string `json:"key"`
	Name   string `json:"name"`
	SpdxID string `json:"spdx_id"`
	URL    string `json:"url"`
	NodeID string `json:"node_id"`
}