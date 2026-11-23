<script lang="ts">
	import { api } from '$lib/api/client';
	import type { components } from '$lib/api/schema';
	import { can, session } from '$lib/auth.svelte';
	import { ROLE_OPTIONS, type Role } from '$lib/roles';

	type Person = components['schemas']['Person'];
	type User = components['schemas']['User'];

	let people = $state<Person[]>([]);
	let users = $state<User[]>([]);
	let error = $state<string | null>(null);
	let banner = $state<{ kind: 'success' | 'error'; text: string } | null>(null);

	// Per-person drafts for new accounts and password resets.
	let newRole = $state<Record<string, Role>>({});
	let newPassword = $state<Record<string, string>>({});
	let resetFor = $state<string | null>(null);
	let resetPassword = $state('');

	async function load() {
		const [p, u] = await Promise.all([
			api.GET('/api/v1/people', { params: { query: { page: 1, pageSize: 100 } } }),
			api.GET('/api/v1/users')
		]);
		if (u.error) error = u.error.message ?? 'Failed to load accounts';
		people = (p.data?.data ?? []).sort((a, b) => `${a.firstName} ${a.lastName}`.localeCompare(`${b.firstName} ${b.lastName}`));
		users = u.data?.data ?? [];
	}

	$effect(() => {
		if (can('admin')) load();
	});

	const accountOf = (personId: string) => users.find((u) => u.personId === personId);

	function flash(kind: 'success' | 'error', text: string) {
		banner = { kind, text };
		setTimeout(() => (banner = null), 4000);
	}

	async function createAccount(p: Person) {
		const { data, error: err } = await api.POST('/api/v1/users', {
			body: { personId: p.id, role: newRole[p.id] ?? 'member', password: newPassword[p.id] ?? '' }
		});
		if (err || !data) return flash('error', err?.message ?? 'Could not create the account.');
		users = [...users, data];
		newPassword[p.id] = '';
		flash('success', `${p.firstName} can now sign in as ${data.email}. Share the password with them.`);
	}

	async function update(u: User, body: components['schemas']['UserUpdate'], done: string) {
		const { data, error: err } = await api.PATCH('/api/v1/users/{id}', { params: { path: { id: u.id } }, body });
		if (err || !data) {
			flash('error', err?.message ?? 'Could not update the account.');
			await load();
			return false;
		}
		users = users.map((x) => (x.id === data.id ? data : x));
		flash('success', done);
		return true;
	}

	async function doReset(u: User) {
		if (await update(u, { password: resetPassword }, 'Password set. Share it with them.')) {
			resetFor = null;
			resetPassword = '';
		}
	}

	const lastLogin = (u: User) => (u.lastLoginAt ? new Date(u.lastLoginAt).toLocaleDateString() : 'never');
</script>

<svelte:head><title>Accounts · Pyramid</title></svelte:head>

<section class="space-y-4">
	<div>
		<h1 class="text-2xl font-semibold tracking-tight">Accounts</h1>
		<p class="mt-1 text-sm text-slate-500">
			Who can sign in, and what they can do. Give someone an account with a first password and share it with them;
			they can change it under their account.
		</p>
	</div>

	{#if !can('admin')}
		<div class="rounded-lg border border-slate-200 bg-white p-6 text-sm text-slate-600">Only admins can manage accounts.</div>
	{:else}
		{#if error}
			<div class="rounded-lg border border-red-200 bg-red-50 p-4 text-sm text-red-700">{error}</div>
		{/if}
		{#if banner}
			<div
				class="rounded-md border p-3 text-sm {banner.kind === 'error'
					? 'border-red-200 bg-red-50 text-red-700'
					: 'border-emerald-200 bg-emerald-50 text-emerald-800'}"
				role="status"
			>
				{banner.text}
			</div>
		{/if}

		<ul class="divide-y divide-slate-100 rounded-lg border border-slate-200 bg-white">
			{#each people as p (p.id)}
				{@const u = accountOf(p.id)}
				<li class="flex flex-col gap-3 p-4 text-sm md:flex-row md:items-center">
					<div class="min-w-0 md:w-64">
						<div class="font-medium text-slate-900">
							{p.firstName}
							{p.lastName}
							{#if u && u.id === session.me?.user.id}<span class="text-xs font-normal text-slate-500">(you)</span>{/if}
						</div>
						<div class="break-all text-xs text-slate-500">{u?.email ?? p.email}</div>
					</div>
					{#if u}
						<div class="flex flex-1 flex-wrap items-center gap-2">
							<select
								class="input w-auto"
								aria-label="Role for {p.firstName}"
								value={u.role}
								onchange={(e) => update(u, { role: e.currentTarget.value as Role }, `${p.firstName} is now ${e.currentTarget.value}.`)}
							>
								{#each ROLE_OPTIONS as r (r.value)}
									<option value={r.value}>{r.label}</option>
								{/each}
							</select>
							<span class="text-xs {u.isActive ? 'text-emerald-700' : 'text-slate-500'}">
								{u.isActive ? 'Can sign in' : 'Turned off'} · last sign-in {lastLogin(u)}
							</span>
							<div class="flex gap-2 md:ml-auto">
								<button type="button" class="btn-secondary" onclick={() => ((resetFor = u.id), (resetPassword = ''))}>
									Set password
								</button>
								<button
									type="button"
									class="btn-secondary"
									title="Turn off two-factor sign-in (when they lost their phone)"
									onclick={async () => {
										if (!confirm(`Turn off two-factor sign-in for ${p.firstName}?`)) return;
										const { error: err } = await api.POST('/api/v1/users/{id}/reset-2fa', { params: { path: { id: u.id } } });
										flash(err ? 'error' : 'success', err ? (err.message ?? 'Could not reset.') : `${p.firstName} can sign in with just a password now.`);
									}}>Reset 2FA</button
								>
								<button
									type="button"
									class="btn-secondary {u.isActive ? 'text-red-600 hover:bg-red-50' : ''}"
									onclick={() =>
										update(u, { isActive: !u.isActive }, u.isActive ? `${p.firstName} can no longer sign in.` : `${p.firstName} can sign in again.`)}
								>
									{u.isActive ? 'Turn off' : 'Turn on'}
								</button>
							</div>
						</div>
						{#if resetFor === u.id}
							<form
								class="flex w-full flex-wrap items-center gap-2 md:w-auto"
								onsubmit={(e) => {
									e.preventDefault();
									doReset(u);
								}}
							>
								<input
									class="input min-w-0 flex-1 md:w-48"
									type="text"
									autocomplete="off"
									minlength="8"
									required
									placeholder="New password (8+)"
									aria-label="New password for {p.firstName}"
									bind:value={resetPassword}
								/>
								<button type="submit" class="btn-primary">Save</button>
								<button type="button" class="btn-ghost" onclick={() => (resetFor = null)}>Cancel</button>
							</form>
						{/if}
					{:else}
						<form
							class="flex flex-1 flex-wrap items-center gap-2"
							onsubmit={(e) => {
								e.preventDefault();
								createAccount(p);
							}}
						>
							<select class="input w-auto" aria-label="Role for {p.firstName}" bind:value={newRole[p.id]}>
								{#each ROLE_OPTIONS as r (r.value)}
									<option value={r.value} selected={r.value === 'member'}>{r.label}</option>
								{/each}
							</select>
							<input
								class="input min-w-0 flex-1 md:max-w-56"
								type="text"
								autocomplete="off"
								minlength="8"
								required
								placeholder="First password (8+)"
								aria-label="First password for {p.firstName}"
								bind:value={newPassword[p.id]}
							/>
							<button type="submit" class="btn-primary">Give access</button>
						</form>
					{/if}
				</li>
			{/each}
		</ul>
	{/if}
</section>
