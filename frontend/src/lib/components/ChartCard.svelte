<script lang="ts">
	import type { Snippet } from 'svelte';

	let {
		title,
		subtitle = '',
		children
	}: {
		title: string;
		subtitle?: string;
		children: Snippet;
	} = $props();
</script>

<div class="rounded-lg border border-slate-200 bg-white p-5">
	<h3 class="text-sm font-semibold text-slate-900">{title}</h3>
	{#if subtitle}
		<p class="mt-0.5 text-xs text-slate-500">{subtitle}</p>
	{/if}
	<div class="mt-4">
		{@render children()}
	</div>
</div>
