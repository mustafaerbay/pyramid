<script lang="ts">
	import { api, baseUrl } from '$lib/api/client';
	import { formatBytes } from '$lib/files';
	import type { components } from '$lib/api/schema';
	import { can, session } from '$lib/auth.svelte';
	import { describeChange, type Names } from '$lib/history';
	import Markdown from './Markdown.svelte';
	import MentionTextarea from './MentionTextarea.svelte';
	import GitEventRow from './GitEventRow.svelte';
	import { itemHref, uploadImage } from '$lib/itemFiles';

	type Task = components['schemas']['Task'];
	type Comment = components['schemas']['TaskComment'];
	type Entry = components['schemas']['TaskHistoryEntry'];

	let {
		task,
		projectTasks = [],
		milestoneOptions = [],
		version = 0
	}: {
		task: Task;
		projectTasks?: Task[];
		milestoneOptions?: { id: string; name: string }[];
		/** Bumped after the item is saved, to reload its history and files. */
		version?: number;
	} = $props();
	const addImage = async (file: File) => {
		const md = await uploadImage(task.id, file);
		loadFiles();
		return md;
	};

	let tab = $state<'comments' | 'history' | 'files' | 'dev'>('comments');
	let dev = $state<components['schemas']['GitEvent'][]>([]);
	$effect(() => {
		const id = task.id;
		api.GET('/api/v1/git-activity', { params: { query: { taskId: id } } }).then((r) => (dev = r.data?.data ?? []));
	});
	let files = $state<components['schemas']['Attachment'][]>([]);
	let uploading = $state(false);
	async function loadFiles() {
		const { data } = await api.GET('/api/v1/tasks/{id}/attachments', { params: { path: { id: task.id } } });
		files = data?.data ?? [];
	}
	async function uploadFiles(list: FileList | null) {
		if (!list?.length) return;
		uploading = true;
		error = null;
		for (const file of Array.from(list)) {
			if (file.size > 10 * 1024 * 1024) {
				error = `${file.name} is larger than 10 MB.`;
				continue;
			}
			const fd = new FormData();
			fd.append('file', file);
			const res = await fetch(`${baseUrl}/api/v1/tasks/${task.id}/attachments`, { method: 'POST', body: fd, credentials: 'include' });
			if (!res.ok) error = (await res.json().catch(() => null))?.message ?? `Could not upload ${file.name}.`;
		}
		uploading = false;
		loadFiles();
	}
	async function removeFile(id: string, name: string) {
		if (!confirm(`Delete ${name}?`)) return;
		const { error: err } = await api.DELETE('/api/v1/attachments/{id}', { params: { path: { id } } });
		if (err) error = err.message ?? 'Could not delete the file.';
		loadFiles();
	}
	let comments = $state<Comment[]>([]);
	let history = $state<Entry[]>([]);
	let people = $state<Record<string, string>>({});
	let followers = $state<components['schemas']['TaskFollowers']>({ following: false, personIds: [] });
	$effect(() => {
		const id = task.id;
		api.GET('/api/v1/tasks/{id}/followers', { params: { path: { id } } }).then((r) => (followers = r.data ?? followers));
	});
	async function toggleFollow() {
		const opts = { params: { path: { id: task.id } } };
		const r = followers.following ? await api.DELETE('/api/v1/tasks/{id}/follow', opts) : await api.PUT('/api/v1/tasks/{id}/follow', opts);
		if (r.data) followers = r.data;
	}
	const personNames = $derived(Object.values(people).sort((a, b) => a.localeCompare(b)));
	const itemLink = (n: number) => {
		const t = projectTasks.find((x) => x.number === n);
		return t ? itemHref(t.projectId, t.number) : null;
	};
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
		loadFiles();
		history = h.data?.data ?? [];
		people = Object.fromEntries((p.data?.data ?? []).map((x) => [x.id, `${x.firstName} ${x.lastName}`]));
	}
	$effect(() => {
		task.id;
		load();
	});
	$effect(() => {
		if (!version) return;
		api.GET('/api/v1/tasks/{id}/history', { params: { path: { id: task.id } } }).then((h) => (history = h.data?.data ?? history));
		loadFiles();
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
	<div class="mb-3 flex flex-wrap items-center gap-1 text-sm" role="tablist">
		{#each [['comments', `Comments (${comments.length})`], ['files', `Files (${files.length})`], ...(dev.length ? [['dev', `Development (${dev.length})`]] : []), ['history', 'History']] as [key, label] (key)}
			<button
				type="button"
				role="tab"
				aria-selected={tab === key}
				class="rounded-md px-3 py-1.5 font-medium {tab === key ? 'bg-slate-100 text-slate-900' : 'text-slate-500 hover:text-slate-700'}"
				onclick={() => (tab = key as 'comments' | 'history' | 'files' | 'dev')}>{label}</button
			>
		{/each}
		<button
			type="button"
			class="ml-auto rounded-md px-2 py-1 text-xs {followers.following ? 'bg-amber-50 text-amber-800 ring-1 ring-inset ring-amber-200' : 'text-slate-600 ring-1 ring-inset ring-slate-200 hover:bg-slate-50'}"
			aria-pressed={followers.following}
			title={followers.personIds.length ? `Followed by ${followers.personIds.map((id) => people[id] ?? 'someone').join(', ')}` : 'Get notified of every change'}
			onclick={toggleFollow}>{followers.following ? '★ Following' : '☆ Follow'}{followers.personIds.length ? ` · ${followers.personIds.length}` : ''}</button
		>
	</div>

	{#if error}
		<p class="mb-2 text-sm text-red-700" role="alert">{error}</p>
	{/if}

	{#if tab === 'dev'}
		<ul class="space-y-2">
			{#each dev as e (e.id)}<li><GitEventRow {e} /></li>{/each}
		</ul>
	{:else if tab === 'files'}
		{#if files.length === 0}<p class="text-sm text-slate-500">No files yet.</p>{/if}
		<ul class="space-y-2">
			{#each files as f (f.id)}
				<li class="flex items-center gap-3 text-sm">
					{#if f.isImage}
						<a href="{baseUrl}{f.url}?inline=1" target="_blank" rel="noopener"
							><img src="{baseUrl}{f.url}?inline=1" alt="" class="h-10 w-10 rounded border border-slate-200 object-cover" /></a
						>
					{:else}
						<span class="flex h-10 w-10 items-center justify-center rounded border border-slate-200 text-[10px] font-semibold uppercase text-slate-500"
							>{f.filename.split('.').pop()?.slice(0, 4) ?? 'file'}</span
						>
					{/if}
					<span class="min-w-0 flex-1">
						<a href="{baseUrl}{f.url}" class="block truncate font-medium text-slate-800 hover:underline">{f.filename}</a>
						<span class="text-xs text-slate-500">{formatBytes(f.sizeBytes)} · {who(f.uploadedByPersonId)} · {when(f.createdAt)}</span>
					</span>
					{#if (f.uploadedByPersonId && f.uploadedByPersonId === session.me?.person.id) || can('manager')}
						<button type="button" class="text-xs text-red-600 hover:underline" onclick={() => removeFile(f.id, f.filename)}>Delete</button>
					{/if}
				</li>
			{/each}
		</ul>
		{#if can('member')}
			<label class="mt-3 flex cursor-pointer items-center justify-center rounded-md border border-dashed border-slate-300 p-3 text-sm text-slate-600 hover:bg-slate-50">
				<input type="file" multiple class="sr-only" disabled={uploading} onchange={(e) => uploadFiles(e.currentTarget.files)} />
				{uploading ? 'Uploading…' : '+ Attach files (up to 10 MB each)'}
			</label>
		{/if}
	{:else if tab === 'comments'}
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
						<div class="mt-1"><MentionTextarea bind:value={editDraft} people={personNames} rows={3} maxlength={5000} label="Edit comment" onImage={addImage} /></div>
						<div class="mt-1 flex gap-2">
							<button type="button" class="btn-primary px-3 py-1 text-xs" onclick={() => saveEdit(c)}>Save</button>
							<button type="button" class="btn-ghost px-3 py-1 text-xs" onclick={() => (editingId = null)}>Cancel</button>
						</div>
					{:else}
						<Markdown text={c.body} people={personNames} item={itemLink} class="mt-0.5 text-slate-700" />
					{/if}
				</li>
			{/each}
		</ul>
		{#if can('member')}
			<form class="mt-3 space-y-2" onsubmit={post}>
				<MentionTextarea bind:value={draft} people={personNames} rows={2} maxlength={5000} placeholder="Write a comment… (Markdown, @ to mention)" label="New comment" onImage={addImage} />
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
