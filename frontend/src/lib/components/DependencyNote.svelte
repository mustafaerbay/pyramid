<script lang="ts">
	import type { components } from '$lib/api/schema';
	import { blocksOf, openBlockers } from '$lib/dependencies';

	type Task = components['schemas']['Task'];

	let { task, all }: { task: Task; all: Task[] } = $props();

	const waiting = $derived(task.status === 'done' || task.status === 'cancelled' ? [] : openBlockers(task, all));
	const blocks = $derived(
		task.status === 'done' || task.status === 'cancelled'
			? []
			: blocksOf(task.id, all).filter((t) => t.status !== 'done' && t.status !== 'cancelled')
	);
	const ref = (t: Task) => `#${t.number} ${t.title}`;
</script>

{#if waiting.length > 0}
	<span class="mt-0.5 block text-xs font-medium text-amber-700" title={waiting.map(ref).join(', ')}>
		⏳ Waiting on {waiting.map(ref).join(', ')}
	</span>
{/if}
{#if blocks.length > 0}
	<span class="mt-0.5 block text-xs text-slate-500" title={blocks.map(ref).join(', ')}>
		Blocks {blocks.length === 1 ? ref(blocks[0]) : `${blocks.length} tasks`}
	</span>
{/if}
