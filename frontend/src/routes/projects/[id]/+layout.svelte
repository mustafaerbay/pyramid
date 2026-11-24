<script lang="ts">
	import { page } from '$app/state';
	import { api } from '$lib/api/client';
	import { can } from '$lib/auth.svelte';

	let { children } = $props();
	const projectId = $derived(page.params.id ?? '');
	let project = $state<{ id: string; key: string; name: string } | null>(null);
	$effect(() => {
		const id = projectId;
		if (!id || project?.id === id) return;
		api.GET('/api/v1/projects/{id}', { params: { path: { id } } }).then((r) => {
			if (r.data && projectId === id) project = { id, key: r.data.key, name: r.data.name };
		});
	});

	// One tab bar for every page of a project. Items open from the board, so
	// an item's page keeps the Board tab lit.
	const tabs = $derived([
		{ href: '', label: 'Overview' },
		{ href: '/board', label: 'Board', also: '/items/' },
		{ href: '/backlog', label: 'Backlog' },
		{ href: '/sprints', label: 'Sprints' },
		{ href: '/code', label: 'Code' },
		{ href: '/artifacts', label: 'Artifacts' },
		{ href: '/wiki', label: 'Wiki' },
		{ href: '/tests', label: 'Tests' },
		{ href: '/analytics', label: 'Analytics' },
		...(can('member') ? [{ href: '/settings', label: 'Settings' }] : [])
	]);
	function current(t: { href: string; also?: string }): boolean {
		const base = `/projects/${projectId}`;
		const path = page.url.pathname.replace(/\/$/, '');
		if (t.href === '') return path === base;
		return path === base + t.href || path.startsWith(base + t.href + '/') || (!!t.also && path.startsWith(base + t.also));
	}
</script>

<div class="mb-6">
	<nav class="flex flex-wrap items-center gap-1 text-sm text-slate-500" aria-label="Breadcrumb">
		<a class="inline-flex min-h-6 items-center hover:text-slate-800 hover:underline" href="/projects">Projects</a>
		<span aria-hidden="true">/</span>
		<a class="inline-flex min-h-6 items-center gap-1.5 font-medium text-slate-800 hover:underline" href="/projects/{projectId}">
			{#if project}<span class="font-mono text-xs text-slate-500">{project.key}</span>{/if}
			{project?.name ?? 'Project'}
		</a>
	</nav>
	<nav class="-mx-4 mt-2 overflow-x-auto border-b border-slate-200 px-4 sm:mx-0 sm:px-0" aria-label="Project">
		<ul class="flex min-w-max gap-1">
			{#each tabs as t (t.label)}
				<li>
					<a
						href="/projects/{projectId}{t.href}"
						aria-current={current(t) ? 'page' : undefined}
						class="flex min-h-11 items-center whitespace-nowrap border-b-2 px-3 text-sm font-medium transition-colors {current(t)
							? 'border-slate-900 text-slate-900'
							: 'border-transparent text-slate-600 hover:border-slate-300 hover:text-slate-900'}">{t.label}</a
					>
				</li>
			{/each}
		</ul>
	</nav>
</div>

{@render children()}
