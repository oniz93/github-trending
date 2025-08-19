-- This script adds a hash to each row and removes duplicates from the repository_stats table.

-- Step 1: Add the row_hash column to the repository_stats table.
-- This hash will be used to identify unique rows based on a composite of fields.
ALTER TABLE default.repository_stats ADD COLUMN row_hash UInt64;

-- Step 2: Calculate and update the row_hash for all existing rows.
-- This is a mutation, which runs asynchronously in the background. You need to wait for it to complete
-- before running the next steps. You can check the status in the system.mutations table.
-- The hash is calculated using sipHash64 on the columns that define a unique state for a repository's stats.
ALTER TABLE default.repository_stats UPDATE row_hash = sipHash64(repository_id, stargazers_count, watchers_count, forks_count, open_issues_count) WHERE 1;

-- Note: Please wait for the mutation from Step 2 to complete before proceeding.
-- You can check the progress with the following query:
SELECT * FROM system.mutations WHERE table = 'repository_stats' AND database = 'default';

-- Step 3: Create a new table to hold the deduplicated data.
-- We select the first occurrence (the oldest) of each unique row_hash.
-- The subquery uses the row_number() window function to identify duplicates.
CREATE TABLE default.repository_stats_deduped
ENGINE = MergeTree
PARTITION BY toYYYYMM(event_date)
ORDER BY (repository_id, event_time)
AS SELECT
    event_date,
    event_time,
    repository_id,
    stargazers_count,
    watchers_count,
    forks_count,
    open_issues_count,
    pushed_at,
    score,
    row_hash
FROM (
    SELECT
        *,
        row_number() OVER (PARTITION BY row_hash ORDER BY event_time ASC) as rn
    FROM default.repository_stats
    WHERE row_hash != '' -- Ensure the hash has been calculated
)
WHERE rn = 1;

-- Step 4: Rename the tables to make the deduplicated table the primary one.
-- The original table is kept as a backup until the process is verified.
RENAME TABLE default.repository_stats TO default.repository_stats_old, default.repository_stats_deduped TO default.repository_stats;

-- Step 5: Drop the old table.
-- IMPORTANT: Run this step manually after you have verified that the new table
-- contains the correct data and the system is working as expected.
-- DROP TABLE default.repository_stats_old;


SELECT COUNT(1) FROM repository_stats_old