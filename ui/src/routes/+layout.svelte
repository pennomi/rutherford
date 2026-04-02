<script lang="ts">
  import { page } from '$app/state';
  import { initAuth, getUserManager } from '$lib/auth';
  import { cluster } from '$lib/state.svelte';
  import { connectWebSocket } from '$lib/ws';
  import { onMount } from 'svelte';
  import '../app.css';

  let { children } = $props();
  let authRequired = $state(false);
  let accessDenied = $state(false);
  let accessDeniedReason = $state('');
  let userName = $state('');
  let userEmail = $state('');
  let userPicture = $state('');
  let isCallback = $derived(page.url.pathname === '/callback');
  let menuOpen = $state(false);
  let menuRef = $state<HTMLElement>(null!);

  interface Crumb {
    label: string;
    href: string;
  }

  let breadcrumbs = $derived.by((): Crumb[] => {
    const path = page.url.pathname;
    const crumbs: Crumb[] = [{ label: 'Rutherford', href: '/' }];
    const match = path.match(/^\/namespace\/([^/]+)/);
    if (match) {
      const ns = decodeURIComponent(match[1]);
      crumbs.push({ label: ns, href: `/namespace/${match[1]}` });
      const podMatch = path.match(/^\/namespace\/[^/]+\/pod\/([^/]+)/);
      if (podMatch) {
        crumbs.push({ label: decodeURIComponent(podMatch[1]), href: path });
      }
    }
    return crumbs;
  });

  onMount(async () => {
    if (isCallback) return;

    const userManager = await initAuth();

    if (!userManager) {
      connectWebSocket();
      return;
    }

    const user = await userManager.getUser();
    if (user && !user.expired) {
      userName = user.profile.name ?? user.profile.preferred_username ?? '';
      userEmail = user.profile.email ?? '';
      userPicture = user.profile.picture ?? '';

      const checkResp = await fetch('/api/auth/check', {
        headers: { Authorization: `Bearer ${user.access_token}` }
      });
      if (!checkResp.ok) {
        accessDenied = true;
        accessDeniedReason = await checkResp.text();
        return;
      }

      connectWebSocket(user.access_token);
    } else {
      authRequired = true;
      await userManager.signinRedirect();
    }
  });

  async function logout() {
    await getUserManager().signoutRedirect();
  }

  function handleClickOutside(event: MouseEvent) {
    if (menuRef && !menuRef.contains(event.target as Node)) {
      menuOpen = false;
    }
  }

  function handleKeydown(event: KeyboardEvent) {
    if (event.key === 'Escape') {
      menuOpen = false;
    }
  }
</script>

<svelte:document onclick={handleClickOutside} onkeydown={handleKeydown} />

{#if isCallback}
  {@render children()}
{:else}
  <div class="bg-neutral-800 min-h-screen">
    <header class="flex items-center justify-between px-6 py-3 bg-neutral-700 border-b border-zinc-700">
      <div class="flex items-center gap-4">
        <span class="text-white font-semibold text-lg">Rutherford</span>
        <span class="w-px h-5 bg-zinc-500"></span>
        <nav class="flex items-center gap-1.5 text-md">
          {#each breadcrumbs as crumb, i}
            {#if i > 0}
              <span class="text-zinc-600">/</span>
            {/if}
            {#if i < breadcrumbs.length - 1}
              <a href={crumb.href} class="text-zinc-400 hover:text-zinc-200">{crumb.label}</a>
            {:else}
              <span class="text-zinc-300">{crumb.label}</span>
            {/if}
          {/each}
        </nav>
      </div>
      <div class="flex items-center gap-5">
        <div class="flex items-center gap-2">
          {#if cluster.connected}
            <span class="w-2.5 h-2.5 rounded-full bg-emerald-500"></span>
            <span class="text-emerald-400 text-sm">Connected</span>
          {:else}
            <span class="w-2.5 h-2.5 rounded-full bg-red-500 animate-pulse"></span>
            <span class="text-red-400 text-sm">{cluster.reconnectStatus || 'Disconnected'}</span>
          {/if}
        </div>
        {#if userEmail}
          <div class="relative" bind:this={menuRef}>
            <button
              onclick={() => menuOpen = !menuOpen}
              class="flex items-center gap-2 text-zinc-400 text-sm hover:text-zinc-200 cursor-pointer rounded px-2 py-1 hover:bg-neutral-600 transition-colors"
            >
              {#if userPicture}
                <img src={userPicture} alt="" class="w-7 h-7 rounded-full" />
              {/if}
              <span>{userEmail}</span>
              <svg class="w-3.5 h-3.5 transition-transform {menuOpen ? 'rotate-180' : ''}" fill="none" viewBox="0 0 24 24" stroke="currentColor" stroke-width="2">
                <path d="M19 9l-7 7-7-7" />
              </svg>
            </button>
            {#if menuOpen}
              <div class="absolute right-0 mt-1 w-56 bg-neutral-700 border border-zinc-600 rounded-lg shadow-xl z-50 overflow-hidden">
                <div class="px-4 py-3 border-b border-zinc-600">
                  {#if userName}
                    <p class="text-zinc-200 text-sm font-medium">{userName}</p>
                  {/if}
                  <p class="text-zinc-400 text-xs truncate">{userEmail}</p>
                </div>
                <button
                  onclick={logout}
                  class="w-full text-left px-4 py-2.5 text-sm text-zinc-300 hover:bg-neutral-600 hover:text-white cursor-pointer flex items-center gap-2 transition-colors"
                >
                  <svg class="w-4 h-4" fill="none" viewBox="0 0 24 24" stroke="currentColor" stroke-width="2">
                    <path d="M17 16l4-4m0 0l-4-4m4 4H7m6 4v1a3 3 0 01-3 3H6a3 3 0 01-3-3V7a3 3 0 013-3h4a3 3 0 013 3v1" />
                  </svg>
                  Sign out
                </button>
              </div>
            {/if}
          </div>
        {/if}
      </div>
    </header>
    {#key page.url.pathname}
      {@render children()}
    {/key}
  </div>

  {#if accessDenied || cluster.authError}
    <div class="fixed inset-0 bg-neutral-900 flex items-center justify-center z-50">
      <div class="text-center">
        <h1 class="text-2xl text-white font-semibold mb-2">Access Denied</h1>
        <p class="text-zinc-400 mb-1">Signed in as <span class="text-zinc-200">{userEmail}</span></p>
        <p class="text-red-400 text-sm mb-4">{accessDeniedReason || cluster.authError}</p>
        <p class="text-zinc-500 mb-6">Contact your cluster administrator to request access.</p>
        <button onclick={logout} class="px-4 py-2 bg-zinc-700 text-zinc-200 rounded hover:bg-zinc-600 cursor-pointer">Sign out</button>
      </div>
    </div>
  {:else if authRequired}
    <div class="fixed inset-0 bg-neutral-900/80 flex items-center justify-center z-50">
      <div class="text-zinc-500">Redirecting to login...</div>
    </div>
  {/if}
{/if}
