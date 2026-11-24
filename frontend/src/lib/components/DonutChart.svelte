<script lang="ts">
	import type { ChartColor } from '$lib/chartColors';

	type Segment = { label: string; value: number; color: ChartColor };

	let {
		data,
		size = 128,
		strokeWidth = 16,
		centerValue = '',
		centerLabel = '',
		showLegend = true
	}: {
		data: Segment[];
		size?: number;
		strokeWidth?: number;
		centerValue?: string;
		centerLabel?: string;
		// Turn off for a single-metric progress ring, where the center label
		// already says everything the legend would.
		showLegend?: boolean;
	} = $props();

	const total = $derived(data.reduce((sum, d) => sum + d.value, 0));
	const radius = $derived((size - strokeWidth) / 2);
	const circumference = $derived(2 * Math.PI * radius);

	// Each segment gets a slice of the ring's circumference proportional to
	// its share of the total, laid end to end via stroke-dashoffset.
	const segments = $derived(
		(() => {
			let cumulative = 0;
			const out: (Segment & { length: number; offset: number })[] = [];
			for (const d of data) {
				if (d.value <= 0) continue;
				const length = (d.value / Math.max(total, 1)) * circumference;
				out.push({ ...d, length, offset: cumulative });
				cumulative += length;
			}
			return out;
		})()
	);
</script>

<div class="flex flex-wrap items-center gap-5">
	<div class="relative shrink-0" style="width: {size}px; height: {size}px">
		<svg width={size} height={size} viewBox="0 0 {size} {size}" class="-rotate-90">
			<circle
				cx={size / 2}
				cy={size / 2}
				r={radius}
				fill="none"
				stroke-width={strokeWidth}
				class="stroke-slate-100"
			/>
			{#each segments as seg (seg.label)}
				<circle
					cx={size / 2}
					cy={size / 2}
					r={radius}
					fill="none"
					stroke-width={strokeWidth}
					stroke-dasharray="{seg.length} {circumference - seg.length}"
					stroke-dashoffset={-seg.offset}
					class={seg.color.stroke}
				/>
			{/each}
		</svg>
		{#if centerValue || centerLabel}
			<div class="absolute inset-0 flex flex-col items-center justify-center">
				<span class="text-lg font-semibold text-slate-900">{centerValue}</span>
				{#if centerLabel}
					<span class="text-xs text-slate-500">{centerLabel}</span>
				{/if}
			</div>
		{/if}
	</div>

	{#if showLegend}
		<div class="flex flex-1 flex-col gap-1.5 text-sm">
			{#if total === 0}
				<p class="text-slate-500">No data yet.</p>
			{:else}
				{#each data as d (d.label)}
					<span class="inline-flex items-center gap-1.5 text-slate-600">
						<span class="h-2 w-2 shrink-0 rounded-full {d.color.dot}"></span>
						{d.label}
						<span class="ml-auto font-medium text-slate-800">{d.value}</span>
					</span>
				{/each}
			{/if}
		</div>
	{/if}
</div>
