<script lang="ts">
	import type { components } from '$lib/api/schema';
	import ChipSelect from './ChipSelect.svelte';

	type Rules = components['schemas']['WorkItemRules'];
	type Field = components['schemas']['CustomField'];
	type State = components['schemas']['WorkflowState'];

	let {
		rules,
		states,
		fields,
		onSave,
		onClose
	}: { rules: Rules; states: State[]; fields: Field[]; onSave: (r: Rules) => Promise<string | null>; onClose: () => void } = $props();

	const BUILT_IN = [
		{ value: 'assignees', label: 'Assignees' },
		{ value: 'dueDate', label: 'Due date' },
		{ value: 'startDate', label: 'Start date' },
		{ value: 'storyPoints', label: 'Story points' },
		{ value: 'manDays', label: 'Man-days' },
		{ value: 'description', label: 'Description' },
		{ value: 'area', label: 'Area' },
		{ value: 'sprint', label: 'Sprint' },
		{ value: 'tags', label: 'Tags' }
	];
	// svelte-ignore state_referenced_locally
	const fieldOptions = [...BUILT_IN, ...fields.map((f) => ({ value: f.key, label: f.label }))];

	// svelte-ignore state_referenced_locally
	let required = $state<Record<string, string[]>>(Object.fromEntries(states.map((s) => [s.key, [...(rules.requiredByState[s.key] ?? [])]])));
	// svelte-ignore state_referenced_locally
	let defaults = $state<Record<string, string>>(Object.fromEntries(Object.entries(rules.defaults).map(([k, v]) => [k, String(v)])));
	// svelte-ignore state_referenced_locally
	let readOnly = $state<string[]>([...rules.readOnly]);
	let error = $state<string | null>(null);
	let saving = $state(false);

	async function save() {
		const d: Record<string, unknown> = {};
		for (const [k, v] of Object.entries(defaults)) {
			if (v === '' || v == null) continue;
			const f = fields.find((x) => x.key === k);
			d[k] = k === 'storyPoints' || f?.type === 'number' ? Number(v) : v;
		}
		const req = Object.fromEntries(Object.entries(required).filter(([, v]) => v.length));
		saving = true;
		error = await onSave({ requiredByState: req, defaults: d, readOnly });
		saving = false;
		if (!error) onClose();
	}
</script>

<div class="fixed inset-0 z-40 flex items-end justify-center bg-slate-900/40 sm:items-center sm:p-4" role="dialog" aria-modal="true" aria-label="Rules">
	<div class="max-h-[90vh] w-full overflow-y-auto rounded-t-xl bg-white p-5 shadow-xl sm:max-w-2xl sm:rounded-xl">
		<div class="mb-1 flex items-center justify-between">
			<h2 class="text-lg font-semibold">Work item rules</h2>
			<button type="button" class="btn-ghost px-2" aria-label="Close" onclick={onClose}>✕</button>
		</div>
		{#if error}<p class="mb-2 rounded-md border border-red-200 bg-red-50 p-2 text-sm text-red-700" role="alert">{error}</p>{/if}

		<h3 class="mt-2 text-sm font-semibold">Fields needed to enter a state</h3>
		<p class="mb-2 text-xs text-slate-500">An item can't move into the state until these are filled.</p>
		<div class="grid gap-3 sm:grid-cols-2">
			{#each states as s (s.key)}
				<div class="text-sm">
					<div class="mb-1 font-medium text-slate-700">{s.label}</div>
					<ChipSelect label="Required" options={fieldOptions} bind:values={required[s.key]} />
				</div>
			{/each}
		</div>

		<h3 class="mt-5 text-sm font-semibold">Defaults for new items</h3>
		<div class="mt-2 grid gap-3 sm:grid-cols-2">
			<label class="block text-sm">
				<span class="mb-1 block text-slate-700">Priority</span>
				<select class="input" bind:value={defaults.priority}>
					<option value="">—</option>
					{#each ['low', 'medium', 'high', 'urgent'] as p (p)}<option value={p}>{p}</option>{/each}
				</select>
			</label>
			<label class="block text-sm">
				<span class="mb-1 block text-slate-700">Story points</span>
				<input class="input" type="number" min="0" max="999" bind:value={defaults.storyPoints} />
			</label>
			{#each fields as f (f.key)}
				<label class="block text-sm">
					<span class="mb-1 block text-slate-700">{f.label}</span>
					{#if f.type === 'select'}
						<select class="input" bind:value={defaults[f.key]}>
							<option value="">—</option>
							{#each f.options ?? [] as o (o)}<option value={o}>{o}</option>{/each}
						</select>
					{:else}
						<input class="input" type={f.type === 'number' ? 'number' : f.type === 'date' ? 'date' : 'text'} bind:value={defaults[f.key]} />
					{/if}
				</label>
			{/each}
		</div>

		{#if fields.length}
			<h3 class="mt-5 text-sm font-semibold">Read-only after creation</h3>
			<p class="mb-2 text-xs text-slate-500">Only managers and admins can change these once an item exists.</p>
			<div class="flex flex-wrap gap-3 text-sm">
				{#each fields as f (f.key)}
					<label class="flex items-center gap-1.5">
						<input type="checkbox" checked={readOnly.includes(f.key)} onchange={(e) => (readOnly = e.currentTarget.checked ? [...readOnly, f.key] : readOnly.filter((k) => k !== f.key))} />
						{f.label}
					</label>
				{/each}
			</div>
		{/if}

		<div class="mt-5 flex gap-2">
			<button type="button" class="btn-primary" disabled={saving} onclick={save}>{saving ? 'Saving…' : 'Save rules'}</button>
			<button type="button" class="btn-ghost" onclick={onClose}>Cancel</button>
		</div>
	</div>
</div>
