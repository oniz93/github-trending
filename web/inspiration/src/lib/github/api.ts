// Purpose: GitHub API operations for fetching stargazer and repository data
// Context: Separates API calls from database operations

import { Octokit } from "@octokit/rest";
import type { StargazerData } from "./types";

export function isWithinLastDays(dateStr: string, days: number): boolean {
  const date = new Date(dateStr);
  const now = new Date();
  const daysAgo = new Date(now.getTime() - (days * 24 * 60 * 60 * 1000));
  return date >= daysAgo;
}

export async function fetchStargazerPage(
  octokit: Octokit,
  owner: string,
  repo: string,
  page: number
): Promise<{ stargazers: StargazerData[]; hasMore: boolean }> {
  const response = await octokit.rest.activity.listStargazersForRepo({
    owner,
    repo,
    sort: "created",
    direction: "desc",
    per_page: 100,
    page: page,
    headers: {
      Accept: 'application/vnd.github.v3.star+json'
    }
  }
  );

  const stargazers: StargazerData[] = [];
  let foundOldStar = false;

  for (const stargazerData of response.data) {
    const user = 'user' in stargazerData ? stargazerData.user : stargazerData;
    const starredAt = 'starred_at' in stargazerData ? stargazerData.starred_at : null;
    
    if (!user || !starredAt) continue;

    // Check if this star is within the last 24 hours
    if (!isWithinLastDays(starredAt, 10)) {
      foundOldStar = true;
      break;
    }

    stargazers.push({
      id: user.id,
      login: user.login,
      avatar_url: user.avatar_url,
      html_url: user.html_url,
      type: user.type ?? 'User',
      starred_at: starredAt,
    });
  }

  console.log(`Found ${stargazers.length} stargazers for ${owner}/${repo}`);

  const hasMore = response.data.length === 100 && !foundOldStar;
  return { stargazers, hasMore };
}

export async function fetchAllRecentStargazers(
  octokit: Octokit,
  owner: string,
  repo: string
): Promise<StargazerData[]> {
  let page = 1;
  let hasMorePages = true;
  const allStargazers: StargazerData[] = [];

  while (hasMorePages) {
    console.log(`Fetching stargazers page ${page}...`);
    const { stargazers, hasMore } = await fetchStargazerPage(octokit, owner, repo, page);
    
    allStargazers.push(...stargazers);
    hasMorePages = hasMore;
    page++;

    // GitHub API has rate limiting, so let's add a small delay
    await new Promise(resolve => setTimeout(resolve, 1000));
  }

  return allStargazers;
} 