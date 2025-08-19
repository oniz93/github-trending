-- This script changes the data type of row_hash from UInt64 to String to prevent integer overflow issues in clients.

-- Step 1: Add a temporary column with the String data type.
ALTER TABLE default.repository_stats ADD COLUMN row_hash_str String;

-- Step 2: Convert the existing UInt64 hashes to strings and populate the new column.
-- This is a mutation and will run in the background. Please wait for it to complete.
ALTER TABLE default.repository_stats UPDATE row_hash_str = toString(row_hash) WHERE 1;

-- Note: Please wait for the mutation from Step 2 to complete before proceeding.
-- You can check the progress with the following query:
SELECT * FROM system.mutations WHERE table = 'repository_stats' AND database = 'default';

-- Step 3: Drop the old UInt64 row_hash column.
ALTER TABLE default.repository_stats DROP COLUMN row_hash;

-- Step 4: Rename the new string column to row_hash.
ALTER TABLE default.repository_stats RENAME COLUMN row_hash_str TO row_hash;

KILL MUTATION WHERE mutation_id = 'mutation_882816.txt';