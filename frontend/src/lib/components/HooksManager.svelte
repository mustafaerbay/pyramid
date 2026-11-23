<script lang="ts">
	import { api } from '$lib/api/client';
	import type { components } from '$lib/api/schema';

	type Hook = components['schemas']['ProjectHook'];
	type Event = components['schemas']['HookEvent'];
	let { projectId, onClose }: { projectId: string; onClose: () => void } = $props();

	const EVENTS: { key: Event; label: string }[] = [
		{ key: 'item_created', label: 'Item created' },
		{ key: 'item_state_changed', label: 'State changed' },
		{ key: 'item_updated', label: 'Any item change' },
		{ key: 'item_commented', label: 'Comment added' },
		{ key: 'item_deleted', label: 'Item deleted' },
		{ key: 'sprint_created', label: 'Sprint planned' },
		{ key: 'sprint_updated', label: 'Sprint changed' }
	];
	let hooks = $state<Hook[]>([]);
	let error = $state<string | null>(null);
	let editing = $state<Hook | null>(null);
	let kind = $state<Hook['kind']>('slack');
	let name = $state('');
	let url = $state('');
	let secret = $state('');
	let events = $state<Event[]>(['item_created', 'item_state_changed']);
	let active = $state(true);
	let open = $state(false);

	const load = async () => (hooks = (await api.GET('/api/v1/project-hooks', { params: { query: { projectId } } })).data?.data ?? []);
	$effect(() => {
		load();
	});
	function startNew() {
		editing = null;
		kind = 'slack';
		name = url = secret = '';
		events = ['item_created', 'item_state_changed'];
		active = true;
		open = true;
	}
	function startEdit(h: Hook) {
		editing = h;
		kind = h.kind;
		name = h.name;
		url = h.url;
		secret = '';
		events = [...h.events];
		active = h.active;
		open = true;
	}
	async function save(e: SubmitEvent) {
		e.preventDefault();
		const body = { projectId, kind, name, url, events, active, ...(kind === 'webhook' && secret ? { secret } : {}) };
		const res = editing
			? await api.PATCH('/api/v1/project-hooks/{id}', { params: { path: { id: editing.id } }, body })
			: await api.POST('/api/v1/project-hooks', { body });
		if (res.error) return (error = res.error.message ?? 'Could not save.');
		error = null;
		open = false;
		load();
	}
	async function remove(h: Hook) {
		if (!confirm(`Remove "${h.name}"?`)) return;
		await api.DELETE('/api/v1/project-hooks/{id}', { params: { path: { id: h.id } } });
		load();
	}
	async function test(h: Hook) {
		const { data } = await api.POST('/api/v1/project-hooks/{id}/test', { params: { path: { id: h.id } } });
		if (data) hooks = hooks.map((x) => (x.id === data.id ? data : x));
	}
	const toggle = (ev: Event) => (events = events.includes(ev) ? events.filter((x) => x !== ev) : [...events, ev]);
</script>

<div class="fixed inset-0 z-40 flex items-end justify-center bg-slate-900/40 sm:items-center sm:p-4" role="dialog" aria-modal="true" aria-label="Integrations">
	<div class="max-h-[90vh] w-full overflow-y-auto rounded-t-xl bg-white p-5 shadow-xl sm:max-w-2xl sm:rounded-xl">
		<div class="mb-1 flex items-center justify-between">
			<h2 class="text-lg font-semibold">Integrations</h2>
			<button type="button" class="btn-ghost px-2" aria-label="Close" onclick={onClose}>✕</button>
		</div>
		<p class="mb-3 text-sm text-slate-500">
			Post to a Slack or Microsoft Teams channel (incoming webhook URL), or send signed JSON to any URL when things happen in this project.
		</p>
		{#if error}<p class="mb-2 rounded-md border border-red-200 bg-red-50 p-2 text-sm text-red-700" role="alert">{error}</p>{/if}

		<ul class="divide-y divide-slate-100 rounded-md border border-slate-200 text-sm">
			{#each hooks as h (h.id)}
				<li class="flex flex-wrap items-center gap-2 px-3 py-2">
					<span class="rounded bg-slate-100 px-1.5 text-xs font-medium uppercase text-slate-600">{h.kind}</span>
					<span class="font-medium {h.active ? '' : 'text-slate-400 line-through'}">{h.name}</span>
					<span class="text-xs text-slate-500">{h.events.length} events</span>
					{#if h.lastAt}
						<span class="text-xs {h.lastError ? 'text-red-700' : 'text-emerald-700'}" title={h.lastError ?? ''}>
							{h.lastError ? `✕ ${h.lastStatus || 'no connection'}` : `✓ ${h.lastStatus}`} · {new Date(h.lastAt).toLocaleString()}
						</span>
					{/if}
					<span class="ml-auto flex gap-1">
						<button type="button" class="btn-ghost px-2 py-1 text-xs" onclick={() => test(h)}>Send test</button>
						<button type="button" class="btn-ghost px-2 py-1 text-xs" onclick={() => startEdit(h)}>Edit</button>
						<button type="button" class="btn-ghost px-2 py-1 text-xs text-red-600" onclick={() => remove(h)}>Remove</button>
					</span>
				</li>
			{:else}
				<li class="px-3 py-4 text-center text-slate-500">No integrations yet.</li>
			{/each}
		</ul>
		{#if !open}
			<button type="button" class="btn-secondary mt-3" onclick={startNew}>+ Add integration</button>
		{:else}
			<form class="mt-4 space-y-3 rounded-md border border-slate-200 p-3 text-sm" onsubmit={save}>
				<div class="flex flex-wrap gap-2">
					<select class="input w-auto" aria-label="Kind" bind:value={kind}>
						<option value="slack">Slack</option>
						<option value="teams">Microsoft Teams</option>
						<option value="webhook">Webhook (JSON)</option>
					</select>
					<input class="input min-w-0 flex-1" placeholder="Name, e.g. #team-chat" aria-label="Name" required maxlength="100" bind:value={name} />
				</div>
				<input class="input" type="url" placeholder="https://hooks.slack.com/services/…" aria-label="URL" required maxlength="2000" bind:value={url} />
				{#if kind === 'webhook'}
					<input
						class="input"
						placeholder={editing?.hasSecret ? 'Secret is set — type to replace it' : 'Secret (optional) for the X-Pyramid-Signature header'}
						aria-label="Secret"
						maxlength="200"
						bind:value={secret}
					/>
				{/if}
				<fieldset>
					<legend class="mb-1 font-medium text-slate-700">Send on</legend>
					<div class="flex flex-wrap gap-x-4 gap-y-1">
						{#each EVENTS as ev (ev.key)}
							<label class="flex items-center gap-1.5"><input type="checkbox" checked={events.includes(ev.key)} onchange={() => toggle(ev.key)} /> {ev.label}</label>
						{/each}
					</div>
				</fieldset>
				<label class="flex items-center gap-1.5"><input type="checkbox" bind:checked={active} /> Active</label>
				<div class="flex gap-2">
					<button type="submit" class="btn-primary" disabled={!events.length}>Save</button>
					<button type="button" class="btn-ghost" onclick={() => (open = false)}>Cancel</button>
				</div>
			</form>
		{/if}
	</div>
</div>
