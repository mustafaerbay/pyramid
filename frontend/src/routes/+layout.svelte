<script lang="ts">
	import '../app.css';
	import { page } from '$app/state';
	import { goto } from '$app/navigation';
	import { can, loadMe, session, signOut } from '$lib/auth.svelte';
	import NotificationBell from '$lib/components/NotificationBell.svelte';
	import SearchBox from '$lib/components/SearchBox.svelte';

	let { children } = $props();

	// One entry per module. New modules get a tab here as they're built.
	const baseNav = [
		{ href: '/', label: 'Dashboard' },
		{ href: '/me', label: 'My Tasks' },
		{ href: '/people', label: 'People' },
		{ href: '/projects', label: 'Projects' },
		{ href: '/queries', label: 'Queries' },
		{ href: '/dashboards', label: 'Dashboards' },
		{ href: '/plans', label: 'Plans' },
		{ href: '/contributions', label: 'Contributions' },
		{ href: '/goals', label: 'Growth' },
		{ href: '/time-off', label: 'Time Off' }
	];
	const navItems = $derived(can('admin') ? [...baseNav, { href: '/users', label: 'Accounts' }, { href: '/audit', label: 'Audit' }] : baseNav);

	// Every page but /login needs a signed-in account.
	// Pages that work signed out.
	const onLogin = $derived(['/login', '/forgot', '/reset'].includes(page.url.pathname));
	$effect(() => {
		if (onLogin || session.checked) return;
		loadMe().then((me) => {
			if (!me) goto(`/login?next=${encodeURIComponent(page.url.pathname + page.url.search)}`, { replaceState: true });
		});
	});
	$effect(() => {
		if (!onLogin && session.checked && !session.me) goto('/login', { replaceState: true });
	});

	// Read-only accounts: CSS hides write controls (see app.css).
	$effect(() => {
		document.documentElement.toggleAttribute('data-viewer', !!session.me && !can('member'));
	});

	async function doSignOut() {
		await signOut();
		goto('/login', { replaceState: true });
	}
	const firstName = $derived(session.me?.person.firstName ?? '');

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

<svelte:head><title>{onLogin ? (page.url.pathname === '/login' ? 'Sign in · Pyramid' : 'Password · Pyramid') : 'Pyramid'}</title></svelte:head>

{#if onLogin}
	<div class="min-h-screen bg-slate-50 px-4 py-6 text-slate-900">{@render children()}</div>
{:else if !session.me}
	<div class="flex min-h-screen items-center justify-center bg-slate-50 text-sm text-slate-500">Loading…</div>
{:else}
<div class="min-h-screen bg-slate-50 text-slate-900">
	<header class="border-b border-slate-200 bg-white">
		<div class="mx-auto max-w-6xl px-4 sm:px-6">
			<div class="flex items-center gap-2 py-3 md:py-4">
				<span class="text-lg font-semibold tracking-tight">▲ Pyramid</span>
				<span class="hidden text-sm text-slate-500 sm:inline">Software Department Management</span>
				<span class="text-sm text-slate-500 sm:hidden">{currentLabel}</span>
				<div class="ml-auto flex items-center gap-2"><SearchBox /><NotificationBell /></div>
				<div class="hidden items-center gap-3 text-sm md:flex">
					<a href="/account" class="font-medium text-slate-700 hover:underline" title="Your account">{firstName}</a>
					<button type="button" class="text-slate-500 hover:text-slate-800 hover:underline" onclick={doSignOut}>Sign out</button>
				</div>
				<button
					type="button"
					class="inline-flex h-10 w-10 items-center justify-center rounded-md text-slate-700 hover:bg-slate-100 md:hidden"
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
					<div class="mt-2 border-t border-slate-100 pt-2">
						<a href="/account" class="flex min-h-11 items-center rounded-md px-3 text-base font-medium text-slate-600 hover:bg-slate-50"
							>Your account ({firstName})</a
						>
						<button
							type="button"
							class="flex min-h-11 w-full items-center rounded-md px-3 text-left text-base font-medium text-slate-600 hover:bg-slate-50"
							onclick={doSignOut}>Sign out</button
						>
					</div>
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
{/if}
