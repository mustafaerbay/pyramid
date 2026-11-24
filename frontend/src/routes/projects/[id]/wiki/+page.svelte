<script lang="ts">
	import { goto } from '$app/navigation';
	import { page } from '$app/state';
	import { api } from '$lib/api/client';
	import type { components } from '$lib/api/schema';
	import { can } from '$lib/auth.svelte';
	import { breadcrumb, wikiTree } from '$lib/wiki';
	import Markdown from '$lib/components/Markdown.svelte';
	import MentionTextarea from '$lib/components/MentionTextarea.svelte';

	type Page = components['schemas']['WikiPage'];
	const projectId = $derived(page.params.id ?? '');
	const pageId = $derived(page.url.searchParams.get('page') ?? '');
	let project = $state<components['schemas']['Project'] | null>(null);
	let pages = $state<Page[]>([]);
	let current = $state<Page | null>(null);
	let people = $state<Record<string, string>>({});
	let editing = $state(false);
	let title = $state('');
	let body = $state('');
	let error = $state<string | null>(null);
	let saving = $state(false);
	let history = $state<components['schemas']['WikiRevision'][] | null>(null);
	let viewing = $state<components['schemas']['WikiRevision'] | null>(null);

	const loadPages = async () => (pages = (await api.GET('/api/v1/wiki-pages', { params: { query: { projectId } } })).data?.data ?? []);
	$effect(() => {
		if (!projectId) return;
		api.GET('/api/v1/projects/{id}', { params: { path: { id: projectId } } }).then((r) => (project = r.data ?? null));
		api.GET('/api/v1/people', { params: { query: { page: 1, pageSize: 100 } } }).then(
			(r) => (people = Object.fromEntries((r.data?.data ?? []).map((x) => [x.id, `${x.firstName} ${x.lastName}`])))
		);
		loadPages();
	});
	$effect(() => {
		const id = pageId || wikiTree(pages)[0]?.page.id;
		editing = false;
		history = viewing = null;
		if (!id) {
			current = null;
			return;
		}
		api.GET('/api/v1/wiki-pages/{id}', { params: { path: { id } } }).then((r) => (current = r.data ?? null));
	});
	const open = (id: string) => goto(`?page=${id}`, { keepFocus: true, noScroll: true });

	async function create(parentId: string | null) {
		const t = prompt(parentId ? 'Title of the sub-page' : 'Title of the new page');
		if (!t?.trim()) return;
		const { data, error: err } = await api.POST('/api/v1/wiki-pages', { body: { projectId, title: t.trim(), parentId, body: `# ${t.trim()}\n\n` } });
		if (err || !data) return (error = err?.message ?? 'Could not add the page.');
		await loadPages();
		open(data.id);
		setTimeout(() => startEdit(), 300);
	}
	function startEdit() {
		if (!current) return;
		title = current.title;
		body = current.body ?? '';
		editing = true;
		error = null;
	}
	async function save() {
		if (!current) return;
		saving = true;
		const { data, error: err } = await api.PATCH('/api/v1/wiki-pages/{id}', {
			params: { path: { id: current.id } },
			body: { title, body, baseUpdatedAt: current.updatedAt }
		});
		saving = false;
		if (err || !data) return (error = err?.message ?? 'Could not save.');
		current = data;
		editing = false;
		loadPages();
	}
	async function remove() {
		if (!current || !confirm(`Delete "${current.title}"? Its sub-pages move up one level.`)) return;
		await api.DELETE('/api/v1/wiki-pages/{id}', { params: { path: { id: current.id } } });
		await loadPages();
		goto('?');
	}
	async function showHistory() {
		if (!current) return;
		history = (await api.GET('/api/v1/wiki-pages/{id}/revisions', { params: { path: { id: current.id } } })).data?.data ?? [];
	}
	async function restore(r: components['schemas']['WikiRevision']) {
		if (!current || !confirm('Bring back this version? The current text is kept in the history.')) return;
		const { data } = await api.PATCH('/api/v1/wiki-pages/{id}', { params: { path: { id: current.id } }, body: { title: r.title, body: r.body } });
		if (data) current = data;
		history = viewing = null;
		loadPages();
	}
	const tree = $derived(wikiTree(pages));
	const crumbs = $derived(current ? breadcrumb(pages, current.id) : []);
	const names = $derived(Object.values(people));
	const writable = $derived(can('member'));
</script>

<svelte:head><title>{current?.title ?? 'Wiki'} · {project?.name ?? 'Project'} · Pyramid</title></svelte:head>

<section class="space-y-4">
	<div>
		<h1 class="mt-1 text-2xl font-semibold tracking-tight">Wiki</h1>
	</div>
	<div class="grid gap-4 md:grid-cols-[14rem_minmax(0,1fr)]">
		<nav class="rounded-lg border border-slate-200 bg-white p-2 text-sm md:self-start" aria-label="Wiki pages">
			<ul>
				{#each tree as { page: p, depth } (p.id)}
					<li>
						<button
							type="button"
							class="block w-full truncate rounded px-2 py-1 text-left {current?.id === p.id ? 'bg-slate-100 font-medium text-slate-900' : 'text-slate-700 hover:bg-slate-50'}"
							style:padding-left="{0.5 + depth * 0.9}rem"
							onclick={() => open(p.id)}>{p.title}</button
						>
					</li>
				{:else}
					<li class="px-2 py-1 text-slate-500">No pages yet.</li>
				{/each}
			</ul>
			{#if writable}<button type="button" class="btn-ghost mt-2 w-full px-2 py-1 text-xs" onclick={() => create(null)}>+ New page</button>{/if}
		</nav>

		<article class="min-w-0 rounded-lg border border-slate-200 bg-white p-4 sm:p-6">
			{#if error}<p class="mb-3 rounded-md border border-red-200 bg-red-50 p-2 text-sm text-red-700">{error}</p>{/if}
			{#if !current}
				<p class="text-sm text-slate-500">
					A wiki keeps the project's docs next to its work: how to set things up, decisions, runbooks. {#if writable}Start with <button
							type="button"
							class="underline"
							onclick={() => create(null)}>a first page</button
						>.{/if}
				</p>
			{:else if editing}
				<input class="input mb-2 text-lg font-semibold" aria-label="Title" maxlength="200" bind:value={title} />
				<MentionTextarea bind:value={body} people={names} rows={18} label="Page text" />
				<div class="mt-3 flex gap-2">
					<button type="button" class="btn-primary" disabled={saving} onclick={save}>{saving ? 'Saving…' : 'Save'}</button>
					<button type="button" class="btn-ghost" onclick={() => (editing = false)}>Cancel</button>
				</div>
			{:else}
				<div class="mb-3 flex flex-wrap items-center gap-2 text-xs text-slate-500">
					<span>{crumbs.map((c) => c.title).join(' / ')}</span>
					<span>· edited {new Date(current.updatedAt).toLocaleString()}{current.updatedBy && people[current.updatedBy] ? ` by ${people[current.updatedBy]}` : ''}</span>
					<span class="ml-auto flex gap-2">
						{#if writable}
							<button type="button" class="btn-secondary px-2 py-1 text-xs" onclick={startEdit}>Edit</button>
							<button type="button" class="btn-ghost px-2 py-1 text-xs" onclick={() => create(current!.id)}>+ Sub-page</button>
						{/if}
						<button type="button" class="btn-ghost px-2 py-1 text-xs" onclick={showHistory}>History</button>
						{#if writable}<button type="button" class="btn-ghost px-2 py-1 text-xs text-red-600" onclick={remove}>Delete</button>{/if}
					</span>
				</div>
				{#if viewing}
					<div class="mb-3 rounded-md border border-amber-200 bg-amber-50 p-2 text-xs text-amber-900">
						Version from {new Date(viewing.createdAt).toLocaleString()} ·
						{#if writable}<button type="button" class="underline" onclick={() => restore(viewing!)}>Restore it</button> ·{/if}
						<button type="button" class="underline" onclick={() => (viewing = null)}>Back to current</button>
					</div>
					<Markdown text={viewing.body} people={names} class="text-slate-800" />
				{:else}
					<Markdown text={current.body ?? ''} people={names} class="text-slate-800" />
				{/if}
				{#if history}
					<div class="mt-6 border-t border-slate-200 pt-3 text-sm">
						<h2 class="mb-1 font-semibold">History</h2>
						<ul class="space-y-1">
							{#each history as r (r.id)}
								<li>
									<button type="button" class="text-slate-700 hover:underline" onclick={() => (viewing = r)}>
										{new Date(r.createdAt).toLocaleString()}{r.authorId && people[r.authorId] ? ` · ${people[r.authorId]}` : ''} · “{r.title}”
									</button>
								</li>
							{:else}
								<li class="text-slate-500">No earlier versions.</li>
							{/each}
						</ul>
					</div>
				{/if}
			{/if}
		</article>
	</div>
</section>
