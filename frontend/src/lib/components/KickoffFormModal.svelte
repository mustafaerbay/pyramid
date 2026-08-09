<script lang="ts">
	import { api } from '$lib/api/client';
	import type { components } from '$lib/api/schema';
	import PersonSelect from './PersonSelect.svelte';

	type ProjectKickoff = components['schemas']['ProjectKickoff'];

	let {
		projectId,
		kickoff = null,
		onClose,
		onSaved,
		onDelete
	}: {
		projectId: string;
		kickoff?: ProjectKickoff | null;
		onClose: () => void;
		onSaved: (kickoff: ProjectKickoff) => void;
		// Shown as a "Delete kickoff" link in edit mode only — same split as
		// MilestoneFormModal/TaskFormModal, the parent owns the confirm+delete
		// flow.
		onDelete?: () => void;
	} = $props();

	// svelte-ignore state_referenced_locally
	const isEdit = kickoff !== null;
	// svelte-ignore state_referenced_locally
	const initial = kickoff;

	// -- Meeting info --
	let meetingDate = $state(initial?.meetingDate ?? '');
	let facilitatorId = $state(initial?.facilitatorId ?? '');
	let attendees = $state(initial?.attendees ?? '');

	// -- Vision & business goals --
	let businessProblem = $state(initial?.businessProblem ?? '');
	let successDefinition = $state(initial?.successDefinition ?? '');

	// -- Target audience & value proposition --
	let primaryUsers = $state(initial?.primaryUsers ?? '');
	let painPoints = $state(initial?.painPoints ?? '');
	let mvpFeature = $state(initial?.mvpFeature ?? '');

	// -- Constraints, risks & compliance --
	let constraintsNotes = $state(initial?.constraintsNotes ?? '');
	let biggestRisk = $state(initial?.biggestRisk ?? '');

	// -- Governance & communication --
	let signOffOwner = $state(initial?.signOffOwner ?? '');
	let meetingCadence = $state(initial?.meetingCadence ?? '');
	let primaryTools = $state(initial?.primaryTools ?? '');

	// -- Next steps & action items --
	let nextSessionDate = $state(initial?.nextSessionDate ?? '');

	// Local draft list, seeded from the loaded kickoff's action items (which
	// already carry any taskId set by a prior "Convert to task" click on the
	// project page — this modal never edits taskId itself, only description/
	// owner, so those links survive a save). Every save resubmits the WHOLE
	// list — same full-set-replace convention as TaskUpdate.assigneeIds — so
	// this draft array must be the complete, current set, not just additions.
	type DraftActionItem = { description: string; ownerId: string; taskId: string | null };
	let actionItems = $state<DraftActionItem[]>(
		(initial?.actionItems ?? []).map((a) => ({
			description: a.description,
			ownerId: a.ownerId ?? '',
			taskId: a.taskId ?? null
		}))
	);

	let newItemDescription = $state('');
	let newItemOwnerId = $state('');

	function addActionItem() {
		if (!newItemDescription.trim()) return;
		actionItems = [
			...actionItems,
			{ description: newItemDescription.trim(), ownerId: newItemOwnerId, taskId: null }
		];
		newItemDescription = '';
		newItemOwnerId = '';
	}

	function removeActionItem(index: number) {
		actionItems = actionItems.filter((_, i) => i !== index);
	}

	let saving = $state(false);
	let formError = $state<string | null>(null);

	async function handleSubmit(e: SubmitEvent) {
		e.preventDefault();
		formError = null;
		saving = true;

		const body = {
			meetingDate: meetingDate || undefined,
			facilitatorId: facilitatorId || undefined,
			attendees: attendees || undefined,
			businessProblem: businessProblem || undefined,
			successDefinition: successDefinition || undefined,
			primaryUsers: primaryUsers || undefined,
			painPoints: painPoints || undefined,
			mvpFeature: mvpFeature || undefined,
			constraintsNotes: constraintsNotes || undefined,
			biggestRisk: biggestRisk || undefined,
			signOffOwner: signOffOwner || undefined,
			meetingCadence: meetingCadence || undefined,
			primaryTools: primaryTools || undefined,
			nextSessionDate: nextSessionDate || undefined,
			actionItems: actionItems.map((item) => ({
				description: item.description,
				ownerId: item.ownerId || undefined,
				taskId: item.taskId || undefined
			}))
		};

		try {
			if (isEdit && kickoff) {
				const { data, error } = await api.PATCH('/api/v1/project-kickoffs/{id}', {
					params: { path: { id: kickoff.id } },
					body
				});
				if (error) {
					formError = error.message ?? 'Failed to update the kickoff.';
					return;
				}
				if (data) onSaved(data);
			} else {
				const { data, error } = await api.POST('/api/v1/project-kickoffs', {
					body: { projectId, ...body }
				});
				if (error) {
					formError = error.message ?? 'Failed to save the kickoff.';
					return;
				}
				if (data) onSaved(data);
			}
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
		aria-labelledby="kickoff-modal-title"
		onclick={(e) => e.stopPropagation()}
	>
		<div class="flex items-center justify-between">
			<h3 id="kickoff-modal-title" class="text-lg font-semibold text-slate-900">
				{isEdit ? 'Edit kickoff' : 'Run kickoff'}
			</h3>
			<button type="button" class="text-slate-400 hover:text-slate-600" onclick={onClose} aria-label="Close">
				✕
			</button>
		</div>

		<form class="mt-4 space-y-6" onsubmit={handleSubmit}>
			{#if formError}
				<div class="rounded-md border border-red-200 bg-red-50 px-3 py-2 text-sm text-red-700">
					{formError}
				</div>
			{/if}

			<div class="space-y-3">
				<h4 class="text-sm font-semibold text-slate-700">Meeting info</h4>
				<div class="grid grid-cols-1 gap-4 sm:grid-cols-2">
					<label class="block text-sm">
						<span class="mb-1 block font-medium text-slate-700">Meeting date</span>
						<input type="date" class="input" bind:value={meetingDate} />
					</label>
					<PersonSelect
						bind:value={facilitatorId}
						label="Facilitator"
						emptyLabel="Not set"
					/>
				</div>
				<label class="block text-sm">
					<span class="mb-1 block font-medium text-slate-700">Attendees</span>
					<textarea
						class="input"
						rows="2"
						bind:value={attendees}
						placeholder="Client stakeholders, PM, Lead developer, UX designer…"
					></textarea>
				</label>
			</div>

			<div class="space-y-3 border-t border-slate-100 pt-4">
				<h4 class="text-sm font-semibold text-slate-700">Vision &amp; business goals</h4>
				<label class="block text-sm">
					<span class="mb-1 block font-medium text-slate-700">
						What business problem are we trying to solve?
					</span>
					<textarea class="input" rows="2" bind:value={businessProblem}></textarea>
				</label>
				<label class="block text-sm">
					<span class="mb-1 block font-medium text-slate-700">
						What does a successful launch look like, and how will it be measured?
					</span>
					<textarea class="input" rows="2" bind:value={successDefinition}></textarea>
				</label>
			</div>

			<div class="space-y-3 border-t border-slate-100 pt-4">
				<h4 class="text-sm font-semibold text-slate-700">Target audience &amp; value proposition</h4>
				<label class="block text-sm">
					<span class="mb-1 block font-medium text-slate-700">Who are the primary end-users?</span>
					<textarea class="input" rows="2" bind:value={primaryUsers}></textarea>
				</label>
				<label class="block text-sm">
					<span class="mb-1 block font-medium text-slate-700">What are their main pain points?</span>
					<textarea class="input" rows="2" bind:value={painPoints}></textarea>
				</label>
				<label class="block text-sm">
					<span class="mb-1 block font-medium text-slate-700">
						Single most critical feature required for launch (MVP)
					</span>
					<input class="input" bind:value={mvpFeature} />
				</label>
			</div>

			<div class="space-y-3 border-t border-slate-100 pt-4">
				<h4 class="text-sm font-semibold text-slate-700">Constraints, risks &amp; compliance</h4>
				<label class="block text-sm">
					<span class="mb-1 block font-medium text-slate-700">
						Hard deadlines, fixed budgets, or compliance standards (GDPR, HIPAA, …)
					</span>
					<textarea class="input" rows="2" bind:value={constraintsNotes}></textarea>
				</label>
				<label class="block text-sm">
					<span class="mb-1 block font-medium text-slate-700">
						Biggest foreseen internal or external roadblock
					</span>
					<textarea class="input" rows="2" bind:value={biggestRisk}></textarea>
				</label>
			</div>

			<div class="space-y-3 border-t border-slate-100 pt-4">
				<h4 class="text-sm font-semibold text-slate-700">Governance &amp; communication</h4>
				<label class="block text-sm">
					<span class="mb-1 block font-medium text-slate-700">Who has final sign-off?</span>
					<input class="input" bind:value={signOffOwner} />
				</label>
				<div class="grid grid-cols-1 gap-4 sm:grid-cols-2">
					<label class="block text-sm">
						<span class="mb-1 block font-medium text-slate-700">Meeting cadence</span>
						<input class="input" bind:value={meetingCadence} placeholder="e.g. Weekly syncs" />
					</label>
					<label class="block text-sm">
						<span class="mb-1 block font-medium text-slate-700">Primary tools</span>
						<input class="input" bind:value={primaryTools} placeholder="e.g. Slack, Jira" />
					</label>
				</div>
			</div>

			<div class="space-y-3 border-t border-slate-100 pt-4">
				<h4 class="text-sm font-semibold text-slate-700">Next steps &amp; action items</h4>
				<label class="block text-sm">
					<span class="mb-1 block font-medium text-slate-700">Next requirements session</span>
					<input type="date" class="input" bind:value={nextSessionDate} />
				</label>

				{#if actionItems.length > 0}
					<ul class="divide-y divide-slate-100 rounded-md border border-slate-200">
						{#each actionItems as item, i (i)}
							<li class="flex items-center justify-between gap-2 px-3 py-2 text-sm">
								<span class="text-slate-800">
									{item.description}
									{#if item.taskId}
										<span class="ml-1.5 rounded-full bg-emerald-100 px-1.5 py-0.5 text-xs text-emerald-700">
											linked to a task
										</span>
									{/if}
								</span>
								<button
									type="button"
									class="shrink-0 text-xs text-red-600 hover:underline"
									onclick={() => removeActionItem(i)}
								>
									Remove
								</button>
							</li>
						{/each}
					</ul>
				{/if}

				<div class="grid grid-cols-1 gap-3 sm:grid-cols-3">
					<label class="block text-sm sm:col-span-2">
						<span class="mb-1 block font-medium text-slate-700">Action item</span>
						<input
							class="input"
							bind:value={newItemDescription}
							placeholder="e.g. Confirm the API rate limits with the vendor"
						/>
					</label>
					<PersonSelect bind:value={newItemOwnerId} label="Owner" emptyLabel="Unassigned" />
				</div>
				<div class="flex justify-end">
					<button type="button" class="btn-secondary" onclick={addActionItem}>+ Add action item</button>
				</div>
			</div>

			<div class="flex items-center justify-between gap-2 border-t border-slate-100 pt-4">
				{#if isEdit && onDelete}
					<button type="button" class="text-sm text-red-600 hover:text-red-700 hover:underline" onclick={onDelete}>
						Delete kickoff
					</button>
				{:else}
					<span></span>
				{/if}
				<div class="flex gap-2">
					<button type="button" class="btn-secondary" onclick={onClose}>Cancel</button>
					<button type="submit" class="btn-primary" disabled={saving}>
						{saving ? 'Saving…' : isEdit ? 'Save changes' : 'Save kickoff'}
					</button>
				</div>
			</div>
		</form>
	</div>
</div>
