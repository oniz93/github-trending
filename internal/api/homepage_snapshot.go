package api

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"log"
	"sync"
	"sync/atomic"
	"time"

	"github.com/redis/go-redis/v9"
	"github.com/teomiscia/github-trending/internal/database"
	"github.com/teomiscia/github-trending/internal/models"
)

const (
	homepageSnapshotName            = "trending"
	homepageSnapshotRedisKey        = "homepage:snapshot:trending:v1"
	homepageSnapshotMaxRepositories = 200
	homepageSnapshotPageSize        = 50
	homepageSnapshotTTL             = 7 * 24 * time.Hour
	homepageTrendingWindowDays      = 30
)

// homepageRepository mirrors the object returned by /retrieveList. Keeping the
// exact same shape means the app does not need to know whether a response came
// from the snapshot or from the regular query path.
type homepageRepository struct {
	Repository models.Repository      `json:"repository"`
	Owner      models.Owner           `json:"owner"`
	Stats      *models.RepositoryStat `json:"stats,omitempty"`
}

type homepageSnapshot struct {
	BuiltAt      time.Time            `json:"built_at"`
	Repositories []homepageRepository `json:"repositories"`
}

// homepageSnapshotService keeps a precomputed trending page in memory and
// refreshes it in the background. The snapshot is also persisted in Redis and
// Postgres so that a restart or a cache flush does not force users to wait for
// the expensive ClickHouse growth query.
type homepageSnapshotService struct {
	redis           *redis.Client
	pgdb            *database.PostgresConnection
	chdb            *database.ClickHouseConnection
	refreshInterval time.Duration

	current  atomic.Pointer[homepageSnapshot]
	building sync.Mutex
}

func newHomepageSnapshotService(refreshInterval time.Duration, redisClient *redis.Client, pgdb *database.PostgresConnection, chdb *database.ClickHouseConnection) *homepageSnapshotService {
	if refreshInterval <= 0 {
		refreshInterval = 30 * time.Minute
	}

	return &homepageSnapshotService{
		redis:           redisClient,
		pgdb:            pgdb,
		chdb:            chdb,
		refreshInterval: refreshInterval,
	}
}

// Start loads the last persisted snapshot (if any) and starts the background
// refresh loop. It never blocks on the expensive trending computation.
func (s *homepageSnapshotService) Start(ctx context.Context) {
	if err := s.pgdb.EnsureHomepageSnapshotTable(); err != nil {
		log.Printf("Failed to ensure homepage snapshot table: %v", err)
	}

	s.loadPersisted(ctx)
	go s.refreshLoop(ctx)
}

// Page returns a page of the current snapshot. The second return value is false
// when the snapshot does not have that page, in which case the caller should
// fall back to the regular query path.
func (s *homepageSnapshotService) Page(page int) ([]homepageRepository, bool) {
	snapshot := s.current.Load()
	if snapshot == nil || page < 0 {
		return nil, false
	}

	start := page * homepageSnapshotPageSize
	if start >= len(snapshot.Repositories) {
		return nil, false
	}

	end := start + homepageSnapshotPageSize
	if end > len(snapshot.Repositories) {
		end = len(snapshot.Repositories)
	}

	return snapshot.Repositories[start:end], true
}

func (s *homepageSnapshotService) refreshLoop(ctx context.Context) {
	// If a fresh snapshot was loaded from Redis/Postgres, wait for it to age
	// out. Otherwise build immediately.
	wait := time.Duration(0)
	if snapshot := s.current.Load(); snapshot != nil {
		if age := time.Since(snapshot.BuiltAt); age < s.refreshInterval {
			wait = s.refreshInterval - age
		}
	}

	for {
		if ctx.Err() != nil {
			return
		}

		if wait > 0 {
			select {
			case <-ctx.Done():
				return
			case <-time.After(wait):
			}
		}

		if s.rebuild() {
			wait = s.refreshInterval
		} else {
			// Retry quickly after a transient database failure instead of
			// leaving the service without a snapshot for a full interval.
			wait = time.Minute
		}
	}
}

func (s *homepageSnapshotService) rebuild() bool {
	// A single build is enough; skip overlapping refreshes.
	if !s.building.TryLock() {
		return true
	}
	defer s.building.Unlock()

	started := time.Now()
	log.Println("Building homepage trending snapshot...")

	snapshot, err := s.build()
	if err != nil {
		log.Printf("Failed to build homepage trending snapshot: %v", err)
		return false
	}

	s.current.Store(snapshot)
	s.persist(snapshot)
	s.warmRepositoryStatsCache(snapshot)

	log.Printf(
		"Homepage trending snapshot ready: %d repositories in %s",
		len(snapshot.Repositories),
		time.Since(started).Round(time.Millisecond),
	)
	return true
}

func (s *homepageSnapshotService) build() (*homepageSnapshot, error) {
	repoIDs, err := s.chdb.GetTrendingRepositoryIDsByGrowth(homepageTrendingWindowDays)
	if err != nil {
		return nil, fmt.Errorf("failed to get trending repository IDs: %w", err)
	}

	if len(repoIDs) > homepageSnapshotMaxRepositories {
		repoIDs = repoIDs[:homepageSnapshotMaxRepositories]
	}
	if len(repoIDs) == 0 {
		return nil, fmt.Errorf("trending query returned no repositories")
	}

	repositories, err := s.pgdb.GetRepositoriesDataByIDs(repoIDs)
	if err != nil {
		return nil, fmt.Errorf("failed to get repository data: %w", err)
	}
	if len(repositories) == 0 {
		return nil, fmt.Errorf("no repository data found for trending IDs")
	}

	repositoriesByID := make(map[int64]models.RepositoryData, len(repositories))
	for _, repo := range repositories {
		repositoriesByID[int64(repo.Repository.ID)] = repo
	}

	latestStats, statsErr := s.chdb.GetLatestRepositoryStatsByIDs(repoIDs)
	if statsErr != nil {
		// Stats are important but not critical: a snapshot without them is
		// still much better than no snapshot at all.
		log.Printf("Failed to get latest stats for homepage snapshot: %v", statsErr)
	}

	entries := make([]homepageRepository, 0, len(repoIDs))
	orderedIDs := make([]int64, 0, len(repoIDs))
	for _, repoID := range repoIDs {
		repo, ok := repositoriesByID[repoID]
		if !ok {
			continue
		}

		entries = append(entries, homepageRepository{
			Repository: repo.Repository,
			Owner:      repo.Owner,
			Stats:      latestStats[repoID],
		})
		orderedIDs = append(orderedIDs, repoID)
	}

	if len(entries) == 0 {
		return nil, fmt.Errorf("none of the trending repositories could be loaded")
	}

	s.warmTrendingIDsCache(orderedIDs)

	return &homepageSnapshot{
		BuiltAt:      time.Now(),
		Repositories: entries,
	}, nil
}

func (s *homepageSnapshotService) persist(snapshot *homepageSnapshot) {
	payload, err := json.Marshal(snapshot)
	if err != nil {
		log.Printf("Failed to marshal homepage snapshot: %v", err)
		return
	}

	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	if err := s.redis.Set(ctx, homepageSnapshotRedisKey, compress(payload), homepageSnapshotTTL).Err(); err != nil {
		log.Printf("Failed to store homepage snapshot in Redis: %v", err)
	}

	if err := s.pgdb.SaveHomepageSnapshot(homepageSnapshotName, payload, snapshot.BuiltAt); err != nil {
		log.Printf("Failed to store homepage snapshot in Postgres: %v", err)
	}
}

func (s *homepageSnapshotService) loadPersisted(ctx context.Context) {
	if compressed, err := s.redis.Get(ctx, homepageSnapshotRedisKey).Bytes(); err == nil {
		snapshot, decodeErr := decodeCompressedHomepageSnapshot(compressed)
		if decodeErr == nil {
			s.current.Store(snapshot)
			log.Printf("Loaded homepage snapshot from Redis (%d repositories, built %s ago)", len(snapshot.Repositories), time.Since(snapshot.BuiltAt).Round(time.Second))
			return
		}
		log.Printf("Ignoring invalid homepage snapshot in Redis: %v", decodeErr)
	}

	payload, builtAt, err := s.pgdb.GetHomepageSnapshot(homepageSnapshotName)
	if err != nil {
		if err != sql.ErrNoRows {
			log.Printf("Failed to load homepage snapshot from Postgres: %v", err)
		}
		return
	}

	snapshot, err := decodeHomepageSnapshot(payload)
	if err != nil {
		log.Printf("Ignoring invalid homepage snapshot in Postgres: %v", err)
		return
	}
	if snapshot.BuiltAt.IsZero() {
		snapshot.BuiltAt = builtAt
	}

	s.current.Store(snapshot)
	log.Printf("Loaded homepage snapshot from Postgres (%d repositories, built %s ago)", len(snapshot.Repositories), time.Since(snapshot.BuiltAt).Round(time.Second))

	// Re-populate Redis so the next instance can skip Postgres.
	if compressed, err := encodeCompressedHomepageSnapshot(snapshot); err == nil {
		if err := s.redis.Set(ctx, homepageSnapshotRedisKey, compressed, homepageSnapshotTTL).Err(); err != nil {
			log.Printf("Failed to re-populate homepage snapshot in Redis: %v", err)
		}
	}
}

// warmRepositoryStatsCache stores the latest stat for each snapshot repository
// using the same cache key and shape as the regular request path. This keeps
// page loads fast for users whose session falls back to the personalized path.
func (s *homepageSnapshotService) warmRepositoryStatsCache(snapshot *homepageSnapshot) {
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	pipeline := s.redis.Pipeline()
	for _, entry := range snapshot.Repositories {
		if entry.Stats == nil {
			continue
		}

		payload, err := json.Marshal([]models.RepositoryStat{*entry.Stats})
		if err != nil {
			continue
		}

		pipeline.Set(ctx, fmt.Sprintf("repo_stats:%d", entry.Repository.ID), compress(payload), 12*time.Hour)
	}

	if _, err := pipeline.Exec(ctx); err != nil {
		log.Printf("Failed to warm repository stats cache: %v", err)
	}
}

// warmTrendingIDsCache keeps the existing filtered-trending cache warm as well.
func (s *homepageSnapshotService) warmTrendingIDsCache(repoIDs []int64) {
	payload, err := json.Marshal(repoIDs)
	if err != nil {
		return
	}

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	if err := s.redis.Set(ctx, fmt.Sprintf("trending_repo_ids_by_growth:%d", homepageTrendingWindowDays), compress(payload), 24*time.Hour).Err(); err != nil {
		log.Printf("Failed to warm trending repository ID cache: %v", err)
	}
}

func encodeCompressedHomepageSnapshot(snapshot *homepageSnapshot) ([]byte, error) {
	payload, err := json.Marshal(snapshot)
	if err != nil {
		return nil, err
	}
	return compress(payload), nil
}

func decodeCompressedHomepageSnapshot(data []byte) (*homepageSnapshot, error) {
	payload, err := decompress(data)
	if err != nil {
		return nil, err
	}
	return decodeHomepageSnapshot(payload)
}

func decodeHomepageSnapshot(payload []byte) (*homepageSnapshot, error) {
	var snapshot homepageSnapshot
	if err := json.Unmarshal(payload, &snapshot); err != nil {
		return nil, err
	}
	if len(snapshot.Repositories) == 0 {
		return nil, fmt.Errorf("snapshot contains no repositories")
	}
	return &snapshot, nil
}
