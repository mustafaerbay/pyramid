<script lang="ts">
	import { page } from '$app/state';
	import { api } from '$lib/api/client';

	const token = $derived(page.url.searchParams.get('token') ?? '');
	let password = $state('');
	let repeat = $state('');
	let done = $state(false);
	let busy = $state(false);
	let error = $state<string | null>(null);

	async function submit(e: SubmitEvent) {
		e.preventDefault();
		error = null;
		if (password !== repeat) return (error = "The passwords don't match.");
		busy = true;
		const { error: err } = await api.POST('/api/v1/auth/reset', { body: { token, newPassword: password } });
		busy = false;
		if (err) return (error = err.message ?? 'Could not set the password.');
		done = true;
	}
</script>

<div class="mx-auto mt-8 max-w-sm sm:mt-16">
	<div class="mb-6 text-center">
		<div class="text-2xl font-semibold tracking-tight">▲ Pyramid</div>
		<p class="mt-1 text-sm text-slate-500">Set a new password</p>
	</div>
	<div class="space-y-4 rounded-lg border border-slate-200 bg-white p-6 text-sm">
		{#if done}
			<p role="status">Your password is set. <a href="/login" class="font-medium underline">Sign in</a> with it.</p>
		{:else if !token}
			<p>This page needs the link from the email. <a href="/forgot" class="underline">Ask for a new one</a>.</p>
		{:else}
			<form class="space-y-4" onsubmit={submit}>
				{#if error}<div class="rounded-md border border-red-200 bg-red-50 p-3 text-red-700" role="alert">{error}</div>{/if}
				<label class="block">
					<span class="mb-1 block font-medium text-slate-700">New password</span>
					<input class="input" type="password" autocomplete="new-password" minlength="8" required bind:value={password} />
					<span class="mt-1 block text-xs text-slate-500">At least 8 characters.</span>
				</label>
				<label class="block">
					<span class="mb-1 block font-medium text-slate-700">Again</span>
					<input class="input" type="password" autocomplete="new-password" minlength="8" required bind:value={repeat} />
				</label>
				<button type="submit" class="btn-primary w-full" disabled={busy}>{busy ? 'Saving…' : 'Set password'}</button>
			</form>
		{/if}
	</div>
</div>
