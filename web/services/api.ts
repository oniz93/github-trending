import axios from 'axios';
import { FeedProject, RepositoryResponse } from '../types/repository';

import Constants from 'expo-constants';

const API_BASE_URL = Constants.expoConfig?.extra?.EXPO_PUBLIC_API_URL as string;

export const getRepositories = async (page: number, sessionId: string): Promise<{repositories: FeedProject[], sessionId: string}> => {
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
      language: repo.repository.language.Valid 
        ? repo.repository.language.String 
        : (Object.keys(repo.repository.languages).length > 0 
            ? Object.keys(repo.repository.languages).reduce((a, b) => repo.repository.languages[a] > repo.repository.languages[b] ? a : b)
            : 'Unknown'),
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

export const fetchReadme = async (repoId: string) => {
    try {
        const response = await axios.get(`${API_BASE_URL}/getReadme?repoId=${repoId}`);
        return response.data;
    } catch (error) {
        console.error('Error fetching README:', error);
        throw error;
    }
};

export const trackOpenRepository = async (sessionId: string, repositoryId: string) => {
    try {
        await axios.post(`${API_BASE_URL}/trackOpenRepository`, {
            sessionId,
            repositoryId: parseInt(repositoryId, 10),
        });
    } catch (error) {
        console.error('Error tracking open repository:', error);
    }
};

export const getRepository = async (repositoryId: string): Promise<FeedProject> => {
    try {
        const response = await axios.get(`${API_BASE_URL}/repository/${repositoryId}`);

        const repo: RepositoryResponse = response.data;

        return {
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
            language: repo.repository.language.Valid
                ? repo.repository.language.String
                : (Object.keys(repo.repository.languages).length > 0
                    ? Object.keys(repo.repository.languages).reduce((a, b) => repo.repository.languages[a] > repo.repository.languages[b] ? a : b)
                    : 'Unknown'),
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
        };
    } catch (error) {
        console.error('Error fetching repository:', error);
        throw error;
    }
}
