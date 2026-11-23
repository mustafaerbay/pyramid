<script lang="ts">
	import { api } from '$lib/api/client';
	import type { components } from '$lib/api/schema';
	import { can } from '$lib/auth.svelte';

	type Project = components['schemas']['Project'];
	type Template = components['schemas']['ProcessTemplate'];

	let { project, onApplied, onClose }: { project: Project; onApplied: (p: Project) => void; onClose: () => void } = $props();

	let templates = $state<Template[]>([]);
	let projects = $state<{ id: string; name: string }[]>([]);
	let source = $state('');
	let name = $state('');
	let error = $state<string | null>(null);
	let note = $state<string | null>(null);

	async function load() {
		const [t, p] = await Promise.all([
			api.GET('/api/v1/process-templates'),
			api.GET('/api/v1/projects', { params: { query: { page: 1, pageSize: 100 } } })
		]);
		templates = t.data?.data ?? [];
		projects = (p.data?.data ?? []).filter((x) => x.id !== project.id).map((x) => ({ id: x.id, name: x.name }));
	}
	$effect(() => {
		load();
	});

	async function saveTemplate(e: SubmitEvent) {
		e.preventDefault();
		error = null;
		const { error: err } = await api.POST('/api/v1/process-templates', { body: { name, fromProjectId: project.id } });
		if (err) return (error = err.message ?? 'Could not save the template.');
		note = `Saved "${name}".`;
		name = '';
		load();
	}
	async function apply() {
		if (!source) return;
		const label = source.startsWith('t:') ? templates.find((t) => t.id === source.slice(2))?.name : projects.find((p) => p.id === source.slice(2))?.name;
		if (!confirm(`Replace this project's workflow, types, fields, rules and board settings with those of "${label}"?`)) return;
		error = null;
		const body = source.startsWith('t:') ? { templateId: source.slice(2) } : { fromProjectId: source.slice(2) };
		const { data, error: err } = await api.POST('/api/v1/projects/{id}/apply-process', { params: { path: { id: project.id } }, body });
		if (err || !data) return (error = err?.message ?? 'Could not apply it.');
		onApplied(data);
		onClose();
	}
	async function remove(t: Template) {
		if (!confirm(`Delete template "${t.name}"? Projects that used it keep their settings.`)) return;
		await api.DELETE('/api/v1/process-templates/{id}', { params: { path: { id: t.id } } });
		load();
	}
	const summary = (t: Template) =>
		`${t.process.workflow.map((s) => s.label).join(' → ')} · ${t.process.workItemTypes.length} own types · ${t.process.customFields.length} fields`;
</script>

<div class="fixed inset-0 z-40 flex items-end justify-center bg-slate-900/40 sm:items-center sm:p-4" role="dialog" aria-modal="true" aria-label="Process">
	<div class="max-h-[90vh] w-full overflow-y-auto rounded-t-xl bg-white p-5 shadow-xl sm:max-w-xl sm:rounded-xl">
		<div class="mb-1 flex items-center justify-between">
			<h2 class="text-lg font-semibold">Process</h2>
			<button type="button" class="btn-ghost px-2" aria-label="Close" onclick={onClose}>✕</button>
		</div>
		<p class="mb-3 text-sm text-slate-500">A process is a project's workflow, work item types, custom fields, rules and board settings.</p>
		{#if error}<p class="mb-2 rounded-md border border-red-200 bg-red-50 p-2 text-sm text-red-700" role="alert">{error}</p>{/if}
		{#if note}<p class="mb-2 text-sm text-emerald-700" role="status">{note}</p>{/if}

		<h3 class="text-sm font-semibold">Use another process here</h3>
		<div class="mt-2 flex flex-wrap gap-2 text-sm">
			<select class="input min-w-0 flex-1" aria-label="Process to use" bind:value={source}>
				<option value="">Pick a template or project…</option>
				{#if templates.length}<optgroup label="Templates">{#each templates as t (t.id)}<option value="t:{t.id}">{t.name}</option>{/each}</optgroup>{/if}
				{#if projects.length}<optgroup label="Copy from a project">{#each projects as p (p.id)}<option value="p:{p.id}">{p.name}</option>{/each}</optgroup>{/if}
			</select>
			<button type="button" class="btn-primary" disabled={!source} onclick={apply}>Apply</button>
		</div>

		<h3 class="mt-5 text-sm font-semibold">Templates</h3>
		<ul class="mt-2 divide-y divide-slate-100 rounded-md border border-slate-200 text-sm">
			{#each templates as t (t.id)}
				<li class="flex items-start gap-2 px-3 py-2">
					<span class="min-w-0 flex-1"><span class="font-medium">{t.name}</span><span class="block truncate text-xs text-slate-500">{summary(t)}</span></span>
					{#if can('manager')}<button type="button" class="text-xs text-red-600 hover:underline" onclick={() => remove(t)}>Delete</button>{/if}
				</li>
			{:else}
				<li class="px-3 py-2 text-slate-500">No templates yet.</li>
			{/each}
		</ul>
		{#if can('manager')}
			<form class="mt-3 flex gap-2 text-sm" onsubmit={saveTemplate}>
				<input class="input min-w-0 flex-1" placeholder="Save this project's process as…" aria-label="Template name" required maxlength="60" bind:value={name} />
				<button type="submit" class="btn-secondary">Save as template</button>
			</form>
		{/if}
	</div>
</div>
