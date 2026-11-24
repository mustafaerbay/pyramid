<script lang="ts">
	import Icon from '$lib/components/Icon.svelte';
	import { page } from '$app/state';
	import { api } from '$lib/api/client';
	import type { components } from '$lib/api/schema';
	import TypeBadge from '$lib/components/TypeBadge.svelte';

	const q = $derived(page.url.searchParams.get('q') ?? '');
	let res = $state<components['schemas']['SearchResults'] | null>(null);
	$effect(() => {
		const term = q.trim();
		if (!term) return;
		api.GET('/api/v1/search', { params: { query: { q: term } } }).then((r) => (res = r.data ?? null));
	});
	const IN: Record<string, string> = { number: 'number', title: 'title', description: 'description', comment: 'comment' };
</script>

<svelte:head><title>Search · Pyramid</title></svelte:head>

<section class="space-y-4">
	<h1 class="text-2xl font-semibold tracking-tight">Results for “{q}”</h1>
	{#if res}
		{#if res.projects.length || res.people.length}
			<div class="flex flex-wrap gap-2 text-sm">
				{#each res.projects as p (p.id)}<a class="rounded-full bg-white px-3 py-1 ring-1 ring-slate-200 hover:bg-slate-50 inline-flex items-center gap-1.5" href="/projects/{p.id}"><Icon name="folder" size={14} class="text-slate-500" />{p.name}</a>{/each}
				{#each res.people as p (p.id)}<a class="rounded-full bg-white px-3 py-1 ring-1 ring-slate-200 hover:bg-slate-50 inline-flex items-center gap-1.5" href="/people/{p.id}"><Icon name="user" size={14} class="text-slate-500" />{p.name}</a>{/each}
			</div>
		{/if}
		<ul class="divide-y divide-slate-100 rounded-lg border border-slate-200 bg-white text-sm">
			{#each res.items as i (i.id)}
				<li class="px-4 py-2.5">
					<a href="/projects/{i.projectId}/items/{i.id}" class="flex min-h-8 items-center gap-2 hover:underline">
						<TypeBadge type={i.type} />
						<span class="text-xs tabular-nums text-slate-500">{i.projectKey}-{i.number}</span>
						<span class="font-medium text-slate-800">{i.title}</span>
						<span class="ml-auto text-xs text-slate-500">{i.status.replace('_', ' ')} · in {IN[i.matchedIn]}</span>
					</a>
					{#if i.snippet}<p class="mt-0.5 text-xs text-slate-500">{i.snippet}</p>{/if}
				</li>
			{:else}
				<li class="px-4 py-6 text-center text-slate-500">No work items match.</li>
			{/each}
		</ul>
		{#if res.items.length >= 30}<p class="text-xs text-slate-500">Showing the first 30 — add words to narrow it down, or use Queries.</p>{/if}
	{/if}
</section>
