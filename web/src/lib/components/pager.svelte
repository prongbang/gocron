<script lang="ts">
	import * as Pagination from '$lib/components/ui/pagination';

	let { count, perPage, page = $bindable(1) }: { count: number; perPage: number; page?: number } =
		$props();

	const pageCount = $derived(Math.max(1, Math.ceil(count / perPage)));
	// Shrinking data (a job stopped, a filter applied) must not leave us on an empty page.
	$effect(() => {
		if (page > pageCount) page = pageCount;
	});
</script>

{#if count > perPage}
	<Pagination.Root {count} {perPage} bind:page>
		{#snippet children({ pages, currentPage })}
			<Pagination.Content>
				<Pagination.Item><Pagination.Previous /></Pagination.Item>
				{#each pages as p (p.key)}
					<Pagination.Item>
						{#if p.type === 'ellipsis'}
							<Pagination.Ellipsis />
						{:else}
							<Pagination.Link page={p} isActive={currentPage === p.value} />
						{/if}
					</Pagination.Item>
				{/each}
				<Pagination.Item><Pagination.Next /></Pagination.Item>
			</Pagination.Content>
		{/snippet}
	</Pagination.Root>
{/if}
