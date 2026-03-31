<script lang="ts">
  import { initAuth } from '$lib/auth';
  import { onMount } from 'svelte';

  onMount(async () => {
    const userManager = await initAuth();
    if (!userManager) {
      window.location.href = '/';
      return;
    }
    await userManager.signinRedirectCallback();
    window.location.href = '/';
  });
</script>

<div class="flex items-center justify-center min-h-screen bg-neutral-800 text-zinc-500">
  Completing login...
</div>
