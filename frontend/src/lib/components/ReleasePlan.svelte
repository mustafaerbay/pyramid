<script lang="ts">
	import type { components } from '$lib/api/schema';
	import Badge from '$lib/components/Badge.svelte';
	import type { BadgeTone } from '$lib/badgeTones';
	import {
		isAfterRelease,
		isOverdue,
		releasePlanGroups,
		taskProgress
	} from '$lib/releasePlan';

	type Version = components['schemas']['Version'];
	type VersionStatus = components['schemas']['VersionStatus'];
	type Milestone = components['schemas']['Milestone'];
	type MilestoneStatus = components['schemas']['MilestoneStatus'];
	type Task = components['schemas']['Task'];

	let {
		versions,
		milestones,
		tasks,
		onSelectVersion,
		onSelectMilestone
	}: {
		versions: Version[];
		milestones: Milestone[];
		tasks: Task[];
		onSelectVersion: (v: Version) => void;
		onSelectMilestone: (m: Milestone) => void;
	} = $props();

	const today = new Date().toISOString().slice(0, 10);

	const groups = $derived(releasePlanGroups(versions, milestones));

	function versionTone(status: VersionStatus): BadgeTone {
		switch (status) {
			case 'planned':
				return 'neutral';
			case 'in_progress':
				return 'warning';
			case 'released':
				return 'success';
			case 'cancelled':
				return 'neutralMuted';
		}
	}

	function milestoneDot(status: MilestoneStatus): string {
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

	function fmt(date: string): string {
		return new Date(`${date}T00:00:00Z`).toLocaleDateString('en-GB', {
			day: 'numeric',
			month: 'short',
			year: 'numeric',
			timeZone: 'UTC'
		});
	}

	function pct(p: { done: number; total: number }): number {
		return p.total === 0 ? 0 : Math.round((p.done / p.total) * 100);
	}
</script>

{#if groups.length === 0}
	<p class="text-sm text-slate-500">No versions yet. Add one to plan releases.</p>
{:else}
	<div class="divide-y divide-slate-100 rounded-lg border border-slate-200 bg-white">
		{#each groups as group (group.version?.id ?? '__none__')}
			{@const v = group.version}
			{@const progress = taskProgress(
				group.milestones.map((m) => m.id),
				tasks
			)}
			<div class="p-4 {v?.status === 'cancelled' ? 'opacity-60' : ''}">
				{#if v}
					<button
						type="button"
						class="flex w-full flex-wrap items-center gap-x-3 gap-y-1 text-left"
						onclick={() => onSelectVersion(v)}
					>
						<span class="font-semibold text-slate-900 hover:underline">{v.name}</span>
						<Badge tone={versionTone(v.status)}>{v.status.replace('_', ' ')}</Badge>
						{#if v.status === 'released' && v.releasedAt}
							<span class="text-xs text-slate-500">Released {fmt(v.releasedAt)}</span>
						{:else if v.targetDate}
							<span class="text-xs {isOverdue(v, today) ? 'font-medium text-red-600' : 'text-slate-500'}">
								Target {fmt(v.targetDate)}{isOverdue(v, today) ? ' · overdue' : ''}
							</span>
						{:else}
							<span class="text-xs text-slate-500">No target date</span>
						{/if}
						<span class="ml-auto flex items-center gap-2">
							<span
								class="h-2 w-28 overflow-hidden rounded-full bg-slate-100"
								role="meter"
								aria-label="{v.name} progress"
								aria-valuemin={0}
								aria-valuemax={100}
								aria-valuenow={pct(progress)}
							>
								<span
									class="block h-full rounded-full {v.status === 'released' ? 'bg-emerald-500' : 'bg-sky-500'}"
									style="width: {pct(progress)}%"
								></span>
							</span>
							<span class="w-14 text-right text-xs font-medium tabular-nums text-slate-600">
								{progress.done}/{progress.total}
							</span>
						</span>
					</button>
					{#if v.description}
						<p class="mt-1 text-xs text-slate-500">{v.description}</p>
					{/if}
				{:else}
					<p class="text-sm font-semibold text-slate-500">No version</p>
				{/if}

				{#if group.milestones.length > 0}
					<ul class="mt-2 space-y-1">
						{#each group.milestones as m (m.id)}
							{@const mp = taskProgress([m.id], tasks)}
							<li>
								<button
									type="button"
									class="flex w-full items-center gap-2 rounded px-2 py-1 text-left text-sm hover:bg-slate-50"
									onclick={() => onSelectMilestone(m)}
								>
									<span class="h-1.5 w-1.5 shrink-0 rounded-full {milestoneDot(m.status)}"></span>
									<span class="truncate text-slate-800">{m.name}</span>
									{#if v && isAfterRelease(m, v)}
										<span
											class="shrink-0 rounded bg-amber-50 px-1.5 text-xs text-amber-700"
											title="Due {m.dueDate}, after the {v.name} target date {v.targetDate}"
										>
											after release
										</span>
									{/if}
									<span class="ml-auto shrink-0 text-xs text-slate-500">{m.status.replace('_', ' ')}</span>
									<span class="w-12 shrink-0 text-right text-xs tabular-nums text-slate-500">
										{mp.done}/{mp.total}
									</span>
								</button>
							</li>
						{/each}
					</ul>
				{:else if v}
					<p class="mt-2 text-xs text-slate-500">No milestones in this version yet.</p>
				{/if}
			</div>
		{/each}
	</div>
{/if}
