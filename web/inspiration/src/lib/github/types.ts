// Purpose: Type definitions for GitHub API responses and data structures
// Context: Used to type GitHub API interactions and data processing

import type { repositories } from '../db/schema';

export interface StargazerData {
  id: number;
  login: string;
  avatar_url: string;
  html_url: string;
  type: string;
  starred_at: string;
}

export interface GraphQLPinnedRepo {
  id: string;
  name: string;
  nameWithOwner: string;
  description: string | null;
  url: string;
  primaryLanguage: {
    name: string;
  } | null;
  stargazerCount: number;
  isFork: boolean;
  defaultBranchRef: {
    name: string;
  } | null;
  createdAt: string;
  updatedAt: string;
}

export interface GraphQLResponse {
  user: {
    pinnedItems: {
      nodes: GraphQLPinnedRepo[];
    };
  };
}

export type Repository = typeof repositories.$inferSelect; 