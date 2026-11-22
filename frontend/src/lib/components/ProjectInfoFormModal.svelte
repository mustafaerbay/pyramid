<script lang="ts">
	import { api } from '$lib/api/client';
	import type { components } from '$lib/api/schema';

	type Project = components['schemas']['Project'];

	// This modal only ever edits an existing project (it's opened from a
	// button on the project detail page, so a project always exists) — PATCH
	// only, no create path, unlike ProjectFormModal.
	let {
		project,
		onClose,
		onSaved
	}: {
		project: Project;
		onClose: () => void;
		onSaved: (project: Project) => void;
	} = $props();

	// svelte-ignore state_referenced_locally
	const initial = project;

	let sponsor = $state(initial.sponsor ?? '');
	let projectDirector = $state(initial.projectDirector ?? '');
	let reportPeriod = $state(initial.reportPeriod ?? '');
	let hqPm = $state(initial.hqPm ?? '');
	let turkeyPm = $state(initial.turkeyPm ?? '');
	let chineseInterface = $state(initial.chineseInterface ?? '');
	let hqInterface = $state(initial.hqInterface ?? '');
	let turkeyQa = $state(initial.turkeyQa ?? '');
	let accountName = $state(initial.accountName ?? '');
	let hqSe = $state(initial.hqSe ?? '');
	let turkeySe = $state(initial.turkeySe ?? '');
	let projectCode = $state(initial.projectCode ?? '');
	// Bound to <input type="number">, which Svelte coerces to an actual
	// number on input (not a string) once something's typed, while an empty
	// box binds back to '' — so the type here has to allow both, and
	// toIntOrUndefined below has to handle either shape.
	let teamSizeActual = $state<number | string>(initial.teamSizeActual ?? '');
	let teamSizePlanned = $state<number | string>(initial.teamSizePlanned ?? '');

	let saving = $state(false);
	let formError = $state<string | null>(null);

	function toIntOrUndefined(v: number | string): number | undefined {
		if (v === '' || v === null || v === undefined) return undefined;
		const n = Number(v);
		return Number.isFinite(n) ? Math.trunc(n) : undefined;
	}

	async function handleSubmit(e: SubmitEvent) {
		e.preventDefault();
		formError = null;
		saving = true;

		try {
			const { data, error } = await api.PATCH('/api/v1/projects/{id}', {
				params: { path: { id: project.id } },
				body: {
					sponsor: sponsor || undefined,
					projectDirector: projectDirector || undefined,
					reportPeriod: reportPeriod || undefined,
					hqPm: hqPm || undefined,
					turkeyPm: turkeyPm || undefined,
					chineseInterface: chineseInterface || undefined,
					hqInterface: hqInterface || undefined,
					turkeyQa: turkeyQa || undefined,
					accountName: accountName || undefined,
					hqSe: hqSe || undefined,
					turkeySe: turkeySe || undefined,
					projectCode: projectCode || undefined,
					teamSizeActual: toIntOrUndefined(teamSizeActual),
					teamSizePlanned: toIntOrUndefined(teamSizePlanned)
				}
			});
			if (error) {
				formError = error.message ?? 'Failed to update project info.';
				return;
			}
			if (data) onSaved(data);
		} finally {
			saving = false;
		}
	}
</script>

<svelte:window onkeydown={(e) => e.key === 'Escape' && !saving && onClose()} />

<!-- svelte-ignore a11y_click_events_have_key_events -->
<!-- svelte-ignore a11y_no_static_element_interactions -->
<div
	class="fixed inset-0 z-50 flex items-center justify-center bg-slate-900/40 p-4"
	role="presentation"
	onclick={() => !saving && onClose()}
>
	<div
		class="max-h-[90vh] w-full max-w-2xl overflow-y-auto rounded-lg bg-white p-6 shadow-xl"
		role="dialog"
		tabindex="-1"
		aria-modal="true"
		aria-labelledby="project-info-modal-title"
		onclick={(e) => e.stopPropagation()}
	>
		<div class="flex items-center justify-between">
			<h3 id="project-info-modal-title" class="text-lg font-semibold text-slate-900">
				Edit project info
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
		<p class="mt-1 text-sm text-slate-500">
			Stakeholder and reporting details shown at the top of the project page.
		</p>

		<form class="mt-4 space-y-4" onsubmit={handleSubmit}>
			{#if formError}
				<div class="rounded-md border border-red-200 bg-red-50 px-3 py-2 text-sm text-red-700">
					{formError}
				</div>
			{/if}

			<div class="grid grid-cols-1 gap-4 sm:grid-cols-2">
				<label class="block text-sm">
					<span class="mb-1 block font-medium text-slate-700">Sponsor</span>
					<input class="input" bind:value={sponsor} />
				</label>
				<label class="block text-sm">
					<span class="mb-1 block font-medium text-slate-700">Project Director</span>
					<input class="input" bind:value={projectDirector} />
				</label>
				<label class="block text-sm">
					<span class="mb-1 block font-medium text-slate-700">Report Period</span>
					<input class="input" bind:value={reportPeriod} placeholder="e.g. Q3 2026" />
				</label>
				<label class="block text-sm">
					<span class="mb-1 block font-medium text-slate-700">Account Name</span>
					<input class="input" bind:value={accountName} />
				</label>
				<label class="block text-sm">
					<span class="mb-1 block font-medium text-slate-700">Project Code</span>
					<input class="input" bind:value={projectCode} />
				</label>
			</div>

			<div class="space-y-3 border-t border-slate-100 pt-4">
				<h4 class="text-sm font-semibold text-slate-700">PM &amp; interface contacts</h4>
				<div class="grid grid-cols-1 gap-4 sm:grid-cols-2">
					<label class="block text-sm">
						<span class="mb-1 block font-medium text-slate-700">HQ PM</span>
						<input class="input" bind:value={hqPm} />
					</label>
					<label class="block text-sm">
						<span class="mb-1 block font-medium text-slate-700">Turkey PM</span>
						<input class="input" bind:value={turkeyPm} />
					</label>
					<label class="block text-sm">
						<span class="mb-1 block font-medium text-slate-700">HQ SE</span>
						<input class="input" bind:value={hqSe} />
					</label>
					<label class="block text-sm">
						<span class="mb-1 block font-medium text-slate-700">Turkey SE</span>
						<input class="input" bind:value={turkeySe} />
					</label>
					<label class="block text-sm">
						<span class="mb-1 block font-medium text-slate-700">HQ Interface</span>
						<input class="input" bind:value={hqInterface} />
					</label>
					<label class="block text-sm">
						<span class="mb-1 block font-medium text-slate-700">Chinese Interface</span>
						<input class="input" bind:value={chineseInterface} />
					</label>
					<label class="block text-sm">
						<span class="mb-1 block font-medium text-slate-700">Turkey QA</span>
						<input class="input" bind:value={turkeyQa} />
					</label>
				</div>
			</div>

			<div class="space-y-3 border-t border-slate-100 pt-4">
				<h4 class="text-sm font-semibold text-slate-700">Team size</h4>
				<div class="grid grid-cols-1 gap-4 sm:grid-cols-2">
					<label class="block text-sm">
						<span class="mb-1 block font-medium text-slate-700">Actual</span>
						<input type="number" min="0" step="1" class="input" bind:value={teamSizeActual} />
					</label>
					<label class="block text-sm">
						<span class="mb-1 block font-medium text-slate-700">Planned</span>
						<input type="number" min="0" step="1" class="input" bind:value={teamSizePlanned} />
					</label>
				</div>
			</div>

			<div class="flex justify-end gap-2 border-t border-slate-100 pt-4">
				<button type="button" class="btn-secondary" onclick={onClose}>Cancel</button>
				<button type="submit" class="btn-primary" disabled={saving}>
					{saving ? 'Saving…' : 'Save changes'}
				</button>
			</div>
		</form>
	</div>
</div>
