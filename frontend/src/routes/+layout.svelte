<script lang="ts">
	import '../app.css';
	import { page } from '$app/state';

	let { children } = $props();

	// One entry per module. New modules get a tab here as they're built.
	const navItems = [
		{ href: '/', label: 'Dashboard' },
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
</script>

<div class="min-h-screen bg-slate-50 text-slate-900">
	<header class="border-b border-slate-200 bg-white">
		<div class="mx-auto max-w-6xl px-6">
			<div class="flex items-center gap-2 py-4">
				<span class="text-lg font-semibold tracking-tight">▲ Pyramid</span>
				<span class="text-sm text-slate-500">Software Department Management</span>
			</div>
			<nav class="-mb-px flex gap-1 overflow-x-auto" aria-label="Primary">
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
	<main class="mx-auto max-w-6xl px-6 py-8">
		{@render children()}
	</main>
</div>
