-- Purpose: Initial database schema for stargazers and repositories
-- Context: Creates tables to store GitHub stargazers and their repositories with HTTP-optimized settings

-- SQLite optimizations for HTTP serving
PRAGMA journal_mode = DELETE;
PRAGMA page_size = 1024;
PRAGMA cache_size = 5000;
PRAGMA temp_store = MEMORY;
PRAGMA locking_mode = EXCLUSIVE;
PRAGMA synchronous = NORMAL;
PRAGMA foreign_keys = ON;

DROP TABLE IF EXISTS stargazers;
DROP TABLE IF EXISTS repositories;

CREATE TABLE IF NOT EXISTS stargazers (
  id INTEGER PRIMARY KEY,
  login TEXT NOT NULL,
  avatar_url TEXT NOT NULL,
  html_url TEXT NOT NULL,
  type TEXT NOT NULL,
  starred_at TEXT,
  created_at TEXT NOT NULL DEFAULT CURRENT_TIMESTAMP,
  updated_at TEXT NOT NULL DEFAULT CURRENT_TIMESTAMP
);

-- Indexes for stargazers table
CREATE INDEX idx_stargazers_login ON stargazers(login);
CREATE INDEX idx_stargazers_type ON stargazers(type);
CREATE INDEX idx_stargazers_starred_at ON stargazers(starred_at);
CREATE INDEX idx_stargazers_created_at ON stargazers(created_at);

CREATE TABLE IF NOT EXISTS repositories (
  id TEXT PRIMARY KEY,
  owner_id INTEGER NOT NULL,
  name TEXT NOT NULL,
  full_name TEXT NOT NULL,
  description TEXT,
  html_url TEXT NOT NULL,
  language TEXT,
  default_branch TEXT,
  featured TEXT,
  stargazers_count INTEGER NOT NULL DEFAULT 0,
  fork INTEGER NOT NULL DEFAULT 0,
  is_pinned INTEGER NOT NULL DEFAULT 0,
  created_at TEXT NOT NULL,
  updated_at TEXT NOT NULL,
  fetched_at TEXT NOT NULL DEFAULT CURRENT_TIMESTAMP,
  FOREIGN KEY (owner_id) REFERENCES stargazers(id) ON DELETE CASCADE
);

-- Indexes for repositories table
CREATE INDEX idx_repositories_owner_id ON repositories(owner_id);
CREATE INDEX idx_repositories_full_name ON repositories(full_name);
CREATE INDEX idx_repositories_language ON repositories(language);
CREATE INDEX idx_repositories_stargazers_count ON repositories(stargazers_count DESC);
CREATE INDEX idx_repositories_is_pinned ON repositories(is_pinned);
CREATE INDEX idx_repositories_created_at ON repositories(created_at);
CREATE INDEX idx_repositories_updated_at ON repositories(updated_at);

-- Compound indexes for common query patterns
CREATE INDEX idx_repositories_owner_language ON repositories(owner_id, language);
CREATE INDEX idx_repositories_owner_stars ON repositories(owner_id, stargazers_count DESC);
CREATE INDEX idx_repositories_pinned_stars ON repositories(is_pinned, stargazers_count DESC); 