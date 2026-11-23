<script lang="ts">
	import { api } from '$lib/api/client';
	import type { components } from '$lib/api/schema';

	type Channels = components['schemas']['NotificationChannels'];
	const TYPES: { key: string; label: string; hint: string }[] = [
		{ key: 'assignment', label: 'Assigned to me', hint: 'Someone puts you on an item' },
		{ key: 'status_change', label: 'My items move', hint: 'Items you work on change state' },
		{ key: 'mention', label: 'Mentions and comments', hint: '@you, and comments on your items' },
		{ key: 'follow', label: 'Items I follow', hint: 'Any change to an item you follow' },
		{ key: 'approval', label: 'Time off', hint: 'Requests to approve and decisions' },
		{ key: 'deadline', label: 'Deadlines', hint: 'Due dates coming up' },
		{ key: 'system', label: 'System', hint: 'Account and admin messages' }
	];
	let types = $state<Record<string, Channels>>({});
	let saved = $state(false);
	$effect(() => {
		api.GET('/api/v1/me/notification-settings').then((r) => (types = r.data?.types ?? {}));
	});
	async function set(key: string, field: keyof Channels, value: boolean) {
		types[key] = { ...(types[key] ?? { inApp: true, email: true }), [field]: value };
		const { data } = await api.PUT('/api/v1/me/notification-settings', { body: { types: { [key]: types[key] } } });
		if (data) types = data.types;
		saved = true;
		setTimeout(() => (saved = false), 1500);
	}
</script>

<div class="space-y-3 rounded-lg border border-slate-200 bg-white p-4 text-sm sm:p-6">
	<div class="flex items-center justify-between">
		<h2 class="text-lg font-semibold tracking-tight">Notifications</h2>
		{#if saved}<span class="text-xs text-emerald-700" role="status">Saved</span>{/if}
	</div>
	<table class="w-full">
		<thead class="text-xs text-slate-500">
			<tr><th class="pb-1 text-left font-medium">When</th><th class="pb-1 font-medium">In app</th><th class="pb-1 font-medium">Email</th></tr>
		</thead>
		<tbody class="divide-y divide-slate-100">
			{#each TYPES as t (t.key)}
				<tr>
					<td class="py-1.5"><div class="text-slate-800">{t.label}</div><div class="text-xs text-slate-500">{t.hint}</div></td>
					<td class="text-center"
						><input type="checkbox" aria-label="{t.label} in app" checked={types[t.key]?.inApp ?? true} onchange={(e) => set(t.key, 'inApp', e.currentTarget.checked)} /></td
					>
					<td class="text-center"
						><input type="checkbox" aria-label="{t.label} by email" checked={types[t.key]?.email ?? true} onchange={(e) => set(t.key, 'email', e.currentTarget.checked)} /></td
					>
				</tr>
			{/each}
		</tbody>
	</table>
	<p class="text-xs text-slate-500">Email goes out only when the server has email set up.</p>
</div>
