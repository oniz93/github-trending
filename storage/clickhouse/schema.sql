CREATE TABLE IF NOT EXISTS repository_stats (
    event_date Date,
    event_time DateTime,
    repository_id UInt64,
    stargazers_count UInt64,
    watchers_count UInt64,
    forks_count UInt64,
    open_issues_count UInt64,
    pushed_at DateTime,
    score Float64
)
ENGINE = MergeTree
PARTITION BY toYYYYMM(event_date)
ORDER BY (repository_id, event_time);