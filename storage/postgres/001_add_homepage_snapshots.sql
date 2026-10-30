-- Homepage snapshot support + a fast lookup index for personalized feeds.
--
-- Run this against an existing database. It is intentionally idempotent.
-- CREATE INDEX CONCURRENTLY cannot run inside a transaction, so do not wrap
-- this file in BEGIN/COMMIT.

CREATE TABLE IF NOT EXISTS homepage_snapshots (
    name TEXT PRIMARY KEY,
    payload JSONB NOT NULL,
    built_at TIMESTAMP WITH TIME ZONE NOT NULL
);

CREATE INDEX CONCURRENTLY IF NOT EXISTS idx_repository_views_session_id
    ON repository_views (session_id, viewed_at DESC);
