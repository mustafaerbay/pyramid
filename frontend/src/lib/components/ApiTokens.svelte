<script lang="ts">
	import { api } from '$lib/api/client';
	import type { components } from '$lib/api/schema';

	type Token = components['schemas']['ApiToken'];
	let tokens = $state<Token[]>([]);
	let name = $state('');
	let scope = $state<'read' | 'write'>('read');
	let days = $state<string>('90');
	let fresh = $state<string | null>(null);
	let error = $state<string | null>(null);
	const load = async () => (tokens = (await api.GET('/api/v1/me/tokens')).data?.data ?? []);
	$effect(() => {
		load();
	});
	async function create(e: SubmitEvent) {
		e.preventDefault();
		const { data, error: err } = await api.POST('/api/v1/me/tokens', { body: { name, scope, ...(days ? { expiresInDays: Number(days) } : {}) } });
		if (err || !data) return (error = err?.message ?? 'Could not make the token.');
		error = null;
		fresh = data.token;
		name = '';
		load();
	}
	async function revoke(t: Token) {
		if (!confirm(`Revoke "${t.name}"? Scripts using it stop working.`)) return;
		await api.DELETE('/api/v1/me/tokens/{id}', { params: { path: { id: t.id } } });
		load();
	}
	const when = (d?: string | null) => (d ? new Date(d).toLocaleDateString() : '—');
</script>

<div class="space-y-3 rounded-lg border border-slate-200 bg-white p-4 text-sm sm:p-6">
	<h2 class="text-lg font-semibold tracking-tight">API tokens</h2>
	<p class="text-slate-500">For scripts and integrations: send <code class="rounded bg-slate-100 px-1">Authorization: Bearer &lt;token&gt;</code>. A read token can only read.</p>
	{#if fresh}
		<div class="rounded-md border border-amber-200 bg-amber-50 p-3">
			<div class="font-medium text-amber-900">Copy your token now — it won't be shown again.</div>
			<div class="mt-1 flex gap-2">
				<code class="min-w-0 flex-1 break-all rounded bg-white px-2 py-1 text-xs">{fresh}</code>
				<button type="button" class="btn-secondary px-2 py-1 text-xs" onclick={() => navigator.clipboard?.writeText(fresh ?? '')}>Copy</button>
				<button type="button" class="btn-ghost px-2 py-1 text-xs" onclick={() => (fresh = null)}>Done</button>
			</div>
		</div>
	{/if}
	{#if error}<p class="text-red-700" role="alert">{error}</p>{/if}
	<ul class="divide-y divide-slate-100 rounded-md border border-slate-200">
		{#each tokens as t (t.id)}
			<li class="flex flex-wrap items-center gap-2 px-3 py-2">
				<span class="font-medium">{t.name}</span>
				<code class="text-xs text-slate-500">{t.prefix}…</code>
				<span class="rounded bg-slate-100 px-1.5 text-xs">{t.scope}</span>
				<span class="text-xs text-slate-500">used {when(t.lastUsedAt)} · expires {t.expiresAt ? when(t.expiresAt) : 'never'}</span>
				<button type="button" class="ml-auto text-xs text-red-600 hover:underline" onclick={() => revoke(t)}>Revoke</button>
			</li>
		{:else}
			<li class="px-3 py-3 text-slate-500">No tokens.</li>
		{/each}
	</ul>
	<form class="flex flex-wrap gap-2" onsubmit={create}>
		<input class="input min-w-0 flex-1" placeholder="Token name, e.g. CI export" aria-label="Token name" required maxlength="100" bind:value={name} />
		<select class="input w-auto" aria-label="Scope" bind:value={scope}>
			<option value="read">Read</option>
			<option value="write">Read and write</option>
		</select>
		<select class="input w-auto" aria-label="Expires" bind:value={days}>
			<option value="30">30 days</option>
			<option value="90">90 days</option>
			<option value="365">1 year</option>
			<option value="">Never</option>
		</select>
		<button type="submit" class="btn-primary">Create token</button>
	</form>
</div>
