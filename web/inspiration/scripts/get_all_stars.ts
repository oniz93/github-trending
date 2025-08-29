// Purpose: Script to fetch recent GitHub stargazers and their repositories
// Context: Used to maintain a list of users who have starred this repository in the last 24 hours

import { Octokit } from "@octokit/rest";
import { fetchRecentStargazers } from "../src/lib/github/stargazers";

async function main() {
  const octokit = new Octokit({
    auth: process.env.GITHUB_TOKEN
  });

  await fetchRecentStargazers(octokit, "BlackShoreTech", "gittok.dev");
}

main().catch(error => {
  console.error('Error running script:', error);
  process.exit(1);
});