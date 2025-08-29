<script lang="ts">
	import { topics } from '$lib/topics'
	import { topicsStore } from '$lib/stores/topics';
	import { goto } from '$app/navigation';
	import { writable } from 'svelte/store';

	let newTopic = '';
	let showSuggestions = false;
	let filteredTopics: string[] = [];

	// Create a local store for expanded state
	const expanded = writable(new Set<string>());

	// Flatten the topics object to get all possible topics
	const getAllTopics = (obj: any): string[] => {
		let result: string[] = [];

		const traverse = (o: any) => {
			if (Array.isArray(o)) {
				result.push(...o);
			} else if (typeof o === 'object') {
				Object.entries(o).forEach(([key, value]) => {
					result.push(key);
					traverse(value);
				});
			}
		};

		traverse(topics);
		return [...new Set(result)]; // Remove duplicates
	};

	const allTopics = getAllTopics(topics);

	// Filter topics based on input
	function filterTopics(input: string) {
		if (!input.trim()) {
			filteredTopics = [];
			return;
		}

		const searchTerm = input.toLowerCase().trim();
		filteredTopics = allTopics
			.filter(topic =>
				topic.toLowerCase().includes(searchTerm) &&
				!$topicsStore.has(topic)
			)
			.slice(0, 5); // Limit to 5 suggestions
	}

	// Update suggestions when input changes
	$: {
		filterTopics(newTopic);
		showSuggestions = filteredTopics.length > 0 && newTopic.trim().length > 0;
	}

	function toggleExpanded(topic: string) {
		expanded.update(state => {
			const newState = new Set(state);
			state.has(topic) ? newState.delete(topic) : newState.add(topic);
			return newState;
		});
	}

	function selectTopic(topic: string) {
		topicsStore.toggle(topic);
		newTopic = '';
		showSuggestions = false;
	}

	function addCustomTopic(e: Event) {
		e.preventDefault();
		if (newTopic.trim()) {
			topicsStore.toggle(newTopic.trim().toLowerCase());
			newTopic = '';
			showSuggestions = false;
		}
	}

	$: canContinue = $topicsStore.size > 0;
</script>

<!-- Replace the existing Add custom topic section with this: -->
<div class="p-6 pb-0 relative">
	<form
		onsubmit={addCustomTopic}
		class="flex gap-2 items-center"
	>
		<div class="relative flex-1">
			<input
				type="text"
				bind:value={newTopic}
					onfocus={() => filterTopics(newTopic)}
				placeholder="Search or add your own topic..."
				class="w-full px-4 py-2 rounded-full backdrop-blur-sm
					   focus:outline-none focus:ring-2 focus:ring-blue-400
					   border border-gray-800/50
					   bg-gray-800/30 text-gray-200 text-sm
					   placeholder:text-gray-500"
			/>

			{#if showSuggestions}
				<div class="absolute top-full left-0 right-0 mt-2
						   bg-gray-800/95 backdrop-blur-sm border border-gray-700/50
						   rounded-lg overflow-hidden z-50">
					{#each filteredTopics as topic}
						<button
							type="button"
							onclick={() => selectTopic(topic)}
							class="w-full px-4 py-2 text-left text-gray-200 text-sm
								   hover:bg-gray-700/50 transition-colors duration-150
								   flex items-center gap-2"
						>
							<span class="text-blue-400">+</span>
							<span class="capitalize font-serif">{topic.replace(/_/g, ' ')}</span>
						</button>
					{/each}
				</div>
			{/if}
		</div>

		<button
			type="submit"
			class="px-4 py-2 rounded-full backdrop-blur-sm
				   focus:outline-none focus:ring-2 focus:ring-blue-400
				   border border-gray-800/50 hover:border-blue-400/50
				   bg-gray-800/30 hover:bg-gray-700/40 text-sm
				   text-gray-200"
		>
			Add
		</button>
	</form>
</div>

<!-- Remove the separate custom topics section and show all selected topics in one place -->
{#if $topicsStore.size > 0}
	<div class="p-6">
		<h3 class="text-sm font-mono text-gray-400 mb-3">Selected Topics</h3>
		<div class="grid grid-cols-1 sm:grid-cols-2 md:grid-cols-3 lg:grid-cols-4 gap-4">
			{#each [...$topicsStore] as topic}
				<div
					class="group w-full px-4 py-2 rounded-full backdrop-blur-sm transition-all duration-300
						   flex items-center justify-between cursor-pointer
						   focus:outline-none focus:ring-2 focus:ring-blue-400
						   border border-gray-800/50 hover:border-blue-400/50
						   bg-gray-800/30 hover:bg-gray-700/40 text-sm
						   bg-blue-900/30 border-blue-400/50"
					onclick={() => topicsStore.toggle(topic)}
					onkeydown={(e) => e.key === 'Enter' && topicsStore.toggle(topic)}
					role="button"
					tabindex="0"
				>
					<div class="flex items-center gap-2">
						<span class="text-yellow-400 text-xs">★</span>
						<span class="capitalize font-serif text-gray-200">{topic.replace(/-/g, ' ')}</span>
					</div>
					<button
						class="opacity-0 group-hover:opacity-100 text-gray-500 hover:text-red-400 transition-opacity"
						onclick={(e) => {
							e.stopPropagation();
							topicsStore.toggle(topic);
						}}
					>
						×
					</button>
				</div>
			{/each}
		</div>
	</div>
{/if}

<!-- Predefined topics grid -->
<div class="grid grid-cols-1 sm:grid-cols-2 md:grid-cols-3 lg:grid-cols-4 gap-4 p-6">
	{#each Object.entries(topics) as [topic, value]}
		<div class="space-y-3">
			<!-- Category button - only toggles expansion -->
			<button
				onclick={() => toggleExpanded(topic)}
				class="w-full px-4 py-2 rounded-full backdrop-blur-sm transition-all duration-300
					   flex items-center justify-between
					   focus:outline-none focus:ring-2 focus:ring-blue-400
					   border border-gray-800/50 hover:border-blue-400/50
					   bg-gray-800/30 hover:bg-gray-700/40 text-sm"
			>
				<span class="capitalize font-serif text-gray-200">{topic.replace(/_/g, ' ')}</span>
				{#if typeof value === 'object'}
					<span class="text-blue-400 text-sm transition-transform duration-300 {$expanded.has(topic) ? 'rotate-90' : ''}">
						→
					</span>
				{/if}
			</button>

			{#if $expanded.has(topic)}
				<div class="pl-4 space-y-3 border-l border-gray-800/50">
					{#if Array.isArray(value)}
						{#each value as item}
							<!-- Leaf node topic - can be selected -->
							<button
								onclick={() => topicsStore.toggle(item)}
								class="w-full px-4 py-2 rounded-full backdrop-blur-sm transition-all duration-300
									   flex items-center justify-between
									   focus:outline-none focus:ring-2 focus:ring-blue-400
									   border border-gray-800/50 hover:border-blue-400/50
									   bg-gray-800/30 hover:bg-gray-700/40 text-sm
									   {$topicsStore.has(item) ? 'bg-blue-900/30 border-blue-400/50' : ''}"
							>
								<div class="flex items-center gap-2">
									{#if $topicsStore.has(item)}
										<span class="text-yellow-400 text-xs">★</span>
									{/if}
									<span class="capitalize font-serif text-gray-200">{item.replace(/-/g, ' ')}</span>
								</div>
							</button>
						{/each}
					{:else}
						{#each Object.entries(value) as [subtopic, subvalue]}
							<!-- Subcategory button - only toggles expansion -->
							<button
								onclick={() => toggleExpanded(subtopic)}
								class="w-full px-4 py-2 rounded-full backdrop-blur-sm transition-all duration-300
									   flex items-center justify-between
									   focus:outline-none focus:ring-2 focus:ring-blue-400
									   border border-gray-800/50 hover:border-blue-400/50
									   bg-gray-800/30 hover:bg-gray-700/40 text-sm"
							>
								<span class="capitalize font-serif text-gray-200">{subtopic.replace(/_/g, ' ')}</span>
								{#if typeof subvalue === 'object'}
									<span class="text-blue-400 text-sm transition-transform duration-300 {$expanded.has(subtopic) ? 'rotate-90' : ''}">
										→
									</span>
								{/if}
							</button>

							{#if $expanded.has(subtopic)}
								<div class="pl-4 space-y-3 border-l border-gray-800/50">
									{#if Array.isArray(subvalue)}
										{#each subvalue as item}
											<!-- Leaf node topic - can be selected -->
											<button
												onclick={() => topicsStore.toggle(item)}
												class="w-full px-4 py-2 rounded-full backdrop-blur-sm transition-all duration-300
													   flex items-center justify-between
													   focus:outline-none focus:ring-2 focus:ring-blue-400
													   border border-gray-800/50 hover:border-blue-400/50
													   bg-gray-800/30 hover:bg-gray-700/40 text-sm
													   {$topicsStore.has(item) ? 'bg-blue-900/30 border-blue-400/50' : ''}"
											>
												<div class="flex items-center gap-2">
													{#if $topicsStore.has(item)}
														<span class="text-yellow-400 text-xs">★</span>
													{/if}
													<span class="capitalize font-serif text-gray-200">{item.replace(/-/g, ' ')}</span>
												</div>
											</button>
										{/each}
									{:else}
										{#each Object.entries(subvalue) as [subsubtopic, subsubvalue]}
											<!-- Leaf node topic - can be selected -->
											<button
												onclick={() => topicsStore.toggle(subsubtopic)}
												class="w-full px-4 py-2 rounded-full backdrop-blur-sm transition-all duration-300
													   flex items-center justify-between
													   focus:outline-none focus:ring-2 focus:ring-blue-400
													   border border-gray-800/50 hover:border-blue-400/50
													   bg-gray-800/30 hover:bg-gray-700/40 text-sm
													   {$topicsStore.has(subsubtopic) ? 'bg-blue-900/30 border-blue-400/50' : ''}"
											>
												<div class="flex items-center gap-2">
													{#if $topicsStore.has(subsubtopic)}
														<span class="text-yellow-400 text-xs">★</span>
													{/if}
													<span class="capitalize font-serif text-gray-200">
														{subsubtopic.replace(/_/g, ' ')}
													</span>
												</div>
											</button>
										{/each}
									{/if}
								</div>
							{/if}
						{/each}
					{/if}
				</div>
			{/if}
		</div>
	{/each}
</div>

<div class="fixed bottom-6 right-6 flex gap-3">
	{#if !canContinue}
		<button
			onclick={() => goto('/feed')}
			class="bg-gray-700/50 hover:bg-gray-600/50 text-gray-300 font-bold py-3 px-6 rounded-lg transition-all duration-300 flex items-center gap-2 backdrop-blur-sm border border-gray-600/30"
		>
			Skip for now
		</button>
	{/if}

	<button
		onclick={() => goto('/feed')}
		class="bg-blue-400 hover:bg-blue-500 text-white font-bold py-3 px-6 rounded-lg transition-all duration-300 flex items-center gap-2"
	>
		{canContinue ? 'Continue to Feed' : 'Continue without topics'}
		<svg xmlns="http://www.w3.org/2000/svg" class="h-5 w-5" viewBox="0 0 20 20" fill="currentColor">
			<path fill-rule="evenodd" d="M10.293 3.293a1 1 0 011.414 0l6 6a1 1 0 010 1.414l-6 6a1 1 0 01-1.414-1.414L14.586 11H3a1 1 0 110-2h11.586l-4.293-4.293a1 1 0 010-1.414z" clip-rule="evenodd" />
		</svg>
	</button>
</div>

