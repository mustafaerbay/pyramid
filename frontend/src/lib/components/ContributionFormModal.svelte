<script lang="ts">
	import { onMount } from 'svelte';
	import { api } from '$lib/api/client';
	import type { components } from '$lib/api/schema';
	import PersonSelect from '$lib/components/PersonSelect.svelte';

	type Contribution = components['schemas']['Contribution'];
	type ContributionType = components['schemas']['ContributionType'];
	type Project = components['schemas']['Project'];

	let {
		contribution = null,
		initialProjectId = '',
		onClose,
		onSaved
	}: {
		contribution?: Contribution | null;
		// Preselects the project when launched from a project's own dashboard.
		// Ignored in edit mode, where the contribution's own project applies.
		initialProjectId?: string;
		onClose: () => void;
		onSaved: (contribution: Contribution) => void;
	} = $props();

	// svelte-ignore state_referenced_locally
	const isEdit = contribution !== null;

	// svelte-ignore state_referenced_locally
	const initial = contribution;

	let personId = $state(initial?.personId ?? '');
	// svelte-ignore state_referenced_locally
	let projectId = $state(initial?.projectId ?? initialProjectId);
	let type = $state<ContributionType>(initial?.type ?? 'commit');
	let title = $state(initial?.title ?? '');
	let url = $state(initial?.url ?? '');
	let weight = $state(initial?.weight ?? 1);
	// occurredAt is date-time in the API; keep the date portion in a plain
	// date input for simplicity and send it back as an ISO string.
	let occurredDate = $state(initial?.occurredAt ? initial.occurredAt.slice(0, 10) : '');

	let projects = $state<Project[]>([]);

	let saving = $state(false);
	let formError = $state<string | null>(null);

	const typeOptions: ContributionType[] = [
		'commit',
		'pull_request',
		'code_review',
		'issue',
		'design',
		'documentation',
		'deployment',
		'support',
		'other'
	];

	onMount(async () => {
		const { data } = await api.GET('/api/v1/projects', { params: { query: { page: 1, pageSize: 100 } } });
		if (data) projects = data.data;
	});

	async function handleSubmit(e: SubmitEvent) {
		e.preventDefault();
		formError = null;

		if (!personId) {
			formError = 'Person is required.';
			return;
		}

		const occurredAt = occurredDate ? new Date(`${occurredDate}T12:00:00Z`).toISOString() : undefined;

		saving = true;
		try {
			if (isEdit && contribution) {
				const { data, error } = await api.PATCH('/api/v1/contributions/{id}', {
					params: { path: { id: contribution.id } },
					body: {
						personId,
						projectId: projectId || undefined,
						type,
						title: title || undefined,
						url: url || undefined,
						weight,
						occurredAt
					}
				});
				if (error) {
					formError = error.message ?? 'Failed to update contribution.';
					return;
				}
				if (data) onSaved(data);
			} else {
				const { data, error } = await api.POST('/api/v1/contributions', {
					body: {
						personId,
						projectId: projectId || undefined,
						type,
						title: title || undefined,
						url: url || undefined,
						weight,
						occurredAt
					}
				});
				if (error) {
					formError = error.message ?? 'Failed to log contribution.';
					return;
				}
				if (data) onSaved(data);
			}
		} finally {
			saving = false;
		}
	}
</script>

<svelte:window onkeydown={(e) => e.key === 'Escape' && onClose()} />

<!-- svelte-ignore a11y_click_events_have_key_events -->
<!-- svelte-ignore a11y_no_static_element_interactions -->
<div
	class="fixed inset-0 z-50 flex items-center justify-center bg-slate-900/40 p-4"
	role="presentation"
	onclick={onClose}
>
	<div
		class="max-h-[90vh] w-full max-w-lg overflow-y-auto rounded-lg bg-white p-6 shadow-xl"
		role="dialog"
		tabindex="-1"
		aria-modal="true"
		aria-labelledby="contribution-modal-title"
		onclick={(e) => e.stopPropagation()}
	>
		<div class="flex items-center justify-between">
			<h3 id="contribution-modal-title" class="text-lg font-semibold text-slate-900">
				{isEdit ? 'Edit contribution' : 'Log contribution'}
			</h3>
			<button
				type="button"
				class="text-slate-400 hover:text-slate-600"
				onclick={onClose}
				aria-label="Close"
			>
				✕
			</button>
		</div>

		<form class="mt-4 space-y-4" onsubmit={handleSubmit}>
			{#if formError}
				<div class="rounded-md border border-red-200 bg-red-50 px-3 py-2 text-sm text-red-700">
					{formError}
				</div>
			{/if}

			<PersonSelect bind:value={personId} label="Person" required emptyLabel="Select a person" />

			<div class="grid grid-cols-1 gap-4 sm:grid-cols-2">
				<label class="block text-sm">
					<span class="mb-1 block font-medium text-slate-700">Type</span>
					<select class="input" bind:value={type}>
						{#each typeOptions as opt (opt)}
							<option value={opt}>{opt.replace('_', ' ')}</option>
						{/each}
					</select>
				</label>
				<label class="block text-sm">
					<span class="mb-1 block font-medium text-slate-700">Project</span>
					<select class="input" bind:value={projectId}>
						<option value="">No project</option>
						{#each projects as project (project.id)}
							<option value={project.id}>{project.key} — {project.name}</option>
						{/each}
					</select>
				</label>
			</div>

			<label class="block text-sm">
				<span class="mb-1 block font-medium text-slate-700">Title</span>
				<input class="input" bind:value={title} placeholder="e.g. Fix login redirect bug" />
			</label>

			<label class="block text-sm">
				<span class="mb-1 block font-medium text-slate-700">URL</span>
				<input class="input" type="url" bind:value={url} placeholder="https://…" />
			</label>

			<div class="grid grid-cols-1 gap-4 sm:grid-cols-2">
				<label class="block text-sm">
					<span class="mb-1 block font-medium text-slate-700">Weight</span>
					<input type="number" step="0.5" min="0" class="input" bind:value={weight} />
				</label>
				<label class="block text-sm">
					<span class="mb-1 block font-medium text-slate-700">Date</span>
					<input type="date" class="input" bind:value={occurredDate} />
				</label>
			</div>

			<div class="flex justify-end gap-2 pt-2">
				<button type="button" class="btn-secondary" onclick={onClose}>Cancel</button>
				<button type="submit" class="btn-primary" disabled={saving}>
					{saving ? 'Saving…' : isEdit ? 'Save changes' : 'Log contribution'}
				</button>
			</div>
		</form>
	</div>
</div>
