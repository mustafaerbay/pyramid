<script lang="ts">
	import { chartColors } from '$lib/chartColors';

	type TimelineProject = {
		id: string;
		key: string;
		name: string;
		status: string;
		health: 'on_track' | 'at_risk' | 'off_track';
		startDate: string;
		targetDate: string;
	};

	let { projects }: { projects: TimelineProject[] } = $props();

	// All date math is done in UTC-midnight milliseconds — parsing a plain
	// "YYYY-MM-DD" as `${d}T00:00:00Z` and building "today" from UTC date
	// parts (not local wall-clock time) keeps the "today" marker lined up
	// with the bars regardless of the viewer's timezone.
	function toUtc(dateStr: string): number {
		return new Date(`${dateStr}T00:00:00Z`).getTime();
	}
	function todayUtc(): number {
		const now = new Date();
		return Date.UTC(now.getUTCFullYear(), now.getUTCMonth(), now.getUTCDate());
	}

	const today = todayUtc();
	const DAY_MS = 86_400_000;

	const healthColor = {
		on_track: chartColors.emerald,
		at_risk: chartColors.amber,
		off_track: chartColors.red
	};

	// Shared axis: spans every project's start..target, widened to include
	// today (so an upcoming or overdue project's "today" marker is never
	// clipped) plus a small pad so bars aren't flush against the edges.
	const range = $derived(
		(() => {
			if (projects.length === 0) return null;
			let min = Infinity;
			let max = -Infinity;
			for (const p of projects) {
				min = Math.min(min, toUtc(p.startDate));
				max = Math.max(max, toUtc(p.targetDate));
			}
			min = Math.min(min, today);
			max = Math.max(max, today);
			const pad = Math.max((max - min) * 0.04, DAY_MS);
			return { min: min - pad, max: max + pad };
		})()
	);

	function pct(ms: number, r: { min: number; max: number }): number {
		return ((ms - r.min) / (r.max - r.min)) * 100;
	}

	const rows = $derived(
		range === null
			? []
			: projects.map((p) => {
					const start = toUtc(p.startDate);
					const target = toUtc(p.targetDate);
					const left = pct(start, range);
					const width = Math.max(pct(target, range) - left, 1);
					const durationMs = Math.max(target - start, DAY_MS);
					const elapsedFraction = Math.min(Math.max((today - start) / durationMs, 0), 1);
					const daysLeft = Math.round((target - today) / DAY_MS);
					// "Overdue" only means something for a project that's still
					// supposed to be moving — a completed project that finished
					// past its original target isn't in trouble, it's just done.
					const overdue = today > target && p.status !== 'completed';
					return {
						...p,
						left,
						width,
						elapsedFraction,
						daysLeft,
						overdue,
						color: healthColor[p.health]
					};
				})
	);

	const todayPct = $derived(range ? pct(today, range) : 0);
</script>

{#if rows.length === 0}
	<p class="text-sm text-slate-500">No projects have both a start and target date yet.</p>
{:else}
	<div class="relative">
		<div class="pointer-events-none absolute top-0 bottom-0 z-10 w-px bg-slate-900/60" style="left: {todayPct}%"
		></div>
		<div class="space-y-3">
			{#each rows as row (row.id)}
				<div class="flex items-center gap-3">
					<a
						href="/projects/{row.id}"
						class="w-36 shrink-0 truncate text-sm font-medium text-slate-700 hover:underline"
						title="{row.key} — {row.name}"
					>
						{row.key}
					</a>
					<div class="relative h-5 flex-1 rounded bg-slate-50">
						<div
							class="absolute top-0 h-full rounded {row.color.soft}"
							style="left: {row.left}%; width: {row.width}%"
						></div>
						<div
							class="absolute top-0 h-full rounded {row.color.fill}"
							style="left: {row.left}%; width: {row.width * row.elapsedFraction}%"
						></div>
					</div>
					<span
						class="w-24 shrink-0 text-right text-xs {row.overdue
							? 'font-medium text-red-600'
							: 'text-slate-500'}"
					>
						{row.status === 'completed'
							? 'Completed'
							: row.overdue
								? `${Math.abs(row.daysLeft)}d overdue`
								: `${row.daysLeft}d left`}
					</span>
				</div>
			{/each}
		</div>
	</div>
	<div class="mt-3 flex items-center gap-1.5 text-xs text-slate-500">
		<span class="inline-block h-3 w-px bg-slate-900/60"></span>
		Today
	</div>
{/if}
