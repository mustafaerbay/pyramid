<script lang="ts">
	let {
		label,
		value,
		danger = false,
		size = 'default'
	}: {
		label: string;
		value: string | number;
		// Red value text for an "overdue"-style metric — mirrors the one place
		// this was already conditional (project detail's "Days to target").
		danger?: boolean;
		// 'default' = project-page/section header stat (uppercase tracking-wide
		// label, larger left-aligned value). 'compact' = card-internal stat
		// (ProjectCard's 3-up row) — centered, smaller, denser. These were two
		// deliberately different treatments before this extraction, not one
		// pattern accidentally drifting into two — kept as two variants of one
		// component rather than forced into a single look.
		size?: 'default' | 'compact';
	} = $props();
</script>

{#if size === 'compact'}
	<div class="text-center">
		<div class="text-sm font-semibold {danger ? 'text-red-600' : 'text-slate-900'}">{value}</div>
		<div class="text-xs text-slate-500">{label}</div>
	</div>
{:else}
	<div>
		<dt class="text-xs font-medium tracking-wide text-slate-500 uppercase">{label}</dt>
		<dd class="mt-1 text-xl font-semibold {danger ? 'text-red-600' : 'text-slate-900'}">{value}</dd>
	</div>
{/if}
