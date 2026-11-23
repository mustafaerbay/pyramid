<script lang="ts">
	import { page } from '$app/state';
	import { api, baseUrl } from '$lib/api/client';
	import type { components } from '$lib/api/schema';
	import { can } from '$lib/auth.svelte';
	import GitEventRow from '$lib/components/GitEventRow.svelte';

	type Repo = components['schemas']['ProjectRepo'];
	type Event = components['schemas']['GitEvent'];
	const projectId = $derived(page.params.id ?? '');
	let project = $state<components['schemas']['Project'] | null>(null);
	let repos = $state<Repo[]>([]);
	let events = $state<Event[]>([]);
	let tab = $state<'all' | 'commit' | 'pull_request' | 'pipeline'>('all');
	let repoId = $state('');
	let provider = $state<'github' | 'gitlab'>('github');
	let url = $state('');
	let fresh = $state<Repo | null>(null);
	let error = $state<string | null>(null);

	const hookUrl = (id: string) => `${baseUrl.replace(/\/$/, '')}/api/v1/git/hooks/${id}`;
	async function loadRepos() {
		repos = (await api.GET('/api/v1/project-repos', { params: { query: { projectId } } })).data?.data ?? [];
	}
	$effect(() => {
		if (!projectId) return;
		api.GET('/api/v1/projects/{id}', { params: { path: { id: projectId } } }).then((r) => (project = r.data ?? null));
		loadRepos();
	});
	$effect(() => {
		const query = { ...(repoId ? { repoId } : { projectId }), ...(tab !== 'all' ? { kind: tab } : {}) };
		if (!projectId) return;
		api.GET('/api/v1/git-activity', { params: { query } }).then((r) => (events = r.data?.data ?? []));
	});
	async function link(e: SubmitEvent) {
		e.preventDefault();
		const { data, error: err } = await api.POST('/api/v1/project-repos', { body: { projectId, provider, url } });
		if (err || !data) return (error = err?.message ?? 'Could not link the repository.');
		error = null;
		fresh = data;
		url = '';
		loadRepos();
	}
	async function unlink(r: Repo) {
		if (!confirm(`Unlink ${r.name}? Its commits, pull requests and runs are forgotten here.`)) return;
		await api.DELETE('/api/v1/project-repos/{id}', { params: { path: { id: r.id } } });
		loadRepos();
	}
	const counts = $derived({
		failed: events.filter((e) => e.kind === 'pipeline' && e.state === 'failed').length,
		open: events.filter((e) => e.kind === 'pull_request' && e.state === 'open').length
	});
</script>

<svelte:head><title>Code · {project?.name ?? 'Project'} · Pyramid</title></svelte:head>

<section class="space-y-4">
	<div>
		<a href="/projects/{projectId}" class="text-sm text-slate-500 hover:text-slate-700">← {project?.name ?? 'Project'}</a>
		<h1 class="mt-1 text-2xl font-semibold tracking-tight">Code: repos and pipelines</h1>
		<p class="mt-1 text-sm text-slate-500">
			Pyramid doesn't host code or run builds. Link this project's GitHub or GitLab repositories: their webhooks bring in commits, pull requests and
			CI runs, linked to items that mention <code class="rounded bg-slate-100 px-1">{project?.key ?? 'KEY'}-12</code> or <code class="rounded bg-slate-100 px-1">#12</code>.
		</p>
	</div>

	<div class="rounded-lg border border-slate-200 bg-white p-4 text-sm">
		<h2 class="mb-2 font-semibold">Repositories</h2>
		<ul class="divide-y divide-slate-100">
			{#each repos as r (r.id)}
				<li class="flex flex-wrap items-center gap-2 py-2">
					<span class="rounded bg-slate-100 px-1.5 text-xs uppercase">{r.provider}</span>
					<a href={r.url} target="_blank" rel="noopener noreferrer" class="font-medium hover:underline">{r.name}</a>
					<span class="text-xs text-slate-500">{r.lastEventAt ? `last event ${new Date(r.lastEventAt).toLocaleString()}` : 'no events yet'}</span>
					<span class="ml-auto flex gap-2 text-xs">
						<button type="button" class="text-slate-600 hover:underline" onclick={() => (repoId = repoId === r.id ? '' : r.id)}>{repoId === r.id ? 'All repos' : 'Only this'}</button>
						{#if can('manager')}<button type="button" class="text-red-600 hover:underline" onclick={() => unlink(r)}>Unlink</button>{/if}
					</span>
				</li>
			{:else}
				<li class="py-2 text-slate-500">No repositories linked.</li>
			{/each}
		</ul>
		{#if fresh}
			<div class="mt-3 space-y-1 rounded-md border border-amber-200 bg-amber-50 p-3 text-xs text-amber-950">
				<div class="font-semibold">Set up the webhook for {fresh.name} now — the secret is shown only once.</div>
				<div>Payload URL: <code class="break-all rounded bg-white px-1">{hookUrl(fresh.id)}</code></div>
				<div>Secret{fresh.provider === 'gitlab' ? ' token' : ''}: <code class="break-all rounded bg-white px-1">{fresh.secret}</code></div>
				{#if fresh.provider === 'github'}
					<div>On GitHub: Settings → Webhooks → Add webhook · content type <b>application/json</b> · events: <b>Pushes, Pull requests, Workflow runs</b>.</div>
				{:else}
					<div>On GitLab: Settings → Webhooks · triggers: <b>Push events, Merge request events, Pipeline events</b>.</div>
				{/if}
				<button type="button" class="mt-1 underline" onclick={() => (fresh = null)}>Done</button>
			</div>
		{/if}
		{#if can('manager')}
			<form class="mt-3 flex flex-wrap gap-2" onsubmit={link}>
				<select class="input w-auto" aria-label="Provider" bind:value={provider}>
					<option value="github">GitHub</option>
					<option value="gitlab">GitLab</option>
				</select>
				<input class="input min-w-0 flex-1" type="url" required placeholder="https://github.com/acme/shop" aria-label="Repository address" bind:value={url} />
				<button type="submit" class="btn-primary">Link repository</button>
			</form>
			{#if error}<p class="mt-2 text-red-700">{error}</p>{/if}
		{/if}
	</div>

	<div class="rounded-lg border border-slate-200 bg-white p-4">
		<div class="mb-3 flex flex-wrap items-center gap-2">
			<div class="inline-flex rounded-md bg-slate-100 p-0.5 text-sm" role="tablist">
				{#each [['all', 'All'], ['commit', 'Commits'], ['pull_request', 'Pull requests'], ['pipeline', 'Pipelines']] as [k, l] (k)}
					<button
						type="button"
						role="tab"
						aria-selected={tab === k}
						class="rounded px-3 py-1.5 font-medium {tab === k ? 'bg-white shadow-sm' : 'text-slate-500'}"
						onclick={() => (tab = k as typeof tab)}>{l}</button
					>
				{/each}
			</div>
			{#if counts.open}<span class="text-xs text-slate-600">{counts.open} open pull requests</span>{/if}
			{#if counts.failed}<span class="text-xs text-red-700">{counts.failed} failed runs</span>{/if}
		</div>
		<ul class="space-y-2">
			{#each events as e (e.id)}
				<li>
					<GitEventRow {e} showRepo={repos.length > 1} />
					{#if e.items.length}<div class="pl-6 text-xs text-slate-500">→ {e.items.map((n) => `${project?.key ?? ''}-${n}`).join(', ')}</div>{/if}
				</li>
			{:else}
				<li class="text-sm text-slate-500">Nothing yet. Activity shows up once a webhook sends it.</li>
			{/each}
		</ul>
	</div>
</section>
