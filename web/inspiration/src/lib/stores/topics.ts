import { writable } from 'svelte/store';
import { browser } from '$app/environment';

// Load initial state from localStorage if available
const getStoredTopics = (): Set<string> => {
    if (!browser) return new Set();

    const stored = localStorage.getItem('topics');
    if (!stored) return new Set();

    return new Set(JSON.parse(stored));
};

// Create the store with initial state
const { subscribe, set, update } = writable<Set<string>>(getStoredTopics());

// Save state to localStorage
const saveToStorage = (topics: Set<string>) => {
    if (!browser) return;
    localStorage.setItem('topics', JSON.stringify([...topics]));
};

// Export the store with actions
export const topicsStore = {
    subscribe,
    set,
    update,
    toggle: (topic: string) => update(topics => {
        const newTopics = new Set(topics);
        topics.has(topic) ? newTopics.delete(topic) : newTopics.add(topic);
        saveToStorage(newTopics);
        return newTopics;
    }),

    reset: () => {
        const newTopics = new Set<string>();
        if (browser) localStorage.removeItem('topics');
        set(newTopics);
    }
};