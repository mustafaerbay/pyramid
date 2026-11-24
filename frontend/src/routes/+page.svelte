<script lang="ts">
	import { onMount } from 'svelte';
	import { api } from '$lib/api/client';
	import type { components } from '$lib/api/schema';
	import { chartColors, type ChartColorName } from '$lib/chartColors';
	import ChartCard from '$lib/components/ChartCard.svelte';
	import DonutChart from '$lib/components/DonutChart.svelte';
	import BarChart from '$lib/components/BarChart.svelte';
	import StackedBar from '$lib/components/StackedBar.svelte';
	import TrendChart from '$lib/components/TrendChart.svelte';
	import ProjectCard from '$lib/components/ProjectCard.svelte';
	import CapacityCard from '$lib/components/CapacityCard.svelte';
	import { DAY_MS, toUtc, todayUtc } from '$lib/dateMath';

	type Person = components['schemas']['Person'];
	type Project = components['schemas']['Project'];
	type Contribution = components['schemas']['Contribution'];
	type Goal = components['schemas']['Goal'];
	type LeaveRequest = components['schemas']['LeaveRequest'];
	type ProjectMember = components['schemas']['ProjectMember'];

	type ModuleKey = 'people' | 'projects' | 'contributions' | 'goals' | 'timeOff';

	// "Today" at UTC midnight, computed once. Every date comparison below
	// parses plain "YYYY-MM-DD" values as `${d}T00:00:00Z` (via the shared
	// dateMath helpers) and compares against this, so results don't shift
	// with the viewer's timezone.
	const TODAY = todayUtc();

	let counts = $state<Record<ModuleKey, number | null>>({
		people: null,
		projects: null,
		contributions: null,
		goals: null,
		timeOff: null
	});

	let people = $state<Person[]>([]);
	let projects = $state<Project[]>([]);
	let contributions = $state<Contribution[]>([]);
	let goals = $state<Goal[]>([]);
	let leaveRequests = $state<LeaveRequest[]>([]);
	let projectMembers = $state<ProjectMember[]>([]);
	let loading = $state(true);
	let loadError = $state<string | null>(null);

	const modules: { key: ModuleKey; href: string; title: string; description: string }[] = [
		{
			key: 'people',
			href: '/people',
			title: 'People',
			description: 'Team members, org structure, and skills.'
		},
		{
			key: 'projects',
			href: '/projects',
			title: 'Projects',
			description: 'Projects, milestones, sprints, and tasks.'
		},
		{
			key: 'contributions',
			href: '/contributions',
			title: 'Contributions',
			description: 'Commits, pull requests, reviews, and logged time.'
		},
		{
			key: 'goals',
			href: '/goals',
			title: 'Growth',
			description: 'Goals, OKRs, and performance reviews.'
		},
		{
			key: 'timeOff',
			href: '/time-off',
			title: 'Time Off',
			description: 'Leave requests and approvals.'
		}
	];

	// Fixed-order category -> color maps, one per breakdown. Order is fixed
	// (not derived from whatever's present in the data) so a chart's legend
	// and segment order never shuffle as counts change, and each color is a
	// literal ChartColor so Tailwind's scanner can see the classes it uses.
	const personStatusOrder: { key: Person['status']; label: string; color: ChartColorName }[] = [
		{ key: 'active', label: 'Active', color: 'emerald' },
		{ key: 'on_leave', label: 'On leave', color: 'amber' },
		{ key: 'offboarding', label: 'Offboarding', color: 'orange' },
		{ key: 'inactive', label: 'Inactive', color: 'slate' }
	];

	const employmentTypeOrder: { key: Person['employmentType']; label: string; color: ChartColorName }[] = [
		{ key: 'full_time', label: 'Full time', color: 'emerald' },
		{ key: 'part_time', label: 'Part time', color: 'sky' },
		{ key: 'contractor', label: 'Contractor', color: 'violet' },
		{ key: 'intern', label: 'Intern', color: 'cyan' }
	];

	const contributionTypeOrder: { key: Contribution['type']; label: string; color: ChartColorName }[] = [
		{ key: 'commit', label: 'Commit', color: 'slate' },
		{ key: 'pull_request', label: 'Pull request', color: 'violet' },
		{ key: 'code_review', label: 'Code review', color: 'sky' },
		{ key: 'issue', label: 'Issue', color: 'amber' },
		{ key: 'design', label: 'Design', color: 'pink' },
		{ key: 'documentation', label: 'Documentation', color: 'indigo' },
		{ key: 'deployment', label: 'Deployment', color: 'emerald' },
		{ key: 'support', label: 'Support', color: 'cyan' },
		{ key: 'other', label: 'Other', color: 'slateDark' }
	];

	const goalStatusOrder: { key: Goal['status']; label: string; color: ChartColorName }[] = [
		{ key: 'draft', label: 'Draft', color: 'slate' },
		{ key: 'active', label: 'Active', color: 'emerald' },
		{ key: 'achieved', label: 'Achieved', color: 'sky' },
		{ key: 'partially_achieved', label: 'Partially achieved', color: 'amber' },
		{ key: 'missed', label: 'Missed', color: 'red' },
		{ key: 'cancelled', label: 'Cancelled', color: 'red' }
	];

	const leaveTypeOrder: { key: LeaveRequest['type']; label: string; color: ChartColorName }[] = [
		{ key: 'vacation', label: 'Vacation', color: 'sky' },
		{ key: 'sick', label: 'Sick', color: 'red' },
		{ key: 'personal', label: 'Personal', color: 'violet' },
		{ key: 'parental', label: 'Parental', color: 'pink' },
		{ key: 'bereavement', label: 'Bereavement', color: 'slateDark' },
		{ key: 'unpaid', label: 'Unpaid', color: 'orange' },
		{ key: 'other', label: 'Other', color: 'slate' }
	];

	function breakdown<K extends string>(
		items: K[],
		order: { key: K; label: string; color: ChartColorName }[]
	) {
		const tally: Record<string, number> = {};
		for (const item of items) tally[item] = (tally[item] ?? 0) + 1;
		return order.map((o) => ({ label: o.label, value: tally[o.key] ?? 0, color: chartColors[o.color] }));
	}

	const personStatusData = $derived(breakdown(people.map((p) => p.status), personStatusOrder));
	const employmentTypeData = $derived(
		breakdown(people.map((p) => p.employmentType), employmentTypeOrder)
	);
	const contributionTypeData = $derived(
		breakdown(contributions.map((c) => c.type), contributionTypeOrder)
	);
	const goalStatusData = $derived(breakdown(goals.map((g) => g.status), goalStatusOrder));
	const leaveTypeData = $derived(breakdown(leaveRequests.map((l) => l.type), leaveTypeOrder));

	const avgGoalProgress = $derived(
		goals.length === 0
			? 0
			: Math.round(goals.reduce((sum, g) => sum + g.progressPct, 0) / goals.length)
	);

	const goalProgressRingData = $derived([
		{ label: 'Progress', value: avgGoalProgress, color: chartColors.emerald },
		{ label: 'Remaining', value: 100 - avgGoalProgress, color: chartColors.slate }
	]);

	// Contributions logged per week, last 8 weeks (Mon–Sun buckets), from
	// whatever page of contributions was fetched (see the pageSize note
	// below). Weeks are labeled by their Monday, "M/D".
	const weeklyContributionData = $derived(
		(() => {
			const weeks: { start: number; label: string; value: number }[] = [];
			const dayOfWeek = new Date(TODAY).getUTCDay(); // 0 = Sunday
			const daysSinceMonday = (dayOfWeek + 6) % 7;
			const thisMonday = TODAY - daysSinceMonday * DAY_MS;

			for (let i = 7; i >= 0; i--) {
				const start = thisMonday - i * 7 * DAY_MS;
				const d = new Date(start);
				weeks.push({
					start,
					label: `${d.getUTCMonth() + 1}/${d.getUTCDate()}`,
					value: 0
				});
			}

			for (const c of contributions) {
				const occurred = new Date(c.occurredAt).getTime();
				for (const w of weeks) {
					if (occurred >= w.start && occurred < w.start + 7 * DAY_MS) {
						w.value += 1;
						break;
					}
				}
			}

			return weeks.map((w) => ({ label: w.label, value: w.value }));
		})()
	);

	// Per-project rollups for the card grid below: active (not-yet-left)
	// team size + average allocation from project-members, and a
	// contribution count from the (capped-at-100, see the pageSize note
	// below) contributions list.
	const projectTeamStats = $derived(
		(() => {
			const map: Record<string, { count: number; totalAllocation: number }> = {};
			for (const m of projectMembers) {
				if (m.leftAt) continue;
				const entry = map[m.projectId] ?? { count: 0, totalAllocation: 0 };
				entry.count += 1;
				entry.totalAllocation += m.allocationPct;
				map[m.projectId] = entry;
			}
			return map;
		})()
	);

	const projectContributionCounts = $derived(
		(() => {
			const map: Record<string, number> = {};
			for (const c of contributions) {
				if (!c.projectId) continue;
				map[c.projectId] = (map[c.projectId] ?? 0) + 1;
			}
			return map;
		})()
	);

	// Active-ish projects first, then by soonest target date within a
	// status group (undated projects sort last within their group) — the
	// point is "what needs attention" reads top-to-bottom, left-to-right.
	const projectStatusRank: Record<Project['status'], number> = {
		active: 0,
		planning: 1,
		on_hold: 2,
		completed: 3,
		cancelled: 4
	};

	const sortedProjects = $derived(
		[...projects].sort((a, b) => {
			const rankDiff = projectStatusRank[a.status] - projectStatusRank[b.status];
			if (rankDiff !== 0) return rankDiff;
			if (!a.targetDate && !b.targetDate) return 0;
			if (!a.targetDate) return 1;
			if (!b.targetDate) return -1;
			return a.targetDate.localeCompare(b.targetDate);
		})
	);

	// People whose approved leave spans today.
	const peopleById = $derived(
		(() => {
			const map: Record<string, Person> = {};
			for (const p of people) map[p.id] = p;
			return map;
		})()
	);

	const offTodayList = $derived(
		leaveRequests
			.filter(
				(l) => l.status === 'approved' && toUtc(l.startDate) <= TODAY && toUtc(l.endDate) >= TODAY
			)
			.map((l) => ({ ...l, person: peopleById[l.personId] }))
			.sort((a, b) => a.endDate.localeCompare(b.endDate))
	);

	onMount(async () => {
		loading = true;
		loadError = null;
		// pageSize 100 is the API max. Counts (the tiles up top) come from
		// `pagination.total`, which is exact even past 100 — the charts below
		// are built from the fetched rows only, so on a department larger
		// than 100 people/projects/etc. they reflect "the first page", not
		// the whole org. Fine at the scale this app targets; worth swapping
		// for server-side aggregate endpoints if that changes.
		//
		// allSettled (not all) so one slow/broken endpoint degrades that
		// section to its empty state instead of blanking the whole page —
		// and everything runs through try/finally so a fully unreachable API
		// still clears `loading` instead of spinning forever.
		try {
			const [peopleRes, projectsRes, contributionsRes, goalsRes, leaveRes, membersRes] =
				await Promise.allSettled([
					api.GET('/api/v1/people', { params: { query: { page: 1, pageSize: 100 } } }),
					api.GET('/api/v1/projects', { params: { query: { page: 1, pageSize: 100 } } }),
					api.GET('/api/v1/contributions', { params: { query: { page: 1, pageSize: 100 } } }),
					api.GET('/api/v1/goals', { params: { query: { page: 1, pageSize: 100 } } }),
					api.GET('/api/v1/leave-requests', { params: { query: { page: 1, pageSize: 100 } } }),
					api.GET('/api/v1/project-members', { params: { query: { page: 1, pageSize: 100 } } })
				]);

			if (peopleRes.status === 'fulfilled' && peopleRes.value.data) {
				people = peopleRes.value.data.data;
				counts = { ...counts, people: peopleRes.value.data.pagination.total };
			}
			if (projectsRes.status === 'fulfilled' && projectsRes.value.data) {
				projects = projectsRes.value.data.data;
				counts = { ...counts, projects: projectsRes.value.data.pagination.total };
			}
			if (contributionsRes.status === 'fulfilled' && contributionsRes.value.data) {
				contributions = contributionsRes.value.data.data;
				counts = { ...counts, contributions: contributionsRes.value.data.pagination.total };
			}
			if (goalsRes.status === 'fulfilled' && goalsRes.value.data) {
				goals = goalsRes.value.data.data;
				counts = { ...counts, goals: goalsRes.value.data.pagination.total };
			}
			if (leaveRes.status === 'fulfilled' && leaveRes.value.data) {
				leaveRequests = leaveRes.value.data.data;
				counts = { ...counts, timeOff: leaveRes.value.data.pagination.total };
			}
			if (membersRes.status === 'fulfilled' && membersRes.value.data) {
				projectMembers = membersRes.value.data.data;
			}

			const allFailed = [peopleRes, projectsRes, contributionsRes, goalsRes, leaveRes].every(
				(r) => r.status === 'rejected' || !r.value.data
			);
			if (allFailed) {
				loadError = "Couldn't reach the API. Make sure the Go backend is running on localhost:8080.";
			}
		} catch (err) {
			loadError = err instanceof Error ? err.message : 'Failed to load dashboard data.';
		} finally {
			loading = false;
		}
	});
</script>

<section class="space-y-6">
	<div>
		<h1 class="text-2xl font-semibold tracking-tight">Welcome back</h1>
		<p class="mt-1 text-slate-500">An overview of your software department.</p>
	</div>

	<div class="grid grid-cols-1 gap-4 sm:grid-cols-2 lg:grid-cols-5">
		{#each modules as m (m.href)}
			<a
				href={m.href}
				class="rounded-lg border border-slate-200 bg-white p-5 transition-shadow hover:shadow-sm"
			>
				<div class="flex items-center justify-between">
					<h2 class="font-medium text-slate-900">{m.title}</h2>
					<span class="rounded-full bg-slate-100 px-2 py-0.5 text-xs text-slate-600">
						{counts[m.key] ?? '…'}
					</span>
				</div>
				<p class="mt-1 text-sm text-slate-500">{m.description}</p>
			</a>
		{/each}
	</div>

	{#if loading}
		<p class="text-slate-500">Loading dashboard…</p>
	{:else if loadError}
		<div class="rounded-lg border border-red-200 bg-red-50 p-4 text-red-700">
			<p class="font-medium">Couldn't load the dashboard.</p>
			<p class="text-sm">{loadError}</p>
		</div>
	{:else}
		<CapacityCard {projects} />

		<div>
			<div class="mb-3 flex items-baseline justify-between">
				<h2 class="text-lg font-semibold tracking-tight">Projects</h2>
				<a href="/projects" class="text-sm text-slate-500 hover:text-slate-700">View all →</a>
			</div>
			{#if sortedProjects.length === 0}
				<div class="rounded-lg border border-slate-200 bg-white p-8 text-center text-slate-500">
					No projects yet.
				</div>
			{:else}
				<div class="grid grid-cols-1 gap-4 md:grid-cols-2 xl:grid-cols-3">
					{#each sortedProjects as project (project.id)}
						<ProjectCard
							{project}
							teamCount={projectTeamStats[project.id]?.count ?? 0}
							avgAllocation={projectTeamStats[project.id]
								? Math.round(
										projectTeamStats[project.id].totalAllocation / projectTeamStats[project.id].count
									)
								: null}
							contributionCount={projectContributionCounts[project.id] ?? 0}
						/>
					{/each}
				</div>
			{/if}
		</div>

		<div>
			<div class="mb-3 flex items-baseline justify-between">
				<h2 class="text-lg font-semibold tracking-tight">People</h2>
				<a href="/people" class="text-sm text-slate-500 hover:text-slate-700">View all →</a>
			</div>
			<div class="grid grid-cols-1 gap-4 lg:grid-cols-2">
				<ChartCard title="By status">
					<DonutChart data={personStatusData} />
				</ChartCard>
				<ChartCard title="By employment type">
					<BarChart data={employmentTypeData} />
				</ChartCard>
			</div>
		</div>

		<div>
			<div class="mb-3 flex items-baseline justify-between">
				<h2 class="text-lg font-semibold tracking-tight">Contributions</h2>
				<a href="/contributions" class="text-sm text-slate-500 hover:text-slate-700">View all →</a>
			</div>
			<div class="grid grid-cols-1 gap-4 lg:grid-cols-2">
				<ChartCard title="By type">
					<BarChart data={contributionTypeData} />
				</ChartCard>
				<ChartCard
					title="Last 8 weeks"
					subtitle={contributions.length >= 100
						? 'Based on the 100 most recently logged.'
						: undefined}
				>
					<TrendChart data={weeklyContributionData} />
				</ChartCard>
			</div>
		</div>

		<div>
			<div class="mb-3 flex items-baseline justify-between">
				<h2 class="text-lg font-semibold tracking-tight">Growth</h2>
				<a href="/goals" class="text-sm text-slate-500 hover:text-slate-700">View all →</a>
			</div>
			<div class="grid grid-cols-1 gap-4 lg:grid-cols-2">
				<ChartCard title="Average goal progress">
					<DonutChart
						data={goalProgressRingData}
						centerValue="{avgGoalProgress}%"
						centerLabel="avg. progress"
						showLegend={false}
					/>
				</ChartCard>
				<ChartCard title="By status">
					<StackedBar data={goalStatusData} />
				</ChartCard>
			</div>
		</div>

		<div>
			<div class="mb-3 flex items-baseline justify-between">
				<h2 class="text-lg font-semibold tracking-tight">Time Off</h2>
				<a href="/time-off" class="text-sm text-slate-500 hover:text-slate-700">View all →</a>
			</div>
			<div class="grid grid-cols-1 gap-4 lg:grid-cols-2">
				<ChartCard title="Who's off today" subtitle={offTodayList.length === 0 ? undefined : `${offTodayList.length} out today`}>
					{#if offTodayList.length === 0}
						<p class="text-sm text-slate-500">No one is off today.</p>
					{:else}
						<ul class="divide-y divide-slate-100">
							{#each offTodayList as leave (leave.id)}
								<li class="flex items-center justify-between gap-3 py-2 text-sm first:pt-0 last:pb-0">
									<div class="min-w-0">
										<p class="truncate font-medium text-slate-800">
											{leave.person ? `${leave.person.firstName} ${leave.person.lastName}` : 'Unknown person'}
										</p>
										<p class="text-xs text-slate-500">{leave.type.replace('_', ' ')}</p>
									</div>
									<span class="shrink-0 text-xs text-slate-500">through {leave.endDate}</span>
								</li>
							{/each}
						</ul>
					{/if}
				</ChartCard>
				<ChartCard title="By leave type">
					<BarChart data={leaveTypeData} />
				</ChartCard>
			</div>
		</div>
	{/if}
</section>
