package database

import (
	"database/sql"
	"fmt"
	"log"
	"strings"
	"time"

	_ "github.com/lib/pq"
	"github.com/teomiscia/github-trending/internal/models"
)

// DBConnection defines the interface for PostgreSQL database operations.
type DBConnection interface {
	GetLastCrawlTime(repoID int64) (time.Time, error)
	InsertRepository(repo models.Repository, lastCrawledAt time.Time) error
	GetRepositoryByID(repoID int64) (models.Repository, error)
	Close() error
}

// PostgresConnection holds the database connection.
type PostgresConnection struct {
	DB *sql.DB
}

// NewPostgresConnection creates a new connection to the PostgreSQL database.
func NewPostgresConnection(host, port, user, password, dbname string) (*PostgresConnection, error) {
	connStr := fmt.Sprintf("host=%s port=%s user=%s password=%s dbname=%s sslmode=disable", host, port, user, password, dbname)
	db, err := sql.Open("postgres", connStr)
	if err != nil {
		return nil, err
	}

	if err = db.Ping(); err != nil {
		return nil, err
	}

	return &PostgresConnection{DB: db}, nil
}

// Close closes the database connection.
func (pc *PostgresConnection) Close() error {
	return pc.DB.Close()
}

// TrackRepositoryView inserts a record of a repository view into the database.
func (pc *PostgresConnection) TrackRepositoryView(sessionID string, repositoryID int64) error {
	_, err := pc.DB.Exec("INSERT INTO repository_views (session_id, repository_id) VALUES ($1, $2)", sessionID, repositoryID)
	return err
}

// GetTrendingRepositoryIDs retrieves a list of trending repository IDs, filtered by languages and tags.
func (pc *PostgresConnection) GetTrendingRepositoryIDs(languages []string, tags []string) ([]int64, error) {
	query := "SELECT id FROM repositories"
	args := []interface{}{}
	whereClauses := []string{}

	if len(languages) > 0 {
		whereClauses = append(whereClauses, "id IN (SELECT repository_id FROM repository_languages WHERE language_id IN (SELECT id FROM languages WHERE name = ANY($1)))")
		args = append(args, languages)
	}

	if len(tags) > 0 {
		whereClauses = append(whereClauses, "id IN (SELECT repository_id FROM repository_tags WHERE tag_id IN (SELECT id FROM tags WHERE name = ANY($2)))")
		args = append(args, tags)
	}

	if len(whereClauses) > 0 {
		query += " WHERE " + strings.Join(whereClauses, " AND ")
	}

	query += " ORDER BY last_crawled_at DESC LIMIT 100"

	rows, err := pc.DB.Query(query, args...)
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

	return ids, nil
}

// GetLastCrawlTime retrieves the last time a repository was crawled.
func (pc *PostgresConnection) GetLastCrawlTime(repoID int64) (time.Time, error) {
	var lastCrawledAt time.Time
	err := pc.DB.QueryRow("SELECT last_crawled_at FROM repositories WHERE id = $1", repoID).Scan(&lastCrawledAt)
	if err != nil {
		if err == sql.ErrNoRows {
			return time.Time{}, nil // Not yet crawled
		}
		return time.Time{}, err
	}
	return lastCrawledAt, nil
}

// InsertRepository inserts a repository into the database.
func (pc *PostgresConnection) InsertRepository(repo models.Repository, lastCrawledAt time.Time) error {
	tx, err := pc.DB.Begin()
	if err != nil {
		return err
	}

	// Insert owner
	_, err = tx.Exec(`
		INSERT INTO owners (id, login, avatar_url, html_url, type)
		VALUES ($1, $2, $3, $4, $5)
		ON CONFLICT (id) DO NOTHING
	`, repo.Owner.ID, repo.Owner.Login, repo.Owner.AvatarURL, repo.Owner.HTMLURL, repo.Owner.Type)
	if err != nil {
		log.Printf("Failed to insert owner: %v", err)
		tx.Rollback()
		return err
	}

	// Insert license
	if repo.License.Key.Valid {
		_, err = tx.Exec(`
			INSERT INTO licenses (key, name, spdx_id, url, node_id)
			VALUES ($1, $2, $3, $4, $5)
			ON CONFLICT (key) DO UPDATE SET
				name = $2, spdx_id = $3, url = $4, node_id = $5
		`, repo.License.Key, repo.License.Name, repo.License.SpdxID, repo.License.URL, repo.License.NodeID)
		if err != nil {
			log.Printf("Failed to insert license: %v", err)
			tx.Rollback()
			return err
		}
	}

	// Insert repository
	_, err = tx.Exec(`
		INSERT INTO repositories (id, node_id, name, full_name, owner_id, description, html_url, homepage, default_branch, license_key, readme_url, created_at, is_fork, is_template, is_archived, is_disabled, last_crawled_at)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12, $13, $14, $15, $16, $17)
		ON CONFLICT (id) DO UPDATE SET
			node_id = $2, name = $3, full_name = $4, owner_id = $5, description = $6, html_url = $7, homepage = $8, default_branch = $9, license_key = $10, readme_url = $11, created_at = $12, is_fork = $13, is_template = $14, is_archived = $15, is_disabled = $16, last_crawled_at = $17
	`, repo.ID, repo.NodeID, repo.Name, repo.FullName, repo.Owner.ID, repo.Description, repo.HTMLURL, repo.Homepage, repo.DefaultBranch, func() interface{} {
		if !repo.License.Key.Valid {
			return nil
		}
		return repo.License.Key
	}(), repo.ReadmeURL, repo.CreatedAt, repo.Fork, repo.IsTemplate, repo.Archived, repo.Disabled, lastCrawledAt)
	if err != nil {
		log.Printf("Failed to insert repository: %v", err)
		tx.Rollback()
		return err
	}

	// Insert tags
	for _, tagName := range repo.Tags {
		var tagID int
		err = tx.QueryRow("INSERT INTO tags (name) VALUES ($1) ON CONFLICT (name) DO UPDATE SET name = $1 RETURNING id", tagName).Scan(&tagID)
		if err != nil {
			log.Printf("Failed to insert tag: %v", err)
			tx.Rollback()
			return err
		}

		_, err = tx.Exec("INSERT INTO repository_tags (repository_id, tag_id) VALUES ($1, $2) ON CONFLICT DO NOTHING", repo.ID, tagID)
		if err != nil {
			log.Printf("Failed to insert repository_tag: %v", err)
			tx.Rollback()
			return err
		}
	}

	// Insert languages
	for langName, size := range repo.Languages {
		var langID int
		err = tx.QueryRow("INSERT INTO languages (name) VALUES ($1) ON CONFLICT (name) DO UPDATE SET name = $1 RETURNING id", langName).Scan(&langID)
		if err != nil {
			log.Printf("Failed to insert language: %v", err)
			tx.Rollback()
			return err
		}

		_, err = tx.Exec("INSERT INTO repository_languages (repository_id, language_id, size) VALUES ($1, $2, $3) ON CONFLICT DO NOTHING", repo.ID, langID, size)
		if err != nil {
			log.Printf("Failed to insert repository_language: %v", err)
			tx.Rollback()
			return err
		}
	}

	return tx.Commit()
}

// GetRepositoryByID retrieves a repository by its ID.
func (pc *PostgresConnection) GetRepositoryByID(repoID int64) (models.Repository, error) {
	var repo models.Repository
	var ownerID int
	var licenseKey sql.NullString
	var repoNodeID sql.NullString
	var ownerNodeID sql.NullString
	var siteAdmin sql.NullBool
	var licenseName sql.NullString

	row := pc.DB.QueryRow(`
		SELECT
			r.id, r.node_id, r.name, r.full_name, r.owner_id, r.description, r.html_url, r.homepage, r.default_branch, r.license_key, r.readme_url, r.created_at, r.is_fork, r.is_template, r.is_archived, r.is_disabled, r.last_crawled_at,
			o.login, o.node_id, o.avatar_url, o.html_url, o.type, o.site_admin,
			l.name, l.spdx_id, l.url, l.node_id
		FROM
			repositories r
		JOIN
			owners o ON r.owner_id = o.id
		LEFT JOIN
			licenses l ON r.license_key = l.key
		WHERE
			r.id = $1
	`, repoID)

	err := row.Scan(
		&repo.ID, &repoNodeID, &repo.Name, &repo.FullName, &ownerID, &repo.Description, &repo.HTMLURL, &repo.Homepage, &repo.DefaultBranch, &licenseKey, &repo.ReadmeURL, &repo.CreatedAt, &repo.Fork, &repo.IsTemplate, &repo.Archived, &repo.Disabled, &repo.LastCrawledAt,
		&repo.Owner.Login, &ownerNodeID, &repo.Owner.AvatarURL, &repo.Owner.HTMLURL, &repo.Owner.Type, &siteAdmin,
		&licenseName, &repo.License.SpdxID, &repo.License.URL, &repo.License.NodeID,
	)

	if err != nil {
		return models.Repository{}, fmt.Errorf("failed to query repository: %w", err)
	}

	if repoNodeID.Valid {
		repo.NodeID = repoNodeID
	}

	if ownerNodeID.Valid {
		repo.Owner.NodeID = ownerNodeID
	}

	if licenseKey.Valid {
		repo.License.Key = licenseKey
	}

	if licenseName.Valid {
		repo.License.Name = licenseName
	}

	if siteAdmin.Valid {
		repo.Owner.SiteAdmin = siteAdmin
	}

	// Fetch tags
	tags, err := pc.getTagsForRepository(repoID)
	if err != nil {
		return models.Repository{}, fmt.Errorf("failed to get tags for repository: %w", err)
	}
	repo.Tags = tags

	// Fetch languages
	languages, err := pc.getLanguagesForRepository(repoID)
	if err != nil {
		return models.Repository{}, fmt.Errorf("failed to get languages for repository: %w", err)
	}
	repo.Languages = languages

	return repo, nil
}

func (pc *PostgresConnection) getTagsForRepository(repoID int64) ([]string, error) {
	rows, err := pc.DB.Query(`
		SELECT
			t.name
		FROM
			tags t
		JOIN
			repository_tags rt ON t.id = rt.tag_id
		WHERE
			rt.repository_id = $1
	`, repoID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var tags []string
	for rows.Next() {
		var tag string
		if err := rows.Scan(&tag); err != nil {
			return nil, err
		}
		tags = append(tags, tag)
	}
	return tags, nil
}

func (pc *PostgresConnection) getLanguagesForRepository(repoID int64) (map[string]int, error) {
	rows, err := pc.DB.Query(`
		SELECT
			l.name, rl.size
		FROM
			languages l
		JOIN
			repository_languages rl ON l.id = rl.language_id
		WHERE
			rl.repository_id = $1
	`, repoID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	languages := make(map[string]int)
	for rows.Next() {
		var name string
		var size int
		if err := rows.Scan(&name, &size); err != nil {
			return nil, err
		}
		languages[name] = size
	}
	return languages, nil
}

// GetRepositoryIDsUpdatedSince retrieves a list of repository IDs that have been updated since a given time.
func (pc *PostgresConnection) GetRepositoryIDsUpdatedSince(since time.Time) ([]int64, error) {
	rows, err := pc.DB.Query("SELECT id FROM repositories WHERE last_crawled_at >= $1", since)
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

	return ids, nil
}

// GetRecentClickedRepositoryIDs retrieves a list of recently clicked repository IDs for a given session.
func (pc *PostgresConnection) GetRecentClickedRepositoryIDs(sessionID string, limit int) ([]int64, error) {
	rows, err := pc.DB.Query("SELECT repository_id FROM repository_views WHERE session_id = $1 ORDER BY viewed_at DESC LIMIT $2", sessionID, limit)
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

	return ids, nil
}

// GetRepositoriesByIDs retrieves a list of repositories by their IDs.
func (pc *PostgresConnection) GetRepositoriesByIDs(repoIDs []int64) ([]models.Repository, error) {
	if len(repoIDs) == 0 {
		return []models.Repository{}, nil
	}

	// Create a string of placeholders for the IN clause
	placeholders := make([]string, len(repoIDs))
	args := make([]interface{}, len(repoIDs))
	for i, id := range repoIDs {
		placeholders[i] = fmt.Sprintf("$%d", i+1)
		args[i] = id
	}
	placeholderStr := strings.Join(placeholders, ",")

	query := fmt.Sprintf(`
		SELECT
			r.id, r.node_id, r.name, r.full_name, r.owner_id, r.description, r.html_url, r.homepage, r.default_branch, r.license_key, r.readme_url, r.created_at, r.is_fork, r.is_template, r.is_archived, r.is_disabled, r.last_crawled_at,
			o.login, o.node_id, o.avatar_url, o.html_url, o.type, o.site_admin,
			l.name, l.spdx_id, l.url, l.node_id
		FROM
			repositories r
		JOIN
			owners o ON r.owner_id = o.id
		LEFT JOIN
			licenses l ON r.license_key = l.key
		WHERE
			r.id IN (%s)
	`, placeholderStr)

	rows, err := pc.DB.Query(query, args...)
	if err != nil {
		return nil, fmt.Errorf("failed to query repositories by IDs: %w", err)
	}
	defer rows.Close()

	var repositories []models.Repository
	for rows.Next() {
		var repo models.Repository
		var ownerID int
		var licenseKey sql.NullString
		var repoNodeID sql.NullString
		var ownerNodeID sql.NullString
		var siteAdmin sql.NullBool
		var licenseName sql.NullString

		err := rows.Scan(
			&repo.ID, &repoNodeID, &repo.Name, &repo.FullName, &ownerID, &repo.Description, &repo.HTMLURL, &repo.Homepage, &repo.DefaultBranch, &licenseKey, &repo.ReadmeURL, &repo.CreatedAt, &repo.Fork, &repo.IsTemplate, &repo.Archived, &repo.Disabled, &repo.LastCrawledAt,
			&repo.Owner.Login, &ownerNodeID, &repo.Owner.AvatarURL, &repo.Owner.HTMLURL, &repo.Owner.Type, &siteAdmin,
			&licenseName, &repo.License.SpdxID, &repo.License.URL, &repo.License.NodeID,
		)
		if err != nil {
			return nil, fmt.Errorf("failed to scan repository row: %w", err)
		}

		if repoNodeID.Valid {
			repo.NodeID = repoNodeID
		}

		if ownerNodeID.Valid {
			repo.Owner.NodeID = ownerNodeID
		}

		if licenseKey.Valid {
			repo.License.Key = licenseKey
		}

		if licenseName.Valid {
			repo.License.Name = licenseName
		}

		if siteAdmin.Valid {
			repo.Owner.SiteAdmin = siteAdmin
		}

		// Fetch tags
		tags, err := pc.getTagsForRepository(int64(repo.ID))
		if err != nil {
			return nil, fmt.Errorf("failed to get tags for repository %d: %w", repo.ID, err)
		}
		repo.Tags = tags

		// Fetch languages
		languages, err := pc.getLanguagesForRepository(int64(repo.ID))
		if err != nil {
			return nil, fmt.Errorf("failed to get languages for repository %d: %w", repo.ID, err)
		}
		repo.Languages = languages

		repositories = append(repositories, repo)
	}

	return repositories, nil
}
