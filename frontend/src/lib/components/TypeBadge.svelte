<script lang="ts">
	import { typeMeta } from '$lib/workItems';

	// A project's own type shows its own name and colour.
	let { type, label = null, color = null }: { type?: string; label?: string | null; color?: string | null } = $props();
	const COLORS: Record<string, string> = {
		orange: 'bg-orange-100 text-orange-800 ring-orange-200',
		violet: 'bg-violet-100 text-violet-800 ring-violet-200',
		sky: 'bg-sky-100 text-sky-800 ring-sky-200',
		red: 'bg-red-100 text-red-800 ring-red-200',
		amber: 'bg-amber-50 text-amber-800 ring-amber-200',
		emerald: 'bg-emerald-100 text-emerald-800 ring-emerald-200',
		pink: 'bg-pink-100 text-pink-800 ring-pink-200',
		slate: 'bg-slate-100 text-slate-700 ring-slate-300'
	};
	const meta = $derived(typeMeta(type));
	const text = $derived(label ?? meta.label);
	const cls = $derived(label && color ? (COLORS[color] ?? meta.badge) : meta.badge);
</script>

<span class="inline-flex shrink-0 rounded px-1.5 py-px text-xs font-semibold uppercase tracking-wide ring-1 ring-inset {cls}">{text}</span>
