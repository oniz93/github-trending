package database

import (
	"context"
	"crypto/sha256"
	"database/sql"
	"encoding/json"
	"fmt"
	"log"
	"strings"
	"time"

	"github.com/golang/snappy"
	_ "github.com/lib/pq"
	"github.com/redis/go-redis/v9"
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
	DB          *sql.DB
	RedisClient *redis.Client
}

// WithRedis sets the Redis client for the PostgresConnection.
func (pc *PostgresConnection) WithRedis(client *redis.Client) *PostgresConnection {
	pc.RedisClient = client
	return pc
}

func compress(data []byte) []byte {
	return snappy.Encode(nil, data)
}

func decompress(data []byte) ([]byte, error) {
	return snappy.Decode(nil, data)
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

// GetTrendingRepositoryIDs retrieves a list of trending repository IDs, filtered by languages, tags and seen repositories.
func (pc *PostgresConnection) GetTrendingRepositoryIDs(languages, tags, topics, seenRepoIDs []string) ([]int64, error) {
	if pc.RedisClient != nil {
		var keyBuilder strings.Builder
		keyBuilder.WriteString("trending_repos:")
		keyBuilder.WriteString(strings.Join(languages, ","))
		keyBuilder.WriteString(":")
		keyBuilder.WriteString(strings.Join(tags, ","))
		keyBuilder.WriteString(":")
		keyBuilder.WriteString(strings.Join(topics, ","))
		keyBuilder.WriteString(":")
		keyBuilder.WriteString(strings.Join(seenRepoIDs, ","))

		hash := sha256.Sum256([]byte(keyBuilder.String()))
		key := fmt.Sprintf("trending:%x", hash)

		val, err := pc.RedisClient.Get(context.Background(), key).Result()
		if err == nil {
			decompressed, err := decompress([]byte(val))
			if err == nil {
				var ids []int64
				if err := json.Unmarshal(decompressed, &ids); err == nil {
					return ids, nil
				}
			}
		}

		ids, err := pc.getTrendingRepositoryIDsFromDB(languages, tags, topics, seenRepoIDs)
		if err != nil {
			return nil, err
		}

		jsonBytes, err := json.Marshal(ids)
		if err == nil {
			compressed := compress(jsonBytes)
			pc.RedisClient.Set(context.Background(), key, compressed, 10*time.Minute)
		}

		return ids, nil
	}

	return pc.getTrendingRepositoryIDsFromDB(languages, tags, topics, seenRepoIDs)
}

func (pc *PostgresConnection) getTrendingRepositoryIDsFromDB(languages, tags, topics, seenRepoIDs []string) ([]int64, error) {
	query := "SELECT id FROM repositories"
	args := []interface{}{}
	whereClauses := []string{}

	if len(languages) > 0 {
		whereClauses = append(whereClauses, fmt.Sprintf("id IN (SELECT repository_id FROM repository_languages WHERE language_id IN (SELECT id FROM languages WHERE name = ANY($%d)))", len(args)+1))
		args = append(args, languages)
	}

	if len(tags) > 0 {
		whereClauses = append(whereClauses, fmt.Sprintf("id IN (SELECT repository_id FROM repository_tags WHERE tag_id IN (SELECT id FROM tags WHERE name = ANY($%d)))", len(args)+1))
		args = append(args, tags)
	}

	if len(topics) > 0 {
		whereClauses = append(whereClauses, fmt.Sprintf("id IN (SELECT repository_id FROM repository_topics WHERE topic_id IN (SELECT id FROM topics WHERE name = ANY($%d)))", len(args)+1))
		args = append(args, topics)
	}

	if len(seenRepoIDs) > 0 {
		whereClauses = append(whereClauses, fmt.Sprintf("id NOT IN ($%d)", len(args)+1))
		args = append(args, seenRepoIDs)
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
	if pc.RedisClient != nil {
		key := fmt.Sprintf("last_crawl_time:%d", repoID)
		val, err := pc.RedisClient.Get(context.Background(), key).Result()
		if err == nil {
			decompressed, err := decompress([]byte(val))
			if err == nil {
				t, err := time.Parse(time.RFC3339Nano, string(decompressed))
				if err == nil {
					return t, nil
				}
			}
		}
	}

	var lastCrawledAt time.Time
	err := pc.DB.QueryRow("SELECT last_crawled_at FROM repositories WHERE id = $1", repoID).Scan(&lastCrawledAt)
	if err != nil {
		if err == sql.ErrNoRows {
			return time.Time{}, nil // Not yet crawled
		}
		return time.Time{}, err
	}

	if pc.RedisClient != nil {
		key := fmt.Sprintf("last_crawl_time:%d", repoID)
		compressed := compress([]byte(lastCrawledAt.Format(time.RFC3339Nano)))
		pc.RedisClient.Set(context.Background(), key, compressed, 1*time.Hour)
	}

	return lastCrawledAt, nil
}

// InsertRepository inserts or updates a repository and its related data in a single transaction.
func (pc *PostgresConnection) InsertRepository(repo models.Repository, lastCrawledAt time.Time) error {
	tx, err := pc.DB.Begin()
	if err != nil {
		return err
	}
	defer tx.Rollback() // Rollback is a no-op if the transaction is committed.

	// Insert owner if not present.
	_, err = tx.Exec(`
		INSERT INTO owners (id, login, avatar_url, html_url, type)
		VALUES ($1, $2, $3, $4, $5)
		ON CONFLICT (id) DO NOTHING
	`, repo.Owner.ID, repo.Owner.Login, repo.Owner.AvatarURL, repo.Owner.HTMLURL, repo.Owner.Type)
	if err != nil {
		log.Printf("Failed to insert owner: %v", err)
		return err
	}

	// Insert or update license, only updating if data has changed.
	if repo.License.Key.Valid {
		_, err = tx.Exec(`
			INSERT INTO licenses (key, name, spdx_id, url, node_id)
			VALUES ($1, $2, $3, $4, $5)
			ON CONFLICT (key) DO UPDATE SET
				name = EXCLUDED.name,
				spdx_id = EXCLUDED.spdx_id,
				url = EXCLUDED.url,
				node_id = EXCLUDED.node_id
			WHERE licenses.name IS DISTINCT FROM EXCLUDED.name
			   OR licenses.spdx_id IS DISTINCT FROM EXCLUDED.spdx_id
			   OR licenses.url IS DISTINCT FROM EXCLUDED.url
		`, repo.License.Key, repo.License.Name, repo.License.SpdxID, repo.License.URL, repo.License.NodeID)
		if err != nil {
			log.Printf("Failed to insert license: %v", err)
			return err
		}
	}

	// Insert or update repository.
	// OPTIMIZATION: The WHERE clause prevents "empty" updates, reducing write load and lock contention.
	_, err = tx.Exec(`
		INSERT INTO repositories (id, node_id, name, full_name, owner_id, description, html_url, homepage, default_branch, license_key, readme_url, created_at, is_fork, is_template, is_archived, is_disabled, last_crawled_at)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12, $13, $14, $15, $16, $17)
		ON CONFLICT (id) DO UPDATE SET
			node_id = EXCLUDED.node_id, 
			name = EXCLUDED.name, 
			full_name = EXCLUDED.full_name, 
			owner_id = EXCLUDED.owner_id, 
			description = EXCLUDED.description, 
			html_url = EXCLUDED.html_url, 
			homepage = EXCLUDED.homepage, 
			default_branch = EXCLUDED.default_branch, 
			license_key = EXCLUDED.license_key, 
			readme_url = EXCLUDED.readme_url, 
			is_fork = EXCLUDED.is_fork, 
			is_template = EXCLUDED.is_template, 
			is_archived = EXCLUDED.is_archived, 
			is_disabled = EXCLUDED.is_disabled, 
			last_crawled_at = EXCLUDED.last_crawled_at
		WHERE repositories.description IS DISTINCT FROM EXCLUDED.description
		   OR repositories.homepage IS DISTINCT FROM EXCLUDED.homepage
		   OR repositories.license_key IS DISTINCT FROM EXCLUDED.license_key
		   OR repositories.is_archived IS DISTINCT FROM EXCLUDED.is_archived
		   OR repositories.is_disabled IS DISTINCT FROM EXCLUDED.is_disabled
		   OR repositories.last_crawled_at < EXCLUDED.last_crawled_at
	`, repo.ID, repo.NodeID, repo.Name, repo.FullName, repo.Owner.ID, repo.Description, repo.HTMLURL, repo.Homepage, repo.DefaultBranch, repo.License.Key, repo.ReadmeURL, repo.CreatedAt, repo.Fork, repo.IsTemplate, repo.Archived, repo.Disabled, lastCrawledAt)
	if err != nil {
		log.Printf("Failed to insert repository: %v", err)
		return err
	}

	// OPTIMIZATION: Changed "DO NOTHING" to "DO UPDATE" to fix a bug where Scan would fail on conflict.
	// This pattern safely inserts or finds the existing ID every time.
	tagUpsertQuery := "INSERT INTO tags (name) VALUES ($1) ON CONFLICT (name) DO UPDATE SET name = EXCLUDED.name RETURNING id"
	for _, tagName := range repo.Tags {
		var tagID int
		if err = tx.QueryRow(tagUpsertQuery, tagName).Scan(&tagID); err != nil {
			log.Printf("Failed to upsert tag: %v", err)
			return err
		}
		_, err = tx.Exec("INSERT INTO repository_tags (repository_id, tag_id) VALUES ($1, $2) ON CONFLICT DO NOTHING", repo.ID, tagID)
		if err != nil {
			log.Printf("Failed to insert repository_tag: %v", err)
			return err
		}
	}

	topicUpsertQuery := "INSERT INTO topics (name) VALUES ($1) ON CONFLICT (name) DO UPDATE SET name = EXCLUDED.name RETURNING id"
	for _, topicName := range repo.Topics {
		var topicID int
		if err = tx.QueryRow(topicUpsertQuery, topicName).Scan(&topicID); err != nil {
			log.Printf("Failed to upsert topic: %v", err)
			return err
		}
		_, err = tx.Exec("INSERT INTO repository_topics (repository_id, topic_id) VALUES ($1, $2) ON CONFLICT DO NOTHING", repo.ID, topicID)
		if err != nil {
			log.Printf("Failed to insert repository_topic: %v", err)
			return err
		}
	}

	langUpsertQuery := "INSERT INTO languages (name) VALUES ($1) ON CONFLICT (name) DO UPDATE SET name = EXCLUDED.name RETURNING id"
	for langName, size := range repo.Languages {
		var langID int
		if err = tx.QueryRow(langUpsertQuery, langName).Scan(&langID); err != nil {
			log.Printf("Failed to upsert language: %v", err)
			return err
		}
		// Use a full upsert here in case language byte counts change on a re-crawl.
		_, err = tx.Exec(`
            INSERT INTO repository_languages (repository_id, language_id, size) VALUES ($1, $2, $3)
            ON CONFLICT (repository_id, language_id) DO UPDATE SET size = EXCLUDED.size
        `, repo.ID, langID, size)
		if err != nil {
			log.Printf("Failed to insert repository_language: %v", err)
			return err
		}
	}

	if err := tx.Commit(); err != nil {
		return err
	}

	if pc.RedisClient != nil {
		ctx := context.Background()
		keys := []string{
			fmt.Sprintf("repository:%d", repo.ID),
			fmt.Sprintf("last_crawl_time:%d", repo.ID),
			fmt.Sprintf("repository_data_by_id:%d", repo.ID),
			fmt.Sprintf("tags:%d", repo.ID),
			fmt.Sprintf("topics:%d", repo.ID),
			fmt.Sprintf("languages:%d", repo.ID),
		}
		pc.RedisClient.Del(ctx, keys...)
	}

	return nil
}

// GetRepositoryByID retrieves a repository by its ID.
func (pc *PostgresConnection) GetRepositoryByID(repoID int64) (models.Repository, error) {
	if pc.RedisClient != nil {
		key := fmt.Sprintf("repository:%d", repoID)
		val, err := pc.RedisClient.Get(context.Background(), key).Result()
		if err == nil {
			decompressed, err := decompress([]byte(val))
			if err == nil {
				var repo models.Repository
				if err := json.Unmarshal(decompressed, &repo); err == nil {
					return repo, nil
				}
			}
		}
	}

	repo, err := pc.getRepositoryByIDFromDB(repoID)
	if err != nil {
		return models.Repository{}, err
	}

	if pc.RedisClient != nil {
		key := fmt.Sprintf("repository:%d", repo.ID)
		jsonBytes, err := json.Marshal(repo)
		if err == nil {
			compressed := compress(jsonBytes)
			pc.RedisClient.Set(context.Background(), key, compressed, 1*time.Hour)
		}
	}

	return repo, nil
}

func (pc *PostgresConnection) getRepositoryByIDFromDB(repoID int64) (models.Repository, error) {
	var repo models.Repository
	// FIX: Scan into int64 to match 'bigint' schema type
	var ownerID int64
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

	// FIX: Cast the int64 to int for the model struct field.
	repo.Owner.ID = int(ownerID)
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

	// Fetch topics
	topics, err := pc.getTopicsForRepository(repoID)
	if err != nil {
		return models.Repository{}, fmt.Errorf("failed to get topics for repository: %w", err)
	}
	repo.Topics = topics

	// Fetch languages
	languages, err := pc.getLanguagesForRepository(repoID)
	if err != nil {
		return models.Repository{}, fmt.Errorf("failed to get languages for repository: %w", err)
	}
	repo.Languages = languages

	return repo, nil
}

func (pc *PostgresConnection) getTagsForRepository(repoID int64) ([]string, error) {
	if pc.RedisClient != nil {
		key := fmt.Sprintf("tags:%d", repoID)
		val, err := pc.RedisClient.Get(context.Background(), key).Result()
		if err == nil {
			decompressed, err := decompress([]byte(val))
			if err == nil {
				var tags []string
				if err := json.Unmarshal(decompressed, &tags); err == nil {
					return tags, nil
				}
			}
		}
	}

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

	if pc.RedisClient != nil {
		key := fmt.Sprintf("tags:%d", repoID)
		jsonBytes, err := json.Marshal(tags)
		if err == nil {
			compressed := compress(jsonBytes)
			pc.RedisClient.Set(context.Background(), key, compressed, 1*time.Hour)
		}
	}

	return tags, nil
}

func (pc *PostgresConnection) getTopicsForRepository(repoID int64) ([]string, error) {
	if pc.RedisClient != nil {
		key := fmt.Sprintf("topics:%d", repoID)
		val, err := pc.RedisClient.Get(context.Background(), key).Result()
		if err == nil {
			decompressed, err := decompress([]byte(val))
			if err == nil {
				var topics []string
				if err := json.Unmarshal(decompressed, &topics); err == nil {
					return topics, nil
				}
			}
		}
	}

	rows, err := pc.DB.Query(`
		SELECT
			t.name
		FROM
			topics t
		JOIN
			repository_topics rt ON t.id = rt.topic_id
		WHERE
			rt.repository_id = $1
	`, repoID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var topics []string
	for rows.Next() {
		var topic string
		if err := rows.Scan(&topic); err != nil {
			return nil, err
		}
		topics = append(topics, topic)
	}

	if pc.RedisClient != nil {
		key := fmt.Sprintf("topics:%d", repoID)
		jsonBytes, err := json.Marshal(topics)
		if err == nil {
			compressed := compress(jsonBytes)
			pc.RedisClient.Set(context.Background(), key, compressed, 1*time.Hour)
		}
	}

	return topics, nil
}

func (pc *PostgresConnection) getLanguagesForRepository(repoID int64) (map[string]int, error) {
	if pc.RedisClient != nil {
		key := fmt.Sprintf("languages:%d", repoID)
		val, err := pc.RedisClient.Get(context.Background(), key).Result()
		if err == nil {
			decompressed, err := decompress([]byte(val))
			if err == nil {
				var languages map[string]int
				if err := json.Unmarshal(decompressed, &languages); err == nil {
					return languages, nil
				}
			}
		}
	}

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

	if pc.RedisClient != nil {
		key := fmt.Sprintf("languages:%d", repoID)
		jsonBytes, err := json.Marshal(languages)
		if err == nil {
			compressed := compress(jsonBytes)
			pc.RedisClient.Set(context.Background(), key, compressed, 1*time.Hour)
		}
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

// GetRepositoryIDs retrieves a slice of repository IDs from the database.
func (pc *PostgresConnection) GetRepositoryIDs(limit, offset int) ([]int64, error) {
	rows, err := pc.DB.Query("SELECT id FROM repositories ORDER BY id LIMIT $1 OFFSET $2", limit, offset)
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

// UpsertRepositorySimilarity inserts or updates the similarity data for a repository.
func (pc *PostgresConnection) UpsertRepositorySimilarity(repoID int64, data []byte) error {
	// OPTIMIZATION: Added a WHERE clause to prevent updating the row if the data is identical.
	query := `
		INSERT INTO repository_similarity (id, data)
		VALUES ($1, $2)
		ON CONFLICT (id) DO UPDATE SET
			data = EXCLUDED.data
		WHERE repository_similarity.data IS DISTINCT FROM EXCLUDED.data
	`
	_, err := pc.DB.Exec(query, repoID, data)
	return err
}

// GetRepositorySimilarity retrieves the similarity data for a repository.
func (pc *PostgresConnection) GetRepositorySimilarity(repoID int64) ([]byte, error) {
	var data []byte
	query := "SELECT data FROM repository_similarity WHERE id = $1"
	err := pc.DB.QueryRow(query, repoID).Scan(&data)
	if err != nil {
		if err == sql.ErrNoRows {
			return nil, nil // No similarity data found
		}
		return nil, err
	}
	return data, nil
}

func (pc *PostgresConnection) GetRepositoriesDataByIDs(repoIDs []int64) ([]models.RepositoryData, error) {
	if len(repoIDs) == 0 {
		return []models.RepositoryData{}, nil
	}

	if pc.RedisClient != nil {
		var results []models.RepositoryData
		var missedIDs []int64
		keys := make([]string, len(repoIDs))
		for i, id := range repoIDs {
			keys[i] = fmt.Sprintf("repository_data_by_id:%d", id)
		}

		cachedResults, err := pc.RedisClient.MGet(context.Background(), keys...).Result()
		if err == nil {
			for i, res := range cachedResults {
				if res != nil {
					var repoData models.RepositoryData
					decompressed, err := decompress([]byte(res.(string)))
					if err == nil {
						if err := json.Unmarshal(decompressed, &repoData); err == nil {
							results = append(results, repoData)
							continue
						}
					}
				}
				missedIDs = append(missedIDs, repoIDs[i])
			}
		} else {
			missedIDs = repoIDs
		}

		if len(missedIDs) > 0 {
			dbResults, err := pc.getRepositoriesDataByIDsFromDB(missedIDs)
			if err != nil {
				return nil, err
			}
			for _, dbRes := range dbResults {
				results = append(results, dbRes)
				key := fmt.Sprintf("repository_data_by_id:%d", dbRes.Repository.ID)
				jsonBytes, err := json.Marshal(dbRes)
				if err == nil {
					compressed := compress(jsonBytes)
					pc.RedisClient.Set(context.Background(), key, compressed, 1*time.Hour)
				}
			}
		}
		return results, nil
	}

	return pc.getRepositoriesDataByIDsFromDB(repoIDs)
}

func (pc *PostgresConnection) getRepositoriesDataByIDsFromDB(repoIDs []int64) ([]models.RepositoryData, error) {
	if len(repoIDs) == 0 {
		return []models.RepositoryData{}, nil
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
			r.id, r.node_id, r.name, r.full_name, r.description, r.html_url, r.homepage, r.default_branch, r.license_key, r.readme_url, r.created_at, r.is_fork, r.is_template, r.is_archived, r.is_disabled, r.last_crawled_at,
			o.id, o.login, o.node_id, o.avatar_url, o.html_url, o.type,
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

	var repositoriesData []models.RepositoryData
	for rows.Next() {
		var repoData models.RepositoryData
		// FIX: Scan into int64 to match 'bigint' schema type
		var ownerID int64
		var licenseKey sql.NullString
		var repoNodeID sql.NullString
		var ownerNodeID sql.NullString
		var licenseName sql.NullString

		err := rows.Scan(
			&repoData.Repository.ID, &repoNodeID, &repoData.Repository.Name, &repoData.Repository.FullName, &repoData.Repository.Description, &repoData.Repository.HTMLURL, &repoData.Repository.Homepage, &repoData.Repository.DefaultBranch, &licenseKey, &repoData.Repository.ReadmeURL, &repoData.Repository.CreatedAt, &repoData.Repository.Fork, &repoData.Repository.IsTemplate, &repoData.Repository.Archived, &repoData.Repository.Disabled, &repoData.Repository.LastCrawledAt,
			&ownerID, &repoData.Owner.Login, &ownerNodeID, &repoData.Owner.AvatarURL, &repoData.Owner.HTMLURL, &repoData.Owner.Type,
			&licenseName, &repoData.Repository.License.SpdxID, &repoData.Repository.License.URL, &repoData.Repository.License.NodeID,
		)
		if err != nil {
			return nil, fmt.Errorf("failed to scan repository row: %w", err)
		}

		// FIX: Cast the int64 to int for the model struct field.
		repoData.Owner.ID = int(ownerID)
		if repoNodeID.Valid {
			repoData.Repository.NodeID = repoNodeID
		}
		if ownerNodeID.Valid {
			repoData.Owner.NodeID = ownerNodeID
		}
		if licenseKey.Valid {
			repoData.Repository.License.Key = licenseKey
		}
		if licenseName.Valid {
			repoData.Repository.License.Name = licenseName
		}

		// Fetch tags
		tags, err := pc.getTagsForRepository(int64(repoData.Repository.ID))
		if err != nil {
			return nil, fmt.Errorf("failed to get tags for repository %d: %w", repoData.Repository.ID, err)
		}
		repoData.Tags = tags

		// Fetch topics
		topics, err := pc.getTopicsForRepository(int64(repoData.Repository.ID))
		if err != nil {
			return nil, fmt.Errorf("failed to get topics for repository %d: %w", repoData.Repository.ID, err)
		}
		repoData.Repository.Topics = topics

		// Fetch languages
		languages, err := pc.getLanguagesForRepository(int64(repoData.Repository.ID))
		if err != nil {
			return nil, fmt.Errorf("failed to get languages for repository %d: %w", repoData.Repository.ID, err)
		}
		repoData.Repository.Languages = languages

		repositoriesData = append(repositoriesData, repoData)
	}

	return repositoriesData, nil
}
