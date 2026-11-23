<script lang="ts">
	import { api } from '$lib/api/client';
	import type { components } from '$lib/api/schema';

	let rows = $state<components['schemas']['AuditEntry'][]>([]);
	let total = $state(0);
	let page = $state(1);
	let q = $state('');
	let error = $state<string | null>(null);
	const SIZE = 50;
	let timer: ReturnType<typeof setTimeout> | undefined;
	$effect(() => {
		const query = { page, pageSize: SIZE, ...(q.trim() ? { q: q.trim() } : {}) };
		clearTimeout(timer);
		timer = setTimeout(async () => {
			const { data, error: err } = await api.GET('/api/v1/audit', { params: { query } });
			if (err) return (error = 'Only admins can read the audit log.');
			rows = data?.data ?? [];
			total = data?.total ?? 0;
		}, 200);
	});
</script>

<svelte:head><title>Audit log · Pyramid</title></svelte:head>

<section class="space-y-4">
	<div>
		<h1 class="text-2xl font-semibold tracking-tight">Audit log</h1>
		<p class="mt-1 text-sm text-slate-500">Every change made through Pyramid and every sign-in: who, what, which fields (never their values), from where.</p>
	</div>
	<input class="input w-full sm:w-72" placeholder="Filter by person or action, e.g. failed" aria-label="Filter" bind:value={q} oninput={() => (page = 1)} />
	{#if error}<p class="text-sm text-red-700">{error}</p>{/if}
	<div class="overflow-x-auto rounded-lg border border-slate-200 bg-white">
		<table class="w-full min-w-[44rem] text-left text-sm">
			<thead class="border-b border-slate-200 text-xs text-slate-500">
				<tr><th class="px-3 py-2 font-medium">When</th><th class="px-3 py-2 font-medium">Who</th><th class="px-3 py-2 font-medium">What</th><th class="px-3 py-2 font-medium">Fields</th><th class="px-3 py-2 font-medium">From</th></tr>
			</thead>
			<tbody class="divide-y divide-slate-100">
				{#each rows as e (e.id)}
					<tr class={e.action === 'failed sign in' ? 'bg-red-50' : ''}>
						<td class="whitespace-nowrap px-3 py-1.5 text-slate-600">{new Date(e.at).toLocaleString()}</td>
						<td class="px-3 py-1.5">{e.actor}</td>
						<td class="px-3 py-1.5">{e.action}{e.entityId ? '' : ''}<span class="ml-1 text-xs text-slate-400">{e.entityId ? e.entityId.slice(0, 8) : ''}</span></td>
						<td class="px-3 py-1.5 text-xs text-slate-500">{e.fields.join(', ')}</td>
						<td class="px-3 py-1.5 text-xs text-slate-500">{e.via}{e.ip ? ` · ${e.ip}` : ''}</td>
					</tr>
				{:else}
					<tr><td colspan="5" class="px-3 py-6 text-center text-slate-500">Nothing yet.</td></tr>
				{/each}
			</tbody>
		</table>
	</div>
	{#if total > SIZE}
		<div class="flex items-center gap-2 text-sm">
			<button type="button" class="btn-ghost" disabled={page === 1} onclick={() => page--}>← Newer</button>
			<span class="text-slate-500">Page {page} of {Math.ceil(total / SIZE)}</span>
			<button type="button" class="btn-ghost" disabled={page * SIZE >= total} onclick={() => page++}>Older →</button>
		</div>
	{/if}
</section>
