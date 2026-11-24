<script lang="ts">
	import type { ChartColor } from '$lib/chartColors';
	import { formatManDays } from '$lib/capacity';

	export type LoadMonth = {
		month: string; // YYYY-MM
		planned: number;
		/** Capacity this month after approved leave. */
		capacity: number;
		leaveDays: number;
		segments: { key: string; label: string; value: number; color: ChartColor }[];
	};

	let {
		months,
		capacity,
		selected,
		onSelect
	}: {
		months: LoadMonth[];
		capacity: number;
		selected: string;
		onSelect: (month: string) => void;
	} = $props();

	// `capacity` is the monthly capacity before leave (the dashed line);
	// months with approved leave get their own lower marker.
	// Leave headroom above the capacity line so a full month doesn't touch
	// the top, and grow when a month is over capacity.
	const max = $derived(Math.max(capacity * 1.25, ...months.map((m) => m.planned), 1));
	const capacityRatio = $derived(capacity / max);
	// Bars grow inside each column's content box, which is 1rem (pt-4)
	// shorter than the column, so lines are placed on the same scale.
	const at = (ratio: number) => `calc((100% - 1rem) * ${ratio})`;

	function initial(key: string): string {
		const [y, m] = key.split('-').map(Number);
		return new Date(y, m - 1, 1).toLocaleDateString('en-US', { month: 'short' });
	}
</script>

<div class="select-none">
	<div class="relative h-44 sm:h-52">
		<div class="absolute inset-0 flex items-end gap-1 sm:gap-2">
			{#each months as m (m.month)}
				{@const over = m.planned > m.capacity + 1e-9}
				{@const reduced = m.capacity < capacity - 1e-9}
				<button
					type="button"
					class="group relative flex h-full min-w-0 flex-1 flex-col justify-end rounded-t-md pt-4 {m.month === selected
						? 'bg-slate-100'
						: 'hover:bg-slate-50'}"
					aria-pressed={m.month === selected}
					aria-label="{initial(m.month)}: {formatManDays(m.planned)} of {formatManDays(m.capacity)} man-days{m.leaveDays
						? `, ${m.leaveDays} leave days`
						: ''}"
					title="{initial(m.month)}: {formatManDays(m.planned)} / {formatManDays(m.capacity)} md{m.leaveDays
						? ` (${m.leaveDays} leave days)`
						: ''}"
					onclick={() => onSelect(m.month)}
				>
					{#if m.planned > 0}
						<span
							class="mb-0.5 hidden text-center text-xs font-medium tabular-nums sm:block {over
								? 'text-red-700'
								: 'text-slate-600'}"
						>
							{formatManDays(m.planned)}
						</span>
					{/if}
					<div
						class="mx-auto flex w-full max-w-10 flex-col-reverse overflow-hidden rounded-t {over
							? 'ring-2 ring-red-500'
							: ''}"
						style="height: {(m.planned / max) * 100}%"
					>
						{#each m.segments as seg (seg.key)}
							<div class={seg.color.fill} style="height: {(seg.value / Math.max(m.planned, 1e-9)) * 100}%"></div>
						{/each}
					</div>
					{#if reduced}
						<!-- this month's capacity after leave -->
						<span
							class="pointer-events-none absolute inset-x-0.5 z-20 border-t-2 border-amber-500"
							style="bottom: {at(m.capacity / max)}"
						></span>
					{/if}
				</button>
			{/each}
		</div>

		<!-- capacity line, drawn over the bars -->
		<div
			class="pointer-events-none absolute inset-x-0 z-10 border-t border-dashed border-slate-500"
			style="bottom: {at(capacityRatio)}"
		>
			<span class="absolute -top-5 left-0 rounded bg-white/90 px-1 text-xs font-medium text-slate-600">
				{formatManDays(capacity)} md capacity
			</span>
		</div>
	</div>
	<div class="mt-1 flex gap-1 border-t border-slate-200 pt-1 sm:gap-2">
		{#each months as m (m.month)}
			<span
				class="min-w-0 flex-1 truncate text-center text-xs {m.month === selected
					? 'font-semibold text-slate-900'
					: 'text-slate-500'}"
			>
				{initial(m.month)}
			</span>
		{/each}
	</div>
</div>
