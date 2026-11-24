<script lang="ts">
	import Icon from '$lib/components/Icon.svelte';
	import { goto } from '$app/navigation';
	import { api } from '$lib/api/client';
	import type { components } from '$lib/api/schema';
	import TypeBadge from './TypeBadge.svelte';

	type Results = components['schemas']['SearchResults'];
	let q = $state('');
	let res = $state<Results | null>(null);
	let open = $state(false);
	let active = $state(0);
	let input = $state<HTMLInputElement | null>(null);
	let timer: ReturnType<typeof setTimeout> | undefined;

	const links = $derived(
		res
			? [
					...res.items.slice(0, 8).map((i) => ({ key: i.id, href: `/projects/${i.projectId}/items/${i.id}`, kind: 'item' as const, item: i })),
					...res.projects.map((p) => ({ key: p.id, href: `/projects/${p.id}`, kind: 'project' as const, label: `${p.name} (${p.key})` })),
					...res.people.map((p) => ({ key: p.id, href: `/people/${p.id}`, kind: 'person' as const, label: p.name }))
				]
			: []
	);
	function search() {
		clearTimeout(timer);
		const term = q.trim();
		if (!term) {
			res = null;
			return;
		}
		timer = setTimeout(async () => {
			const { data } = await api.GET('/api/v1/search', { params: { query: { q: term } } });
			if (q.trim() === term) {
				res = data ?? null;
				active = 0;
				open = true;
			}
		}, 200);
	}
	function go(href: string) {
		open = false;
		q = '';
		res = null;
		goto(href);
	}
	function keydown(e: KeyboardEvent) {
		if (e.key === 'ArrowDown' || e.key === 'ArrowUp') {
			e.preventDefault();
			if (links.length) active = (active + (e.key === 'ArrowDown' ? 1 : links.length - 1)) % links.length;
		} else if (e.key === 'Enter') {
			e.preventDefault();
			if (open && links[active]) go(links[active].href);
			else if (q.trim()) go(`/search?q=${encodeURIComponent(q.trim())}`);
		} else if (e.key === 'Escape') {
			open = false;
			input?.blur();
		}
	}
	function globalKey(e: KeyboardEvent) {
		const t = e.target as HTMLElement;
		if (e.key === '/' && !e.metaKey && !e.ctrlKey && !/INPUT|TEXTAREA|SELECT/.test(t.tagName) && !t.isContentEditable) {
			e.preventDefault();
			input?.focus();
		}
	}
</script>

<svelte:window onkeydown={globalKey} />

<div class="relative">
	<input
		bind:this={input}
		type="search"
		class="h-9 w-36 rounded-md border border-slate-200 bg-slate-50 px-3 text-sm focus:w-56 focus:bg-white sm:w-48 sm:focus:w-72"
		placeholder="Search  ( / )"
		aria-label="Search work items, projects and people"
		bind:value={q}
		oninput={search}
		onkeydown={keydown}
		onfocus={() => res && (open = true)}
		onblur={() => setTimeout(() => (open = false), 150)}
	/>
	{#if open && res}
		<div class="absolute right-0 z-50 mt-1 w-[min(28rem,90vw)] overflow-hidden rounded-md border border-slate-200 bg-white text-sm shadow-lg">
			{#if links.length === 0}
				<p class="px-3 py-3 text-slate-500">Nothing found.</p>
			{:else}
				<ul role="listbox" aria-label="Search results">
					{#each links as l, i (l.key)}
						<li role="option" aria-selected={i === active}>
							<a
								href={l.href}
								class="block px-3 py-2 {i === active ? 'bg-indigo-50' : 'hover:bg-slate-50'}"
								onmousedown={(e) => e.preventDefault()}
								onclick={(e) => {
									e.preventDefault();
									go(l.href);
								}}
							>
								{#if l.kind === 'item'}
									<span class="flex items-center gap-1.5">
										<TypeBadge type={l.item.type} />
										<span class="text-xs tabular-nums text-slate-500">{l.item.projectKey}-{l.item.number}</span>
										<span class="truncate font-medium text-slate-800">{l.item.title}</span>
									</span>
									{#if l.item.snippet}<span class="mt-0.5 line-clamp-1 block text-xs text-slate-500">{#if l.item.matchedIn === 'comment'}<Icon name="message" size={12} class="mr-1 align-[-1px]" /><span class="sr-only">In a comment: </span>{/if}{l.item.snippet}</span>{/if}
								{:else}
									<span class="text-xs uppercase text-slate-400">{l.kind}</span> <span class="text-slate-800">{l.label}</span>
								{/if}
							</a>
						</li>
					{/each}
				</ul>
				<a
					href="/search?q={encodeURIComponent(q.trim())}"
					class="block border-t border-slate-100 px-3 py-2 text-xs text-slate-600 hover:bg-slate-50"
					onmousedown={(e) => e.preventDefault()}
					onclick={(e) => {
						e.preventDefault();
						go(`/search?q=${encodeURIComponent(q.trim())}`);
					}}>All results for “{q.trim()}” →</a
				>
			{/if}
		</div>
	{/if}
</div>
