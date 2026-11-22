<script lang="ts">
	import { api } from '$lib/api/client';
	import type { components } from '$lib/api/schema';

	type WeeklySummary = components['schemas']['WeeklySummary'];

	let {
		projectId,
		entry = null,
		onClose,
		onSaved,
		onDelete
	}: {
		projectId: string;
		entry?: WeeklySummary | null;
		onClose: () => void;
		onSaved: (entry: WeeklySummary) => void;
		// Shown as a "Delete entry" link in edit mode only. The parent owns the
		// actual confirm-and-delete flow (same split as Milestone/Task).
		onDelete?: () => void;
	} = $props();

	// svelte-ignore state_referenced_locally
	const isEdit = entry !== null;

	// svelte-ignore state_referenced_locally
	const initial = entry;

	// Defaults a new entry's week to the Monday of the current week, since
	// this is meant to be a "log a weekly update" flow, not a blank-slate date
	// picker every time.
	function mondayOfCurrentWeek(): string {
		const now = new Date();
		const day = now.getUTCDay(); // 0 = Sunday .. 6 = Saturday
		const diffToMonday = day === 0 ? -6 : 1 - day;
		const monday = new Date(Date.UTC(now.getUTCFullYear(), now.getUTCMonth(), now.getUTCDate() + diffToMonday));
		return monday.toISOString().slice(0, 10);
	}

	let weekOf = $state(initial?.weekOf ?? mondayOfCurrentWeek());
	let summary = $state(initial?.summary ?? '');

	let saving = $state(false);
	let formError = $state<string | null>(null);

	async function handleSubmit(e: SubmitEvent) {
		e.preventDefault();
		formError = null;

		if (!weekOf) {
			formError = 'Week is required.';
			return;
		}
		if (!summary.trim()) {
			formError = 'Summary is required.';
			return;
		}

		saving = true;
		try {
			if (isEdit && entry) {
				const { data, error } = await api.PATCH('/api/v1/weekly-summaries/{id}', {
					params: { path: { id: entry.id } },
					body: { weekOf, summary }
				});
				if (error) {
					formError = error.message ?? 'Failed to update weekly summary.';
					return;
				}
				if (data) onSaved(data);
			} else {
				const { data, error } = await api.POST('/api/v1/weekly-summaries', {
					body: { projectId, weekOf, summary }
				});
				if (error) {
					formError = error.message ?? 'Failed to create weekly summary.';
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
		aria-labelledby="weekly-summary-modal-title"
		onclick={(e) => e.stopPropagation()}
	>
		<div class="flex items-center justify-between">
			<h3 id="weekly-summary-modal-title" class="text-lg font-semibold text-slate-900">
				{isEdit ? 'Edit weekly summary' : 'Add weekly summary'}
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
				<span class="mb-1 block font-medium text-slate-700">Week of *</span>
				<input type="date" class="input" bind:value={weekOf} required />
			</label>

			<label class="block text-sm">
				<span class="mb-1 block font-medium text-slate-700">Summary *</span>
				<textarea
					class="input"
					rows="6"
					bind:value={summary}
					placeholder="What shipped, what's in progress, what's blocked…"
					required
				></textarea>
			</label>

			<div class="flex items-center justify-between gap-2 pt-2">
				{#if isEdit && onDelete}
					<button
						type="button"
						class="text-sm text-red-600 hover:text-red-700 hover:underline"
						onclick={onDelete}
					>
						Delete entry
					</button>
				{:else}
					<span></span>
				{/if}
				<div class="flex gap-2">
					<button type="button" class="btn-secondary" onclick={onClose}>Cancel</button>
					<button type="submit" class="btn-primary" disabled={saving}>
						{saving ? 'Saving…' : isEdit ? 'Save changes' : 'Add entry'}
					</button>
				</div>
			</div>
		</form>
	</div>
</div>
