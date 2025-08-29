// Purpose: Test setup configuration for Vitest
// Context: Configures global test settings and environment

import { expect, afterEach } from 'vitest';
import { cleanup } from '@testing-library/svelte';
import '@testing-library/jest-dom';

// Extend Vitest's expect with Testing Library matchers
expect.extend({});

// Clean up after each test
afterEach(() => {
    cleanup();
}); 