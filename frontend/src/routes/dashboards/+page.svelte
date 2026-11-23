<script lang="ts">
	import { goto } from '$app/navigation';
	import { api } from '$lib/api/client';
	import type { components } from '$lib/api/schema';
	import { can } from '$lib/auth.svelte';

	let dashboards = $state<components['schemas']['Dashboard'][]>([]);
	let people = $state<Record<string, string>>({});
	let name = $state('');
	let error = $state<string | null>(null);

	$effect(() => {
		api.GET('/api/v1/dashboards').then((r) => (dashboards = r.data?.data ?? []));
		api.GET('/api/v1/people', { params: { query: { page: 1, pageSize: 100 } } }).then(
			(r) => (people = Object.fromEntries((r.data?.data ?? []).map((x) => [x.id, `${x.firstName} ${x.lastName}`])))
		);
	});
	async function create(e: SubmitEvent) {
		e.preventDefault();
		const { data, error: err } = await api.POST('/api/v1/dashboards', { body: { name, widgets: [] } });
		if (err || !data) return (error = err?.message ?? 'Could not make the dashboard.');
		goto(`/dashboards/${data.id}?edit=1`);
	}
</script>

<svelte:head><title>Dashboards · Pyramid</title></svelte:head>

<section class="space-y-4">
	<div>
		<h1 class="text-2xl font-semibold tracking-tight">Dashboards</h1>
		<p class="mt-1 text-sm text-slate-500">Charts of saved queries, sprint burndown and velocity, cumulative flow, capacity and notes on one page.</p>
	</div>
	{#if can('member')}
		<form class="flex flex-wrap gap-2" onsubmit={create}>
			<input class="input w-full sm:w-64" placeholder="New dashboard name" aria-label="Dashboard name" required maxlength="100" bind:value={name} />
			<button class="btn-primary" type="submit">Create dashboard</button>
		</form>
	{/if}
	{#if error}<p class="text-sm text-red-700">{error}</p>{/if}
	<ul class="grid gap-3 sm:grid-cols-2 lg:grid-cols-3">
		{#each dashboards as d (d.id)}
			<li>
				<a href="/dashboards/{d.id}" class="block rounded-lg border border-slate-200 bg-white p-4 hover:border-slate-300 hover:shadow-sm">
					<div class="font-medium text-slate-900">{d.name}</div>
					<div class="mt-1 text-xs text-slate-500">
						{d.widgets.length} {d.widgets.length === 1 ? 'widget' : 'widgets'} ·
						{d.shared ? (d.mine ? 'shared by you' : `shared by ${people[d.ownerPersonId ?? ''] ?? 'someone'}`) : 'only you'}
					</div>
				</a>
			</li>
		{:else}
			<li class="text-sm text-slate-500">No dashboards yet.</li>
		{/each}
	</ul>
</section>
