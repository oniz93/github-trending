// Purpose: Functions for fetching GitHub repository data
// Context: Used to fetch repository and pinned repository information from GitHub API

import type { Octokit } from "@octokit/rest";
import type { GraphQLResponse, Repository } from "./types";

export async function fetchPinnedRepositories(octokit: Octokit, username: string): Promise<Repository[]> {
  try {
    const query = `
      query($username: String!) {
        user(login: $username) {
          pinnedItems(first: 6, types: REPOSITORY) {
            nodes {
              ... on Repository {
                id
                name
                nameWithOwner
                description
                url
                primaryLanguage {
                  name
                }
                stargazerCount
                isFork
                defaultBranchRef {
                  name
                }
                createdAt
                updatedAt
              }
            }
          }
        }
      }
    `;

    const response = await octokit.graphql<GraphQLResponse>(query, { username });
    const pinnedRepos = response.user?.pinnedItems?.nodes ?? [];

    return pinnedRepos.map(repo => ({
      id: repo.id.toString(),
      name: repo.name,
      full_name: repo.nameWithOwner,
      description: repo.description,
      html_url: repo.url,
      language: repo.primaryLanguage?.name ?? null,
      stargazers_count: repo.stargazerCount,
      fork: repo.isFork ? 1 : 0,
      default_branch: repo.defaultBranchRef?.name ?? null,
      created_at: repo.createdAt,
      updated_at: repo.updatedAt,
      is_pinned: 1,
      owner_id: 0, // This will be set later when we know the stargazer's ID
      fetched_at: new Date().toISOString()
    }));
  } catch (error) {
    console.error(`Error fetching pinned repositories for ${username}:`, error);
    return [];
  }
}

export async function fetchUserRepositories(octokit: Octokit, username: string): Promise<Repository[]> {
  try {
    const repos: Repository[] = [];
    let page = 1;
    let hasMorePages = true;

    // First, get pinned repositories to compare against
    const pinnedRepos = await fetchPinnedRepositories(octokit, username);
    const pinnedRepoNames = new Set(pinnedRepos.map(repo => repo.name));

    while (hasMorePages) {
      console.log(`Fetching repositories for ${username}, page ${page}...`);

      const response = await octokit.rest.repos.listForUser({
        username,
        per_page: 100,
        page: page,
        type: 'owner',
        sort: 'updated'
      });

      const filteredRepos = response.data.map(repo => ({
        id: repo.id.toString(),
        name: repo.name,
        full_name: repo.full_name,
        description: repo.description,
        html_url: repo.html_url,
        language: repo.language ?? null,
        stargazers_count: repo.stargazers_count ?? 0,
        fork: repo.fork ? 1 : 0,
        default_branch: repo.default_branch ?? null,
        created_at: repo.created_at ?? new Date().toISOString(),
        updated_at: repo.updated_at ?? new Date().toISOString(),
        is_pinned: pinnedRepoNames.has(repo.name) ? 1 : 0,
        owner_id: 0, // This will be set later when we know the stargazer's ID
        fetched_at: new Date().toISOString()
      }));

      repos.push(...filteredRepos);

      hasMorePages = response.data.length === 100;
      page++;

      // Add delay to respect rate limits
      await new Promise(resolve => setTimeout(resolve, 1000));
    }

    return repos;
  } catch (error) {
    console.error(`Error fetching repositories for ${username}:`, error);
    return [];
  }
} 