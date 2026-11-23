<script lang="ts">
	import { api } from '$lib/api/client';
	import type { components } from '$lib/api/schema';
	import { can, session } from '$lib/auth.svelte';
	import { describeChange, type Names } from '$lib/history';

	type Task = components['schemas']['Task'];
	type Comment = components['schemas']['TaskComment'];
	type Entry = components['schemas']['TaskHistoryEntry'];

	let {
		task,
		projectTasks = [],
		milestoneOptions = []
	}: { task: Task; projectTasks?: Task[]; milestoneOptions?: { id: string; name: string }[] } = $props();

	let tab = $state<'comments' | 'history'>('comments');
	let comments = $state<Comment[]>([]);
	let history = $state<Entry[]>([]);
	let people = $state<Record<string, string>>({});
	let sprintNames = $state<Record<string, string>>({});
	let areaNames = $state<Record<string, string>>({});
	let draft = $state('');
	let editingId = $state<string | null>(null);
	let editDraft = $state('');
	let busy = $state(false);
	let error = $state<string | null>(null);

	async function load() {
		const [c, h, p, sp, ar] = await Promise.all([
			api.GET('/api/v1/tasks/{id}/comments', { params: { path: { id: task.id } } }),
			api.GET('/api/v1/tasks/{id}/history', { params: { path: { id: task.id } } }),
			api.GET('/api/v1/people', { params: { query: { page: 1, pageSize: 100 } } }),
			api.GET('/api/v1/sprints', { params: { query: { projectId: task.projectId } } }),
			api.GET('/api/v1/areas', { params: { query: { projectId: task.projectId } } })
		]);
		sprintNames = Object.fromEntries((sp.data?.data ?? []).map((x) => [x.id, x.name]));
		areaNames = Object.fromEntries((ar.data?.data ?? []).map((x) => [x.id, x.path]));
		comments = c.data?.data ?? [];
		history = h.data?.data ?? [];
		people = Object.fromEntries((p.data?.data ?? []).map((x) => [x.id, `${x.firstName} ${x.lastName}`]));
	}
	$effect(() => {
		task.id;
		load();
	});

	const names: Names = {
		person: (id) => people[id] ?? 'Someone',
		milestone: (id) => milestoneOptions.find((m) => m.id === id)?.name ?? 'a deleted milestone',
		sprint: (id) => sprintNames[id] ?? 'a deleted sprint',
		area: (id) => areaNames[id] ?? 'a deleted area',
		task: (id) => {
			const t = projectTasks.find((x) => x.id === id);
			return t ? `#${t.number} ${t.title}` : 'a deleted task';
		}
	};
	const who = (id: string | null | undefined) => (id ? (people[id] ?? 'Someone') : 'Someone');
	const when = (iso: string) =>
		new Date(iso).toLocaleString(undefined, { day: 'numeric', month: 'short', hour: '2-digit', minute: '2-digit' });
	const mine = (c: Comment) => !!c.authorPersonId && c.authorPersonId === session.me?.person.id;

	async function post(e: SubmitEvent) {
		e.preventDefault();
		if (!draft.trim()) return;
		busy = true;
		error = null;
		const { data, error: err } = await api.POST('/api/v1/tasks/{id}/comments', {
			params: { path: { id: task.id } },
			body: { body: draft }
		});
		busy = false;
		if (err || !data) {
			error = err?.message ?? 'Could not post the comment.';
			return;
		}
		comments = [...comments, data];
		draft = '';
	}

	async function saveEdit(c: Comment) {
		const { data, error: err } = await api.PATCH('/api/v1/task-comments/{id}', {
			params: { path: { id: c.id } },
			body: { body: editDraft }
		});
		if (err || !data) {
			error = err?.message ?? 'Could not save the comment.';
			return;
		}
		comments = comments.map((x) => (x.id === data.id ? data : x));
		editingId = null;
	}

	async function remove(c: Comment) {
		if (!confirm('Delete this comment?')) return;
		const { error: err } = await api.DELETE('/api/v1/task-comments/{id}', { params: { path: { id: c.id } } });
		if (err) {
			error = err.message ?? 'Could not delete the comment.';
			return;
		}
		comments = comments.filter((x) => x.id !== c.id);
	}
</script>

<div class="mt-6 border-t border-slate-200 pt-4">
	<div class="mb-3 flex gap-1 text-sm" role="tablist">
		{#each [['comments', `Comments (${comments.length})`], ['history', 'History']] as [key, label] (key)}
			<button
				type="button"
				role="tab"
				aria-selected={tab === key}
				class="rounded-md px-3 py-1.5 font-medium {tab === key ? 'bg-slate-100 text-slate-900' : 'text-slate-500 hover:text-slate-700'}"
				onclick={() => (tab = key as 'comments' | 'history')}>{label}</button
			>
		{/each}
	</div>

	{#if error}
		<p class="mb-2 text-sm text-red-700" role="alert">{error}</p>
	{/if}

	{#if tab === 'comments'}
		{#if comments.length === 0}
			<p class="text-sm text-slate-500">No comments yet.</p>
		{/if}
		<ul class="space-y-3">
			{#each comments as c (c.id)}
				<li class="text-sm">
					<div class="flex flex-wrap items-baseline gap-x-2">
						<span class="font-medium text-slate-800">{who(c.authorPersonId)}</span>
						<span class="text-xs text-slate-500">
							{when(c.createdAt)}{c.updatedAt !== c.createdAt ? ' · edited' : ''}
						</span>
						{#if mine(c) || can('manager')}
							<span class="ml-auto flex gap-2 text-xs">
								{#if mine(c)}
									<button type="button" class="text-slate-500 hover:underline" onclick={() => ((editingId = c.id), (editDraft = c.body))}>
										Edit
									</button>
								{/if}
								<button type="button" class="text-red-600 hover:underline" onclick={() => remove(c)}>Delete</button>
							</span>
						{/if}
					</div>
					{#if editingId === c.id}
						<textarea class="input mt-1" rows="3" maxlength="5000" aria-label="Edit comment" bind:value={editDraft}></textarea>
						<div class="mt-1 flex gap-2">
							<button type="button" class="btn-primary px-3 py-1 text-xs" onclick={() => saveEdit(c)}>Save</button>
							<button type="button" class="btn-ghost px-3 py-1 text-xs" onclick={() => (editingId = null)}>Cancel</button>
						</div>
					{:else}
						<p class="mt-0.5 whitespace-pre-wrap break-words text-slate-700">{c.body}</p>
					{/if}
				</li>
			{/each}
		</ul>
		{#if can('member')}
			<form class="mt-3 space-y-2" onsubmit={post}>
				<textarea
					class="input"
					rows="2"
					maxlength="5000"
					placeholder="Write a comment…"
					aria-label="New comment"
					bind:value={draft}
					onkeydown={(e) => {
						if (e.key === 'Enter' && (e.metaKey || e.ctrlKey)) e.currentTarget.form?.requestSubmit();
					}}
				></textarea>
				<button type="submit" class="btn-secondary" disabled={busy || !draft.trim()}>{busy ? 'Posting…' : 'Comment'}</button>
			</form>
		{/if}
	{:else}
		{#if history.length === 0}
			<p class="text-sm text-slate-500">No changes recorded yet. Changes are recorded from now on.</p>
		{/if}
		<ol class="space-y-3">
			{#each history as h (h.id)}
				<li class="text-sm">
					<div class="flex flex-wrap items-baseline gap-x-2">
						<span class="font-medium text-slate-800">{who(h.actorPersonId)}</span>
						<span class="text-slate-600">{h.action === 'created' ? 'created the task' : 'changed'}</span>
						<span class="text-xs text-slate-500">{when(h.createdAt)}</span>
					</div>
					{#if h.action === 'updated'}
						<ul class="mt-0.5 space-y-0.5 text-slate-600">
							{#each Object.entries(h.changes) as [field, change] (field)}
								<li class="break-words">{describeChange(field, change, names)}</li>
							{/each}
						</ul>
					{/if}
				</li>
			{/each}
		</ol>
	{/if}
</div>
