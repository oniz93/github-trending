import axios from 'axios';
import { FeedProject, RepositoryResponse } from '../types/repository';

const API_BASE_URL = 'http://192.168.1.106:8080'; // Using a placeholder

export const getRepositories = async (page: number, sessionId?: string): Promise<{repositories: FeedProject[], sessionId: string}> => {
  try {
    const response = await axios.get(`${API_BASE_URL}/retrieveList`, {
      params: {
        page,
        sessionId,
      },
    });

    const { repositories: rawRepositories, sessionId: newSessionId } = response.data;

    const repositories: FeedProject[] = rawRepositories.map((repo: RepositoryResponse) => ({
      id: repo.repository.id.toString(),
      name: repo.repository.name,
      full_name: repo.repository.full_name,
      owner: {
        login: repo.owner.login,
        avatar_url: repo.owner.avatar_url,
      },
      html_url: repo.repository.html_url,
      description: repo.repository.description.Valid ? repo.repository.description.String : '',
      stargazers_count: repo.stats ? repo.stats.stargazers_count : 0,
      forks_count: repo.stats ? repo.stats.forks_count : 0,
      watchers_count: repo.stats ? repo.stats.watchers_count : 0,
      open_issues_count: repo.stats ? repo.stats.open_issues_count : 0,
      pushed_at: repo.stats ? repo.stats.pushed_at : '',
      language: repo.repository.language.Valid ? repo.repository.language.String : 'Unknown',
      stats: repo.stats,
      created_at: repo.repository.created_at,
      updated_at: repo.repository.updated_at,
      stargazersUrl: `${repo.repository.html_url}/stargazers`,
      forksUrl: `${repo.repository.html_url}/forks`,
      default_branch: repo.repository.default_branch,
      type: 'repo',
      topics: repo.repository.topics || [],
      languages: repo.repository.languages || {},
      license: repo.repository.license && repo.repository.license.name.Valid ? { name: repo.repository.license.name.String } : null,
    }));

    return { repositories, sessionId: newSessionId };
  } catch (error) {
    console.error('Error fetching repositories:', error);
    throw error;
  }
};

export const fetchReadme = async (fullName: string) => {
    try {
        // 1. Fetch raw markdown
        const readmeUrl = `https://api.github.com/repos/${fullName}/readme`;
        const readmeResponse = await axios.get(readmeUrl, {
            headers: {
                Accept: 'application/vnd.github.raw',
            }
        });
        const markdown = readmeResponse.data;

        // 2. Render markdown using the /markdown endpoint
        const markdownUrl = 'https://api.github.com/markdown';
        const markdownResponse = await axios.post(markdownUrl, {
            text: markdown,
            mode: 'gfm',
            context: fullName
        }, {
            headers: {
                Accept: 'application/vnd.github.html+json',
            }
        });

        return markdownResponse.data;
    } catch (error) {
        console.error('Error fetching or rendering README:', error);
        return 'Could not load README.';
    }
};
