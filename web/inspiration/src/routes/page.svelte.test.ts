import { describe, test, expect } from 'vitest';
import '@testing-library/jest-dom/vitest';
import { render, screen } from '@testing-library/svelte';
import Page from './feed/+page.svelte';

describe('/feed/+page.svelte', () => {
	test('should render project list container', () => {
		render(Page);
		expect(screen.getByRole('list')).toBeInTheDocument();
	});
});
