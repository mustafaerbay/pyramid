<script lang="ts">
	import { goto } from '$app/navigation';
	import { page } from '$app/state';
	import { api } from '$lib/api/client';
	import type { components } from '$lib/api/schema';
	import { can } from '$lib/auth.svelte';
	import TestCaseEditor from '$lib/components/TestCaseEditor.svelte';

	type Plan = components['schemas']['TestPlan'];
	type Detail = components['schemas']['TestPlanDetail'];
	type Run = components['schemas']['TestRunDetail'];
	type Result = components['schemas']['TestResult'];
	type Outcome = components['schemas']['TestOutcome'];
	type Case = components['schemas']['TestCase'];

	const projectId = $derived(page.params.id ?? '');
	const planId = $derived(page.url.searchParams.get('plan') ?? '');
	const runId = $derived(page.url.searchParams.get('run') ?? '');
	let project = $state<components['schemas']['Project'] | null>(null);
	let plans = $state<Plan[]>([]);
	let detail = $state<Detail | null>(null);
	let run = $state<Run | null>(null);
	let items = $state<{ id: string; label: string }[]>([]);
	let error = $state<string | null>(null);
	let newPlan = $state('');
	// Case editor
	let editingCase = $state<{ suiteId: string; id: string | null } | null>(null);
	let cTitle = $state('');
	let cSteps = $state<components['schemas']['TestStep'][]>([]);
	let cTask = $state('');

	const writable = $derived(can('member'));
	const loadPlans = async () => (plans = (await api.GET('/api/v1/test-plans', { params: { query: { projectId } } })).data?.data ?? []);
	const loadDetail = async (id: string) => (detail = (await api.GET('/api/v1/test-plans/{id}', { params: { path: { id } } })).data ?? null);
	$effect(() => {
		if (!projectId) return;
		api.GET('/api/v1/projects/{id}', { params: { path: { id: projectId } } }).then((r) => (project = r.data ?? null));
		api.GET('/api/v1/tasks', { params: { query: { projectId, pageSize: 100 } } }).then(
			(r) => (items = (r.data?.data ?? []).map((t) => ({ id: t.id, label: `#${t.number} ${t.title}` })))
		);
		loadPlans();
	});
	$effect(() => {
		if (planId) loadDetail(planId);
		else detail = null;
	});
	$effect(() => {
		if (runId) api.GET('/api/v1/test-runs/{id}', { params: { path: { id: runId } } }).then((r) => (run = r.data ?? null));
		else run = null;
	});
	const fail = (m?: string) => (error = m ?? 'Something went wrong.');

	async function createPlan(e: SubmitEvent) {
		e.preventDefault();
		const { data, error: err } = await api.POST('/api/v1/test-plans', { body: { projectId, name: newPlan } });
		if (err || !data) return fail(err?.message);
		newPlan = '';
		await loadPlans();
		goto(`?plan=${data.id}`);
	}
	async function addSuite() {
		const name = prompt('Suite name, e.g. Sign-in');
		if (!name?.trim() || !detail) return;
		const { error: err } = await api.POST('/api/v1/test-suites', { body: { planId: detail.id, name: name.trim() } });
		if (err) return fail(err.message);
		loadDetail(detail.id);
	}
	async function renameSuite(s: components['schemas']['TestSuite']) {
		const name = prompt('Suite name', s.name);
		if (!name?.trim() || !detail) return;
		await api.PATCH('/api/v1/test-suites/{id}', { params: { path: { id: s.id } }, body: { planId: detail.id, name: name.trim() } });
		loadDetail(detail.id);
	}
	async function deleteSuite(s: components['schemas']['TestSuite']) {
		if (!detail || !confirm(`Delete the suite "${s.name}" and its ${s.cases.length} cases?`)) return;
		await api.DELETE('/api/v1/test-suites/{id}', { params: { path: { id: s.id } } });
		loadDetail(detail.id);
	}
	function editCase(suiteId: string, c: Case | null) {
		editingCase = { suiteId, id: c?.id ?? null };
		cTitle = c?.title ?? '';
		cSteps = c ? c.steps.map((s) => ({ ...s })) : [{ action: '', expected: '' }];
		cTask = c?.taskId ?? '';
	}
	async function saveCase() {
		if (!editingCase || !detail) return;
		const body = { suiteId: editingCase.suiteId, title: cTitle, steps: cSteps.filter((s) => s.action.trim() || s.expected.trim()), taskId: cTask || null };
		const res = editingCase.id
			? await api.PATCH('/api/v1/test-cases/{id}', { params: { path: { id: editingCase.id } }, body })
			: await api.POST('/api/v1/test-cases', { body });
		if (res.error) return fail(res.error.message);
		editingCase = null;
		loadDetail(detail.id);
	}
	async function deleteCase(c: Case) {
		if (!detail || !confirm(`Delete the test case "${c.title}"?`)) return;
		await api.DELETE('/api/v1/test-cases/{id}', { params: { path: { id: c.id } } });
		loadDetail(detail.id);
	}
	async function startRun(suiteId?: string) {
		if (!detail) return;
		const { data, error: err } = await api.POST('/api/v1/test-runs', { body: { planId: detail.id, ...(suiteId ? { suiteId } : {}) } });
		if (err || !data) return fail(err?.message);
		goto(`?plan=${detail.id}&run=${data.id}`);
	}
	async function deletePlan() {
		if (!detail || !confirm(`Delete the plan "${detail.name}" with all its suites, cases and runs?`)) return;
		await api.DELETE('/api/v1/test-plans/{id}', { params: { path: { id: detail.id } } });
		await loadPlans();
		goto('?');
	}
	async function setResult(r: Result, body: components['schemas']['TestResultUpdate']) {
		const { data, error: err } = await api.PATCH('/api/v1/test-results/{id}', { params: { path: { id: r.id } }, body });
		if (err || !data) return fail(err?.message);
		if (run) {
			const fresh = await api.GET('/api/v1/test-runs/{id}', { params: { path: { id: run.id } } });
			run = fresh.data ?? run;
		}
	}
	function setStep(r: Result, i: number, o: Outcome) {
		const so = [...r.stepOutcomes];
		so[i] = so[i] === o ? 'not_run' : o;
		setResult(r, { stepOutcomes: so });
	}
	async function fileBug(r: Result) {
		const { data, error: err } = await api.POST('/api/v1/test-results/{id}/bug', { params: { path: { id: r.id } } });
		if (err || !data) return fail(err?.message);
		if (run) run = (await api.GET('/api/v1/test-runs/{id}', { params: { path: { id: run.id } } })).data ?? run;
	}
	const TONE: Record<Outcome, string> = {
		passed: 'bg-emerald-600 text-white',
		failed: 'bg-red-600 text-white',
		blocked: 'bg-amber-500 text-white',
		not_run: 'bg-slate-100 text-slate-600'
	};
	const LABEL: Record<Outcome, string> = { passed: 'Passed', failed: 'Failed', blocked: 'Blocked', not_run: 'Not run' };
	const bar = (x: { passed: number; failed: number; blocked: number; notRun: number }) => {
		const total = x.passed + x.failed + x.blocked + x.notRun || 1;
		return [
			{ k: 'passed', n: x.passed, c: '#1baf7a' },
			{ k: 'failed', n: x.failed, c: '#e34948' },
			{ k: 'blocked', n: x.blocked, c: '#eda100' },
			{ k: 'not run', n: x.notRun, c: '#cbd5e1' }
		].map((s) => ({ ...s, w: (s.n / total) * 100 }));
	};
</script>

<svelte:head><title>Test plans · {project?.name ?? 'Project'} · Pyramid</title></svelte:head>

{#snippet progress(x: { passed: number; failed: number; blocked: number; notRun: number })}
	<span class="flex h-2 w-32 overflow-hidden rounded-full bg-slate-100" title="{x.passed} passed · {x.failed} failed · {x.blocked} blocked · {x.notRun} not run">
		{#each bar(x) as s (s.k)}<span style:width="{s.w}%" style:background={s.c}></span>{/each}
	</span>
{/snippet}

<section class="space-y-4">
	<div>
		<a href={run ? `?plan=${planId}` : planId ? '?' : `/projects/${projectId}`} class="text-sm text-slate-500 hover:text-slate-700"
			>← {run ? detail?.name : planId ? 'All test plans' : (project?.name ?? 'Project')}</a
		>
		<h1 class="mt-1 text-2xl font-semibold tracking-tight">{run ? run.name : detail ? detail.name : 'Test plans'}</h1>
	</div>
	{#if error}<div class="rounded-md border border-red-200 bg-red-50 p-3 text-sm text-red-700">{error}</div>{/if}

	{#if run}
		<div class="flex flex-wrap items-center gap-3 text-sm">
			{@render progress(run)}
			<span class="text-slate-600">{run.passed} passed · {run.failed} failed · {run.blocked} blocked · {run.notRun} not run</span>
			{#if run.completedAt}<span class="text-emerald-700">✓ complete</span>{/if}
		</div>
		<ul class="space-y-3">
			{#each run.results as r (r.id)}
				<li class="rounded-lg border border-slate-200 bg-white p-4 text-sm">
					<div class="flex flex-wrap items-center gap-2">
						<span class="font-medium text-slate-900">{r.title}</span>
						<span class="ml-auto flex gap-1">
							{#each ['passed', 'failed', 'blocked'] as const as o (o)}
								<button
									type="button"
									disabled={!writable}
									class="rounded px-2 py-0.5 text-xs font-medium {r.outcome === o ? TONE[o] : 'bg-white text-slate-600 ring-1 ring-inset ring-slate-200 hover:bg-slate-50'}"
									onclick={() => setResult(r, { outcome: r.outcome === o ? 'not_run' : o })}>{LABEL[o]}</button
								>
							{/each}
						</span>
					</div>
					{#if r.steps.length}
						<ol class="mt-2 space-y-1">
							{#each r.steps as st, i (i)}
								<li class="grid grid-cols-[1.5rem_minmax(0,1fr)_minmax(0,1fr)_auto] items-start gap-2">
									<span class="pt-0.5 text-right text-xs text-slate-500">{i + 1}.</span>
									<span>{st.action}</span>
									<span class="text-slate-500">→ {st.expected}</span>
									<span class="flex gap-0.5">
										{#each ['passed', 'failed', 'blocked'] as const as o (o)}
											<button
												type="button"
												disabled={!writable}
												aria-label="Step {i + 1} {LABEL[o]}"
												title={LABEL[o]}
												class="h-6 w-6 rounded text-xs {r.stepOutcomes[i] === o ? TONE[o] : 'bg-white text-slate-500 ring-1 ring-inset ring-slate-200 hover:bg-slate-50'}"
												onclick={() => setStep(r, i, o)}>{o === 'passed' ? '✓' : o === 'failed' ? '✕' : '!'}</button
											>
										{/each}
									</span>
								</li>
							{/each}
						</ol>
					{/if}
					<div class="mt-2 flex flex-wrap items-center gap-2">
						<input
							class="input min-w-0 flex-1 py-1 text-xs"
							placeholder="Comment (what you saw)"
							aria-label="Comment on {r.title}"
							value={r.comment ?? ''}
							disabled={!writable}
							onchange={(e) => setResult(r, { comment: e.currentTarget.value })}
						/>
						{#if r.bugTaskId}
							<a class="text-xs text-red-700 hover:underline" href="/projects/{projectId}?task={r.bugTaskId}">🐞 Bug filed</a>
						{:else if r.outcome === 'failed' && writable}
							<button type="button" class="btn-secondary px-2 py-1 text-xs" onclick={() => fileBug(r)}>🐞 File bug</button>
						{/if}
					</div>
				</li>
			{/each}
		</ul>
	{:else if detail}
		<div class="flex flex-wrap items-center gap-2 text-sm">
			{@render progress(detail)}
			<span class="text-slate-600">{detail.caseCount} cases · latest run: {detail.passed} passed, {detail.failed} failed</span>
			{#if writable}
				<span class="ml-auto flex gap-2">
					<button type="button" class="btn-primary" disabled={!detail.caseCount} onclick={() => startRun()}>▶ Run all</button>
					<button type="button" class="btn-secondary" onclick={addSuite}>+ Suite</button>
					<button type="button" class="btn-ghost text-red-600" onclick={deletePlan}>Delete plan</button>
				</span>
			{/if}
		</div>
		{#each detail.suites as s (s.id)}
			<div class="rounded-lg border border-slate-200 bg-white p-4 text-sm">
				<div class="mb-2 flex flex-wrap items-center gap-2">
					<h2 class="font-semibold">{s.name}</h2>
					<span class="text-xs text-slate-500">{s.cases.length} cases</span>
					{#if writable}
						<span class="ml-auto flex gap-2 text-xs">
							<button type="button" class="hover:underline" disabled={!s.cases.length} onclick={() => startRun(s.id)}>▶ Run suite</button>
							<button type="button" class="hover:underline" onclick={() => editCase(s.id, null)}>+ Case</button>
							<button type="button" class="hover:underline" onclick={() => renameSuite(s)}>Rename</button>
							<button type="button" class="text-red-600 hover:underline" onclick={() => deleteSuite(s)}>Delete</button>
						</span>
					{/if}
				</div>
				<ul class="divide-y divide-slate-100">
					{#each s.cases as c (c.id)}
						<li class="py-2">
							{#if editingCase?.id === c.id}
								<TestCaseEditor bind:title={cTitle} bind:steps={cSteps} bind:taskId={cTask} {items} onSave={saveCase} onCancel={() => (editingCase = null)} />
							{:else}
								<div class="flex flex-wrap items-center gap-2">
									<span class="font-medium">{c.title}</span>
									<span class="text-xs text-slate-500">{c.steps.length} steps</span>
									{#if c.taskId}<a class="text-xs text-indigo-700 hover:underline" href="/projects/{projectId}?task={c.taskId}">{items.find((i) => i.id === c.taskId)?.label ?? 'item'}</a>{/if}
									{#if writable}
										<span class="ml-auto flex gap-2 text-xs">
											<button type="button" class="hover:underline" onclick={() => editCase(s.id, c)}>Edit</button>
											<button type="button" class="text-red-600 hover:underline" onclick={() => deleteCase(c)}>Delete</button>
										</span>
									{/if}
								</div>
							{/if}
						</li>
					{/each}
					{#if editingCase && editingCase.suiteId === s.id && !editingCase.id}
						<li class="py-2"><TestCaseEditor bind:title={cTitle} bind:steps={cSteps} bind:taskId={cTask} {items} onSave={saveCase} onCancel={() => (editingCase = null)} /></li>
					{/if}
				</ul>
			</div>
		{:else}
			<p class="text-sm text-slate-500">Add a suite (a group of test cases, e.g. "Sign-in") to start.</p>
		{/each}
		{#if detail.runs.length}
			<div class="rounded-lg border border-slate-200 bg-white p-4 text-sm">
				<h2 class="mb-2 font-semibold">Runs</h2>
				<ul class="divide-y divide-slate-100">
					{#each detail.runs as r (r.id)}
						<li class="flex flex-wrap items-center gap-3 py-2">
							<a class="font-medium hover:underline" href="?plan={detail.id}&run={r.id}">{r.name}</a>
							{@render progress(r)}
							<span class="text-xs text-slate-500">{r.passed}/{r.passed + r.failed + r.blocked + r.notRun} passed{r.completedAt ? ' · complete' : ''}</span>
						</li>
					{/each}
				</ul>
			</div>
		{/if}
	{:else}
		{#if writable}
			<form class="flex flex-wrap gap-2" onsubmit={createPlan}>
				<input class="input w-full sm:w-72" placeholder="New test plan, e.g. Release 1.4" aria-label="Test plan name" required maxlength="200" bind:value={newPlan} />
				<button type="submit" class="btn-primary">Create plan</button>
			</form>
		{/if}
		<ul class="divide-y divide-slate-100 rounded-lg border border-slate-200 bg-white text-sm">
			{#each plans as p (p.id)}
				<li class="flex flex-wrap items-center gap-3 px-4 py-3">
					<a class="font-medium text-slate-900 hover:underline" href="?plan={p.id}">{p.name}</a>
					<span class="text-xs text-slate-500">{p.caseCount} cases</span>
					<span class="ml-auto flex items-center gap-2">{@render progress(p)}<span class="text-xs text-slate-500">{p.failed ? `${p.failed} failing` : ''}</span></span>
				</li>
			{:else}
				<li class="px-4 py-6 text-center text-slate-500">No test plans yet. A plan groups test cases (with steps) that you run before a release.</li>
			{/each}
		</ul>
	{/if}
</section>
