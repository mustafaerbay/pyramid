<script lang="ts">
	type MilestoneStatus = 'pending' | 'in_progress' | 'completed' | 'missed';

	type TimelineMilestone = {
		id: string;
		name: string;
		status: MilestoneStatus;
		startDate?: string | null;
		dueDate?: string | null;
	};

	let {
		milestones,
		projectStartDate = null,
		projectTargetDate = null,
		onSelect
	}: {
		milestones: TimelineMilestone[];
		// Anchor the displayed range even before any milestone has a date set,
		// so the timeline isn't just "today ± a couple months" for a brand-new
		// project.
		projectStartDate?: string | null;
		projectTargetDate?: string | null;
		onSelect?: (milestone: TimelineMilestone) => void;
	} = $props();

	// The whole timeline is laid out with percentages against a single
	// container width, so it always fits the screen without horizontal
	// scrolling — wide date spans just compress the pixels-per-day ratio
	// instead of growing the DOM sideways. This cap keeps month labels from
	// compressing into illegible slivers on an extreme outlier date; dates
	// beyond it are surfaced via the "further out" note below, never silently
	// dropped.
	const MAX_MONTHS = 24;
	// Point-in-time milestones (due date only, or same-day start/due) still
	// need a visible minimum width to read as a chip — wide enough to show a
	// few characters of the name, not just the status dot.
	const MIN_BAR_PCT = 9;

	function statusDotClass(status: MilestoneStatus): string {
		switch (status) {
			case 'pending':
				return 'bg-slate-400';
			case 'in_progress':
				return 'bg-amber-500';
			case 'completed':
				return 'bg-emerald-500';
			case 'missed':
				return 'bg-red-500';
		}
	}

	function statusChipClass(status: MilestoneStatus): string {
		switch (status) {
			case 'pending':
				return 'border-slate-200 bg-slate-50';
			case 'in_progress':
				return 'border-amber-200 bg-amber-50';
			case 'completed':
				return 'border-emerald-200 bg-emerald-50';
			case 'missed':
				return 'border-red-200 bg-red-50';
		}
	}

	function monthKey(year: number, month: number): string {
		return `${year}-${String(month + 1).padStart(2, '0')}`;
	}
	function monthLabel(year: number, month: number): string {
		return new Date(Date.UTC(year, month, 1)).toLocaleDateString('en-US', {
			month: 'short',
			year: 'numeric',
			timeZone: 'UTC'
		});
	}
	function monthIndexOf(year: number, month: number): number {
		return year * 12 + month;
	}
	function partsFromDateStr(dateStr: string): { year: number; month: number; day: number } {
		const [y, m, d] = dateStr.split('-').map(Number);
		return { year: y, month: m - 1, day: d };
	}
	// Whole-day index (days since the Unix epoch) — the common unit every
	// position/width on the timeline is computed against.
	function dayIndex(year: number, month: number, day: number): number {
		return Math.floor(Date.UTC(year, month, day) / 86_400_000);
	}
	function daysInMonth(year: number, month: number): number {
		return new Date(Date.UTC(year, month + 1, 0)).getUTCDate();
	}

	const now = new Date();
	const today = { year: now.getUTCFullYear(), month: now.getUTCMonth(), day: now.getUTCDate() };
	const todayDay = dayIndex(today.year, today.month, today.day);

	const dated = $derived(milestones.filter((m) => m.startDate || m.dueDate));
	const undated = $derived(milestones.filter((m) => !m.startDate && !m.dueDate));

	// Resolve a milestone to a concrete day range. Due-only or start-only
	// milestones resolve to a zero-length point range at that single date.
	function resolveRange(m: TimelineMilestone): { startDay: number; endDay: number } | null {
		const s = m.startDate ? partsFromDateStr(m.startDate) : null;
		const d = m.dueDate ? partsFromDateStr(m.dueDate) : null;
		const sDay = s ? dayIndex(s.year, s.month, s.day) : d ? dayIndex(d.year, d.month, d.day) : null;
		const eDay = d ? dayIndex(d.year, d.month, d.day) : s ? dayIndex(s.year, s.month, s.day) : null;
		if (sDay === null || eDay === null) return null;
		return sDay <= eDay ? { startDay: sDay, endDay: eDay } : { startDay: eDay, endDay: sDay };
	}

	const range = $derived(
		(() => {
			const points: { year: number; month: number }[] = [{ year: today.year, month: today.month }];
			if (projectStartDate) {
				const p = partsFromDateStr(projectStartDate);
				points.push({ year: p.year, month: p.month });
			}
			if (projectTargetDate) {
				const p = partsFromDateStr(projectTargetDate);
				points.push({ year: p.year, month: p.month });
			}
			for (const m of dated) {
				if (m.startDate) {
					const p = partsFromDateStr(m.startDate);
					points.push({ year: p.year, month: p.month });
				}
				if (m.dueDate) {
					const p = partsFromDateStr(m.dueDate);
					points.push({ year: p.year, month: p.month });
				}
			}
			let min = points[0];
			let max = points[0];
			for (const p of points) {
				if (monthIndexOf(p.year, p.month) < monthIndexOf(min.year, min.month)) min = p;
				if (monthIndexOf(p.year, p.month) > monthIndexOf(max.year, max.month)) max = p;
			}
			if (monthIndexOf(max.year, max.month) - monthIndexOf(min.year, min.month) + 1 > MAX_MONTHS) {
				const clampedIndex = monthIndexOf(min.year, min.month) + MAX_MONTHS - 1;
				max = { year: Math.floor(clampedIndex / 12), month: clampedIndex % 12 };
			}
			return { min, max };
		})()
	);

	const rangeStartDay = $derived(dayIndex(range.min.year, range.min.month, 1));
	const rangeEndDay = $derived(
		dayIndex(range.max.year, range.max.month, daysInMonth(range.max.year, range.max.month))
	);
	const totalDays = $derived(Math.max(rangeEndDay - rangeStartDay + 1, 1));

	function pct(day: number): number {
		return ((day - rangeStartDay) / totalDays) * 100;
	}

	const months = $derived(
		(() => {
			const out: {
				year: number;
				month: number;
				key: string;
				label: string;
				isCurrent: boolean;
				leftPct: number;
				widthPct: number;
			}[] = [];
			const start = monthIndexOf(range.min.year, range.min.month);
			const end = monthIndexOf(range.max.year, range.max.month);
			for (let i = start; i <= end; i++) {
				const year = Math.floor(i / 12);
				const month = i % 12;
				out.push({
					year,
					month,
					key: monthKey(year, month),
					label: monthLabel(year, month),
					isCurrent: year === today.year && month === today.month,
					leftPct: pct(dayIndex(year, month, 1)),
					widthPct: (daysInMonth(year, month) / totalDays) * 100
				});
			}
			return out;
		})()
	);

	const todayVisible = $derived(todayDay >= rangeStartDay && todayDay <= rangeEndDay);
	const todayPct = $derived(pct(todayDay));

	// Positioned, lane-packed dated milestones. Lanes only multiply when
	// milestones' date ranges genuinely overlap — otherwise everything sits
	// on a single row.
	const positioned = $derived(
		(() => {
			type Positioned = TimelineMilestone & {
				leftPct: number;
				widthPct: number;
				sortStart: number;
				sortEnd: number;
				isRange: boolean;
				lane: number;
			};
			const items: Positioned[] = [];
			for (const m of dated) {
				const r = resolveRange(m);
				if (!r || r.startDay > rangeEndDay) continue;
				const clampedStart = Math.max(r.startDay, rangeStartDay);
				const clampedEnd = Math.min(r.endDay, rangeEndDay);
				const leftPct = pct(clampedStart);
				const rawWidthPct = pct(clampedEnd) - leftPct;
				const widthPct = Math.min(Math.max(rawWidthPct, MIN_BAR_PCT), 100 - leftPct);
				items.push({
					...m,
					leftPct,
					widthPct,
					sortStart: r.startDay,
					sortEnd: r.endDay,
					isRange: !!(m.startDate && m.dueDate),
					lane: 0
				});
			}
			items.sort((a, b) => a.sortStart - b.sortStart || a.sortEnd - b.sortEnd);
			const laneEnds: number[] = [];
			for (const item of items) {
				let placed = false;
				for (let lane = 0; lane < laneEnds.length; lane++) {
					if (laneEnds[lane] <= item.sortStart) {
						item.lane = lane;
						laneEnds[lane] = item.sortEnd;
						placed = true;
						break;
					}
				}
				if (!placed) {
					item.lane = laneEnds.length;
					laneEnds.push(item.sortEnd);
				}
			}
			return items;
		})()
	);

	const laneCount = $derived(Math.max(1, ...positioned.map((p) => p.lane + 1)));

	// Milestones whose date range falls entirely after the (possibly capped)
	// visible range — surfaced as a note rather than silently dropped.
	const clippedCount = $derived(
		dated.filter((m) => {
			const r = resolveRange(m);
			return r !== null && r.startDay > rangeEndDay;
		}).length
	);
</script>

<div class="space-y-3">
	{#if milestones.length === 0}
		<p class="text-sm text-slate-400">No milestones yet.</p>
	{:else}
		<div class="w-full">
			<div class="relative h-6 border-b border-slate-200 text-xs">
				{#each months as m (m.key)}
					<div
						class="absolute top-0 h-full overflow-hidden border-l border-slate-100 pl-1.5 font-medium text-nowrap {m.isCurrent
							? 'text-slate-900'
							: 'text-slate-500'}"
						style="left: {m.leftPct}%; width: {m.widthPct}%;"
					>
						{m.label}
					</div>
				{/each}
			</div>

			<div class="relative" style="height: {laneCount * 2.25 + 0.25}rem;">
				{#each months as m (m.key)}
					<div class="absolute top-0 bottom-0 border-l border-slate-100" style="left: {m.leftPct}%;"></div>
				{/each}
				{#if todayVisible}
					<div class="absolute top-0 bottom-0 w-px bg-slate-400" style="left: {todayPct}%;"></div>
				{/if}

				{#each positioned as milestone (milestone.id)}
					<button
						type="button"
						class="absolute flex items-center gap-1.5 overflow-hidden rounded-md border px-2 text-left text-xs {statusChipClass(
							milestone.status
						)} hover:shadow-sm"
						style="left: {milestone.leftPct}%; width: {milestone.widthPct}%; top: {milestone.lane *
							2.25}rem; height: 2rem;"
						onclick={() => onSelect?.(milestone)}
						title={milestone.name}
					>
						<span class="h-1.5 w-1.5 shrink-0 rounded-full {statusDotClass(milestone.status)}"></span>
						<span class="truncate font-medium text-slate-800">{milestone.name}</span>
					</button>
				{/each}
			</div>
		</div>

		{#if clippedCount > 0}
			<p class="text-xs text-slate-400">
				{clippedCount}
				{clippedCount === 1 ? 'milestone is' : 'milestones are'} further out than this view shows.
			</p>
		{/if}

		{#if undated.length > 0}
			<div>
				<p class="mb-1.5 text-xs font-medium text-slate-500">No due date set</p>
				<div class="flex flex-wrap gap-1.5">
					{#each undated as milestone (milestone.id)}
						<button
							type="button"
							class="rounded-md border px-2 py-1 text-left text-xs {statusChipClass(milestone.status)} hover:shadow-sm"
							onclick={() => onSelect?.(milestone)}
						>
							<span class="flex items-center gap-1.5">
								<span class="h-1.5 w-1.5 shrink-0 rounded-full {statusDotClass(milestone.status)}"></span>
								<span class="text-slate-800">{milestone.name}</span>
							</span>
						</button>
					{/each}
				</div>
			</div>
		{/if}
	{/if}
</div>
