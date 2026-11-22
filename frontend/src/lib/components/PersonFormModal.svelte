<script lang="ts">
	import { api } from '$lib/api/client';
	import type { components } from '$lib/api/schema';
	import { PROJECT_ROLE_OPTIONS, projectRoleLabel } from '$lib/projectRoles';

	type Person = components['schemas']['Person'];
	type SeniorityLevel = components['schemas']['SeniorityLevel'];
	type EmploymentType = components['schemas']['EmploymentType'];
	type PersonStatus = components['schemas']['PersonStatus'];

	// Parent mounts this component inside an {#if}, so a fresh instance is
	// created each time the modal opens — local state below is naturally
	// re-initialized from `person` without needing an effect to reset it.
	let {
		person = null,
		onClose,
		onSaved
	}: {
		person?: Person | null;
		onClose: () => void;
		onSaved: (person: Person) => void;
	} = $props();

	// svelte-ignore state_referenced_locally
	const isEdit = person !== null;

	// Snapshot the prop once: this component is mounted fresh (via {#if} in
	// the parent) each time the modal opens, so only the initial value ever
	// matters — capturing it in a plain const makes that explicit.
	// svelte-ignore state_referenced_locally
	const initial = person;

	let firstName = $state(initial?.firstName ?? '');
	let lastName = $state(initial?.lastName ?? '');
	let email = $state(initial?.email ?? '');
	let phone = $state(initial?.phone ?? '');
	let title = $state(initial?.title ?? '');
	let seniority = $state<SeniorityLevel | ''>(initial?.seniority ?? '');
	let employmentType = $state<EmploymentType>(initial?.employmentType ?? 'full_time');
	let status = $state<PersonStatus>(initial?.status ?? 'active');
	let location = $state(initial?.location ?? '');
	let timezone = $state(initial?.timezone ?? '');
	let hireDate = $state(initial?.hireDate ?? '');

	let saving = $state(false);
	let formError = $state<string | null>(null);

	const seniorityOptions: SeniorityLevel[] = [
		'intern',
		'junior',
		'mid',
		'senior',
		'staff',
		'principal',
		'lead',
		'manager',
		'director'
	];
	const employmentTypeOptions: EmploymentType[] = [
		'full_time',
		'part_time',
		'contractor',
		'intern'
	];
	const statusOptions: PersonStatus[] = ['active', 'on_leave', 'offboarding', 'inactive'];

	// Job title is a free-text column (not tied to the ProjectRole enum at the
	// DB level, since a Person doesn't have to be on any project), but the
	// user asked for the same 10 title strings used for Project Role here too
	// so the two lists read consistently. Reuses projectRoles.ts's labels
	// rather than the raw enum keys, since this field stores display strings.
	const titleOptions = PROJECT_ROLE_OPTIONS.map((opt) => projectRoleLabel[opt]);
	// If an existing person's title isn't one of the 10 (legacy free-text
	// data), keep it selectable so editing doesn't silently clobber it.
	// svelte-ignore state_referenced_locally
	const hasLegacyTitle = title !== '' && !titleOptions.includes(title);

	async function handleSubmit(e: SubmitEvent) {
		e.preventDefault();
		formError = null;

		if (!firstName.trim() || !lastName.trim() || !email.trim()) {
			formError = 'First name, last name, and email are required.';
			return;
		}

		saving = true;
		try {
			if (isEdit && person) {
				const { data, error } = await api.PATCH('/api/v1/people/{id}', {
					params: { path: { id: person.id } },
					body: {
						firstName,
						lastName,
						email,
						phone: phone || undefined,
						title: title || undefined,
						seniority: seniority || undefined,
						employmentType,
						status,
						location: location || undefined,
						timezone: timezone || undefined,
						hireDate: hireDate || undefined
					}
				});
				if (error) {
					formError = error.message ?? 'Failed to update person.';
					return;
				}
				if (data) onSaved(data);
			} else {
				const { data, error } = await api.POST('/api/v1/people', {
					body: {
						firstName,
						lastName,
						email,
						phone: phone || undefined,
						title: title || undefined,
						seniority: seniority || undefined,
						employmentType,
						location: location || undefined,
						timezone: timezone || undefined,
						hireDate: hireDate || undefined
					}
				});
				if (error) {
					formError = error.message ?? 'Failed to create person.';
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
	class="fixed inset-0 z-50 flex items-center justify-center bg-slate-900/40 sm:p-4"
	role="presentation"
	onclick={onClose}
>
	<div
		class="h-dvh max-h-dvh w-full max-w-lg overflow-y-auto bg-white p-4 sm:p-6 shadow-xl sm:h-auto sm:max-h-[90vh] sm:rounded-lg"
		role="dialog"
		tabindex="-1"
		aria-modal="true"
		aria-labelledby="person-modal-title"
		onclick={(e) => e.stopPropagation()}
	>
		<div class="flex items-center justify-between">
			<h3 id="person-modal-title" class="text-lg font-semibold text-slate-900">
				{isEdit ? 'Edit person' : 'Add person'}
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

			<div class="grid grid-cols-1 gap-4 sm:grid-cols-2">
				<label class="block text-sm">
					<span class="mb-1 block font-medium text-slate-700">First name *</span>
					<input class="input" bind:value={firstName} required />
				</label>
				<label class="block text-sm">
					<span class="mb-1 block font-medium text-slate-700">Last name *</span>
					<input class="input" bind:value={lastName} required />
				</label>
			</div>

			<label class="block text-sm">
				<span class="mb-1 block font-medium text-slate-700">Email *</span>
				<input type="email" class="input" bind:value={email} required />
			</label>

			<div class="grid grid-cols-1 gap-4 sm:grid-cols-2">
				<label class="block text-sm">
					<span class="mb-1 block font-medium text-slate-700">Title</span>
					<select class="input" bind:value={title}>
						<option value="">—</option>
						{#each titleOptions as opt (opt)}
							<option value={opt}>{opt}</option>
						{/each}
						{#if hasLegacyTitle}
							<option value={title}>{title}</option>
						{/if}
					</select>
				</label>
				<label class="block text-sm">
					<span class="mb-1 block font-medium text-slate-700">Phone</span>
					<input class="input" bind:value={phone} />
				</label>
			</div>

			<div class="grid grid-cols-1 gap-4 sm:grid-cols-2">
				<label class="block text-sm">
					<span class="mb-1 block font-medium text-slate-700">Seniority</span>
					<select class="input" bind:value={seniority}>
						<option value="">—</option>
						{#each seniorityOptions as opt (opt)}
							<option value={opt}>{opt}</option>
						{/each}
					</select>
				</label>
				<label class="block text-sm">
					<span class="mb-1 block font-medium text-slate-700">Employment type</span>
					<select class="input" bind:value={employmentType}>
						{#each employmentTypeOptions as opt (opt)}
							<option value={opt}>{opt.replace('_', ' ')}</option>
						{/each}
					</select>
				</label>
			</div>

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

			<div class="grid grid-cols-1 gap-4 sm:grid-cols-2">
				<label class="block text-sm">
					<span class="mb-1 block font-medium text-slate-700">Location</span>
					<input class="input" bind:value={location} placeholder="e.g. Istanbul" />
				</label>
				<label class="block text-sm">
					<span class="mb-1 block font-medium text-slate-700">Timezone</span>
					<input class="input" bind:value={timezone} placeholder="e.g. Europe/Istanbul" />
				</label>
			</div>

			<label class="block text-sm">
				<span class="mb-1 block font-medium text-slate-700">Hire date</span>
				<input type="date" class="input" bind:value={hireDate} />
			</label>

			<div class="flex justify-end gap-2 pt-2">
				<button type="button" class="btn-secondary" onclick={onClose}>Cancel</button>
				<button type="submit" class="btn-primary" disabled={saving}>
					{saving ? 'Saving…' : isEdit ? 'Save changes' : 'Add person'}
				</button>
			</div>
		</form>
	</div>
</div>
