
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
  tags: string[];
  stats: RepositoryStat[];
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
    };
    owner: {
        login: string;
        avatar_url: string;
    };
    stats: RepositoryStat[];
    tags: string[];
}
