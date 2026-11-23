<script lang="ts">
	import type { components } from '$lib/api/schema';
	import { fieldKey } from '$lib/customFields';

	type Field = components['schemas']['CustomField'];

	let { fields, onSave, onClose }: { fields: Field[]; onSave: (f: Field[]) => Promise<string | null>; onClose: () => void } = $props();

	// svelte-ignore state_referenced_locally
	let list = $state<Field[]>(fields.map((f) => ({ ...f, options: f.options ? [...f.options] : undefined })));
	let label = $state('');
	let type = $state<Field['type']>('text');
	let options = $state('');
	let required = $state(false);
	let error = $state<string | null>(null);
	let saving = $state(false);

	function add(e: SubmitEvent) {
		e.preventDefault();
		error = null;
		const opts = options
			.split(',')
			.map((o) => o.trim())
			.filter(Boolean);
		if (type === 'select' && opts.length === 0) return (error = 'A list needs at least one option.');
		list = [
			...list,
			{ key: fieldKey(label, list.map((f) => f.key)), label: label.trim(), type, required, ...(type === 'select' ? { options: [...new Set(opts)] } : {}) }
		];
		label = '';
		options = '';
		required = false;
	}
	async function save() {
		saving = true;
		error = await onSave(list);
		saving = false;
		if (!error) onClose();
	}
	const TYPE_LABEL: Record<string, string> = { text: 'Text', number: 'Number', date: 'Date', select: 'List' };
</script>

<div class="fixed inset-0 z-40 flex items-end justify-center bg-slate-900/40 sm:items-center sm:p-4" role="dialog" aria-modal="true" aria-label="Custom fields">
	<div class="max-h-[90vh] w-full overflow-y-auto rounded-t-xl bg-white p-5 shadow-xl sm:max-w-lg sm:rounded-xl">
		<div class="mb-1 flex items-center justify-between">
			<h2 class="text-lg font-semibold">Custom fields</h2>
			<button type="button" class="btn-ghost px-2" aria-label="Close" onclick={onClose}>✕</button>
		</div>
		<p class="mb-3 text-sm text-slate-500">Extra fields on every work item in this project (up to 20). Removing a field hides its values.</p>
		{#if error}<p class="mb-2 rounded-md border border-red-200 bg-red-50 p-2 text-sm text-red-700" role="alert">{error}</p>{/if}

		<ul class="mb-4 divide-y divide-slate-100 rounded-md border border-slate-200 text-sm">
			{#each list as f, i (f.key)}
				<li class="flex items-center gap-2 px-3 py-2">
					<span class="min-w-0 flex-1">
						<span class="font-medium">{f.label}</span>
						<span class="text-xs text-slate-500">· {TYPE_LABEL[f.type]}{f.options ? `: ${f.options.join(', ')}` : ''}{f.required ? ' · required' : ''}</span>
					</span>
					<label class="flex items-center gap-1 text-xs text-slate-600"><input type="checkbox" bind:checked={list[i].required} /> Required</label>
					<button type="button" class="text-xs text-red-600 hover:underline" onclick={() => (list = list.filter((x) => x.key !== f.key))}>Remove</button>
				</li>
			{:else}
				<li class="px-3 py-2 text-slate-500">No custom fields yet.</li>
			{/each}
		</ul>

		{#if list.length < 20}
			<form class="space-y-2 rounded-md bg-slate-50 p-3 text-sm" onsubmit={add}>
				<div class="flex gap-2">
					<input class="input flex-1" placeholder="Field name, e.g. Customer" aria-label="Field name" required maxlength="60" bind:value={label} />
					<select class="input w-auto" aria-label="Field type" bind:value={type}>
						<option value="text">Text</option>
						<option value="number">Number</option>
						<option value="date">Date</option>
						<option value="select">List</option>
					</select>
				</div>
				{#if type === 'select'}
					<input class="input" placeholder="Options, split with commas: Low, Medium, High" aria-label="Options" bind:value={options} />
				{/if}
				<div class="flex items-center justify-between">
					<label class="flex items-center gap-1.5 text-slate-600"><input type="checkbox" bind:checked={required} /> Required</label>
					<button type="submit" class="btn-secondary">Add field</button>
				</div>
			</form>
		{/if}

		<div class="mt-4 flex gap-2">
			<button type="button" class="btn-primary" disabled={saving} onclick={save}>{saving ? 'Saving…' : 'Save fields'}</button>
			<button type="button" class="btn-ghost" onclick={onClose}>Cancel</button>
		</div>
	</div>
</div>
