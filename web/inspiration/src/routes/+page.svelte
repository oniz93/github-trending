<!--
  Purpose: Landing page that automatically redirects based on topic selection status
  Context: Entry point of the application that manages initial user flow
-->

<script lang="ts">
  import { onMount } from 'svelte';
  import { fade, fly, slide } from 'svelte/transition';
  import { quintOut } from 'svelte/easing';
  import { Twitter, MessageSquareQuote } from 'lucide-svelte';

  let hasVisited = false;
  let visible = false;
  let currentQuoteIndex = 0;
  let slideDirection = 1; // 1 for right, -1 for left

  const quotes = [
    { text: "I spent 2 hours scrolling again. But this time, it was neither mindlessly, nor a waste of time :)", author: "WittyWithoutWorry" },
    { text: "Actually addicting lol", author: "AkhilSundaram" },
    { text: "Very cool idea, was fun scrolling through it for a bit", author: "Previous-Tune-8896" },
    { text: "Pretty cool. Would love to be able to filter by tags, keywords etc.", author: "AvidCoco" },
    { text: "Does it also send all my data to the Chinese government?", author: "Maskdask" },
  ];

  // Cycle through quotes every 5 seconds
  onMount(() => {
    hasVisited = localStorage.getItem('hasVisitedGitTok') === 'true';
    visible = true;

    const interval = setInterval(() => {
      slideDirection = 1;
      currentQuoteIndex = (currentQuoteIndex + 1) % quotes.length;
    }, 5000);

    return () => clearInterval(interval);
  });

  const features = [
    { icon: '🚀', text: '"No, I\'m not procrastinating, I\'m researching coding patterns!"' },
    { icon: '⚡', text: '"The only infinite scroll that makes you better at your job*"' },
    { icon: '⭐', text: '"Just star this repo to get your pinned projects featured on GitTok!"' },
  ];
</script>

<div class="min-h-screen relative overflow-hidden bg-gradient-to-br from-gray-900 via-purple-900/20 to-black">
  <!-- Animated background elements -->
  <div class="absolute inset-0 overflow-hidden">
    {#each Array(20) as _, i}
      <div
        class="absolute rounded-full mix-blend-screen animate-float"
        style="
          left: {Math.random() * 100}%;
          top: {Math.random() * 100}%;
          width: {Math.random() * 300 + 50}px;
          height: {Math.random() * 300 + 50}px;
          background: radial-gradient(circle at center,
            rgba(59, 130, 246, 0.1) 0%,
            rgba(59, 130, 246, 0) 70%);
          animation-delay: -{Math.random() * 5}s;
          animation-duration: {Math.random() * 10 + 15}s;
        "
      >
      </div>
    {/each}
  </div>

  <!-- Main content -->
  <div class="relative min-h-screen flex flex-col items-center justify-center p-6">
    {#if visible}
      <div
        class="glass-card text-center p-8 max-w-md w-full"
        in:fly={{ y: 50, duration: 1000 }}
      >
        <div in:fade={{ delay: 200, duration: 800 }}>
          <h1 class="text-5xl font-serif mb-2 text-white font-bold bg-gradient-to-r from-blue-400 to-purple-400 bg-clip-text text-transparent">
            GitTok
          </h1>
          <div class="flex items-center justify-center gap-2 mb-8">
            <p class="text-xl text-gray-300">Get addicted to code</p>
            <span class="text-gray-500">•</span>
            <a
              href="https://twitter.com/brsc2909"
              target="_blank"
              rel="noopener noreferrer"
              class="flex items-center gap-1.5 text-gray-400 hover:text-blue-400 transition-colors duration-200 group"
            >
              <span class="text-sm">by</span>
              <Twitter class="h-4 w-4 group-hover:text-blue-400 transition-colors duration-200" />
              <span class="text-sm font-mono group-hover:text-blue-400 transition-colors duration-200">@brsc2909</span>
            </a>
          </div>

          <!-- Community Quote Section -->
          <div class="relative h-32 mb-8 overflow-hidden">
            {#key currentQuoteIndex}
              <div
                class="absolute inset-0 p-4 rounded-lg bg-white/5 backdrop-blur-sm"
                in:slide|local={{ duration: 400, delay: 0, axis: 'x', easing: quintOut }}
                out:slide|local={{ duration: 400, delay: 0, axis: 'x', easing: quintOut }}
              >
                <div class="flex items-start gap-3">
                  <MessageSquareQuote class="h-5 w-5 text-blue-400 flex-shrink-0 mt-1" />
                  <div class="text-left">
                    <p class="text-gray-200 font-serif italic">{quotes[currentQuoteIndex].text}</p>
                    <p class="text-sm text-gray-400 mt-2 font-mono">- {quotes[currentQuoteIndex].author}</p>
                  </div>
                </div>
              </div>
            {/key}
          </div>

          <div class="space-y-6 text-gray-300 font-serif">
            {#each features as feature, i}
              <div
                class="text-left transform hover:scale-105 transition-transform duration-200"
                in:fly={{ y: 20, delay: 300 + i * 100, duration: 800 }}
              >
                <p class="flex items-center space-x-3 p-3 rounded-lg hover:bg-white/5">
                  <span class="text-2xl">{feature.icon}</span>
                  <span>{feature.text}</span>
                </p>
              </div>
            {/each}

            <p
              class="text-sm text-gray-400 mt-6 italic"
              in:fade={{ delay: 800, duration: 800 }}
            >
              *Results may vary. Side effects include improved git skills and random urges to refactor everything.
            </p>
          </div>

          <div class="flex flex-col items-center gap-4 mt-8">
            <a
              href="/feed"
              class="px-8 py-4 bg-gradient-to-r from-blue-400 to-blue-500 text-white font-medium rounded-lg transition-all duration-200 hover:scale-105 hover:shadow-lg hover:shadow-blue-500/25"
              in:fade={{ delay: 1000, duration: 800 }}
            >
              Start Scrolling
            </a>
          </div>
        </div>
      </div>
    {/if}
  </div>
</div>

<style>
  .glass-card {
    background: rgba(31, 41, 55, 0.3);
    backdrop-filter: blur(12px);
    border-radius: 1rem;
    border: 1px solid rgba(255, 255, 255, 0.1);
    box-shadow: 0 25px 50px -12px rgba(0, 0, 0, 0.25);
  }

  @keyframes float {
    0%, 100% {
      transform: translateY(0) scale(1);
    }
    50% {
      transform: translateY(-20px) scale(1.1);
    }
  }

  .animate-float {
    animation: float linear infinite;
  }

  /* Add hover effects to the glass card */
  .glass-card:hover {
    background: rgba(31, 41, 55, 0.4);
    border: 1px solid rgba(255, 255, 255, 0.2);
    transition: all 0.3s ease;
  }
</style>
