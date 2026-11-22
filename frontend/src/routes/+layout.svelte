<script lang="ts">
	import '../app.css';
	import { page } from '$app/state';

	let { children } = $props();

	// One entry per module. New modules get a tab here as they're built.
	const navItems = [
		{ href: '/', label: 'Dashboard' },
		{ href: '/me', label: 'My Tasks' },
		{ href: '/people', label: 'People' },
		{ href: '/projects', label: 'Projects' },
		{ href: '/contributions', label: 'Contributions' },
		{ href: '/goals', label: 'Growth' },
		{ href: '/time-off', label: 'Time Off' }
	];

	function isActive(href: string): boolean {
		if (href === '/') return page.url.pathname === '/';
		return page.url.pathname.startsWith(href);
	}

	// Phone menu: closed again whenever the route changes.
	let menuOpen = $state(false);
	$effect(() => {
		page.url.pathname;
		menuOpen = false;
	});

	const currentLabel = $derived(navItems.find((i) => isActive(i.href))?.label ?? '');
</script>

<div class="min-h-screen bg-slate-50 text-slate-900">
	<header class="border-b border-slate-200 bg-white">
		<div class="mx-auto max-w-6xl px-4 sm:px-6">
			<div class="flex items-center gap-2 py-3 md:py-4">
				<span class="text-lg font-semibold tracking-tight">▲ Pyramid</span>
				<span class="hidden text-sm text-slate-500 sm:inline">Software Department Management</span>
				<span class="text-sm text-slate-500 sm:hidden">{currentLabel}</span>
				<button
					type="button"
					class="ml-auto inline-flex h-10 w-10 items-center justify-center rounded-md text-slate-700 hover:bg-slate-100 md:hidden"
					aria-label={menuOpen ? 'Close menu' : 'Open menu'}
					aria-expanded={menuOpen}
					aria-controls="mobile-nav"
					onclick={() => (menuOpen = !menuOpen)}
				>
					{#if menuOpen}
						<svg class="h-6 w-6" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2" aria-hidden="true"><path d="M6 6l12 12M18 6L6 18" /></svg>
					{:else}
						<svg class="h-6 w-6" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2" aria-hidden="true"><path d="M4 7h16M4 12h16M4 17h16" /></svg>
					{/if}
				</button>
			</div>
			{#if menuOpen}
				<nav id="mobile-nav" class="border-t border-slate-100 pb-3 pt-2 md:hidden" aria-label="Primary">
					{#each navItems as item (item.href)}
						<a
							href={item.href}
							class="flex min-h-11 items-center rounded-md px-3 text-base font-medium {isActive(item.href)
								? 'bg-slate-100 text-slate-900'
								: 'text-slate-600 hover:bg-slate-50'}"
							aria-current={isActive(item.href) ? 'page' : undefined}
						>
							{item.label}
						</a>
					{/each}
				</nav>
			{/if}
			<nav class="-mb-px hidden gap-1 md:flex" aria-label="Primary">
				{#each navItems as item (item.href)}
					<a
						href={item.href}
						class="whitespace-nowrap border-b-2 px-3 py-2.5 text-sm font-medium transition-colors {isActive(
							item.href
						)
							? 'border-slate-900 text-slate-900'
							: 'border-transparent text-slate-500 hover:border-slate-300 hover:text-slate-700'}"
						aria-current={isActive(item.href) ? 'page' : undefined}
					>
						{item.label}
					</a>
				{/each}
			</nav>
		</div>
	</header>
	<main class="mx-auto max-w-6xl px-4 py-6 sm:px-6 md:py-8">
		{@render children()}
	</main>
</div>
