<script lang="ts">
	import { api } from '$lib/api/client';
	import type { components } from '$lib/api/schema';
	import PersonSelect from '$lib/components/PersonSelect.svelte';

	type LeaveRequest = components['schemas']['LeaveRequest'];
	type LeaveType = components['schemas']['LeaveType'];
	type LeaveStatus = components['schemas']['LeaveStatus'];

	let {
		leaveRequest = null,
		onClose,
		onSaved
	}: {
		leaveRequest?: LeaveRequest | null;
		onClose: () => void;
		onSaved: (leaveRequest: LeaveRequest) => void;
	} = $props();

	// svelte-ignore state_referenced_locally
	const isEdit = leaveRequest !== null;

	// svelte-ignore state_referenced_locally
	const initial = leaveRequest;

	let personId = $state(initial?.personId ?? '');
	let type = $state<LeaveType>(initial?.type ?? 'vacation');
	let status = $state<LeaveStatus>(initial?.status ?? 'pending');
	let startDate = $state(initial?.startDate ?? '');
	let endDate = $state(initial?.endDate ?? '');
	let reason = $state(initial?.reason ?? '');

	let saving = $state(false);
	let formError = $state<string | null>(null);

	const typeOptions: LeaveType[] = [
		'vacation',
		'sick',
		'personal',
		'parental',
		'bereavement',
		'unpaid',
		'other'
	];
	const statusOptions: LeaveStatus[] = ['pending', 'approved', 'rejected', 'cancelled'];

	async function handleSubmit(e: SubmitEvent) {
		e.preventDefault();
		formError = null;

		if (!personId || !startDate || !endDate) {
			formError = 'Person, start date, and end date are required.';
			return;
		}
		if (endDate < startDate) {
			formError = 'End date must be on or after the start date.';
			return;
		}

		saving = true;
		try {
			if (isEdit && leaveRequest) {
				const { data, error } = await api.PATCH('/api/v1/leave-requests/{id}', {
					params: { path: { id: leaveRequest.id } },
					body: {
						personId,
						type,
						status,
						startDate,
						endDate,
						reason: reason || undefined
					}
				});
				if (error) {
					formError = error.message ?? 'Failed to update leave request.';
					return;
				}
				if (data) onSaved(data);
			} else {
				const { data, error } = await api.POST('/api/v1/leave-requests', {
					body: {
						personId,
						type,
						startDate,
						endDate,
						reason: reason || undefined
					}
				});
				if (error) {
					formError = error.message ?? 'Failed to create leave request.';
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
		aria-labelledby="leave-modal-title"
		onclick={(e) => e.stopPropagation()}
	>
		<div class="flex items-center justify-between">
			<h3 id="leave-modal-title" class="text-lg font-semibold text-slate-900">
				{isEdit ? 'Edit leave request' : 'New leave request'}
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
							<option value={opt}>{opt}</option>
						{/each}
					</select>
				</label>
				{#if isEdit}
					<label class="block text-sm">
						<span class="mb-1 block font-medium text-slate-700">Status</span>
						<select class="input" bind:value={status}>
							{#each statusOptions as opt (opt)}
								<option value={opt}>{opt}</option>
							{/each}
						</select>
					</label>
				{/if}
			</div>

			<div class="grid grid-cols-1 gap-4 sm:grid-cols-2">
				<label class="block text-sm">
					<span class="mb-1 block font-medium text-slate-700">Start date *</span>
					<input type="date" class="input" bind:value={startDate} required />
				</label>
				<label class="block text-sm">
					<span class="mb-1 block font-medium text-slate-700">End date *</span>
					<input type="date" class="input" bind:value={endDate} required />
				</label>
			</div>

			<label class="block text-sm">
				<span class="mb-1 block font-medium text-slate-700">Reason</span>
				<textarea class="input" rows="2" bind:value={reason}></textarea>
			</label>

			<div class="flex justify-end gap-2 pt-2">
				<button type="button" class="btn-secondary" onclick={onClose}>Cancel</button>
				<button type="submit" class="btn-primary" disabled={saving}>
					{saving ? 'Saving…' : isEdit ? 'Save changes' : 'Submit request'}
				</button>
			</div>
		</form>
	</div>
</div>
