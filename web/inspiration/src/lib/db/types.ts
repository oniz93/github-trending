// Purpose: Type definitions for database entities
// Context: Used to provide TypeScript types for database tables and query results

export interface Stargazer {
    id: number;
    login: string;
    avatar_url: string;
    html_url: string;
    type: string;
    starred_at: string | null;
    created_at: string;
    updated_at: string;
}

export interface Repository {
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

export type QueryParams = string | number | boolean | null; 