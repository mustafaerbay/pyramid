<script lang="ts">
	import { goto } from '$app/navigation';
	import { api } from '$lib/api/client';
	import type { components } from '$lib/api/schema';

	type Notification = components['schemas']['Notification'];

	let items = $state<Notification[]>([]);
	let unread = $state(0);
	let open = $state(false);
	let root: HTMLDivElement | undefined = $state();

	async function load() {
		const { data } = await api.GET('/api/v1/notifications');
		if (!data) return;
		items = data.data;
		unread = data.unreadCount;
	}

	// Checked on open, every minute, and when the tab comes back to front.
	$effect(() => {
		load();
		const timer = setInterval(load, 60_000);
		const onVisible = () => document.visibilityState === 'visible' && load();
		document.addEventListener('visibilitychange', onVisible);
		return () => {
			clearInterval(timer);
			document.removeEventListener('visibilitychange', onVisible);
		};
	});

	// Close when clicking elsewhere or pressing Escape.
	$effect(() => {
		if (!open) return;
		const onClick = (e: MouseEvent) => root && !root.contains(e.target as Node) && (open = false);
		const onKey = (e: KeyboardEvent) => e.key === 'Escape' && (open = false);
		document.addEventListener('click', onClick);
		document.addEventListener('keydown', onKey);
		return () => {
			document.removeEventListener('click', onClick);
			document.removeEventListener('keydown', onKey);
		};
	});

	async function select(n: Notification) {
		open = false;
		if (!n.readAt) {
			await api.POST('/api/v1/notifications/{id}/read', { params: { path: { id: n.id } } });
			load();
		}
		if (n.link) goto(n.link);
	}

	async function readAll() {
		await api.POST('/api/v1/notifications/read-all');
		load();
	}

	function ago(iso: string): string {
		const mins = Math.round((Date.now() - new Date(iso).getTime()) / 60_000);
		if (mins < 1) return 'just now';
		if (mins < 60) return `${mins} min ago`;
		const hours = Math.round(mins / 60);
		if (hours < 24) return `${hours} h ago`;
		return new Date(iso).toLocaleDateString(undefined, { day: 'numeric', month: 'short' });
	}
</script>

<div class="relative" bind:this={root}>
	<button
		type="button"
		class="relative inline-flex h-10 w-10 items-center justify-center rounded-md text-slate-600 hover:bg-slate-100"
		aria-label={unread ? `Notifications, ${unread} unread` : 'Notifications'}
		aria-expanded={open}
		onclick={() => {
			open = !open;
			if (open) load();
		}}
	>
		<svg class="h-5 w-5" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2" aria-hidden="true">
			<path d="M18 8a6 6 0 1 0-12 0c0 7-3 9-3 9h18s-3-2-3-9M13.7 21a2 2 0 0 1-3.4 0" />
		</svg>
		{#if unread > 0}
			<span
				class="absolute right-1 top-1 min-w-4 rounded-full bg-red-600 px-1 text-center text-[10px] font-semibold leading-4 text-white"
				aria-hidden="true">{unread > 99 ? '99+' : unread}</span
			>
		{/if}
	</button>

	{#if open}
		<div
			class="fixed inset-x-4 top-14 z-50 overflow-hidden rounded-lg border border-slate-200 bg-white shadow-lg sm:absolute sm:inset-x-auto sm:right-0 sm:top-auto sm:mt-1 sm:w-88"
		>
			<div class="flex items-center justify-between border-b border-slate-100 px-3 py-2">
				<span class="text-sm font-semibold text-slate-800">Notifications</span>
				{#if unread > 0}
					<button type="button" class="text-xs text-slate-500 hover:text-slate-800 hover:underline" onclick={readAll}>
						Mark all read
					</button>
				{/if}
			</div>
			{#if items.length === 0}
				<p class="px-3 py-6 text-center text-sm text-slate-500">Nothing yet.</p>
			{:else}
				<ul class="max-h-96 divide-y divide-slate-100 overflow-y-auto">
					{#each items as n (n.id)}
						<li>
							<button
								type="button"
								class="flex w-full gap-2 px-3 py-2.5 text-left text-sm hover:bg-slate-50 {n.readAt ? 'text-slate-500' : 'text-slate-800'}"
								onclick={() => select(n)}
							>
								<span
									class="mt-1.5 h-2 w-2 shrink-0 rounded-full {n.readAt ? 'bg-transparent' : 'bg-sky-500'}"
									aria-hidden="true"
								></span>
								<span class="min-w-0">
									<span class="block {n.readAt ? '' : 'font-medium'}">{n.title}</span>
									{#if n.body}<span class="mt-0.5 line-clamp-2 block text-xs text-slate-500">{n.body}</span>{/if}
									<span class="mt-0.5 block text-xs text-slate-400">{ago(n.createdAt)}</span>
								</span>
							</button>
						</li>
					{/each}
				</ul>
			{/if}
		</div>
	{/if}
</div>
