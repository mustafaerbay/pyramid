<script lang="ts">
	import { onMount } from 'svelte';
	import { api } from '$lib/api/client';
	import type { components } from '$lib/api/schema';

	// Types come straight from the OpenAPI contract — no hand-written interfaces.
	type Person = components['schemas']['Person'];

	let people = $state<Person[]>([]);
	let total = $state(0);
	let loading = $state(true);
	let error = $state<string | null>(null);

	async function load() {
		loading = true;
		error = null;

		const { data, error: err } = await api.GET('/api/v1/people', {
			params: { query: { page: 1, pageSize: 25 } }
		});

		if (err) {
			error = err.message ?? 'Failed to load people';
		} else if (data) {
			people = data.data;
			total = data.pagination.total;
		}
		loading = false;
	}

	function fullName(p: Person): string {
		return `${p.firstName} ${p.lastName}`;
	}

	onMount(load);
</script>

<section class="space-y-4">
	<div class="flex items-center justify-between">
		<h1 class="text-2xl font-semibold tracking-tight">People</h1>
		<span class="rounded-full bg-slate-200 px-3 py-1 text-sm text-slate-700">
			{total} total
		</span>
	</div>

	{#if loading}
		<p class="text-slate-500">Loading…</p>
	{:else if error}
		<div class="rounded-lg border border-red-200 bg-red-50 p-4 text-red-700">
			<p class="font-medium">Couldn’t reach the API.</p>
			<p class="text-sm">{error}</p>
			<p class="mt-2 text-sm text-red-600">
				Make sure the Go backend is running on <code>localhost:8080</code>.
			</p>
		</div>
	{:else if people.length === 0}
		<div class="rounded-lg border border-slate-200 bg-white p-8 text-center text-slate-500">
			No people yet. Create one via <code>POST /api/v1/people</code>.
		</div>
	{:else}
		<div class="overflow-hidden rounded-lg border border-slate-200 bg-white">
			<table class="w-full text-left text-sm">
				<thead class="border-b border-slate-200 bg-slate-50 text-slate-600">
					<tr>
						<th class="px-4 py-3 font-medium">Name</th>
						<th class="px-4 py-3 font-medium">Email</th>
						<th class="px-4 py-3 font-medium">Title</th>
						<th class="px-4 py-3 font-medium">Seniority</th>
						<th class="px-4 py-3 font-medium">Status</th>
					</tr>
				</thead>
				<tbody class="divide-y divide-slate-100">
					{#each people as person (person.id)}
						<tr class="hover:bg-slate-50">
							<td class="px-4 py-3 font-medium">{fullName(person)}</td>
							<td class="px-4 py-3 text-slate-600">{person.email}</td>
							<td class="px-4 py-3 text-slate-600">{person.title ?? '—'}</td>
							<td class="px-4 py-3 text-slate-600">{person.seniority ?? '—'}</td>
							<td class="px-4 py-3">
								<span class="rounded-full bg-emerald-100 px-2 py-0.5 text-xs text-emerald-700">
									{person.status}
								</span>
							</td>
						</tr>
					{/each}
				</tbody>
			</table>
		</div>
	{/if}
</section>
