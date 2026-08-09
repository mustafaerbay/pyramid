<script lang="ts">
	import { onMount } from 'svelte';
	import { api } from '$lib/api/client';
	import type { components } from '$lib/api/schema';

	type Person = components['schemas']['Person'];

	let {
		value = $bindable(''),
		label = 'Person',
		required = false,
		emptyLabel = '—',
		excludeIds = [],
		onchange
	}: {
		value?: string;
		label?: string;
		required?: boolean;
		emptyLabel?: string;
		// People to hide from the list — e.g. those already on a project, to
		// avoid duplicate-assignment errors before they happen.
		excludeIds?: string[];
		onchange?: (personId: string) => void;
	} = $props();

	let people = $state<Person[]>([]);
	let loading = $state(true);

	onMount(async () => {
		// pageSize 100 is the API max; fine for the department-scale data this
		// app is built for. A typeahead search can replace this once the
		// people list grows past a single page.
		const { data } = await api.GET('/api/v1/people', {
			params: { query: { page: 1, pageSize: 100 } }
		});
		if (data) people = data.data;
		loading = false;
	});

	const selectable = $derived(people.filter((p) => !excludeIds.includes(p.id)));

	function fullName(p: Person): string {
		return `${p.firstName} ${p.lastName}`;
	}
</script>

<label class="block text-sm">
	<span class="mb-1 block font-medium text-slate-700">{label}{required ? ' *' : ''}</span>
	<select
		class="input"
		bind:value
		{required}
		disabled={loading}
		onchange={(e) => onchange?.(e.currentTarget.value)}
	>
		<option value="">{loading ? 'Loading…' : emptyLabel}</option>
		{#each selectable as person (person.id)}
			<option value={person.id}>{fullName(person)}</option>
		{/each}
	</select>
</label>
