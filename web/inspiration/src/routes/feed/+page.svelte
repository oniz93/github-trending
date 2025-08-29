<!--
  Purpose: Homepage with TikTok-style vertical scrolling for GitHub projects
  Context: Main landing page that showcases interesting GitHub repositories in a scrollable format
-->

<script lang="ts">
	import { onMount } from 'svelte';
	import { marked } from 'marked';
	import DOMPurify from 'isomorphic-dompurify';
	import { topicsStore } from '$lib/stores/topics';
	import { topics as allTopics } from '$lib/all_topics';
	import { Octokit } from '@octokit/rest';
	import { baseUrl } from 'marked-base-url';
	import { markedEmoji } from 'marked-emoji';
	import { Globe, Settings } from 'lucide-svelte';
	import type { FeedProject } from '$lib/github/feed';
	import { fetchProject, fetchReadme, getRandomSearchQuery, languageColors, searchRepositories } from '$lib/github/feed';
	
	// Import our new components
	import RepoCard from '$lib/components/RepoCard.svelte';
	import PromotedRepoCard from '$lib/components/PromotedRepoCard.svelte';
	import SpecialMessageCard from '$lib/components/SpecialMessageCard.svelte';

	let projects: FeedProject[] = [];
	let viewedIndices = new Set<number>();
	let isLoading = false;
	let hasMore = true;
	let hasShownFollowMessage = false;
	let hasShownFeaturedMessage = false;
	let featuredRepos: FeedProject[] = [];

	let seenQueries: Record<string, { current_page: number; total_projects: number }> = {};

	// Function to load more projects and randomly merge them after the current index
	const loadMoreProjects = async (index: number) => {
		if (isLoading || !hasMore) return;

		isLoading = true;
		try {
			const octokit = new Octokit();
			const query = getRandomSearchQuery(
				$topicsStore.size > 0 ? [...$topicsStore] : allTopics,
				seenQueries
			);
			const queryKey = query.get('q');

			if (!queryKey) {
				throw new Error('No query key found');
			}

			const newProjects = await searchRepositories(octokit, query.toString());

			// Split existing projects into before and after index
			const beforeProjects = projects.slice(0, index + 1);
			const afterProjects = projects.slice(index + 1);

			// Randomly merge new projects with remaining projects after index
			const mergedProjects = [];
			const combined = [...afterProjects, ...newProjects];

			while (combined.length > 0) {
				const randomIndex = Math.floor(Math.random() * combined.length);
				mergedProjects.push(combined.splice(randomIndex, 1)[0]);
			}

			// Combine all parts
			projects = [...beforeProjects, ...mergedProjects];
		} catch (error) {
			console.error('Error loading more projects:', error);
		} finally {
			isLoading = false;
		}
	};

	function seturlparams(project: FeedProject) {
		const url = new URL(window.location.href);
		url.searchParams.set('project', project.name);
		url.searchParams.set('author', project.full_name.split('/')[0]);
		window.history.replaceState({}, '', url.toString());
	}

	// Function to load featured repositories from JSON file
	const loadFeaturedRepos = async (): Promise<FeedProject[]> => {
		try {
			const response = await fetch('/data/featured_repos.json');
			if (!response.ok) {
				throw new Error(`Failed to fetch featured repos: ${response.status}`);
			}
			
			const data = await response.json();
			
			// Transform the data to match FeedProject type
			return Promise.all(data.map(async (repo: any) => {
				// Fetch README for each featured repo
				let readmeContent = '';
				try {
					readmeContent = await fetchReadme(
						repo.full_name.split('/')[0],
						repo.name,
						repo.default_branch
					);
				} catch (error) {
					console.error(`Error fetching README for ${repo.full_name}:`, error);
				}
				
				return {
					id: '-2',
					name: repo.name,
					full_name: repo.full_name,
					description: repo.description || '',
					html_url: repo.html_url,
					language: repo.language,
					stargazers_count: repo.stargazers_count,
					fork: repo.fork,
					created_at: repo.created_at,
					updated_at: repo.updated_at,
					is_pinned: repo.is_pinned,
					owner_id: repo.owner_id,
					fetched_at: repo.fetched_at,
					readmeSnippet: readmeContent,
					avatar: repo.avatar_url,
					stargazersUrl: `${repo.html_url}/stargazers`,
					forksUrl: `${repo.html_url}/fork`,
					default_branch: repo.default_branch
				};
			}));
		} catch (error) {
			console.error('Error loading featured repos:', error);
			return getDefaultFeaturedRepos();
		}
	};

	// Function to get default featured repos when JSON loading fails
	const getDefaultFeaturedRepos = (): FeedProject[] => {
		return [
			{
				id: '-2',
				name: 'gittok.dev',
				full_name: 'BlackShoreTech/gittok.dev',
				description: 'A TikTok-style interface for discovering amazing open source projects. Built with SvelteKit and Tailwind CSS.',
				html_url: 'https://github.com/BlackShoreTech/gittok.dev',
				language: 'Svelte',
				stargazers_count: 1337,
				fork: 0,
				created_at: new Date().toISOString(),
				updated_at: new Date().toISOString(),
				is_pinned: 1,
				owner_id: 0,
				fetched_at: new Date().toISOString(),
				readmeSnippet: '# GitTok\n\nGitTok is an innovative way to discover open source projects. With its TikTok-inspired interface, you can effortlessly scroll through carefully curated GitHub repositories.\n\n## Features\n\n- Vertical scrolling interface\n- Real-time README previews\n- GitHub statistics integration\n- Beautiful dark mode design\n- Mobile-first responsive layout',
				avatar: 'https://avatars.githubusercontent.com/u/583231?v=4',
				stargazersUrl: 'https://github.com/gittok/gittok/stargazers',
				forksUrl: 'https://github.com/gittok/gittok/fork',
				default_branch: 'main'
			},
			{
				id: '-3',
				name: 'Sponsor GitTok',
				full_name: 'sponsor/gittok',
				description: 'Want to promote your open source project here? Reach thousands of developers daily!',
				html_url: 'mailto:sponsor@gittok.dev',
				language: null,
				stargazers_count: 0,
				fork: 0,
				created_at: new Date().toISOString(),
				updated_at: new Date().toISOString(),
				is_pinned: 1,
				owner_id: 0,
				fetched_at: new Date().toISOString(),
				readmeSnippet: '# Promote Your Project\n\nReach thousands of developers who are actively discovering new open source projects.\n\n## Why Sponsor?\n\n- Increase project visibility\n- Reach active developers\n- Support open source\n- Boost community engagement\n\nContact us at sponsor@gittok.dev',
				avatar: 'https://avatars.githubusercontent.com/u/583231?v=4',
				stargazersUrl: 'mailto:sponsor@gittok.dev',
				forksUrl: 'mailto:sponsor@gittok.dev',
				default_branch: 'main'
			}
		];
	};

	// Simplified function to insert a featured repo at a specific position in the feed
	const insertFeaturedRepo = (index: number) => {
		// Only insert if we're at a position divisible by 10 (every 10th item)
		if ((index + 1) % 10 === 0 && featuredRepos.length > 0) {
			const position = Math.floor(index / 10);
			const featuredRepo = featuredRepos[position % featuredRepos.length];
			
			// Insert the featured repo after the current index
			projects = [
				...projects.slice(0, index + 1),
				featuredRepo,
				...projects.slice(index + 1)
			];
		}
	};

	// Update the intersection observer to handle pagination and featured repos
	const observeElement = (element: HTMLElement, index: number) => {
		const observer = new IntersectionObserver(
			async (entries) => {
				entries.forEach(async (entry) => {
					if (entry.isIntersecting) {
						seturlparams(projects[index]);
						console.log('isIntersecting', index);
						viewedIndices.add(index);
						viewedIndices = viewedIndices; // trigger reactivity

						// Show follow message after viewing 10 items
						if (viewedIndices.size === 10 && !hasShownFollowMessage) {
							hasShownFollowMessage = true;
							// Insert the follow message card after the current item
							const followMessageProject: FeedProject = {
								id: '-1',
								name: 'Enjoying GitTok?',
								full_name: '@brsc2909/gittok',
								description:
									"If you're finding this useful, consider following me on Twitter for more cool projects!",
								html_url: 'https://twitter.com/brsc2909',
								language: null,
								stargazers_count: 0,
								fork: 0,
								created_at: new Date().toISOString(),
								updated_at: new Date().toISOString(),
								is_pinned: 1,
								owner_id: 0,
								fetched_at: new Date().toISOString(),
								readmeSnippet: '',
								avatar: 'https://avatars.githubusercontent.com/u/1?v=4',
								stargazersUrl: '',
								forksUrl: '',
								default_branch: ''
							};
							projects = [
								...projects.slice(0, index + 1),
								followMessageProject,
								...projects.slice(index + 1)
							];
						}
						
						// Show "Get Featured" message after viewing 5 items
						if (viewedIndices.size === 5 && !hasShownFeaturedMessage) {
							hasShownFeaturedMessage = true;
							// Insert the get featured message card after the current item
							const getFeaturedMessageProject: FeedProject = {
								id: '-4',
								name: 'Get Your Project Featured',
								full_name: 'BlackShoreTech/gittok.dev',
								description:
									"Want your open source project to be featured on GitTok? Here's how to get more visibility for your work!",
								html_url: 'https://github.com/BlackShoreTech/gittok.dev',
								language: null,
								stargazers_count: 0,
								fork: 0,
								created_at: new Date().toISOString(),
								updated_at: new Date().toISOString(),
								is_pinned: 1,
								owner_id: 0,
								fetched_at: new Date().toISOString(),
								readmeSnippet: '',
								avatar: 'https://avatars.githubusercontent.com/u/583231?v=4',
								stargazersUrl: 'https://github.com/BlackShoreTech/gittok.dev',
								forksUrl: 'https://github.com/BlackShoreTech/gittok.dev/fork',
								default_branch: 'main'
							};
							projects = [
								...projects.slice(0, index + 1),
								getFeaturedMessageProject,
								...projects.slice(index + 1)
							];
						}

						// Insert featured repos at intervals
						insertFeaturedRepo(index);

						// Fetch README when item comes into view
						// Fetch current and next 3 READMEs when an item comes into view
						if (projects[index] && !projects[index].readmeSnippet) {
							// Fetch current README
							const readme = await fetchReadme(
								projects[index].full_name.split('/')[0],
								projects[index].name,
								projects[index].default_branch
							);
							projects[index].readmeSnippet = readme;

							projects = projects; // trigger reactivity
						}

						// Fetch next 2 READMEs
						for (let i = 1; i <= 2; i++) {
							const nextIndex = index + i;
							if (projects[nextIndex] && !projects[nextIndex].readmeSnippet) {
								const nextReadme = await fetchReadme(
									projects[nextIndex].full_name.split('/')[0],
									projects[nextIndex].name,
									projects[nextIndex].default_branch
								);
								projects[nextIndex].readmeSnippet = nextReadme;
							}
						}

						// If we're getting close to the end, load more projects
						if (index >= projects.length - 20) {
							loadMoreProjects(index);
						}
					}
				});
			},
			{ threshold: 0.5 }
		);

		observer.observe(element);
		return {
			destroy() {
				observer.disconnect();
			}
		};
	};

	onMount(async () => {
		const octokit = new Octokit();
		// Get all the emojis available to use on GitHub.
		const res = await octokit.rest.emojis.get();

		const emojis = res.data;

		marked.use(
			markedEmoji({
				emojis,
				renderer: (token) =>
					`<img alt="${token.name}" src="${token.emoji}" class="marked-emoji-img">`
			})
		);

		// Load featured repos from JSON file
		featuredRepos = await loadFeaturedRepos();
		
		// Load initial projects
		const url = new URL(window.location.href);
		const project = url.searchParams.get('project');
		const author = url.searchParams.get('author');
		const initialProjects = [];
		if (project && author) {
			const initialProject = await fetchProject(author, project);
			initialProjects.push(initialProject);
		}

		const query = getRandomSearchQuery(
			$topicsStore.size > 0 ? [...$topicsStore] : allTopics,
			seenQueries
		);
		initialProjects.push(...(await searchRepositories(octokit, query.toString())));
		
		projects = initialProjects;
	});

	// Format numbers to human readable format (e.g., 73.5k)
	const formatNumber = (num: number): string => {
		if (num >= 1000) {
			return (num / 1000).toFixed(1) + 'k';
		}
		return num.toString();
	};

	// Create a function to safely render markdown
	const renderMarkdown = (content: string, repo: string): string => {
		// Force marked to return a string synchronously
		marked.use({
			gfm: true
		});
		marked.use(baseUrl(repo));
		const rawHtml = marked.parse(content, { async: false }) as string;
		return DOMPurify.sanitize(rawHtml, {
			USE_PROFILES: { html: true },
			ALLOWED_TAGS: [
				'h1',
				'h2',
				'h3',
				'h4',
				'h5',
				'h6',
				'p',
				'a',
				'ul',
				'ol',
				'li',
				'code',
				'pre',
				'strong',
				'em',
				'blockquote',
				'table',
				'thead',
				'tbody',
				'tr',
				'th',
				'td',
				'br',
				'hr'
			],
			ALLOWED_ATTR: ['href', 'target', 'rel', 'class']
		});
	};

	// Add share functionality
	const shareProject = async (project: FeedProject) => {
		const shareUrl = `https://gittok.dev/feed?project=${project.name}&author=${project.full_name.split('/')[0]}`;
		const shareText = `Check out ${project.name} on gittok.dev`;

		if (navigator.share) {
			try {
				await navigator.share({
					title: project.name,
					text: shareText,
					url: shareUrl
				});
			} catch (err) {
				if (err instanceof Error && err.name !== 'AbortError') {
					console.error('Error sharing:', err);
					// Fallback to clipboard
					await navigator.clipboard.writeText(shareUrl);
					alert('Link copied to clipboard!');
				}
			}
		} else {
			// Fallback for browsers that don't support Web Share API
			await navigator.clipboard.writeText(shareUrl);
			alert('Link copied to clipboard!');
		}
	};
</script>

<div class="h-screen w-full snap-y snap-mandatory overflow-y-scroll svelte-16xb542" role="list" aria-label="GitHub Projects">
	<!-- Add Settings and About Links -->
	<div class="fixed top-4 right-4 z-50 flex gap-4">
		<a
			href="/about"
			class="rounded-full bg-gray-800/50 p-2 backdrop-blur-sm transition-colors hover:bg-gray-700/50"
			aria-label="About"
		>
			<Globe class="h-6 w-6 text-gray-400" />
		</a>
		<a
			href="/setup"
			class="rounded-full bg-gray-800/50 p-2 backdrop-blur-sm transition-colors hover:bg-gray-700/50"
			aria-label="Settings"
		>
			<Settings class="h-6 w-6 text-gray-400" />
		</a>
	</div>

	{#each projects as project, index}
		<div
			class="project-container relative flex h-screen w-full snap-start items-center justify-center bg-gradient-to-b from-gray-900 to-black"
			use:observeElement={index}
		>
			<!-- Main Content Container -->
			<div class="mx-auto flex h-full w-full max-w-3xl flex-col p-6">
				{#if project.id === '-1'}
					<!-- Special Follow Message Card -->
					<SpecialMessageCard {project} />
				{:else if project.id === '-2'}
					<!-- Promoted Repository Card -->
					<PromotedRepoCard 
						{project} 
						{renderMarkdown} 
						{formatNumber} 
						{shareProject} 
					/>
				{:else if project.id === '-4'}
					<!-- Get Featured Message Card -->
					<SpecialMessageCard {project} />
				{:else}
					<!-- Regular Repository Card -->
					<RepoCard 
						{project} 
						{renderMarkdown} 
						{formatNumber} 
						{shareProject} 
					/>
				{/if}
			</div>
		</div>
	{/each}

	{#if isLoading}
		<div
			class="flex h-screen w-full items-center justify-center bg-gradient-to-b from-gray-900 to-black"
		>
			<div class="font-mono text-white">Loading more repositories...</div>
		</div>
	{/if}

	{#if !hasMore && !isLoading}
		<div
			class="flex h-screen w-full items-center justify-center bg-gradient-to-b from-gray-900 to-black"
		>
			<div class="font-mono text-white">You've reached the end!</div>
		</div>
	{/if}
</div>

<style>
	/* Hide scrollbar for Chrome, Safari and Opera */
	.snap-y::-webkit-scrollbar {
		display: none;
	}

	/* Hide scrollbar for IE, Edge and Firefox */
	.snap-y {
		-ms-overflow-style: none; /* IE and Edge */
		scrollbar-width: none; /* Firefox */
	}
	/* Style markdown content */
	:global(.prose) {
		color: rgb(229 231 235); /* gray-200 */
	}

	:global(.prose a) {
		color: rgb(96 165 250); /* blue-400 */
	}

	:global(.prose code) {
		color: rgb(249 168 212); /* pink-300 */
		background: rgb(31 41 55); /* gray-800 */
		padding: 0.2em 0.4em;
		border-radius: 0.25em;
	}

	:global(.prose pre) {
		background: rgb(17 24 39); /* gray-900 */
		padding: 1em;
		border-radius: 0.5em;
	}
</style>
