<!--
  Purpose: Component for displaying special message cards in the feed
  Context: Used to show promotional content, follow prompts, or other special messages
-->

<script lang="ts">
	import { Twitter, Star, Github, ArrowRight, Sparkles } from 'lucide-svelte';
	import type { FeedProject } from '$lib/github/feed';
	import { onMount } from 'svelte';

	export let project: FeedProject;
	
	// Animation for the card
	let isVisible = false;
	
	onMount(() => {
		// Add a small delay before showing the animation
		setTimeout(() => {
			isVisible = true;
		}, 300);
	});
	
	// Check if this is a "get featured" message
	const isGetFeaturedMessage = project.name.toLowerCase().includes('featured') || 
	                            (project.description && project.description.toLowerCase().includes('featured'));
</script>

<!-- Special Message Card with enhanced styling -->
<div class="relative flex-1 flex flex-col transition-all duration-500 max-h-full {isVisible ? 'scale-100 opacity-100' : 'scale-95 opacity-90'}">
	<!-- Animated gradient border for featured message -->
	{#if isGetFeaturedMessage}
		<div class="absolute inset-0 rounded-xl bg-gradient-to-r from-yellow-400 via-purple-500 to-blue-500 opacity-40 blur-md animate-pulse"></div>
	{/if}
	
	<!-- Card content -->
	<div class="relative flex-1 flex flex-col z-10 rounded-xl {isGetFeaturedMessage ? 'bg-gray-900/90 backdrop-blur-md shadow-xl shadow-purple-500/20' : ''} p-2 overflow-hidden">
		<div class="flex h-full flex-col items-center justify-center space-y-6 text-center p-4">
			{#if isGetFeaturedMessage}
				<!-- Get Featured Message -->
				<div class="relative">
					<span class="absolute -inset-1 rounded-full bg-gradient-to-r from-yellow-400 via-purple-400 to-blue-400 opacity-75 blur-sm animate-pulse"></span>
					<span class="relative rounded-full bg-gradient-to-r from-yellow-400 via-purple-400 to-blue-400 px-4 py-2 text-sm font-bold text-black flex items-center gap-2">
						<Sparkles class="h-4 w-4" />
						Get Featured
						<Sparkles class="h-4 w-4" />
					</span>
				</div>
				
				<h1 class="font-serif text-3xl md:text-4xl text-transparent bg-clip-text bg-gradient-to-r from-yellow-300 via-purple-300 to-blue-300 font-bold">
					{project.name}
				</h1>
				
				<p class="max-w-lg font-serif text-lg text-gray-200">
					{project.description || ''}
				</p>
				
				<div class="w-full max-w-md bg-gray-800/50 backdrop-blur-sm rounded-xl p-5 border border-purple-500/20 shadow-lg shadow-purple-500/10">
					<h2 class="text-xl font-serif text-white mb-4">How to get featured:</h2>
					<ul class="text-left space-y-3">
						<li class="flex items-start gap-2">
							<span class="mt-1 text-yellow-400"><Star class="h-5 w-5" /></span>
							<div>
								<span class="flex items-center gap-2">
									<span>Star our repository on <a href="https://github.com/BlackShoreTech/gittok.dev" target="_blank" rel="noopener noreferrer" class="text-blue-400 hover:underline">GitHub</a></span>
									<span class="bg-yellow-500/20 text-xs px-2 py-0.5 rounded">This is all you need!</span>
								</span>
							</div>
						</li>
						<li class="flex items-start gap-2 opacity-80">
							<span class="mt-1 text-purple-400"><Github class="h-5 w-5" /></span>
							<span>We also appreciate if you create a pull request with your project details</span>
						</li>
						<li class="flex items-start gap-2 opacity-80">
							<span class="mt-1 text-blue-400"><Twitter class="h-5 w-5" /></span>
							<span>Consider sharing GitTok on social media to help us grow</span>
						</li>
					</ul>
				</div>
				
				<div class="flex flex-wrap gap-4 justify-center">
					<a
						href="https://github.com/BlackShoreTech/gittok.dev"
						target="_blank"
						rel="noopener noreferrer"
						class="flex items-center gap-2 rounded-full bg-gradient-to-r from-yellow-500/30 to-purple-500/30 hover:from-yellow-500/50 hover:to-purple-500/50 px-5 py-2.5 text-white backdrop-blur-sm transition-all duration-300 transform hover:scale-105 shadow-md shadow-yellow-500/20"
					>
						<Star class="h-5 w-5 text-yellow-300" />
						<span class="font-semibold">Star on GitHub</span>
					</a>
					
					<a
						href="https://twitter.com/intent/tweet?text=Check%20out%20GitTok%20-%20A%20TikTok-style%20interface%20for%20discovering%20amazing%20open%20source%20projects!%20https://gittok.dev"
						target="_blank"
						rel="noopener noreferrer"
						class="flex items-center gap-2 rounded-full bg-gradient-to-r from-blue-500/30 to-purple-500/30 hover:from-blue-500/50 hover:to-purple-500/50 px-5 py-2.5 text-white backdrop-blur-sm transition-all duration-300 transform hover:scale-105 shadow-md shadow-blue-500/20"
					>
						<Twitter class="h-5 w-5 text-blue-300" />
						<span class="font-semibold">Share on Twitter</span>
					</a>
				</div>
			{:else if project.html_url.includes('twitter.com')}
				<!-- Twitter Follow Message -->
				<h1 class="font-serif text-4xl text-white">{project.name}</h1>
				<p class="max-w-lg font-serif text-xl text-gray-200">{project.description || ''}</p>
				<a
					href={project.html_url}
					target="_blank"
					rel="noopener noreferrer"
					class="flex items-center gap-3 rounded-full bg-blue-400 px-6 py-3 text-white transition-colors duration-200 hover:bg-blue-500"
				>
					<Twitter class="h-5 w-5" />
					<span class="font-mono">Follow @{project.html_url.split('/').pop()}</span>
				</a>
			{:else}
				<!-- Generic Message -->
				<h1 class="font-serif text-4xl text-white">{project.name}</h1>
				<p class="max-w-lg font-serif text-xl text-gray-200">{project.description || ''}</p>
				<a
					href={project.html_url}
					target="_blank"
					rel="noopener noreferrer"
					class="flex items-center gap-3 rounded-full bg-blue-400 px-6 py-3 text-white transition-colors duration-200 hover:bg-blue-500"
				>
					<span class="font-mono">Learn More</span>
					<ArrowRight class="h-5 w-5" />
				</a>
			{/if}
			
			<p class="mt-4 text-sm text-gray-400">Keep scrolling for more awesome projects!</p>
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
</style> 