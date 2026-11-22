<script lang="ts">
	import { api } from '$lib/api/client';
	import type { components } from '$lib/api/schema';

	type Version = components['schemas']['Version'];
	type VersionStatus = components['schemas']['VersionStatus'];

	let {
		projectId,
		version = null,
		onClose,
		onSaved,
		onDelete
	}: {
		projectId: string;
		version?: Version | null;
		onClose: () => void;
		onSaved: (version: Version) => void;
		// "Delete version" link in edit mode; the parent owns the confirm flow.
		onDelete?: () => void;
	} = $props();

	// svelte-ignore state_referenced_locally
	const isEdit = version !== null;

	// svelte-ignore state_referenced_locally
	const initial = version;

	let name = $state(initial?.name ?? '');
	let description = $state(initial?.description ?? '');
	let targetDate = $state(initial?.targetDate ?? '');
	// New versions always start planned server-side; status and released
	// date are edit-only, like a milestone's status.
	let status = $state<VersionStatus>(initial?.status ?? 'planned');
	let releasedAt = $state(initial?.releasedAt ?? '');

	let saving = $state(false);
	let formError = $state<string | null>(null);

	const statusOptions: VersionStatus[] = ['planned', 'in_progress', 'released', 'cancelled'];

	async function handleSubmit(e: SubmitEvent) {
		e.preventDefault();
		formError = null;

		if (!name.trim()) {
			formError = 'Name is required.';
			return;
		}

		saving = true;
		try {
			if (isEdit && version) {
				const { data, error } = await api.PATCH('/api/v1/versions/{id}', {
					params: { path: { id: version.id } },
					body: {
						name: name.trim(),
						description: description || undefined,
						status,
						targetDate: targetDate || undefined,
						releasedAt: releasedAt || undefined
					}
				});
				if (error) {
					formError = error.message ?? 'Failed to update version.';
					return;
				}
				if (data) onSaved(data);
			} else {
				const { data, error } = await api.POST('/api/v1/versions', {
					body: {
						projectId,
						name: name.trim(),
						description: description || undefined,
						targetDate: targetDate || undefined
					}
				});
				if (error) {
					formError = error.message ?? 'Failed to create version.';
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
		class="max-h-[90vh] w-full max-w-md overflow-y-auto rounded-lg bg-white p-6 shadow-xl"
		role="dialog"
		tabindex="-1"
		aria-modal="true"
		aria-labelledby="version-modal-title"
		onclick={(e) => e.stopPropagation()}
	>
		<div class="flex items-center justify-between">
			<h3 id="version-modal-title" class="text-lg font-semibold text-slate-900">
				{isEdit ? 'Edit version' : 'Add version'}
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

			<label class="block text-sm">
				<span class="mb-1 block font-medium text-slate-700">Name *</span>
				<input class="input" bind:value={name} placeholder="e.g. v1.0" required />
			</label>

			<label class="block text-sm">
				<span class="mb-1 block font-medium text-slate-700">Target release date</span>
				<input type="date" class="input" bind:value={targetDate} />
			</label>

			<label class="block text-sm">
				<span class="mb-1 block font-medium text-slate-700">Description</span>
				<textarea class="input" rows="2" bind:value={description}></textarea>
			</label>

			{#if isEdit}
				<div class="grid grid-cols-1 gap-4 sm:grid-cols-2">
					<label class="block text-sm">
						<span class="mb-1 block font-medium text-slate-700">Status</span>
						<select class="input" bind:value={status}>
							{#each statusOptions as opt (opt)}
								<option value={opt}>{opt.replace('_', ' ')}</option>
							{/each}
						</select>
					</label>
					<label class="block text-sm">
						<span class="mb-1 block font-medium text-slate-700">Released on</span>
						<input type="date" class="input" bind:value={releasedAt} />
						<span class="mt-1 block text-xs text-slate-400">Set to today when marked released, if empty.</span>
					</label>
				</div>
			{/if}

			<div class="flex items-center justify-between gap-2 pt-2">
				{#if isEdit && onDelete}
					<button
						type="button"
						class="text-sm text-red-600 hover:text-red-700 hover:underline"
						onclick={onDelete}
					>
						Delete version
					</button>
				{:else}
					<span></span>
				{/if}
				<div class="flex gap-2">
					<button type="button" class="btn-secondary" onclick={onClose}>Cancel</button>
					<button type="submit" class="btn-primary" disabled={saving}>
						{saving ? 'Saving…' : isEdit ? 'Save changes' : 'Add version'}
					</button>
				</div>
			</div>
		</form>
	</div>
</div>
