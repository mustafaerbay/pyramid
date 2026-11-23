<script lang="ts">
	import { page } from '$app/state';
	import { api } from '$lib/api/client';
	import type { components } from '$lib/api/schema';
	import { can } from '$lib/auth.svelte';
	import { formatBytes } from '$lib/files';
	import Markdown from '$lib/components/Markdown.svelte';
	import TypeBadge from '$lib/components/TypeBadge.svelte';

	type Pkg = components['schemas']['ProjectPackage'];
	type AV = components['schemas']['ArtifactVersion'];
	const projectId = $derived(page.params.id ?? '');
	let project = $state<components['schemas']['Project'] | null>(null);
	let packages = $state<Pkg[]>([]);
	let defaultServer = $state<string | null>(null);
	let versions = $state<{ id: string; name: string }[]>([]);
	let selected = $state<string>('');
	let list = $state<AV[]>([]);
	let loading = $state(false);
	let error = $state<string | null>(null);
	let server = $state('');
	let pkgName = $state('');

	async function loadPackages() {
		const r = await api.GET('/api/v1/project-packages', { params: { query: { projectId } } });
		packages = r.data?.data ?? [];
		defaultServer = r.data?.defaultServerUrl ?? null;
		if (!selected || !packages.some((p) => p.id === selected)) selected = packages[0]?.id ?? '';
	}
	$effect(() => {
		if (!projectId) return;
		api.GET('/api/v1/projects/{id}', { params: { path: { id: projectId } } }).then((r) => (project = r.data ?? null));
		api.GET('/api/v1/versions', { params: { query: { projectId, page: 1, pageSize: 100 } } }).then(
			(r) => (versions = (r.data?.data ?? []).map((v) => ({ id: v.id, name: v.name })))
		);
		loadPackages();
	});
	async function loadVersions(id: string) {
		if (!id) return (list = []);
		loading = true;
		error = null;
		const { data, error: err } = await api.GET('/api/v1/project-packages/{id}/versions', { params: { path: { id } } });
		loading = false;
		if (err) {
			list = [];
			return (error = err.message ?? 'arti did not answer.');
		}
		list = data?.data ?? [];
	}
	$effect(() => {
		loadVersions(selected);
	});
	async function link(e: SubmitEvent) {
		e.preventDefault();
		const { data, error: err } = await api.POST('/api/v1/project-packages', {
			body: { projectId, package: pkgName, ...(server.trim() ? { serverUrl: server.trim() } : {}) }
		});
		if (err || !data) return (error = err?.message ?? 'Could not link the package.');
		pkgName = '';
		await loadPackages();
		selected = data.id;
	}
	async function unlink(p: Pkg) {
		if (!confirm(`Unlink ${p.package}? Nothing changes in arti.`)) return;
		await api.DELETE('/api/v1/project-packages/{id}', { params: { path: { id: p.id } } });
		loadPackages();
	}
	async function setVersion(v: AV, versionId: string) {
		await api.PUT('/api/v1/project-packages/{id}/links', {
			params: { path: { id: selected } },
			body: { artifactVersion: v.version, versionId: versionId || null }
		});
		loadVersions(selected);
	}
	const current = $derived(packages.find((p) => p.id === selected) ?? null);
</script>

<svelte:head><title>Artifacts · {project?.name ?? 'Project'} · Pyramid</title></svelte:head>

<section class="space-y-4">
	<div>
		<a href="/projects/{projectId}" class="text-sm text-slate-500 hover:text-slate-700">← {project?.name ?? 'Project'}</a>
		<h1 class="mt-1 text-2xl font-semibold tracking-tight">Artifacts</h1>
		<p class="mt-1 text-sm text-slate-500">
			Build outputs live in <b>arti</b>. Link this project's arti packages to see their versions, files and release notes here, and say which Pyramid
			version each one shipped — with the items finished in it.
		</p>
	</div>

	<div class="rounded-lg border border-slate-200 bg-white p-4 text-sm">
		<div class="flex flex-wrap items-center gap-2">
			{#each packages as p (p.id)}
				<button
					type="button"
					class="rounded-full px-3 py-1 ring-1 ring-inset {selected === p.id ? 'bg-slate-900 text-white ring-slate-900' : 'bg-white text-slate-700 ring-slate-200 hover:bg-slate-50'}"
					onclick={() => (selected = p.id)}>📦 {p.package}</button
				>
			{:else}
				<span class="text-slate-500">No packages linked yet.</span>
			{/each}
		</div>
		{#if current}
			<div class="mt-2 flex flex-wrap gap-x-4 gap-y-1 text-xs text-slate-500">
				<a class="hover:underline" href={current.webUrl} target="_blank" rel="noopener noreferrer">Open in arti ↗</a>
				<span>Update URL for apps: <code class="rounded bg-slate-100 px-1">{current.updateUrl}</code></span>
				<span>server {current.serverUrl}</span>
				{#if can('manager')}<button type="button" class="text-red-600 hover:underline" onclick={() => unlink(current!)}>Unlink</button>{/if}
			</div>
		{/if}
		{#if can('manager')}
			<form class="mt-3 flex flex-wrap gap-2" onsubmit={link}>
				<input class="input w-full sm:w-64" placeholder={defaultServer ?? 'https://arti.example.com'} aria-label="arti server" bind:value={server} />
				<input class="input min-w-0 flex-1" placeholder="Package, e.g. WMG/Desktop/SmartDock" aria-label="Package" required bind:value={pkgName} />
				<button type="submit" class="btn-primary">Link package</button>
			</form>
		{/if}
	</div>

	{#if error}<div class="rounded-md border border-red-200 bg-red-50 p-3 text-sm text-red-700">{error}</div>{/if}
	{#if loading}<p class="text-sm text-slate-500">Asking arti…</p>{/if}

	<ul class="space-y-3">
		{#each list as v (v.version)}
			<li class="rounded-lg border border-slate-200 bg-white p-4 text-sm">
				<div class="flex flex-wrap items-center gap-2">
					<span class="text-base font-semibold">{v.version}</span>
					{#if v.locked}<span class="rounded bg-slate-100 px-1.5 text-xs" title="Locked in arti">🔒 locked</span>{/if}
					<span class="text-xs text-slate-500">{new Date(v.created).toLocaleDateString()}</span>
					<label class="ml-auto flex items-center gap-1.5 text-xs text-slate-600">
						Shipped
						<select class="input w-auto py-1 text-xs" aria-label="Pyramid version for {v.version}" value={v.versionId ?? ''} disabled={!can('member')} onchange={(e) => setVersion(v, e.currentTarget.value)}>
							<option value="">— no Pyramid version —</option>
							{#each versions as pv (pv.id)}<option value={pv.id}>{pv.name}</option>{/each}
						</select>
					</label>
				</div>
				{#if v.files.length}
					<ul class="mt-2 flex flex-wrap gap-2 text-xs">
						{#each v.files as f (f.name)}
							<li><a class="rounded border border-slate-200 px-2 py-0.5 hover:bg-slate-50" href={f.url}>⬇ {f.name} <span class="text-slate-500">{formatBytes(f.size)}</span></a></li>
						{/each}
					</ul>
				{/if}
				{#if v.notes}<Markdown text={v.notes} class="mt-3 border-l-2 border-slate-200 pl-3 text-slate-700" />{/if}
				{#if v.versionName}
					<div class="mt-3">
						<div class="text-xs font-medium text-slate-600">{v.versionName}: {v.items.length} finished {v.items.length === 1 ? 'item' : 'items'}</div>
						<ul class="mt-1 space-y-0.5">
							{#each v.items as t (t.id)}
								<li><TypeBadge type={t.type} /> <a class="hover:underline" href="/projects/{projectId}?task={t.id}">#{t.number} {t.title}</a></li>
							{/each}
						</ul>
					</div>
				{/if}
			</li>
		{:else}
			{#if current && !loading && !error}<li class="text-sm text-slate-500">This package has no versions in arti yet.</li>{/if}
		{/each}
	</ul>
</section>
