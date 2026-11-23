<script lang="ts">
	import { api } from '$lib/api/client';
	import type { components } from '$lib/api/schema';

	let status = $state<components['schemas']['TwoFactorStatus'] | null>(null);
	let setup = $state<components['schemas']['TwoFactorSetup'] | null>(null);
	let code = $state('');
	let password = $state('');
	let codes = $state<string[] | null>(null);
	let error = $state<string | null>(null);
	let disabling = $state(false);
	const load = async () => (status = (await api.GET('/api/v1/auth/2fa')).data ?? null);
	$effect(() => {
		load();
	});
	async function start() {
		error = null;
		const { data, error: err } = await api.POST('/api/v1/auth/2fa/setup');
		if (err || !data) return (error = 'Could not start.');
		setup = data;
	}
	async function enable(e: SubmitEvent) {
		e.preventDefault();
		const { data, error: err } = await api.POST('/api/v1/auth/2fa/enable', { body: { code } });
		if (err || !data) return (error = err?.message ?? 'That code does not match.');
		error = null;
		codes = data.recoveryCodes;
		setup = null;
		code = '';
		load();
	}
	async function disable(e: SubmitEvent) {
		e.preventDefault();
		const { error: err } = await api.POST('/api/v1/auth/2fa/disable', { body: { password } });
		if (err) return (error = err.message ?? 'Could not turn it off.');
		error = null;
		password = '';
		disabling = false;
		load();
	}
	const grouped = (s: string) => s.match(/.{1,4}/g)?.join(' ') ?? s;
</script>

<div class="space-y-3 rounded-lg border border-slate-200 bg-white p-4 text-sm sm:p-6">
	<div class="flex items-center justify-between">
		<h2 class="text-lg font-semibold tracking-tight">Two-factor sign-in</h2>
		{#if status}<span class="rounded-full px-2 py-0.5 text-xs font-medium {status.enabled ? 'bg-emerald-50 text-emerald-800' : 'bg-slate-100 text-slate-600'}">{status.enabled ? 'On' : 'Off'}</span>{/if}
	</div>
	{#if error}<p class="text-red-700" role="alert">{error}</p>{/if}
	{#if codes}
		<div class="rounded-md border border-amber-200 bg-amber-50 p-3">
			<div class="font-medium text-amber-900">Save these recovery codes somewhere safe. Each works once if you lose your phone.</div>
			<ul class="mt-2 grid grid-cols-2 gap-1 font-mono text-xs">{#each codes as c (c)}<li>{c}</li>{/each}</ul>
			<button type="button" class="mt-2 text-xs underline" onclick={() => (codes = null)}>I saved them</button>
		</div>
	{/if}
	{#if status?.enabled}
		<p class="text-slate-600">Signing in asks for a code from your authenticator app. {status.recoveryCodesLeft} recovery codes left.</p>
		{#if disabling}
			<form class="flex flex-wrap gap-2" onsubmit={disable}>
				<input class="input min-w-0 flex-1" type="password" placeholder="Your password" aria-label="Password" required autocomplete="current-password" bind:value={password} />
				<button type="submit" class="btn-secondary text-red-700">Turn off</button>
				<button type="button" class="btn-ghost" onclick={() => (disabling = false)}>Cancel</button>
			</form>
		{:else}
			<button type="button" class="btn-ghost px-2 py-1 text-xs text-red-600" onclick={() => (disabling = true)}>Turn off…</button>
		{/if}
	{:else if setup}
		<ol class="list-decimal space-y-2 pl-5 text-slate-700">
			<li>
				In your authenticator app (Google Authenticator, Microsoft Authenticator, 1Password …) add an account with this key:
				<code class="mt-1 block break-all rounded bg-slate-100 px-2 py-1 font-mono text-sm">{grouped(setup.secret)}</code>
				<a class="text-xs text-indigo-700 underline" href={setup.otpauthUrl}>or open it in the app on this device</a>
			</li>
			<li>
				Type the 6-digit code it shows:
				<form class="mt-1 flex gap-2" onsubmit={enable}>
					<input class="input w-32 tracking-widest" inputmode="numeric" autocomplete="one-time-code" maxlength="6" aria-label="Code" required bind:value={code} />
					<button type="submit" class="btn-primary">Turn on</button>
				</form>
			</li>
		</ol>
	{:else}
		<p class="text-slate-600">Add a code from your phone to signing in, so a stolen password isn't enough.</p>
		<button type="button" class="btn-secondary" onclick={start}>Set up</button>
	{/if}
</div>
