// Purpose: Test suite for the SQLite database client
// Context: Tests database operations and queries using Vitest

import { describe, expect, test, beforeAll, afterAll, vi } from 'vitest';

// Mock $app/environment
vi.mock('$app/environment', () => ({
    dev: true,
    browser: true,
    building: false
}));

// Mock sql.js-httpvfs
vi.mock('sql.js-httpvfs', () => ({
    createDbWorker: vi.fn().mockResolvedValue({
        db: {
            query: vi.fn().mockImplementation((query: string) => {
                // Mock query responses based on the query
                if (query.includes('SELECT * FROM stargazers')) {
                    return [{ id: 1, login: 'user1', type: 'User' }];
                }
                if (query.includes('SELECT * FROM repositories')) {
                    return [{ id: 1, name: 'repo1', language: 'TypeScript' }];
                }
                if (query.includes('COUNT(*) as total')) {
                    return [{ total: 10, pinned: 2, avgStars: 100 }];
                }
                if (query.includes('SELECT language, COUNT(*) as count')) {
                    return [{ language: 'TypeScript', count: 5 }];
                }
                return [];
            })
        }
    })
}));

import { initDatabase, getStargazers, getRepositories, getRepositoryStats, searchRepositories, getRecentRepositories, closeDatabase } from './client';

describe('Database Client', () => {
    beforeAll(async () => {
        // Initialize the client with existing database
        await initDatabase('stars.db');
    });

    afterAll(async () => {
        await closeDatabase();
    });

    test('getStargazers returns data', async () => {
        const stargazers = await getStargazers({});
        expect(Array.isArray(stargazers)).toBe(true);
        expect(stargazers[0]).toHaveProperty('login');
    });

    test('getRepositories returns data', async () => {
        const repos = await getRepositories({});
        expect(Array.isArray(repos)).toBe(true);
        expect(repos[0]).toHaveProperty('name');
    });

    test('getRepositoryStats returns statistics', async () => {
        const stats = await getRepositoryStats();
        expect(stats).toEqual(expect.objectContaining({
            totalRepos: expect.any(Number),
            totalPinnedRepos: expect.any(Number),
            avgStarsPerRepo: expect.any(Number),
            topLanguages: expect.arrayContaining([
                expect.objectContaining({
                    language: expect.any(String),
                    count: expect.any(Number)
                })
            ])
        }));
    });

    test('searchRepositories can search repositories', async () => {
        const results = await searchRepositories('test');
        expect(Array.isArray(results)).toBe(true);
    });

    test('getRecentRepositories returns recent repositories', async () => {
        const recentRepos = await getRecentRepositories(7);
        expect(Array.isArray(recentRepos)).toBe(true);
    });
}); 