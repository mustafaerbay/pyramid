<script lang="ts">
	import { api } from '$lib/api/client';
	import { parseCsv } from '$lib/csv';
	import { readXlsx } from '$lib/xlsx';
	import { runLimited } from '$lib/bulk';
	import { EXPORT_HEADER, importWaves, rowsToTasks, type ImportRow, type Lookups } from '$lib/taskCsv';

	let { projectId, lookups, onDone, onClose }: { projectId: string; lookups: Lookups; onDone: (message: string) => void; onClose: () => void } = $props();

	let parsed = $state<{ ok: ImportRow[]; errors: { line: number; message: string }[] } | null>(null);
	let fileName = $state('');
	let busy = $state(false);
	let progress = $state(0);

	async function read(e: Event) {
		const file = (e.currentTarget as HTMLInputElement).files?.[0];
		if (!file) return;
		if (file.size > 2_000_000) {
			parsed = { ok: [], errors: [{ line: 0, message: 'The file is larger than 2 MB.' }] };
			return;
		}
		fileName = file.name;
		let rows: string[][];
		try {
			rows = /\.xlsx$/i.test(file.name) ? (await readXlsx(new Uint8Array(await file.arrayBuffer()))).filter((r) => r.some((c) => c.trim())) : parseCsv(await file.text());
		} catch (err) {
			parsed = { ok: [], errors: [{ line: 0, message: (err as Error).message || 'Could not read the file.' }] };
			return;
		}
		parsed = rowsToTasks(rows, projectId, lookups);
	}

	async function run() {
		if (!parsed) return;
		busy = true;
		progress = 0;
		// Parents first: each wave waits for the one before, so a row can be
		// put under the item its parent row just became.
		const createdId = new Map<number, string>();
		const results: { ok: boolean; error?: unknown }[] = [];
		for (const wave of importWaves(parsed.ok)) {
			const out = await runLimited(wave, 5, async (r) => {
				const body = { ...r.body };
				if (r.parentLine !== undefined) {
					const pid = createdId.get(r.parentLine);
					if (!pid) throw new Error(`Line ${r.line}: its parent row (line ${r.parentLine}) wasn't created`);
					body.parentId = pid;
				}
				// eslint-disable-next-line @typescript-eslint/no-explicit-any
				const { data, error } = await api.POST('/api/v1/tasks', { body: body as any });
				progress++;
				if (error || !data) throw new Error(`Line ${r.line}: ${error?.message ?? 'failed'}`);
				createdId.set(r.line, data.id);
			});
			results.push(...out);
		}
		const notReached = parsed.ok.length - results.length;
		for (let i = 0; i < notReached; i++) results.push({ ok: false, error: new Error('Parent row was never created') });
		busy = false;
		const failed = results.filter((x) => !x.ok);
		onDone(
			failed.length
				? `Imported ${results.length - failed.length}; ${failed.length} failed — ${(failed[0] as { error: Error }).error.message}`
				: `Imported ${results.length} item${results.length === 1 ? '' : 's'}.`
		);
	}
</script>

<div class="fixed inset-0 z-40 flex items-end justify-center bg-slate-900/40 sm:items-center sm:p-4" role="dialog" aria-modal="true" aria-label="Import CSV">
	<div class="max-h-[90vh] w-full overflow-y-auto rounded-t-xl bg-white p-5 shadow-xl sm:max-w-2xl sm:rounded-xl">
		<div class="mb-1 flex items-center justify-between">
			<h2 class="text-lg font-semibold">Import work items from CSV or Excel</h2>
			<button type="button" class="btn-ghost px-2" aria-label="Close" onclick={onClose}>✕</button>
		</div>
		<p class="mb-3 text-sm text-slate-500">
			Needs a <b>Title</b> column. Also read: {EXPORT_HEADER.slice(3).join(', ')}. People by email or name, several split with
			";". Parent as #number, the title of an existing item, or the title of another row in the file. A file exported from Pyramid imports as is.
		</p>
		<input type="file" accept=".csv,text/csv,.xlsx,application/vnd.openxmlformats-officedocument.spreadsheetml.sheet" class="block w-full text-sm" onchange={read} disabled={busy} />

		{#if parsed}
			<div class="mt-4 space-y-2 text-sm">
				<p>
					<b>{fileName}</b>: {parsed.ok.length} ready{parsed.errors.length ? `, ${parsed.errors.length} with problems (left out)` : ''}.
				</p>
				{#if parsed.errors.length}
					<ul class="max-h-40 overflow-y-auto rounded-md border border-red-200 bg-red-50 p-2 text-xs text-red-800">
						{#each parsed.errors as e (e.line)}<li>Line {e.line}: {e.message}</li>{/each}
					</ul>
				{/if}
				{#if parsed.ok.length}
					<ul class="max-h-48 divide-y divide-slate-100 overflow-y-auto rounded-md border border-slate-200 text-xs">
						{#each parsed.ok.slice(0, 50) as r (r.line)}
							<li class="flex gap-2 px-2 py-1"><span class="text-slate-400">{r.line}</span><span class="truncate">{r.body.title}</span><span class="ml-auto shrink-0 text-slate-500">{r.body.type ?? 'task'} · {r.body.status ?? 'backlog'}</span></li>
						{/each}
						{#if parsed.ok.length > 50}<li class="px-2 py-1 text-slate-500">…and {parsed.ok.length - 50} more</li>{/if}
					</ul>
				{/if}
				<div class="flex items-center gap-2 pt-2">
					<button type="button" class="btn-primary" disabled={busy || parsed.ok.length === 0} onclick={run}>
						{busy ? `Importing ${progress}/${parsed.ok.length}…` : `Import ${parsed.ok.length}`}
					</button>
					<button type="button" class="btn-ghost" onclick={onClose} disabled={busy}>Cancel</button>
				</div>
			</div>
		{/if}
	</div>
</div>
