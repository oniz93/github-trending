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
      stargazers_count: repo.stats.length > 0 ? repo.stats[0].stargazers_count : 0,
      language: repo.repository.language.Valid ? repo.repository.language.String : 'Unknown',
      readme_url: repo.repository.readme_url.Valid ? repo.repository.readme_url.String : '',
      tags: repo.tags || [],
      stats: repo.stats || [],
      forks_count: repo.repository.forks_count,
      created_at: repo.repository.created_at,
      updated_at: repo.repository.updated_at,
      stargazersUrl: `${repo.repository.html_url}/stargazers`,
      forksUrl: `${repo.repository.html_url}/forks`,
      default_branch: repo.repository.default_branch,
      type: 'repo',
    }));

    return { repositories, sessionId: newSessionId };
  } catch (error) {
    console.error('Error fetching repositories:', error);
    throw error;
  }
};

export const fetchReadme = async (fullName: string) => {
    try {
        const url = `https://api.github.com/repos/${fullName}/readme`;
        const response = await axios.get(url, {
            headers: {
                Accept: 'application/vnd.github.raw',
            }
        });
        return response.data;
    } catch (error) {
        console.error('Error fetching README:', error);
        return 'Could not load README.';
    }
};
