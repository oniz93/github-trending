CREATE TABLE IF NOT EXISTS owners (
    id BIGINT PRIMARY KEY,
    login VARCHAR(255) UNIQUE NOT NULL,
    avatar_url VARCHAR(255),
    html_url VARCHAR(255),
    type VARCHAR(255)
);

CREATE TABLE IF NOT EXISTS licenses (
    key VARCHAR(255) PRIMARY KEY,
    name VARCHAR(255),
    spdx_id VARCHAR(255),
    url VARCHAR(255),
    node_id VARCHAR(255)
);

CREATE TABLE IF NOT EXISTS repositories (
    id BIGINT PRIMARY KEY,
    node_id VARCHAR(255),
    name VARCHAR(255) NOT NULL,
    full_name VARCHAR(255) UNIQUE NOT NULL,
    owner_id BIGINT REFERENCES owners(id),
    description TEXT,
    html_url VARCHAR(255),
    homepage VARCHAR(255),
    default_branch VARCHAR(255),
    license_key VARCHAR(255) REFERENCES licenses(key),
    readme_url VARCHAR(255),
    created_at TIMESTAMP WITH TIME ZONE,
    is_fork BOOLEAN,
    is_template BOOLEAN,
    is_archived BOOLEAN,
    is_disabled BOOLEAN,
    last_crawled_at TIMESTAMP WITH TIME ZONE
);

CREATE TABLE IF NOT EXISTS languages (
    id SERIAL PRIMARY KEY,
    name VARCHAR(255) UNIQUE NOT NULL
);

CREATE TABLE IF NOT EXISTS repository_languages (
    repository_id BIGINT REFERENCES repositories(id) ON DELETE CASCADE,
    language_id INT REFERENCES languages(id) ON DELETE CASCADE,
    size BIGINT NOT NULL,
    PRIMARY KEY (repository_id, language_id)
);

CREATE TABLE IF NOT EXISTS tags (
    id SERIAL PRIMARY KEY,
    name VARCHAR(255) UNIQUE NOT NULL
);

CREATE TABLE IF NOT EXISTS repository_tags (
    repository_id BIGINT REFERENCES repositories(id) ON DELETE CASCADE,
    tag_id INT REFERENCES tags(id) ON DELETE CASCADE,
    PRIMARY KEY (repository_id, tag_id)
);

CREATE TABLE IF NOT EXISTS topics (
    id SERIAL PRIMARY KEY,
    name VARCHAR(255) UNIQUE NOT NULL
);

CREATE TABLE IF NOT EXISTS repository_topics (
    repository_id BIGINT REFERENCES repositories(id) ON DELETE CASCADE,
    topic_id INT REFERENCES topics(id) ON DELETE CASCADE,
    PRIMARY KEY (repository_id, topic_id)
);

CREATE TABLE IF NOT EXISTS repository_views (
    id SERIAL PRIMARY KEY,
    session_id VARCHAR(255) NOT NULL,
    repository_id BIGINT NOT NULL,
    viewed_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP
);

CREATE TABLE IF NOT EXISTS repository_similarity (
    id BIGINT PRIMARY KEY,
    data JSONB NOT NULL
);