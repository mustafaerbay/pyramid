<script lang="ts">
	let { title, description }: { title: string; description: string } = $props();
</script>

<section class="space-y-4">
	<div>
		<h1 class="text-2xl font-semibold tracking-tight">{title}</h1>
		<p class="mt-1 text-slate-500">{description}</p>
	</div>

	<div class="rounded-lg border border-dashed border-slate-300 bg-white p-10 text-center">
		<p class="text-slate-500">This module isn't wired up yet.</p>
		<p class="mt-1 text-sm text-slate-400">
			The database schema already covers it — ask to have it built next, following the same
			pattern as People.
		</p>
	</div>
</section>
