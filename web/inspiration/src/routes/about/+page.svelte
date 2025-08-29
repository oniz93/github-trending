<!--
  Purpose: About page with information about the creator and relevant links
  Context: Created to provide information about the developer and link to their online presence
-->

<script lang="ts">
  import { Globe, Twitter, Sparkles } from 'lucide-svelte';
</script>

<div class="min-h-screen w-full bg-gradient-to-b from-gray-900 to-black">
  <div class="mx-auto max-w-3xl px-6 py-16">
    <div class="rounded-2xl bg-gray-800/30 p-8 backdrop-blur-sm">
      <h1 class="mb-6 font-serif text-4xl text-white">About GitTok</h1>

      <p class="mb-8 font-serif text-lg text-gray-200">
        GitTok is a TikTok-style interface for discovering interesting GitHub repositories.
        Scroll through projects, read their READMEs, and find your next favorite open source project.
      </p>

      <h2 class="mb-4 font-serif text-2xl text-white">Get Your Project Featured</h2>
      
      <div class="mb-8 p-4 rounded-xl bg-gradient-to-r from-gray-800/50 to-gray-700/50 border border-yellow-500/20">
        <div class="flex items-start gap-3">
          <Sparkles class="h-6 w-6 text-yellow-400 flex-shrink-0 mt-1" />
          <div>
            <p class="font-serif text-lg text-gray-200 mb-4">
              Want your open-source project to be showcased on GitTok with special styling and increased visibility?
            </p>
            
            <h3 class="font-serif text-xl text-white mb-2">Here's how:</h3>
            
            <ol class="space-y-2 font-serif text-gray-200 list-decimal list-inside">
              <li>
                <span class="text-yellow-400 font-bold">Star this repository</span> - 
                <span class="bg-yellow-500/20 px-2 py-0.5 rounded">This is all you need to do!</span> 
                Your pinned repositories become eligible for featuring
              </li>
            </ol>
            
            <h3 class="font-serif text-xl text-white mt-4 mb-2">We also appreciate if you:</h3>
            
            <ul class="space-y-2 font-serif text-gray-200 list-disc list-inside">
              <li><span class="text-blue-400 font-bold">Create a pull request</span> - Add your project details for better promotion chances</li>
              <li><span class="text-purple-400 font-bold">Share GitTok</span> - Help us grow the community by sharing on social media</li>
              <li><span class="text-blue-400 font-bold">Follow on Twitter</span> - Stay updated with the latest features</li>
            </ul>
            
            <div class="mt-4">
              <a
                href="https://github.com/BlackShoreTech/gittok.dev"
                target="_blank"
                rel="noopener noreferrer"
                class="inline-flex items-center gap-2 rounded-full bg-gradient-to-r from-yellow-500/30 to-yellow-600/30 hover:from-yellow-500/50 hover:to-yellow-600/50 px-4 py-2 text-white backdrop-blur-sm transition-all duration-300"
              >
                <Sparkles class="h-4 w-4 text-yellow-300" />
                <span class="font-semibold">Star on GitHub</span>
              </a>
            </div>
          </div>
        </div>
      </div>

      <h2 class="mb-4 font-serif text-2xl text-white">Creator</h2>

      <div class="space-y-8">
        <p class="font-serif text-lg text-gray-200">
          Built by Brendan Scullion, a caffeine-powered developer who thought scrolling through GitHub could be as addictive as TikTok.
        </p>

        <div class="space-y-4">
          <h3 class="font-serif text-xl text-white">Other Cool Projects</h3>
          <a
            href="https://coso.ai"
            target="_blank"
            rel="noopener noreferrer"
            class="flex items-center gap-3 rounded-lg bg-gray-700/50 px-4 py-3 text-white transition-colors hover:bg-gray-600/50"
          >
            <Sparkles class="h-5 w-5 text-yellow-400" />
            <span class="font-mono">coso.ai</span>
            <span class="ml-2 text-sm text-gray-300">- Social media co-pilot</span>
          </a>
        </div>

        <div class="space-y-4">
          <h3 class="font-serif text-xl text-white">Find Me Online</h3>
          <div class="flex flex-col gap-4">
            <a
              href="https://brendanscullion.com"
              target="_blank"
              rel="noopener noreferrer"
              class="flex items-center gap-3 rounded-lg bg-gray-700/50 px-4 py-3 text-white transition-colors hover:bg-gray-600/50"
            >
              <Globe class="h-5 w-5 text-blue-400" />
              <span class="font-mono">brendanscullion.com</span>
            </a>

            <a
              href="https://twitter.com/brsc2909"
              target="_blank"
              rel="noopener noreferrer"
              class="flex items-center gap-3 rounded-lg bg-gray-700/50 px-4 py-3 text-white transition-colors hover:bg-gray-600/50"
            >
              <Twitter class="h-5 w-5 text-blue-400" />
              <span class="font-mono">@brsc2909</span>
            </a>
          </div>
        </div>
      </div>
    </div>
  </div>
</div>