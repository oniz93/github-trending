# SQLite Database Client Module

## Purpose
Client-side SQLite database access using sql.js-httpvfs for querying stargazers and repositories data over HTTP.

## Features
- 🔍 Query and filter stargazers and repositories
- 📊 Get repository statistics and analytics
- 🔎 Full-text search capabilities
- 📅 Time-based filtering
- 📝 Type-safe queries with TypeScript

## Installation

1. Install the required dependencies:
```bash
npm install sql.js-httpvfs
```

2. Copy the SQLite worker file to your public directory:
```bash
cp node_modules/sql.js-httpvfs/dist/sqlite.worker.js public/sql.js-httpvfs/
cp node_modules/sql.js-httpvfs/dist/sql-wasm.wasm public/sql.js-httpvfs/
```

## Usage

### Initialize the Database

```typescript
import { initDatabase } from '$lib/db/client';

// Initialize at app startup
await initDatabase('https://your-domain.com/path/to/stars.db');
```

### Query Stargazers

```typescript
import { getStargazers, getStargazerById } from '$lib/db/client';

// Get all stargazers with filtering
const stargazers = await getStargazers({
    limit: 20,
    offset: 0,
    type: 'User',
    orderBy: 'starred_at',
    order: 'desc'
});

// Get a specific stargazer
const stargazer = await getStargazerById(123);
```

### Query Repositories

```typescript
import { getRepositories } from '$lib/db/client';

// Get repositories with filters
const repositories = await getRepositories({
    limit: 50,
    offset: 0,
    language: 'typescript',
    ownerId: 123,
    isPinned: true,
    minStars: 10,
    orderBy: 'stargazers_count',
    order: 'desc'
});
```

### Search Repositories

```typescript
import { searchRepositories } from '$lib/db/client';

// Search repositories by name or description
const results = await searchRepositories('svelte', {
    limit: 10,
    offset: 0
});
```

### Get Recent Repositories

```typescript
import { getRecentRepositories } from '$lib/db/client';

// Get repositories from the last 7 days
const recentRepos = await getRecentRepositories(7, {
    limit: 20,
    offset: 0
});
```

### Get Repository Statistics

```typescript
import { getRepositoryStats } from '$lib/db/client';

// Get overall repository statistics
const stats = await getRepositoryStats();
console.log(stats);
// {
//     totalRepos: 1000,
//     totalPinnedRepos: 50,
//     avgStarsPerRepo: 25.5,
//     topLanguages: [
//         { language: 'JavaScript', count: 300 },
//         { language: 'TypeScript', count: 200 },
//         // ...
//     ]
// }
```

## API Reference

### Types

```typescript
interface Stargazer {
    id: number;
    login: string;
    avatar_url: string;
    html_url: string;
    type: string;
    starred_at: string | null;
    created_at: string;
    updated_at: string;
}

interface Repository {
    id: number;
    owner_id: number;
    name: string;
    full_name: string;
    description: string | null;
    html_url: string;
    language: string | null;
    stargazers_count: number;
    fork: number;
    is_pinned: number;
    created_at: string;
    updated_at: string;
    fetched_at: string;
}
```

### Functions

#### `initDatabase(dbUrl: string)`
Initializes the database connection. Must be called before using any other functions.

#### `getStargazers(options)`
Get stargazers with optional filtering and pagination.
- `limit`: Number of results (default: 50)
- `offset`: Pagination offset (default: 0)
- `type`: Filter by user type
- `orderBy`: Sort field ('starred_at', 'created_at', 'updated_at', 'login')
- `order`: Sort order ('asc' or 'desc')

#### `getRepositories(options)`
Get repositories with optional filtering and pagination.
- `limit`: Number of results (default: 50)
- `offset`: Pagination offset (default: 0)
- `language`: Filter by programming language
- `ownerId`: Filter by owner ID
- `isPinned`: Filter pinned repositories
- `minStars`: Minimum number of stars
- `orderBy`: Sort field ('stargazers_count', 'created_at', 'updated_at', 'name')
- `order`: Sort order ('asc' or 'desc')

#### `searchRepositories(query, options)`
Search repositories by name or description.
- `query`: Search term
- `limit`: Number of results (default: 50)
- `offset`: Pagination offset (default: 0)

#### `getRecentRepositories(days, options)`
Get repositories created within a specific time period.
- `days`: Number of days to look back
- `limit`: Number of results (default: 50)
- `offset`: Pagination offset (default: 0)

#### `getRepositoryStats()`
Get repository statistics including total counts and top languages.

## Error Handling

All functions will throw an error if the database is not initialized. Make sure to call `initDatabase()` before using any other functions.

```typescript
try {
    const repos = await getRepositories();
} catch (error) {
    if (error.message === 'Database not initialized') {
        // Handle initialization error
    } else {
        // Handle other errors
    }
}
```

## Performance Considerations

- All queries use parameterized statements to prevent SQL injection
- Results are paginated by default to prevent loading too much data
- The module uses a Web Worker for database operations to avoid blocking the main thread
- Queries are optimized to use indexes where possible

## Browser Support

This module requires browsers that support:
- Web Workers
- WebAssembly
- Fetch API
- ES2017+ features

## License

MIT 