package database

import (
	"database/sql"
	"fmt"
	"net/url"
	"strconv"
	"time"

	"github.com/ClickHouse/clickhouse-go"
	"github.com/mitchellh/hashstructure"
	"github.com/teomiscia/github-trending/internal/models"
)

// CHConnection defines the interface for ClickHouse connection operations.
type CHConnection interface {
	InsertRepositoryStats(repo models.Repository) error
	Close() error
}

// ClickHouseConnection holds the database connection.
type ClickHouseConnection struct {
	DB *sql.DB
}

// NewClickHouseConnection creates a new connection to the ClickHouse database.
func NewClickHouseConnection(host, port, user, password, database string) (*ClickHouseConnection, error) {
	encodedPassword := url.QueryEscape(password)
	connect, err := sql.Open("clickhouse", fmt.Sprintf("tcp://%s:%s?username=%s&password=%s&database=%s", host, port, user, encodedPassword, database))
	if err != nil {
		return nil, err
	}

	if err := connect.Ping(); err != nil {
		if exception, ok := err.(*clickhouse.Exception); ok {
			fmt.Printf("[%d] %s\n%s\n", exception.Code, exception.Message, exception.StackTrace)
		}
		return nil, err
	}

	return &ClickHouseConnection{DB: connect}, nil
}

// Close closes the database connection.
func (ch *ClickHouseConnection) Close() error {
	return ch.DB.Close()
}

// InsertRepositoryStats inserts repository statistics into the database.
func (ch *ClickHouseConnection) InsertRepositoryStats(repo models.Repository) error {
	newHash, err := ch.calculateRowHash(repo)
	if err != nil {
		return fmt.Errorf("failed to calculate row hash: %w", err)
	}
	newHashStr := strconv.FormatUint(newHash, 10)

	latestHash, err := ch.GetLatestRowHash(repo.ID)
	if err != nil && err != sql.ErrNoRows {
		return fmt.Errorf("failed to get latest row hash: %w", err)
	}

	if newHashStr == latestHash {
		return nil // Skip insert, no changes
	}

	tx, err := ch.DB.Begin()
	if err != nil {
		return err
	}

	stmt, err := tx.Prepare("INSERT INTO repository_stats (event_date, event_time, repository_id, stargazers_count, watchers_count, forks_count, open_issues_count, pushed_at, score, row_hash) VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?)")
	if err != nil {
		return err
	}

	_, err = stmt.Exec(
		time.Now(),
		time.Now(),
		repo.ID,
		repo.StargazersCount,
		repo.WatchersCount,
		repo.ForksCount,
		repo.OpenIssuesCount,
		repo.PushedAt,
		repo.Score,
		newHashStr,
	)
	if err != nil {
		return err
	}

	return tx.Commit()
}

func (ch *ClickHouseConnection) calculateRowHash(repo models.Repository) (uint64, error) {
	// The fields used here should match the ones in the SQL migration script's sipHash64 function
	hash, err := hashstructure.Hash(struct {
		RepoID          int
		StargazersCount int
		WatchersCount   int
		ForksCount      int
		OpenIssuesCount int
	}{
		RepoID:          repo.ID,
		StargazersCount: repo.StargazersCount,
		WatchersCount:   repo.WatchersCount,
		ForksCount:      repo.ForksCount,
		OpenIssuesCount: repo.OpenIssuesCount,
	}, nil)
	if err != nil {
		return 0, err
	}
	return hash,
		nil
}

func (ch *ClickHouseConnection) GetLatestRowHash(repoID int) (string, error) {
	var hash string
	query := "SELECT row_hash FROM repository_stats WHERE repository_id = ? ORDER BY event_time DESC LIMIT 1"
	err := ch.DB.QueryRow(query, repoID).Scan(&hash)
	if err != nil {
		return "", err
	}
	return hash, nil
}

// GetLastUpdateForRepository retrieves the last update timestamp for a given repository from ClickHouse.
func (ch *ClickHouseConnection) GetLastUpdateForRepository(repoID int64) (time.Time, error) {
	var lastUpdate time.Time
	query := "SELECT event_time FROM repository_stats WHERE repository_id = ? ORDER BY event_time DESC LIMIT 1"
	err := ch.DB.QueryRow(query, repoID).Scan(&lastUpdate)
	if err != nil {
		if err == sql.ErrNoRows {
			return time.Time{}, nil // No data for this repository
		}
		return time.Time{}, err
	}
	return lastUpdate, nil
}

// GetRepositoryStats retrieves all statistics for a given repository from ClickHouse.
func (ch *ClickHouseConnection) GetRepositoryStats(repoID int64) ([]models.RepositoryStat, error) {
	rows, err := ch.DB.Query("SELECT event_date, event_time, stargazers_count, watchers_count, forks_count, open_issues_count, pushed_at, score FROM repository_stats WHERE repository_id = ? ORDER BY event_time DESC", repoID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var stats []models.RepositoryStat
	for rows.Next() {
		var stat models.RepositoryStat
		if err := rows.Scan(&stat.EventDate, &stat.EventTime, &stat.StargazersCount, &stat.WatchersCount, &stat.ForksCount, &stat.OpenIssuesCount, &stat.PushedAt, &stat.Score); err != nil {
			return nil, err
		}
		stats = append(stats, stat)
	}

	return stats, nil
}

// GetLatestRepositoryStat retrieves the latest statistic for a given repository from ClickHouse.
func (ch *ClickHouseConnection) GetLatestRepositoryStat(repoID int64) (*models.RepositoryStat, error) {
	row := ch.DB.QueryRow("SELECT event_date, event_time, stargazers_count, watchers_count, forks_count, open_issues_count, pushed_at, score FROM repository_stats WHERE repository_id = ? ORDER BY event_time DESC LIMIT 1", repoID)

	var stat models.RepositoryStat
	if err := row.Scan(&stat.EventDate, &stat.EventTime, &stat.StargazersCount, &stat.WatchersCount, &stat.ForksCount, &stat.OpenIssuesCount, &stat.PushedAt, &stat.Score); err != nil {
		if err == sql.ErrNoRows {
			return nil, nil // No stats found for this repository
		}
		return nil, err
	}

	return &stat, nil
}

// GetRepositoryIDsToUpdate retrieves the repository IDs from ClickHouse that have been pushed to recently.
func (ch *ClickHouseConnection) GetRepositoryIDsToUpdate(since time.Time) ([]int64, error) {
	query := `
		SELECT repository_id
		FROM repository_stats
		WHERE (repository_id, event_time) IN (
			SELECT repository_id, max(event_time)
			FROM repository_stats
			GROUP BY repository_id
		) AND pushed_at > ?
	`
	rows, err := ch.DB.Query(query, since)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var repoIDs []int64
	for rows.Next() {
		var repoID int64
		if err := rows.Scan(&repoID); err != nil {
			return nil, err
		}
		repoIDs = append(repoIDs, repoID)
	}

	return repoIDs, nil
}

// GetTrendingRepositoryIDsByGrowth retrieves repository IDs based on star and fork growth over a period.
func (ch *ClickHouseConnection) GetTrendingRepositoryIDsByGrowth(days int) ([]int64, error) {
	startTime := time.Now().AddDate(0, 0, -days)

	query := `
		WITH
			repo_latest_stats AS (
				SELECT
					repository_id,
					argMax(stargazers_count, event_time) AS latest_stars,
					argMax(forks_count, event_time) AS latest_forks
				FROM repository_stats
				GROUP BY repository_id
			),
			repo_past_stats AS (
				SELECT
					repository_id,
					argMax(stargazers_count, event_time) AS past_stars,
					argMax(forks_count, event_time) AS past_forks
				FROM repository_stats
				WHERE event_time <= ?
				GROUP BY repository_id
			)
		SELECT
			l.repository_id
		FROM repo_latest_stats AS l
		JOIN repo_past_stats AS p ON l.repository_id = p.repository_id
		WHERE (l.latest_stars > p.past_stars OR l.latest_forks > p.past_forks)
		ORDER BY (l.latest_stars - p.past_stars) + (l.latest_forks - p.past_forks) DESC
		LIMIT 200
	`
	// log.Printf("Executing ClickHouse trending query with start time: %v\nQuery: %s", startTime, query)

	rows, err := ch.DB.Query(query, startTime)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var ids []int64
	for rows.Next() {
		var id int64
		if err := rows.Scan(&id); err != nil {
			return nil, err
		}
		ids = append(ids, id)
	}

	// log.Printf("ClickHouse trending query returned %d repository IDs.", len(ids))
	return ids, nil
}
