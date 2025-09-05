export interface FeedProject {
    id: string;
    name: string;
    full_name: string;
    description: string | null;
    html_url: string;
    language: string | null;
    stargazers_count: number;
    forks_count: number;
    watchers_count: number;
    open_issues_count: number;
    created_at: string;
    updated_at: string;
    pushed_at: string;
    owner: {
        login: string;
        avatar_url: string;
    };
    readmeSnippet?: string;
    stargazersUrl: string;
    forksUrl: string;
    default_branch: string;
    type: 'repo' | 'ad' | 'message';
    stats: RepositoryStat;
    topics: string[];
    languages: { [key: string]: number };
    license: { name: string } | null;
}

export interface Repository {
    id: number;
    name: string;
    full_name: string;
    owner: {
        login: string;
        avatar_url: string;
    };
    html_url: string;
    description: string;
    stargazers_count: number;
    language: string;
    languages: { [key: string]: number };
    readme_url: string;
    stats: RepositoryStat;
}

export interface RepositoryStat {
    event_date: string;
    event_time: string;
    stargazers_count: number;
    watchers_count: number;
    forks_count: number;
    open_issues_count: number;
    pushed_at: string;
    score: number;
}

export interface RepositoryResponse {
    repository: {
        id: number;
        name: string;
        full_name: string;
        description: {
            String: string;
            Valid: boolean;
        };
        html_url: string;
        language: {
            String: string;
            Valid: boolean;
        };
        readme_url: {
            String: string;
            Valid: boolean;
        };
        default_branch: string;
        forks_count: number;
        created_at: string;
        updated_at: string;
        topics: string[];
        languages: { [key: string]: number };
        license: {
            name: {
                String: string;
                Valid: boolean;
            };
        } | null;
    };
    owner: {
        login: string;
        avatar_url: string;
    };
    stats: RepositoryStat;
}