package database

import (
	"context"
	"database/sql"
	"fmt"
	"log"
	"time"

	"github.com/ClickHouse/clickhouse-go/v2"
	"github.com/teomiscia/github-trending/internal/models"
)

// ClickHouseConnection holds the database connection.
type ClickHouseConnection struct {
	DB *sql.DB
}

// NewClickHouseConnection creates a new connection to the ClickHouse database.
func NewClickHouseConnection(host string, port int, user, password, database string) (*ClickHouseConnection, error) {
	db := clickhouse.OpenDB(&clickhouse.Options{
		Addr: []string{fmt.Sprintf("%s:%d", host, port)},
		Auth: clickhouse.Auth{
			Database: database,
			Username: user,
			Password: password,
		},
	})

	if err := db.PingContext(context.Background()); err != nil {
		return nil, err
	}

	return &ClickHouseConnection{DB: db}, nil
}

// InsertRepositoryStats inserts repository statistics into ClickHouse.
func (ch *ClickHouseConnection) InsertRepositoryStats(repo models.Repository, discoveredAt time.Time) error {
	tx, err := ch.DB.Begin()
	if err != nil {
		return err
	}

	stmt, err := tx.Prepare(`
		INSERT INTO repository_stats (
			event_date, event_time, repository_id, stargazers_count, watchers_count, forks_count, open_issues_count, pushed_at, score
		) VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?)
	`)
	if err != nil {
		return err
	}
	defer stmt.Close()

	_, err = stmt.Exec(
		discoveredAt.Format("2006-01-02"),
		discoveredAt,
		repo.ID,
		repo.StargazersCount,
		repo.WatchersCount,
		repo.ForksCount,
		repo.OpenIssuesCount,
		repo.PushedAt,
		repo.Score,
	)
	if err != nil {
		log.Printf("Failed to insert repository stats: %v", err)
		return tx.Rollback()
	}

	return tx.Commit()
}
