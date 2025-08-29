// Purpose: Database operations for storing GitHub stargazer and repository data
// Context: Separates database operations from API calls

import { db } from "../db";
import { eq, sql } from "drizzle-orm";
import { stargazers, repositories } from "../db/schema";
import type { StargazerData } from "./types";
import { fetchPinnedRepositories, fetchUserRepositories } from "./repositories";
import type { Octokit } from "@octokit/rest";

export async function upsertStargazer(stargazer: StargazerData) {
  const existingStargazer = await db.select()
    .from(stargazers)
    .where(eq(stargazers.login, stargazer.login))
    .get();

  if (existingStargazer) {
    await db.update(stargazers)
      .set(stargazer)
      .where(eq(stargazers.id, stargazer.id))
      .run();
  } else {
    await db.insert(stargazers).values(stargazer).run();
  }
}

export async function upsertRepository(repo: typeof repositories.$inferSelect) {
  const existingRepo = await db.select()
    .from(repositories)
    .where(eq(repositories.id, repo.id))
    .get();

  if (existingRepo) {
    await db.update(repositories)
      .set(repo)
      .where(eq(repositories.id, repo.id))
      .run();
  } else {
    await db.insert(repositories).values(repo).run();
  }
}

export async function processStargazerRepositories(octokit: Octokit, stargazer: StargazerData) {
  // Fetch and store repositories
//   const pinnedRepos = await fetchPinnedRepositories(octokit, stargazer.login);
  const userRepos = await fetchUserRepositories(octokit, stargazer.login);
  
  // Set owner_id for all repositories
  const reposWithOwner = userRepos.map(repo => ({
    ...repo,
    owner_id: stargazer.id
  }));

  // Upsert repositories
  for (const repo of reposWithOwner) {
    await upsertRepository(repo);
  }
}

export async function printRecentStatistics() {
  const oneDayAgo = new Date(Date.now() - 24 * 60 * 60 * 1000).toISOString();
  
  const recentStargazers = await db.select({ count: sql<number>`count(*)` })
    .from(stargazers)
    .where(sql`starred_at >= ${oneDayAgo}`)
    .get();

  const recentStargazerIds = await db.select({ id: stargazers.id })
    .from(stargazers)
    .where(sql`starred_at >= ${oneDayAgo}`);

  const recentRepos = await db.select({ count: sql<number>`count(*)` })
    .from(repositories)
    .where(sql`owner_id IN ${recentStargazerIds.map(s => s.id)}`)
    .get();

  const recentPinnedRepos = await db.select({ count: sql<number>`count(*)` })
    .from(repositories)
    .where(sql`owner_id IN ${recentStargazerIds.map(s => s.id)} AND is_pinned = 1`)
    .get();

  console.log(`
Statistics for the last 24 hours:
- New stargazers: ${recentStargazers?.count ?? 0}
- Their repositories: ${recentRepos?.count ?? 0}
- Their pinned repositories: ${recentPinnedRepos?.count ?? 0}
- Average repositories per new stargazer: ${((recentRepos?.count ?? 0) / (recentStargazers?.count ?? 1)).toFixed(2)}
  `);
} 