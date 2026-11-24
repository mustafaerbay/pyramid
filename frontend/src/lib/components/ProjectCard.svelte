<script lang="ts">
	import Badge from '$lib/components/Badge.svelte';
	import MetricCard from '$lib/components/MetricCard.svelte';
	import { healthDotClass, type BadgeTone } from '$lib/badgeTones';
	import { DAY_MS, toUtc, todayUtc } from '$lib/dateMath';

	type ProjectStatus = 'planning' | 'active' | 'on_hold' | 'completed' | 'cancelled';
	type ProjectHealth = 'on_track' | 'at_risk' | 'off_track';

	let {
		project,
		teamCount,
		avgAllocation,
		contributionCount
	}: {
		project: {
			id: string;
			key: string;
			name: string;
			status: ProjectStatus;
			health: ProjectHealth;
			startDate?: string | null;
			targetDate?: string | null;
		};
		teamCount: number;
		avgAllocation: number | null;
		contributionCount: number;
	} = $props();

	function statusTone(status: ProjectStatus): BadgeTone {
		switch (status) {
			case 'active':
				return 'success';
			case 'planning':
				return 'info';
			case 'on_hold':
				return 'warning';
			case 'completed':
				return 'neutralMuted';
			case 'cancelled':
				return 'danger';
		}
	}

	// healthDotClass (from badgeTones) also doubles as the timeline's
	// "elapsed" fill color — same health/color association used everywhere
	// else in the app, just applied per-card now.

	const today = todayUtc();

	// Each card's timeline uses its OWN start..target range (not a shared
	// axis across projects) — the point here is a quick per-project glance,
	// not cross-project comparison.
	const timeline = $derived(
		(() => {
			if (!project.startDate || !project.targetDate) return null;
			const start = toUtc(project.startDate);
			const target = toUtc(project.targetDate);
			const durationMs = Math.max(target - start, DAY_MS);
			const elapsedFraction = Math.min(Math.max((today - start) / durationMs, 0), 1);
			const daysLeft = Math.round((target - today) / DAY_MS);
			// A completed project past its original target isn't "overdue," it's
			// done — and a cancelled one was called off, not late.
			const overdue = today > target && project.status !== 'completed' && project.status !== 'cancelled';
			return { elapsedFraction, daysLeft, overdue };
		})()
	);
</script>

<a
	href="/projects/{project.id}"
	class="block rounded-lg border border-slate-200 bg-white p-5 transition-shadow hover:shadow-sm"
>
	<div class="flex items-start justify-between gap-2">
		<div class="min-w-0">
			<span class="font-mono text-xs font-medium text-slate-500">{project.key}</span>
			<p class="truncate font-medium text-slate-900">{project.name}</p>
		</div>
		<div class="flex shrink-0 items-center gap-1.5">
			<Badge tone={statusTone(project.status)}>{project.status.replace('_', ' ')}</Badge>
			<span
				class="h-2 w-2 rounded-full {healthDotClass[project.health]}"
				title={project.health.replace('_', ' ')}
			></span>
		</div>
	</div>

	<div class="mt-3">
		{#if timeline}
			<div class="relative h-1.5 overflow-hidden rounded-full bg-slate-100">
				<div
					class="absolute inset-y-0 left-0 rounded-full {healthDotClass[project.health]}"
					style="width: {timeline.elapsedFraction * 100}%"
				></div>
			</div>
			<div class="mt-1 flex items-center justify-between text-xs text-slate-500">
				<span>{project.startDate}</span>
				<span class={timeline.overdue ? 'font-medium text-red-600' : ''}>
					{project.status === 'completed'
						? 'Completed'
						: project.status === 'cancelled'
							? 'Cancelled'
							: timeline.overdue
								? `${Math.abs(timeline.daysLeft)}d overdue`
								: `${timeline.daysLeft}d left`}
				</span>
				<span>{project.targetDate}</span>
			</div>
		{:else}
			<p class="text-xs text-slate-500">No timeline set.</p>
		{/if}
	</div>

	<div class="mt-4 grid grid-cols-3 gap-2 border-t border-slate-100 pt-3">
		<MetricCard size="compact" label="Team" value={teamCount} />
		<MetricCard size="compact" label="Avg. alloc." value={avgAllocation !== null ? `${avgAllocation}%` : '—'} />
		<MetricCard size="compact" label="Contributions" value={contributionCount} />
	</div>
</a>
