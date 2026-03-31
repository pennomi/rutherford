<script lang="ts">
  interface Props {
    src: string;
    size: string;
  }

  let { src, size }: Props = $props();
</script>

<div class="relative {size} shrink-0 overflow-hidden rounded-md border border-white/5 bg-white/10">
  <img class="absolute inset-0 w-full h-full object-contain blur-xl opacity-50 scale-150" src={src} alt="" />
  <img class="relative w-full h-full object-contain p-2 drop-shadow-lg" src={src} alt="" />
</div>
