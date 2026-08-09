<script lang="ts">
	import type { Snippet } from 'svelte';
	import { badgeToneClass, type BadgeTone } from '$lib/badgeTones';

	let {
		tone,
		capitalize = false,
		children
	}: {
		tone: BadgeTone;
		// A few call sites (e.g. a goal's "scope") show the raw enum value
		// as-is rather than a pre-formatted label — capitalize just that case
		// instead of making every caller remember `class="capitalize"`.
		capitalize?: boolean;
		children: Snippet;
	} = $props();
</script>

<span class="rounded-full px-2 py-0.5 text-xs {badgeToneClass[tone]} {capitalize ? 'capitalize' : ''}">
	{@render children()}
</span>
