<!--
  Purpose: Component for displaying a regular GitHub repository card
  Context: Used in the feed page to show repository information in a TikTok-style format
-->

<script lang="ts">
	import { Star, GitFork, Share2 } from 'lucide-svelte';
	import type { FeedProject } from '$lib/github/feed';
	import { languageColors } from '$lib/github/feed';

	export let project: FeedProject;
	export let renderMarkdown: (content: string, repo: string) => string;
	export let formatNumber: (num: number) => string;
	export let shareProject: (project: FeedProject) => Promise<void>;

	// Function to scroll markdown content to top when loaded
	const scrollMarkdownToTop = (node: HTMLElement) => {
		node.scrollTop = 0;
		return {
			update() {
				node.scrollTop = 0;
			}
		};
	};

	// Get language color for a project
	const getLanguageColor = (language: string | null): string => {
		if (!language) return '#808080'; // Default gray for undefined languages
		return languageColors[language as keyof typeof languageColors] || '#808080';
	};
</script>

<!-- Regular Repository Card -->
<!-- Top: Repository Name and Description -->
<div class="flex-none space-y-2">
	<h1 class="flex items-center gap-2 font-serif text-2xl text-white md:text-3xl">
		{project.name}
		{#if project.language}
			<span class="h-3 w-3 rounded-full" style="background-color: {getLanguageColor(project.language)}"></span>
			<span class="font-mono text-sm text-gray-400">{project.language}</span>
		{/if}
	</h1>
	<p class="font-serif text-lg text-gray-200">{project.description}</p>
</div>

<!-- Middle: README Content -->
<div class="my-4 flex min-h-0 flex-1">
	<div
		class="markdown-content readme-container w-full overflow-y-clip overflow-x-hidden rounded-xl bg-gray-800/30 p-6 backdrop-blur-sm"
		use:scrollMarkdownToTop
	>
		{#if !project.readmeSnippet}
			<div class="animate-pulse text-gray-400">Loading README...</div>
		{:else}
			{@html renderMarkdown(
				project.readmeSnippet,
				`https://github.com/${project.full_name.split('/')[0]}/${project.name}/${project.default_branch}/`
			)}
		{/if}
	</div>
</div>

<!-- Right Side: Stats -->
<div class="absolute right-4 bottom-24 flex flex-col items-center gap-6">
	<!-- Stars -->
	<div class="flex flex-col items-center">
		<a
			href={project.stargazersUrl}
			target="_blank"
			rel="noopener noreferrer"
			class="rounded-full bg-gray-800/50 p-2 backdrop-blur-sm transition-colors hover:bg-gray-700/50"
			aria-label={`Star repository (${formatNumber(project.stargazers_count)} stars)`}
		>
			<Star class="h-8 w-8 text-yellow-400" />
		</a>
		<span class="mt-1 font-mono text-sm text-white">{formatNumber(project.stargazers_count)}</span>
	</div>

	<!-- Forks -->
	<div class="flex flex-col items-center">
		<a
			href={project.forksUrl}
			target="_blank"
			rel="noopener noreferrer"
			class="rounded-full bg-gray-800/50 p-2 backdrop-blur-sm transition-colors hover:bg-gray-700/50"
			aria-label={`Fork repository (${formatNumber(project.fork)} forks)`}
		>
			<GitFork class="h-8 w-8 text-gray-400" />
		</a>
		<span class="mt-1 font-mono text-sm text-white">{formatNumber(project.fork)}</span>
	</div>

	<!-- Share Button -->
	<div class="flex flex-col items-center">
		<button
			on:click={() => shareProject(project)}
			class="rounded-full bg-gray-800/50 p-2 backdrop-blur-sm transition-colors hover:bg-gray-700/50"
			aria-label="Share repository"
		>
			<Share2 class="h-8 w-8 text-purple-400" />
		</button>
		<span class="mt-1 font-mono text-sm text-white">Share</span>
	</div>
</div>

<!-- Bottom: Author Info -->
<div class="mb-4 flex flex-none items-center gap-3">
	<img
		src={project.avatar}
		alt={`${project.full_name.split('/')[0]}'s avatar`}
		class="h-10 w-10 rounded-full"
	/>
	<div>
		<h2 class="font-mono text-lg text-white">{project.full_name.split('/')[0]}</h2>
	</div>
	<a
		href={`https://github.com/${project.full_name.split('/')[0]}/${project.name}`}
		target="_blank"
		rel="noopener noreferrer"
		class="ml-auto rounded-lg bg-white/10 hover:bg-white/20 px-4 py-2 font-mono text-white transition-colors duration-200"
	>
		View
	</a>
</div> 