<script lang="ts">
	import '../app.css';
	import { page } from '$app/state';
	import { goto } from '$app/navigation';
	import { can, loadMe, session, signOut } from '$lib/auth.svelte';
	import NotificationBell from '$lib/components/NotificationBell.svelte';
	import SearchBox from '$lib/components/SearchBox.svelte';
	import { i18n, setLocale, watchTranslations } from '$lib/i18n.svelte';
	import { onMount } from 'svelte';

	let { children } = $props();
	onMount(() => watchTranslations());

	// Top-level sections. A section with pages shows them as a second row of
	// tabs, so the top row stays short enough to fit on a tablet.
	type NavLink = { href: string; label: string };
	type NavSection = NavLink & { pages?: NavLink[] };
	const baseNav: NavSection[] = [
		{ href: '/', label: 'Home' },
		{ href: '/me', label: 'My Tasks' },
		{ href: '/projects', label: 'Projects' },
		{
			href: '/queries',
			label: 'Work',
			pages: [
				{ href: '/queries', label: 'Queries' },
				{ href: '/dashboards', label: 'Dashboards' },
				{ href: '/plans', label: 'Plans' }
			]
		},
		{
			href: '/people',
			label: 'Team',
			pages: [
				{ href: '/people', label: 'People' },
				{ href: '/contributions', label: 'Contributions' },
				{ href: '/goals', label: 'Growth' },
				{ href: '/time-off', label: 'Time Off' }
			]
		}
	];
	const adminNav: NavSection = {
		href: '/users',
		label: 'Admin',
		pages: [
			{ href: '/users', label: 'Accounts' },
			{ href: '/audit', label: 'Audit' }
		]
	};
	const navItems = $derived(can('admin') ? [...baseNav, adminNav] : baseNav);

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
		const path = page.url.pathname;
		if (href === '/') return path === '/';
		return path === href || path.startsWith(href + '/');
	}
	function sectionActive(s: NavSection): boolean {
		return s.pages ? s.pages.some((p) => isActive(p.href)) : isActive(s.href);
	}
	const section = $derived(navItems.find(sectionActive));

	// Phone menu: closed again whenever the route changes.
	let menuOpen = $state(false);
	$effect(() => {
		page.url.pathname;
		menuOpen = false;
	});

	const currentLabel = $derived(section?.pages?.find((p) => isActive(p.href))?.label ?? section?.label ?? '');
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
				<a href="/" class="inline-flex min-h-8 shrink-0 items-center gap-1.5 text-lg font-semibold tracking-tight"><span aria-hidden="true">▲</span> Pyramid</a>
				<span class="hidden text-sm text-slate-500 sm:inline">Software Department Management</span>
				<span class="min-w-0 truncate text-sm text-slate-500 sm:hidden">{currentLabel}</span>
				<div class="ml-auto flex shrink-0 items-center gap-1 sm:gap-2"><SearchBox /><NotificationBell /></div>
				<div class="hidden items-center gap-1 text-sm md:flex">
					<button
						type="button"
						class="inline-flex h-8 min-w-8 items-center justify-center rounded px-1.5 text-xs font-medium text-slate-600 ring-1 ring-inset ring-slate-200 hover:bg-slate-50"
						title={i18n.locale === 'tr' ? 'Switch to English' : 'Türkçe’ye geç'}
						onclick={() => setLocale(i18n.locale === 'tr' ? 'en' : 'tr')}>{i18n.locale === 'tr' ? 'EN' : 'TR'}</button
					>
					<a href="/account" class="inline-flex h-8 items-center rounded px-1.5 font-medium text-slate-700 hover:bg-slate-50 hover:underline" title="Your account">{firstName}</a>
					<button type="button" class="inline-flex h-8 items-center rounded px-1.5 text-slate-600 hover:bg-slate-50 hover:text-slate-900" onclick={doSignOut}>Sign out</button>
				</div>
				<button
					type="button"
					class="inline-flex h-10 w-10 shrink-0 items-center justify-center rounded-md text-slate-700 hover:bg-slate-100 md:hidden"
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
					{#each navItems as item (item.label)}
						{#if item.pages}
							<p class="px-3 pb-1 pt-3 text-xs font-semibold uppercase tracking-wide text-slate-500">{item.label}</p>
						{/if}
						{#each item.pages ?? [item] as link (link.href)}
							<a
								href={link.href}
								class="flex min-h-11 items-center rounded-md text-base font-medium {item.pages ? 'pl-5 pr-3' : 'px-3'} {isActive(link.href)
									? 'bg-slate-100 text-slate-900'
									: 'text-slate-600 hover:bg-slate-50'}"
								aria-current={isActive(link.href) ? 'page' : undefined}
							>
								{link.label}
							</a>
						{/each}
					{/each}
					<div class="mt-2 border-t border-slate-100 pt-2">
						<a href="/account" class="flex min-h-11 items-center rounded-md px-3 text-base font-medium text-slate-600 hover:bg-slate-50"
							>Your account ({firstName})</a
						>
						<button
							type="button"
							class="flex min-h-11 w-full items-center rounded-md px-3 text-left text-base font-medium text-slate-600 hover:bg-slate-50"
							onclick={() => setLocale(i18n.locale === 'tr' ? 'en' : 'tr')}>{i18n.locale === 'tr' ? 'English' : 'Türkçe'}</button
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
				{#each navItems as item (item.label)}
					{@const on = sectionActive(item)}
					<a
						href={item.href}
						class="whitespace-nowrap border-b-2 px-3 py-2.5 text-sm font-medium transition-colors {on
							? 'border-slate-900 text-slate-900'
							: 'border-transparent text-slate-600 hover:border-slate-300 hover:text-slate-900'}"
						aria-current={on && !item.pages ? 'page' : undefined}
					>
						{item.label}
					</a>
				{/each}
			</nav>
		</div>
	</header>
	{#if section?.pages}
		<nav class="border-b border-slate-200 bg-white" aria-label={section.label}>
			<ul class="mx-auto flex max-w-6xl gap-1 overflow-x-auto px-4 py-1.5 sm:px-6">
				{#each section.pages as sub (sub.href)}
					<li>
						<a
							href={sub.href}
							class="inline-flex min-h-9 items-center whitespace-nowrap rounded-md px-3 text-sm font-medium {isActive(sub.href)
								? 'bg-slate-100 text-slate-900'
								: 'text-slate-600 hover:bg-slate-50 hover:text-slate-900'}"
							aria-current={isActive(sub.href) ? 'page' : undefined}
						>
							{sub.label}
						</a>
					</li>
				{/each}
			</ul>
		</nav>
	{/if}
	<main class="mx-auto max-w-6xl px-4 py-6 sm:px-6 md:py-8">
		{@render children()}
	</main>
</div>
{/if}
