<script lang="ts">
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

	function statusBadgeClass(status: ProjectStatus): string {
		switch (status) {
			case 'active':
				return 'bg-emerald-100 text-emerald-700';
			case 'planning':
				return 'bg-sky-100 text-sky-700';
			case 'on_hold':
				return 'bg-amber-100 text-amber-700';
			case 'completed':
				return 'bg-slate-200 text-slate-600';
			case 'cancelled':
				return 'bg-red-100 text-red-700';
		}
	}

	// Also doubles as the timeline's "elapsed" fill color — same health/color
	// association used everywhere else in the app (badges, the old shared
	// timeline), just applied per-card now.
	function healthColorClass(health: ProjectHealth): string {
		switch (health) {
			case 'on_track':
				return 'bg-emerald-500';
			case 'at_risk':
				return 'bg-amber-500';
			case 'off_track':
				return 'bg-red-500';
		}
	}

	const DAY_MS = 86_400_000;
	function toUtc(dateStr: string): number {
		return new Date(`${dateStr}T00:00:00Z`).getTime();
	}
	function todayUtc(): number {
		const now = new Date();
		return Date.UTC(now.getUTCFullYear(), now.getUTCMonth(), now.getUTCDate());
	}
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
			<span class="rounded-full px-2 py-0.5 text-xs {statusBadgeClass(project.status)}">
				{project.status.replace('_', ' ')}
			</span>
			<span
				class="h-2 w-2 rounded-full {healthColorClass(project.health)}"
				title={project.health.replace('_', ' ')}
			></span>
		</div>
	</div>

	<div class="mt-3">
		{#if timeline}
			<div class="relative h-1.5 overflow-hidden rounded-full bg-slate-100">
				<div
					class="absolute inset-y-0 left-0 rounded-full {healthColorClass(project.health)}"
					style="width: {timeline.elapsedFraction * 100}%"
				></div>
			</div>
			<div class="mt-1 flex items-center justify-between text-[11px] text-slate-400">
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
			<p class="text-xs text-slate-400">No timeline set.</p>
		{/if}
	</div>

	<div class="mt-4 grid grid-cols-3 gap-2 border-t border-slate-100 pt-3 text-center">
		<div>
			<div class="text-sm font-semibold text-slate-900">{teamCount}</div>
			<div class="text-[11px] text-slate-500">Team</div>
		</div>
		<div>
			<div class="text-sm font-semibold text-slate-900">
				{avgAllocation !== null ? `${avgAllocation}%` : '—'}
			</div>
			<div class="text-[11px] text-slate-500">Avg. alloc.</div>
		</div>
		<div>
			<div class="text-sm font-semibold text-slate-900">{contributionCount}</div>
			<div class="text-[11px] text-slate-500">Contributions</div>
		</div>
	</div>
</a>
