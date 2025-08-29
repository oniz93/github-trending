// Purpose: Database schema for stargazers and repositories
// Context: Used to store GitHub stargazers and their repositories in a relational structure

import { sqliteTable, text, integer } from "drizzle-orm/sqlite-core";
import { sql } from "drizzle-orm";

export const stargazers = sqliteTable('stargazers', {
  id: integer('id').primaryKey(),
  login: text('login').notNull(),
  avatar_url: text('avatar_url').notNull(),
  html_url: text('html_url').notNull(),
  type: text('type').notNull(),
  starred_at: text('starred_at'),
  created_at: text('created_at').notNull().default(sql`CURRENT_TIMESTAMP`),
  updated_at: text('updated_at').notNull().default(sql`CURRENT_TIMESTAMP`),
});

export const repositories = sqliteTable('repositories', {
  id: text('id').primaryKey(),
  owner_id: integer('owner_id').notNull().references(() => stargazers.id),
  name: text('name').notNull(),
  full_name: text('full_name').notNull(),
  description: text('description'),
  html_url: text('html_url').notNull(),
  language: text('language'),
  stargazers_count: integer('stargazers_count').notNull().default(0),
  fork: integer('fork').notNull().default(0), // SQLite doesn't have boolean, using 0/1
  is_pinned: integer('is_pinned').notNull().default(0),
  default_branch: text('default_branch'),
  created_at: text('created_at').notNull(),
  updated_at: text('updated_at').notNull(),
  fetched_at: text('fetched_at').notNull().default(sql`CURRENT_TIMESTAMP`),
}); 