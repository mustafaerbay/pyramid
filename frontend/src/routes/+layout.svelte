<script lang="ts">
	import '../app.css';

	let { children } = $props();
</script>

<div class="min-h-screen bg-slate-50 text-slate-900">
	<header class="border-b border-slate-200 bg-white">
		<div class="mx-auto flex max-w-5xl items-center gap-2 px-6 py-4">
			<span class="text-lg font-semibold tracking-tight">▲ Pyramid</span>
			<span class="text-sm text-slate-500">Software Department Management</span>
		</div>
	</header>
	<main class="mx-auto max-w-5xl px-6 py-8">
		{@render children()}
	</main>
</div>
