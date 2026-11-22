<script lang="ts">
	import { onMount } from 'svelte';
	import { api } from '$lib/api/client';
	import type { components } from '$lib/api/schema';
	import ContributionFormModal from '$lib/components/ContributionFormModal.svelte';
	import ConfirmDialog from '$lib/components/ConfirmDialog.svelte';
	import Badge from '$lib/components/Badge.svelte';
	import { badgeToneClass, type BadgeTone } from '$lib/badgeTones';

	type Contribution = components['schemas']['Contribution'];
	type ContributionType = components['schemas']['ContributionType'];
	type Project = components['schemas']['Project'];

	const pageSize = 50;

	let contributions = $state<Contribution[]>([]);
	let total = $state(0);
	let currentPage = $state(1);
	let loading = $state(true);
	let error = $state<string | null>(null);

	// Toggleable chips, not a dropdown — several types can be filtered at
	// once. Empty array = no filter (show every type), same default meaning
	// the old single-select's '' value had. A plain array reassigned on
	// every toggle (not a Set — see the Tasks-section status filter for why
	// $state(new Set()) silently fails to trigger re-renders on .add()/
	// .delete()). Server-side filter: every toggle resets to page 1 and
	// re-fetches, since this page paginates on the backend rather than
	// filtering an already-loaded list client-side.
	let typeFilters = $state<ContributionType[]>([]);
	const typeFilterOptions: { value: ContributionType; label: string }[] = [
		{ value: 'commit', label: 'Commit' },
		{ value: 'pull_request', label: 'Pull request' },
		{ value: 'code_review', label: 'Code review' },
		{ value: 'issue', label: 'Issue' },
		{ value: 'design', label: 'Design' },
		{ value: 'documentation', label: 'Documentation' },
		{ value: 'deployment', label: 'Deployment' },
		{ value: 'support', label: 'Support' },
		{ value: 'other', label: 'Other' }
	];
	let peopleNames = $state<Record<string, string>>({});
	let projectKeys = $state<Record<string, string>>({});

	let modalMode = $state<'create' | 'edit' | null>(null);
	let editingContribution = $state<Contribution | null>(null);

	let deletingContribution = $state<Contribution | null>(null);
	let deleting = $state(false);
	let deleteError = $state<string | null>(null);

	let banner = $state<{ kind: 'success' | 'error'; text: string } | null>(null);
	let bannerTimer: ReturnType<typeof setTimeout> | undefined;

	function flash(kind: 'success' | 'error', text: string) {
		banner = { kind, text };
		clearTimeout(bannerTimer);
		bannerTimer = setTimeout(() => (banner = null), 4000);
	}

	async function loadLookups() {
		const [peopleRes, projectsRes] = await Promise.all([
			api.GET('/api/v1/people', { params: { query: { page: 1, pageSize: 100 } } }),
			api.GET('/api/v1/projects', { params: { query: { page: 1, pageSize: 100 } } })
		]);
		if (peopleRes.data) {
			const map: Record<string, string> = {};
			for (const p of peopleRes.data.data) map[p.id] = `${p.firstName} ${p.lastName}`;
			peopleNames = map;
		}
		if (projectsRes.data) {
			const map: Record<string, string> = {};
			for (const p of projectsRes.data.data as Project[]) map[p.id] = p.key;
			projectKeys = map;
		}
	}

	async function load() {
		loading = true;
		error = null;

		const { data, error: err } = await api.GET('/api/v1/contributions', {
			params: {
				query: {
					page: currentPage,
					pageSize,
					...(typeFilters.length > 0 ? { type: typeFilters } : {})
				}
			}
		});

		if (err) {
			error = err.message ?? 'Failed to load contributions';
		} else if (data) {
			contributions = data.data;
			total = data.pagination.total;
		}
		loading = false;
	}

	const totalPages = $derived(Math.max(1, Math.ceil(total / pageSize)));

	function personName(id: string): string {
		return peopleNames[id] ?? 'Unknown person';
	}

	function openCreate() {
		editingContribution = null;
		modalMode = 'create';
	}

	function openEdit(c: Contribution) {
		editingContribution = c;
		modalMode = 'edit';
	}

	function closeModal() {
		modalMode = null;
		editingContribution = null;
	}

	function handleSaved(saved: Contribution) {
		const wasEdit = editingContribution !== null;
		closeModal();
		flash('success', wasEdit ? 'Contribution updated.' : `Logged a contribution for ${personName(saved.personId)}.`);
		load();
	}

	function askDelete(c: Contribution) {
		deleteError = null;
		deletingContribution = c;
	}

	async function confirmDelete() {
		if (!deletingContribution) return;
		const target = deletingContribution;
		deleting = true;
		deleteError = null;

		const { error: err } = await api.DELETE('/api/v1/contributions/{id}', {
			params: { path: { id: target.id } }
		});

		deleting = false;
		if (err) {
			deleteError = err.message ?? 'Failed to delete contribution.';
			return;
		}
		deletingContribution = null;
		flash('success', 'Contribution removed.');
		load();
	}

	function toggleTypeFilter(type: ContributionType) {
		typeFilters = typeFilters.includes(type)
			? typeFilters.filter((t) => t !== type)
			: [...typeFilters, type];
		currentPage = 1;
		load();
	}

	function clearTypeFilters() {
		typeFilters = [];
		currentPage = 1;
		load();
	}

	function typeTone(type: ContributionType): BadgeTone {
		switch (type) {
			case 'commit':
				return 'neutralMuted';
			case 'pull_request':
				return 'violet';
			case 'code_review':
				return 'info';
			case 'issue':
				return 'warning';
			case 'deployment':
				return 'success';
			default:
				return 'neutral';
		}
	}

	function prevPage() {
		if (currentPage > 1) {
			currentPage -= 1;
			load();
		}
	}

	function nextPage() {
		if (currentPage < totalPages) {
			currentPage += 1;
			load();
		}
	}

	onMount(() => {
		load();
		loadLookups();
	});
</script>

<section class="space-y-4">
	<div class="flex flex-wrap items-center justify-between gap-3">
		<div>
			<h1 class="text-2xl font-semibold tracking-tight">Contributions</h1>
			<p class="mt-0.5 text-sm text-slate-500">
				{total}
				{total === 1 ? 'entry' : 'entries'} in the activity feed
			</p>
		</div>
		<button type="button" class="btn-primary" onclick={openCreate}>+ Log contribution</button>
	</div>

	{#if banner}
		<div
			class="rounded-md border px-3 py-2 text-sm {banner.kind === 'success'
				? 'border-emerald-200 bg-emerald-50 text-emerald-700'
				: 'border-red-200 bg-red-50 text-red-700'}"
		>
			{banner.text}
		</div>
	{/if}

	<div class="flex flex-wrap items-center gap-1.5" role="group" aria-label="Filter by type">
		{#each typeFilterOptions as opt (opt.value)}
			{@const selected = typeFilters.includes(opt.value)}
			<button
				type="button"
				class="rounded-full px-2.5 py-1 text-xs font-medium transition-colors {selected
					? badgeToneClass[typeTone(opt.value)] + ' ring-1 ring-inset ring-transparent'
					: 'bg-white text-slate-500 ring-1 ring-inset ring-slate-200 hover:bg-slate-50'}"
				aria-pressed={selected}
				onclick={() => toggleTypeFilter(opt.value)}
			>
				{opt.label}
			</button>
		{/each}
		{#if typeFilters.length > 0}
			<button
				type="button"
				class="px-1 text-xs text-slate-500 hover:text-slate-700 hover:underline"
				onclick={clearTypeFilters}
			>
				Clear
			</button>
		{/if}
	</div>

	{#if loading}
		<p class="text-slate-500">Loading…</p>
	{:else if error}
		<div class="rounded-lg border border-red-200 bg-red-50 p-4 text-red-700">
			<p class="font-medium">Couldn't reach the API.</p>
			<p class="text-sm">{error}</p>
			<p class="mt-2 text-sm text-red-600">
				Make sure the Go backend is running on <code>localhost:8080</code>.
			</p>
		</div>
	{:else if contributions.length === 0}
		<div class="rounded-lg border border-slate-200 bg-white p-8 text-center text-slate-500">
			No contributions logged yet. Click "Log contribution" to add one.
		</div>
	{:else}
		<ul class="m-list">
			{#each contributions as contribution (contribution.id)}
				<li class="m-card">
					<div class="flex items-start justify-between gap-2">
						<p class="font-medium text-slate-900">{personName(contribution.personId)}</p>
						<Badge tone={typeTone(contribution.type)}>{contribution.type.replace('_', ' ')}</Badge>
					</div>
					<p class="mt-1 break-words text-slate-700">
						{#if contribution.url}
							<a
								href={contribution.url}
								target="_blank"
								rel="noopener noreferrer"
								class="underline decoration-slate-300 hover:decoration-slate-500"
							>
								{contribution.title ?? contribution.url}
							</a>
						{:else}
							{contribution.title ?? '—'}
						{/if}
					</p>
					<div class="m-meta">
						<span class="font-mono text-xs">
							{contribution.projectId ? (projectKeys[contribution.projectId] ?? '—') : '—'}
						</span>
						<span>{contribution.occurredAt.slice(0, 10)}</span>
					</div>
					<div class="m-actions">
						<button type="button" class="btn-secondary" onclick={() => openEdit(contribution)}>Edit</button>
						<button
							type="button"
							class="btn-secondary text-red-600 hover:bg-red-50"
							onclick={() => askDelete(contribution)}
						>
							Delete
						</button>
					</div>
				</li>
			{/each}
		</ul>

		<div class="table-wrap">
			<table class="w-full text-left text-sm">
				<thead class="border-b border-slate-200 bg-slate-50 text-slate-600">
					<tr>
						<th class="px-4 py-3 font-medium">Person</th>
						<th class="px-4 py-3 font-medium">Type</th>
						<th class="px-4 py-3 font-medium">Title</th>
						<th class="px-4 py-3 font-medium">Project</th>
						<th class="px-4 py-3 font-medium">Date</th>
						<th class="px-4 py-3 font-medium"><span class="sr-only">Actions</span></th>
					</tr>
				</thead>
				<tbody class="divide-y divide-slate-100">
					{#each contributions as contribution (contribution.id)}
						<!-- svelte-ignore a11y_click_events_have_key_events -->
						<!-- svelte-ignore a11y_no_static_element_interactions -->
						<tr class="cursor-pointer hover:bg-slate-50" onclick={() => openEdit(contribution)}>
							<td class="px-4 py-3 font-medium">{personName(contribution.personId)}</td>
							<td class="px-4 py-3">
								<Badge tone={typeTone(contribution.type)}>{contribution.type.replace('_', ' ')}</Badge>
							</td>
							<td class="px-4 py-3 text-slate-600">
								{#if contribution.url}
									<a
										href={contribution.url}
										target="_blank"
										rel="noopener noreferrer"
										class="text-slate-700 underline decoration-slate-300 hover:decoration-slate-500"
										onclick={(e) => e.stopPropagation()}
									>
										{contribution.title ?? contribution.url}
									</a>
								{:else}
									{contribution.title ?? '—'}
								{/if}
							</td>
							<td class="px-4 py-3 font-mono text-xs text-slate-500">
								{contribution.projectId ? (projectKeys[contribution.projectId] ?? '—') : '—'}
							</td>
							<td class="px-4 py-3 text-slate-600">{contribution.occurredAt.slice(0, 10)}</td>
							<td class="px-4 py-3 text-right whitespace-nowrap" onclick={(e) => e.stopPropagation()}>
								<button
									type="button"
									class="btn-ghost px-2 py-1"
									onclick={() => openEdit(contribution)}
								>
									Edit
								</button>
								<button
									type="button"
									class="btn-ghost px-2 py-1 text-red-600 hover:bg-red-50 hover:text-red-700"
									onclick={() => askDelete(contribution)}
								>
									Delete
								</button>
							</td>
						</tr>
					{/each}
				</tbody>
			</table>
		</div>

		{#if totalPages > 1}
			<div class="flex items-center justify-between pt-2 text-sm text-slate-600">
				<span>Page {currentPage} of {totalPages}</span>
				<div class="flex gap-2">
					<button type="button" class="btn-secondary" disabled={currentPage <= 1} onclick={prevPage}>
						Previous
					</button>
					<button
						type="button"
						class="btn-secondary"
						disabled={currentPage >= totalPages}
						onclick={nextPage}
					>
						Next
					</button>
				</div>
			</div>
		{/if}
	{/if}
</section>

{#if modalMode}
	<ContributionFormModal
		contribution={editingContribution}
		onClose={closeModal}
		onSaved={handleSaved}
	/>
{/if}

<ConfirmDialog
	open={deletingContribution !== null}
	title="Delete contribution"
	message="Remove this contribution entry? This can't be undone."
	confirmLabel={deleting ? 'Deleting…' : 'Delete'}
	danger
	error={deleteError}
	onConfirm={confirmDelete}
	onCancel={() => (deletingContribution = null)}
/>
