<!--
  Purpose: Component for displaying a promoted/featured GitHub repository card
  Context: Used in the feed page to highlight special repositories with enhanced styling
-->

<script lang="ts">
	import { Star, GitFork, Share2, Sparkles } from 'lucide-svelte';
	import type { FeedProject } from '$lib/github/feed';
	import { languageColors } from '$lib/github/feed';
	import { onMount } from 'svelte';

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
	
	// Animation for the card
	let isVisible = false;
	
	onMount(() => {
		// Add a small delay before showing the animation
		setTimeout(() => {
			isVisible = true;
		}, 300);
	});
</script>

<!-- Promoted Repository Card with enhanced styling -->
<!-- Outer container with border glow effect - constrained to fit viewport -->
<div class="relative flex-1 flex flex-col rounded-xl transition-all duration-500 max-h-full {isVisible ? 'scale-100 opacity-100' : 'scale-95 opacity-90'}">
	<!-- Animated gradient border -->
	<div class="absolute inset-0 rounded-xl bg-gradient-to-r from-yellow-400 via-purple-500 to-blue-500 opacity-50 blur-md animate-pulse"></div>
	
	<!-- Card content with inner shadow -->
	<div class="relative flex-1 flex flex-col z-10 rounded-xl bg-gray-900/90 backdrop-blur-md shadow-xl shadow-purple-500/20 p-2 overflow-hidden">
		<!-- Top: Repository Name and Description with Featured Badge - more compact -->
		<div class="flex-none space-y-1 p-3">
			<!-- Promoted Repository Badge and Star CTA in a more compact layout -->
			<div class="mb-3 flex flex-wrap items-center justify-between gap-2">
				<div class="flex items-center">
					<div class="relative">
						<span class="absolute -inset-1 rounded-full bg-gradient-to-r from-yellow-400 via-purple-400 to-blue-400 opacity-75 blur-sm animate-pulse"></span>
						<span class="relative rounded-full bg-gradient-to-r from-yellow-400 via-purple-400 to-blue-400 px-3 py-1 text-sm font-bold text-black flex items-center gap-1">
							<Sparkles class="h-3 w-3" />
							Featured
						</span>
					</div>
				</div>
				<a
					href="https://github.com/BlackShoreTech/gittok.dev"
					target="_blank"
					rel="noopener noreferrer"
					class="flex items-center gap-1 rounded-full bg-gradient-to-r from-yellow-500/30 to-purple-500/30 hover:from-yellow-500/50 hover:to-purple-500/50 px-3 py-1.5 text-sm text-white backdrop-blur-sm transition-all duration-300 transform hover:scale-105 shadow-md shadow-yellow-500/20"
				>
					<Star class="h-4 w-4 text-yellow-300 animate-pulse" />
					<span class="font-semibold">Star us to get featured!</span>
				</a>
			</div>
			
			<h1 class="flex items-center gap-2 font-serif text-2xl text-white md:text-3xl text-transparent bg-clip-text bg-gradient-to-r from-yellow-300 via-purple-300 to-blue-300 font-bold">
				{project.name}
				{#if project.language}
					<span class="h-3 w-3 rounded-full" style="background-color: {getLanguageColor(project.language)}"></span>
					<span class="font-mono text-xs text-gray-300">{project.language}</span>
				{/if}
			</h1>
			<p class="font-serif text-base text-gray-100 line-clamp-2">{project.description}</p>
		</div>

		<!-- Middle: README Content with enhanced styling - flex-1 to take available space -->
		<div class="flex min-h-0 flex-1 px-3">
			<div
				class="markdown-content readme-container w-full overflow-y-clip overflow-x-hidden rounded-xl bg-gradient-to-br from-gray-800/40 via-gray-800/50 to-gray-800/40 shadow-lg shadow-purple-500/20 p-4 backdrop-blur-sm border border-purple-500/20"
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

		<!-- Right Side: Stats with enhanced styling - positioned more compactly -->
		<div class="absolute right-3 bottom-20 flex flex-col items-center gap-4 z-20">
			<!-- Stars -->
			<div class="flex flex-col items-center">
				<a
					href={project.stargazersUrl}
					target="_blank"
					rel="noopener noreferrer"
					class="rounded-full bg-gradient-to-br from-gray-800/70 to-purple-800/70 shadow-lg shadow-yellow-500/30 p-2 backdrop-blur-sm transition-all duration-300 hover:bg-gray-700/70 hover:scale-110"
					aria-label={`Star repository (${formatNumber(project.stargazers_count)} stars)`}
				>
					<Star class="h-6 w-6 text-yellow-300" />
				</a>
				<span class="mt-1 font-mono text-xs text-white font-bold">{formatNumber(project.stargazers_count)}</span>
			</div>

			<!-- Forks -->
			<div class="flex flex-col items-center">
				<a
					href={project.forksUrl}
					target="_blank"
					rel="noopener noreferrer"
					class="rounded-full bg-gradient-to-br from-gray-800/70 to-purple-800/70 shadow-lg shadow-blue-500/30 p-2 backdrop-blur-sm transition-all duration-300 hover:bg-gray-700/70 hover:scale-110"
					aria-label={`Fork repository (${formatNumber(project.fork)} forks)`}
				>
					<GitFork class="h-6 w-6 text-blue-300" />
				</a>
				<span class="mt-1 font-mono text-xs text-white font-bold">{formatNumber(project.fork)}</span>
			</div>

			<!-- Share Button -->
			<div class="flex flex-col items-center">
				<button
					on:click={() => shareProject(project)}
					class="rounded-full bg-gradient-to-br from-gray-800/70 to-purple-800/70 shadow-lg shadow-purple-500/30 p-2 backdrop-blur-sm transition-all duration-300 hover:bg-gray-700/70 hover:scale-110"
					aria-label="Share repository"
				>
					<Share2 class="h-6 w-6 text-purple-300" />
				</button>
				<span class="mt-1 font-mono text-xs text-white font-bold">Share</span>
			</div>
		</div>

		<!-- Bottom: Author Info with enhanced styling - more compact -->
		<div class="mt-2 mb-2 flex flex-none items-center gap-2 px-3">
			<div class="relative">
				<div class="absolute -inset-1 rounded-full bg-gradient-to-r from-yellow-400 to-purple-400 opacity-75 blur-sm"></div>
				<img
					src={project.avatar}
					alt={`${project.full_name.split('/')[0]}'s avatar`}
					class="relative h-8 w-8 rounded-full ring-1 ring-purple-400/50 ring-offset-1 ring-offset-black"
				/>
			</div>
			<div>
				<h2 class="font-mono text-sm text-white font-semibold">{project.full_name.split('/')[0]}</h2>
			</div>
			<a
				href={`https://github.com/${project.full_name.split('/')[0]}/${project.name}`}
				target="_blank"
				rel="noopener noreferrer"
				class="ml-auto rounded-lg bg-gradient-to-r from-purple-500/30 to-blue-500/30 hover:from-purple-500/50 hover:to-blue-500/50 px-4 py-1.5 font-mono text-sm text-white transition-all duration-300 transform hover:scale-105 shadow-md shadow-purple-500/20"
			>
				View Project
			</a>
		</div>
	</div>
</div>

<style>
	/* Add keyframes for the gradient animation */
	@keyframes gradient {
		0% {
			background-position: 0% 50%;
		}
		50% {
			background-position: 100% 50%;
		}
		100% {
			background-position: 0% 50%;
		}
	}
	
	/* Ensure the readme container doesn't grow too large */
	.readme-container {
		max-height: calc(100vh - 250px);
	}
</style> 