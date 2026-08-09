<script lang="ts">
	import { onMount } from 'svelte';
	import { goto } from '$app/navigation';
	import { api } from '$lib/api/client';
	import type { components } from '$lib/api/schema';
	import ProjectFormModal from '$lib/components/ProjectFormModal.svelte';
	import ProjectWizardModal from '$lib/components/ProjectWizardModal.svelte';
	import ConfirmDialog from '$lib/components/ConfirmDialog.svelte';
	import Badge from '$lib/components/Badge.svelte';
	import { healthDotClass } from '$lib/badgeTones';
	import type { BadgeTone } from '$lib/badgeTones';

	type Project = components['schemas']['Project'];
	type ProjectStatus = components['schemas']['ProjectStatus'];

	const pageSize = 50;

	let projects = $state<Project[]>([]);
	let total = $state(0);
	let currentPage = $state(1);
	let loading = $state(true);
	let error = $state<string | null>(null);

	let search = $state('');
	let statusFilter = $state<ProjectStatus | ''>('');

	// id -> "First Last", so the table can show lead names instead of raw ids.
	let peopleNames = $state<Record<string, string>>({});

	let modalMode = $state<'edit' | null>(null);
	let editingProject = $state<Project | null>(null);
	let showWizard = $state(false);

	let deletingProject = $state<Project | null>(null);
	let deleting = $state(false);
	let deleteError = $state<string | null>(null);

	let banner = $state<{ kind: 'success' | 'error'; text: string } | null>(null);
	let bannerTimer: ReturnType<typeof setTimeout> | undefined;

	function flash(kind: 'success' | 'error', text: string) {
		banner = { kind, text };
		clearTimeout(bannerTimer);
		bannerTimer = setTimeout(() => (banner = null), 4000);
	}

	async function loadPeopleNames() {
		const { data } = await api.GET('/api/v1/people', { params: { query: { page: 1, pageSize: 100 } } });
		if (data) {
			const map: Record<string, string> = {};
			for (const p of data.data) map[p.id] = `${p.firstName} ${p.lastName}`;
			peopleNames = map;
		}
	}

	async function load() {
		loading = true;
		error = null;

		const { data, error: err } = await api.GET('/api/v1/projects', {
			params: {
				query: {
					page: currentPage,
					pageSize,
					...(statusFilter ? { status: statusFilter } : {})
				}
			}
		});

		if (err) {
			error = err.message ?? 'Failed to load projects';
		} else if (data) {
			projects = data.data;
			total = data.pagination.total;
		}
		loading = false;
	}

	const filtered = $derived(
		projects.filter((p) => {
			if (!search.trim()) return true;
			const q = search.trim().toLowerCase();
			return (
				p.key.toLowerCase().includes(q) ||
				p.name.toLowerCase().includes(q) ||
				(p.description ?? '').toLowerCase().includes(q)
			);
		})
	);

	const totalPages = $derived(Math.max(1, Math.ceil(total / pageSize)));

	function openProject(p: Project) {
		goto(`/projects/${p.id}`);
	}

	function openCreate() {
		showWizard = true;
	}

	function openEdit(p: Project) {
		editingProject = p;
		modalMode = 'edit';
	}

	function closeModal() {
		modalMode = null;
		editingProject = null;
	}

	function closeWizard() {
		showWizard = false;
	}

	function handleSaved(saved: Project) {
		closeModal();
		flash('success', `${saved.name} was updated.`);
		load();
	}

	function handleWizardCreated(created: Project) {
		closeWizard();
		flash('success', `${created.name} was added.`);
		goto(`/projects/${created.id}`);
	}

	function askDelete(p: Project) {
		deleteError = null;
		deletingProject = p;
	}

	async function confirmDelete() {
		if (!deletingProject) return;
		const target = deletingProject;
		deleting = true;
		deleteError = null;

		const { error: err } = await api.DELETE('/api/v1/projects/{id}', {
			params: { path: { id: target.id } }
		});

		deleting = false;
		if (err) {
			deleteError = err.message ?? 'Failed to delete project.';
			return;
		}
		deletingProject = null;
		flash('success', `${target.name} was removed.`);
		load();
	}

	function applyStatusFilter(value: string) {
		statusFilter = value as ProjectStatus | '';
		currentPage = 1;
		load();
	}

	function statusTone(status: ProjectStatus): BadgeTone {
		switch (status) {
			case 'active':
				return 'success';
			case 'planning':
				return 'info';
			case 'on_hold':
				return 'warning';
			case 'completed':
				return 'neutralMuted';
			case 'cancelled':
				return 'danger';
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
		loadPeopleNames();
	});
</script>

<section class="space-y-4">
	<div class="flex flex-wrap items-center justify-between gap-3">
		<div>
			<h1 class="text-2xl font-semibold tracking-tight">Projects</h1>
			<p class="mt-0.5 text-sm text-slate-500">
				{total}
				{total === 1 ? 'project' : 'projects'}
			</p>
		</div>
		<button type="button" class="btn-primary" onclick={openCreate}>+ Add project</button>
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

	<div class="flex flex-wrap items-center gap-3">
		<input
			class="input max-w-xs"
			type="search"
			placeholder="Search by key, name, or description…"
			bind:value={search}
		/>
		<select
			class="input w-auto"
			value={statusFilter}
			onchange={(e) => applyStatusFilter(e.currentTarget.value)}
		>
			<option value="">All statuses</option>
			<option value="planning">Planning</option>
			<option value="active">Active</option>
			<option value="on_hold">On hold</option>
			<option value="completed">Completed</option>
			<option value="cancelled">Cancelled</option>
		</select>
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
	{:else if filtered.length === 0}
		<div class="rounded-lg border border-slate-200 bg-white p-8 text-center text-slate-500">
			{#if projects.length === 0}
				No projects yet. Click "Add project" to create the first one.
			{:else}
				No projects match your search.
			{/if}
		</div>
	{:else}
		<div class="overflow-hidden rounded-lg border border-slate-200 bg-white">
			<table class="w-full text-left text-sm">
				<thead class="border-b border-slate-200 bg-slate-50 text-slate-600">
					<tr>
						<th class="px-4 py-3 font-medium">Key</th>
						<th class="px-4 py-3 font-medium">Name</th>
						<th class="px-4 py-3 font-medium">Lead</th>
						<th class="px-4 py-3 font-medium">Priority</th>
						<th class="px-4 py-3 font-medium">Health</th>
						<th class="px-4 py-3 font-medium">Status</th>
						<th class="px-4 py-3 font-medium"><span class="sr-only">Actions</span></th>
					</tr>
				</thead>
				<tbody class="divide-y divide-slate-100">
					{#each filtered as project (project.id)}
						<!-- svelte-ignore a11y_click_events_have_key_events -->
						<!-- svelte-ignore a11y_no_static_element_interactions -->
						<tr
							class="cursor-pointer hover:bg-slate-50"
							onclick={() => openProject(project)}
						>
							<td class="px-4 py-3 font-mono text-xs font-medium text-slate-700">
								<a href="/projects/{project.id}" class="hover:underline">{project.key}</a>
							</td>
							<td class="px-4 py-3 font-medium">
								<a href="/projects/{project.id}" class="hover:underline">{project.name}</a>
							</td>
							<td class="px-4 py-3 text-slate-600">
								{project.leadId ? (peopleNames[project.leadId] ?? '—') : '—'}
							</td>
							<td class="px-4 py-3 text-slate-600 capitalize">{project.priority}</td>
							<td class="px-4 py-3">
								<span class="inline-flex items-center gap-1.5 text-slate-600">
									<span class="h-2 w-2 rounded-full {healthDotClass[project.health]}"></span>
									{project.health.replace('_', ' ')}
								</span>
							</td>
							<td class="px-4 py-3">
								<Badge tone={statusTone(project.status)}>{project.status.replace('_', ' ')}</Badge>
							</td>
							<td class="px-4 py-3 text-right whitespace-nowrap" onclick={(e) => e.stopPropagation()}>
								<button type="button" class="btn-ghost px-2 py-1" onclick={() => openEdit(project)}>
									Edit
								</button>
								<button
									type="button"
									class="btn-ghost px-2 py-1 text-red-600 hover:bg-red-50 hover:text-red-700"
									onclick={() => askDelete(project)}
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

{#if modalMode === 'edit'}
	<ProjectFormModal project={editingProject} onClose={closeModal} onSaved={handleSaved} />
{/if}

{#if showWizard}
	<ProjectWizardModal onClose={closeWizard} onCreated={handleWizardCreated} />
{/if}

<ConfirmDialog
	open={deletingProject !== null}
	title="Delete project"
	message={deletingProject ? `Remove "${deletingProject.name}"? This can't be undone.` : ''}
	confirmLabel={deleting ? 'Deleting…' : 'Delete'}
	danger
	error={deleteError}
	onConfirm={confirmDelete}
	onCancel={() => (deletingProject = null)}
/>
