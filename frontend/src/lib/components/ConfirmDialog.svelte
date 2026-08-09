<script lang="ts">
	let {
		open,
		title,
		message,
		confirmLabel = 'Confirm',
		danger = false,
		error = null,
		onConfirm,
		onCancel
	}: {
		open: boolean;
		title: string;
		message: string;
		confirmLabel?: string;
		danger?: boolean;
		error?: string | null;
		onConfirm: () => void;
		onCancel: () => void;
	} = $props();
</script>

<svelte:window onkeydown={(e) => open && e.key === 'Escape' && onCancel()} />

{#if open}
	<!-- svelte-ignore a11y_click_events_have_key_events -->
	<!-- svelte-ignore a11y_no_static_element_interactions -->
	<div
		class="fixed inset-0 z-50 flex items-center justify-center bg-slate-900/40 p-4"
		role="presentation"
		onclick={onCancel}
	>
		<div
			class="w-full max-w-sm rounded-lg bg-white p-6 shadow-xl"
			role="dialog"
			tabindex="-1"
			aria-modal="true"
			aria-labelledby="confirm-dialog-title"
			onclick={(e) => e.stopPropagation()}
		>
			<h3 id="confirm-dialog-title" class="text-base font-semibold text-slate-900">{title}</h3>
			<p class="mt-2 text-sm text-slate-600">{message}</p>

			{#if error}
				<p class="mt-3 rounded-md border border-red-200 bg-red-50 px-3 py-2 text-sm text-red-700">
					{error}
				</p>
			{/if}

			<div class="mt-6 flex justify-end gap-2">
				<button type="button" class="btn-secondary" onclick={onCancel}>Cancel</button>
				<button
					type="button"
					class={danger ? 'btn-danger' : 'btn-primary'}
					onclick={onConfirm}
				>
					{confirmLabel}
				</button>
			</div>
		</div>
	</div>
{/if}
