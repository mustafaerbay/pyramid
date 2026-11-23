<script lang="ts">
	import { api } from '$lib/api/client';

	let email = $state('');
	let sent = $state(false);
	let busy = $state(false);
	let error = $state<string | null>(null);
	let emailOn = $state<boolean | null>(null);

	$effect(() => {
		api.GET('/api/v1/auth/options').then((r) => (emailOn = r.data?.emailReset ?? false));
	});

	async function submit(e: SubmitEvent) {
		e.preventDefault();
		busy = true;
		error = null;
		const { error: err } = await api.POST('/api/v1/auth/forgot', { body: { email } });
		busy = false;
		if (err) return (error = err.message ?? 'Something went wrong.');
		sent = true;
	}
</script>

<div class="mx-auto mt-8 max-w-sm sm:mt-16">
	<div class="mb-6 text-center">
		<div class="text-2xl font-semibold tracking-tight">▲ Pyramid</div>
		<p class="mt-1 text-sm text-slate-500">Forgot your password?</p>
	</div>
	<div class="space-y-4 rounded-lg border border-slate-200 bg-white p-6 text-sm">
		{#if emailOn === false}
			<p>Password emails aren't set up here yet. Ask an admin to set a new password for you under Accounts.</p>
		{:else if sent}
			<p role="status">If <b>{email}</b> has an account, a link to set a new password is on its way. It works once, for an hour.</p>
		{:else}
			<form class="space-y-4" onsubmit={submit}>
				{#if error}<div class="rounded-md border border-red-200 bg-red-50 p-3 text-red-700" role="alert">{error}</div>{/if}
				<label class="block">
					<span class="mb-1 block font-medium text-slate-700">Your email</span>
					<input class="input" type="email" autocomplete="username" required bind:value={email} />
				</label>
				<button type="submit" class="btn-primary w-full" disabled={busy || emailOn === null}>{busy ? 'Sending…' : 'Email me a link'}</button>
			</form>
		{/if}
		<a href="/login" class="block text-center text-slate-500 hover:underline">Back to sign in</a>
	</div>
</div>
