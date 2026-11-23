<script lang="ts">
	import { goto } from '$app/navigation';
	import { page } from '$app/state';
	import { api } from '$lib/api/client';
	import { session } from '$lib/auth.svelte';
	import { safeNext } from '$lib/roles';

	let email = $state('');
	let password = $state('');
	let error = $state<string | null>(null);
	let busy = $state(false);

	async function submit(e: SubmitEvent) {
		e.preventDefault();
		busy = true;
		error = null;
		const { data, error: err } = await api.POST('/api/v1/auth/login', { body: { email, password } });
		busy = false;
		if (err || !data) {
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
		<label class="block text-sm">
			<span class="mb-1 block font-medium text-slate-700">Email</span>
			<input class="input" type="email" autocomplete="username" required bind:value={email} />
		</label>
		<label class="block text-sm">
			<span class="mb-1 block font-medium text-slate-700">Password</span>
			<input class="input" type="password" autocomplete="current-password" required bind:value={password} />
		</label>
		<button type="submit" class="btn-primary w-full" disabled={busy}>{busy ? 'Signing in…' : 'Sign in'}</button>
		<p class="text-center text-xs text-slate-500">Forgot your password? Ask an admin to set a new one.</p>
	</form>
</div>
