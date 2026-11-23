<script lang="ts">
	import type { components } from '$lib/api/schema';

	type Step = components['schemas']['TestStep'];
	let {
		title = $bindable(''),
		steps = $bindable<Step[]>([]),
		taskId = $bindable(''),
		items,
		onSave,
		onCancel
	}: {
		title: string;
		steps: Step[];
		taskId: string;
		items: { id: string; label: string }[];
		onSave: () => void;
		onCancel: () => void;
	} = $props();
</script>

<form
	class="space-y-2 rounded-md border border-slate-200 bg-slate-50 p-3 text-sm"
	onsubmit={(e) => {
		e.preventDefault();
		onSave();
	}}
>
	<input class="input" placeholder="Test case title" aria-label="Test case title" required maxlength="300" bind:value={title} />
	<select class="input" aria-label="Item it tests" bind:value={taskId}>
		<option value="">Tests no particular item</option>
		{#each items as i (i.id)}<option value={i.id}>{i.label}</option>{/each}
	</select>
	<ol class="space-y-1.5">
		{#each steps as st, i (i)}
			<li class="flex gap-2">
				<span class="w-5 pt-2 text-right text-xs text-slate-500">{i + 1}.</span>
				<input class="input flex-1" placeholder="Action" aria-label="Step {i + 1} action" maxlength="2000" bind:value={st.action} />
				<input class="input flex-1" placeholder="Expected result" aria-label="Step {i + 1} expected" maxlength="2000" bind:value={st.expected} />
				<button type="button" class="px-1 text-slate-400 hover:text-red-600" aria-label="Remove step {i + 1}" onclick={() => (steps = steps.filter((_, j) => j !== i))}>✕</button>
			</li>
		{/each}
	</ol>
	<div class="flex gap-2">
		<button type="button" class="btn-ghost px-2 py-1 text-xs" onclick={() => (steps = [...steps, { action: '', expected: '' }])}>+ Step</button>
		<span class="ml-auto flex gap-2">
			<button type="submit" class="btn-primary px-3 py-1 text-xs">Save case</button>
			<button type="button" class="btn-ghost px-3 py-1 text-xs" onclick={onCancel}>Cancel</button>
		</span>
	</div>
</form>
