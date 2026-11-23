<script lang="ts">
	import type { components } from '$lib/api/schema';

	let { e, showRepo = true }: { e: components['schemas']['GitEvent']; showRepo?: boolean } = $props();
	const ICON: Record<string, string> = { commit: '●', pull_request: '⇄', pipeline: '▶' };
	const TONE: Record<string, string> = {
		open: 'bg-emerald-50 text-emerald-800 ring-emerald-200',
		merged: 'bg-violet-50 text-violet-800 ring-violet-200',
		closed: 'bg-slate-100 text-slate-700 ring-slate-300',
		success: 'bg-emerald-50 text-emerald-800 ring-emerald-200',
		failed: 'bg-red-50 text-red-800 ring-red-200',
		running: 'bg-amber-50 text-amber-800 ring-amber-200',
		cancelled: 'bg-slate-100 text-slate-700 ring-slate-300'
	};
	const LABEL: Record<string, string> = { success: '✓ passed', failed: '✕ failed', running: '… running', cancelled: 'cancelled' };
</script>

<div class="flex flex-wrap items-center gap-x-2 gap-y-0.5 text-sm">
	<span class="w-4 text-center text-slate-400" title={e.kind.replace('_', ' ')} aria-hidden="true">{ICON[e.kind]}</span>
	{#if e.url}<a href={e.url} target="_blank" rel="noopener noreferrer" class="min-w-0 truncate font-medium text-slate-800 hover:underline">{e.title}</a>
	{:else}<span class="min-w-0 truncate font-medium text-slate-800">{e.title}</span>{/if}
	{#if e.state}<span class="rounded-full px-1.5 text-[11px] font-medium ring-1 ring-inset {TONE[e.state] ?? TONE.closed}">{LABEL[e.state] ?? e.state}</span>{/if}
	<span class="w-full pl-6 text-xs text-slate-500 sm:ml-auto sm:w-auto sm:pl-0">
		{#if e.kind === 'commit' && e.sha}<code>{e.sha.slice(0, 7)}</code> · {/if}{e.author ?? ''}{e.ref ? ` · ${e.ref}` : ''}{showRepo ? ` · ${e.repoName}` : ''} · {new Date(e.occurredAt).toLocaleString()}
	</span>
</div>
