<script lang="ts">
	import { api } from '$lib/api/client';
	import type { components } from '$lib/api/schema';
	import PersonSelect from '$lib/components/PersonSelect.svelte';

	type Project = components['schemas']['Project'];
	type ProjectStatus = components['schemas']['ProjectStatus'];
	type ProjectPriority = components['schemas']['ProjectPriority'];
	type ProjectHealth = components['schemas']['ProjectHealth'];

	let {
		project = null,
		onClose,
		onSaved
	}: {
		project?: Project | null;
		onClose: () => void;
		onSaved: (project: Project) => void;
	} = $props();

	// svelte-ignore state_referenced_locally
	const isEdit = project !== null;

	// svelte-ignore state_referenced_locally
	const initial = project;

	let key = $state(initial?.key ?? '');
	let name = $state(initial?.name ?? '');
	let description = $state(initial?.description ?? '');
	let status = $state<ProjectStatus>(initial?.status ?? 'planning');
	let priority = $state<ProjectPriority>(initial?.priority ?? 'medium');
	let health = $state<ProjectHealth>(initial?.health ?? 'on_track');
	let leadId = $state(initial?.leadId ?? '');
	let startDate = $state(initial?.startDate ?? '');
	let targetDate = $state(initial?.targetDate ?? '');

	let saving = $state(false);
	let formError = $state<string | null>(null);

	const statusOptions: ProjectStatus[] = ['planning', 'active', 'on_hold', 'completed', 'cancelled'];
	const priorityOptions: ProjectPriority[] = ['low', 'medium', 'high', 'critical'];
	const healthOptions: ProjectHealth[] = ['on_track', 'at_risk', 'off_track'];

	async function handleSubmit(e: SubmitEvent) {
		e.preventDefault();
		formError = null;

		if (!key.trim() || !name.trim()) {
			formError = 'Key and name are required.';
			return;
		}

		saving = true;
		try {
			if (isEdit && project) {
				const { data, error } = await api.PATCH('/api/v1/projects/{id}', {
					params: { path: { id: project.id } },
					body: {
						key,
						name,
						description: description || undefined,
						status,
						priority,
						health,
						leadId: leadId || undefined,
						startDate: startDate || undefined,
						targetDate: targetDate || undefined
					}
				});
				if (error) {
					formError = error.message ?? 'Failed to update project.';
					return;
				}
				if (data) onSaved(data);
			} else {
				const { data, error } = await api.POST('/api/v1/projects', {
					body: {
						key,
						name,
						description: description || undefined,
						priority,
						leadId: leadId || undefined,
						startDate: startDate || undefined,
						targetDate: targetDate || undefined
					}
				});
				if (error) {
					formError = error.message ?? 'Failed to create project.';
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
		aria-labelledby="project-modal-title"
		onclick={(e) => e.stopPropagation()}
	>
		<div class="flex items-center justify-between">
			<h3 id="project-modal-title" class="text-lg font-semibold text-slate-900">
				{isEdit ? 'Edit project' : 'Add project'}
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

			<div class="grid grid-cols-1 gap-4 sm:grid-cols-3">
				<label class="block text-sm sm:col-span-1">
					<span class="mb-1 block font-medium text-slate-700">Key *</span>
					<input class="input uppercase" bind:value={key} placeholder="PYR" required />
				</label>
				<label class="block text-sm sm:col-span-2">
					<span class="mb-1 block font-medium text-slate-700">Name *</span>
					<input class="input" bind:value={name} required />
				</label>
			</div>

			<label class="block text-sm">
				<span class="mb-1 block font-medium text-slate-700">Description</span>
				<textarea class="input" rows="2" bind:value={description}></textarea>
			</label>

			<div class="grid grid-cols-1 gap-4 sm:grid-cols-3">
				{#if isEdit}
					<label class="block text-sm">
						<span class="mb-1 block font-medium text-slate-700">Status</span>
						<select class="input" bind:value={status}>
							{#each statusOptions as opt (opt)}
								<option value={opt}>{opt.replace('_', ' ')}</option>
							{/each}
						</select>
					</label>
				{/if}
				<label class="block text-sm">
					<span class="mb-1 block font-medium text-slate-700">Priority</span>
					<select class="input" bind:value={priority}>
						{#each priorityOptions as opt (opt)}
							<option value={opt}>{opt}</option>
						{/each}
					</select>
				</label>
				{#if isEdit}
					<label class="block text-sm">
						<span class="mb-1 block font-medium text-slate-700">Health</span>
						<select class="input" bind:value={health}>
							{#each healthOptions as opt (opt)}
								<option value={opt}>{opt.replace('_', ' ')}</option>
							{/each}
						</select>
					</label>
				{/if}
			</div>

			<PersonSelect bind:value={leadId} label="Lead" emptyLabel="No lead assigned" />

			<div class="grid grid-cols-1 gap-4 sm:grid-cols-2">
				<label class="block text-sm">
					<span class="mb-1 block font-medium text-slate-700">Start date</span>
					<input type="date" class="input" bind:value={startDate} />
				</label>
				<label class="block text-sm">
					<span class="mb-1 block font-medium text-slate-700">Target date</span>
					<input type="date" class="input" bind:value={targetDate} />
				</label>
			</div>

			<div class="flex justify-end gap-2 pt-2">
				<button type="button" class="btn-secondary" onclick={onClose}>Cancel</button>
				<button type="submit" class="btn-primary" disabled={saving}>
					{saving ? 'Saving…' : isEdit ? 'Save changes' : 'Add project'}
				</button>
			</div>
		</form>
	</div>
</div>
