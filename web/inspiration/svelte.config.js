import adapter from '@sveltejs/adapter-static';
import { vitePreprocess } from '@sveltejs/vite-plugin-svelte';

/** @type {import('@sveltejs/kit').Config} */
const config = {
	// Consult https://svelte.dev/docs/kit/integrations
	// for more information about preprocessors
	preprocess: vitePreprocess(),

	kit: {
		adapter: adapter({
			// Enable SPA fallback
			fallback: '200.html',
			// Pages will be built into this directory
			pages: 'build',
			// Assets will be built into this directory
			assets: 'build',
			// Precompress files
			precompress: false,
			// Strict mode ensures all pages are prerenderable
			strict: true
		}),
		// Add paths configuration for GitHub Pages
		paths: {
			base: ''
		}
	}
};

export default config;
