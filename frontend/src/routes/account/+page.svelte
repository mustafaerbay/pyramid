<script lang="ts">
	import { api } from '$lib/api/client';
	import { session } from '$lib/auth.svelte';
	import { ROLE_OPTIONS } from '$lib/roles';
	import NotificationSettings from '$lib/components/NotificationSettings.svelte';
	import ApiTokens from '$lib/components/ApiTokens.svelte';

	let current = $state('');
	let next = $state('');
	let repeat = $state('');
	let busy = $state(false);
	let message = $state<{ kind: 'success' | 'error'; text: string } | null>(null);

	const me = $derived(session.me);
	const role = $derived(ROLE_OPTIONS.find((r) => r.value === me?.user.role));

	async function submit(e: SubmitEvent) {
		e.preventDefault();
		message = null;
		if (next !== repeat) {
			message = { kind: 'error', text: "The new passwords don't match." };
			return;
		}
		busy = true;
		const { error } = await api.POST('/api/v1/auth/password', {
			body: { currentPassword: current, newPassword: next }
		});
		busy = false;
		if (error) {
			message = { kind: 'error', text: error.message ?? 'Could not change the password.' };
			return;
		}
		current = next = repeat = '';
		message = { kind: 'success', text: 'Password changed. You stay signed in here; other devices are signed out.' };
	}
</script>

<svelte:head><title>Your account · Pyramid</title></svelte:head>

<section class="mx-auto max-w-lg space-y-6">
	<h1 class="text-2xl font-semibold tracking-tight">Your account</h1>

	{#if me}
		<div class="rounded-lg border border-slate-200 bg-white p-4 text-sm sm:p-6">
			<div class="text-base font-medium text-slate-900">{me.person.firstName} {me.person.lastName}</div>
			<div class="break-all text-slate-600">{me.user.email}</div>
			<div class="mt-2 text-slate-600">
				Role: <span class="font-medium text-slate-900">{role?.label}</span>
				<span class="text-slate-500">· {role?.hint}</span>
			</div>
		</div>
	{/if}

	<NotificationSettings />

	<ApiTokens />

	<form class="space-y-4 rounded-lg border border-slate-200 bg-white p-4 sm:p-6" onsubmit={submit}>
		<h2 class="text-lg font-semibold tracking-tight">Change password</h2>
		{#if message}
			<div
				class="rounded-md border p-3 text-sm {message.kind === 'error'
					? 'border-red-200 bg-red-50 text-red-700'
					: 'border-emerald-200 bg-emerald-50 text-emerald-800'}"
				role="status"
			>
				{message.text}
			</div>
		{/if}
		<label class="block text-sm">
			<span class="mb-1 block font-medium text-slate-700">Current password</span>
			<input class="input" type="password" autocomplete="current-password" required bind:value={current} />
		</label>
		<label class="block text-sm">
			<span class="mb-1 block font-medium text-slate-700">New password</span>
			<input class="input" type="password" autocomplete="new-password" minlength="8" required bind:value={next} />
			<span class="mt-1 block text-xs text-slate-500">At least 8 characters.</span>
		</label>
		<label class="block text-sm">
			<span class="mb-1 block font-medium text-slate-700">New password again</span>
			<input class="input" type="password" autocomplete="new-password" minlength="8" required bind:value={repeat} />
		</label>
		<button type="submit" class="btn-primary w-full sm:w-auto" disabled={busy}>
			{busy ? 'Saving…' : 'Change password'}
		</button>
	</form>
</section>
