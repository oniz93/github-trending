package database

import (
	"database/sql"
	"fmt"
	"time"

	"github.com/ClickHouse/clickhouse-go"
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
	connect, err := sql.Open("clickhouse", fmt.Sprintf("tcp://%s:%s?username=%s&password=%s&database=%s", host, port, user, password, database))
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
	tx, err := ch.DB.Begin()
	if err != nil {
		return err
	}

	stmt, err := tx.Prepare("INSERT INTO repository_stats (event_date, event_time, repository_id, stargazers_count, watchers_count, forks_count, open_issues_count, pushed_at, score) VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?)")
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
	)
	if err != nil {
		return err
	}

	return tx.Commit()
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
