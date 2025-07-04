-- storage/clickhouse/schema.sql

-- This table stores time-series data for repository metrics.
-- Each row is a snapshot of a repository's stats at a specific point in time.
CREATE TABLE IF NOT EXISTS repository_stats (
    -- The date of the event, used for partitioning.
    event_date Date,

    -- The exact timestamp of when the data was crawled.
    event_time DateTime,

    -- The GitHub ID of the repository, linking back to the PostgreSQL metadata.
    repository_id UInt64,

    -- Core metrics that change over time.
    stargazers_count UInt64,
    watchers_count UInt64,
    forks_count UInt64,
    open_issues_count UInt64,

    -- The timestamp of the last push event, a key indicator of recent activity.
    pushed_at DateTime,

    -- The search score from the GitHub API, if available.
    score Float64

) ENGINE = MergeTree()
-- Partitioning by month is a good balance for time-based queries.
PARTITION BY toYYYYMM(event_date)
-- This order key is optimized for querying a single repository's history
-- or for querying all repositories at a specific point in time.
ORDER BY (repository_id, event_time);