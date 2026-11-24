<script lang="ts">
	import { goto } from '$app/navigation';
	import { page } from '$app/state';
	import { api, baseUrl } from '$lib/api/client';
	import { session } from '$lib/auth.svelte';
	import { safeNext } from '$lib/roles';

	let email = $state('');
	let password = $state('');
	let code = $state('');
	let needCode = $state(false);
	let error = $state<string | null>(null);
	let busy = $state(false);
	let sso = $state<{ label: string } | null>(null);

	const SSO_ERRORS: Record<string, string> = {
		failed: 'Single sign-on did not complete. Try again.',
		no_account: 'Single sign-on worked, but there is no Pyramid account with that email. Ask an admin to add you.',
		unavailable: 'Single sign-on is not reachable right now.'
	};
	$effect(() => {
		api.GET('/api/v1/auth/options').then((r) => (sso = r.data?.sso ? { label: r.data.ssoLabel ?? 'Sign in with single sign-on' } : null));
		const s = page.url.searchParams.get('sso');
		if (s) error = SSO_ERRORS[s] ?? SSO_ERRORS.failed;
	});

	async function submit(e: SubmitEvent) {
		e.preventDefault();
		busy = true;
		error = null;
		const { data, error: err } = await api.POST('/api/v1/auth/login', { body: { email, password, ...(needCode ? { code } : {}) } });
		busy = false;
		if (err || !data) {
			if (err?.details === 'totp_required') {
				if (needCode) error = err.message ?? 'That code does not match.';
				needCode = true;
				code = '';
				return;
			}
			error = err?.message ?? 'Could not sign in.';
			return;
		}
		session.me = data;
		session.checked = true;
		goto(safeNext(page.url.searchParams.get('next')), { replaceState: true });
	}
</script>

<div class="mx-auto mt-8 max-w-sm sm:mt-16">
	<div class="mb-6 text-center">
		<div class="text-2xl font-semibold tracking-tight">▲ Pyramid</div>
		<p class="mt-1 text-sm text-slate-500">Sign in to continue</p>
	</div>
	<form class="space-y-4 rounded-lg border border-slate-200 bg-white p-6" onsubmit={submit}>
		{#if error}
			<div class="rounded-md border border-red-200 bg-red-50 p-3 text-sm text-red-700" role="alert">{error}</div>
		{/if}
		{#if needCode}
			<p class="text-sm text-slate-600">Enter the 6-digit code from your authenticator app, or one of your recovery codes.</p>
			<label class="block text-sm">
				<span class="mb-1 block font-medium text-slate-700">Code</span>
				<!-- svelte-ignore a11y_autofocus -->
				<input class="input tracking-widest" inputmode="numeric" autocomplete="one-time-code" required autofocus maxlength="20" bind:value={code} />
			</label>
			<button type="submit" class="btn-primary w-full" disabled={busy}>{busy ? 'Checking…' : 'Verify'}</button>
			<button type="button" class="w-full text-center text-xs text-slate-500 hover:underline" onclick={() => ((needCode = false), (error = null))}>Back</button>
		{:else}
			<label class="block text-sm">
				<span class="mb-1 block font-medium text-slate-700">Email</span>
				<input class="input" type="email" autocomplete="username" required bind:value={email} />
			</label>
			<label class="block text-sm">
				<span class="mb-1 block font-medium text-slate-700">Password</span>
				<input class="input" type="password" autocomplete="current-password" required bind:value={password} />
			</label>
			<button type="submit" class="btn-primary w-full" disabled={busy}>{busy ? 'Signing in…' : 'Sign in'}</button>
			{#if sso}
				<div class="flex items-center gap-2 text-xs text-slate-500"><span class="h-px flex-1 bg-slate-200"></span>or<span class="h-px flex-1 bg-slate-200"></span></div>
				<a class="btn-secondary block w-full text-center" href="{baseUrl.replace(/\/$/, '')}/api/v1/auth/oidc/start">{sso.label}</a>
			{/if}
			<p class="text-center text-xs text-slate-500"><a href="/forgot" class="hover:underline">Forgot your password?</a></p>
		{/if}
	</form>
</div>
