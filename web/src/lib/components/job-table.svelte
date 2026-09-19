<script lang="ts">
	import * as Table from '$lib/components/ui/table';
	import * as AlertDialog from '$lib/components/ui/alert-dialog';
	import Pager from '$lib/components/pager.svelte';
	import { buttonVariants } from '$lib/components/ui/button';
	import { Badge } from '$lib/components/ui/badge';
	import CircleStopIcon from '@lucide/svelte/icons/circle-stop';
	import HistoryIcon from '@lucide/svelte/icons/history';
	import type { Job } from '$lib/api';
	import { describeCron, fromNow, serverZone } from '$lib/cron';

	let { jobs, onstop }: { jobs: Job[]; onstop: (job: string) => void } = $props();

	const PER_PAGE = 10;
	let page = $state(1);
	const rows = $derived(jobs.slice((page - 1) * PER_PAGE, page * PER_PAGE));
</script>

<div class="flex flex-col gap-4">
	<Table.Root>
		<Table.Header>
			<Table.Row>
				<Table.Head>Schedule</Table.Head>
				<Table.Head>Next run</Table.Head>
				<Table.Head>Request</Table.Head>
				<Table.Head>Job</Table.Head>
				<Table.Head>Status</Table.Head>
				<Table.Head class="text-right">Action</Table.Head>
			</Table.Row>
		</Table.Header>
		<Table.Body>
			{#each rows as j (j.job)}
				<Table.Row>
					<Table.Cell>
						<div class="flex flex-col">
							<code class="font-mono">{j.cron}</code>
							<span class="text-muted-foreground text-xs">
								{describeCron(j.cron).text}
								{#if j.next_run}· server {serverZone(j.next_run)}{/if}
							</span>
						</div>
					</Table.Cell>
					<Table.Cell>
						{#if j.next_run}
							{@const next = new Date(j.next_run)}
							<div class="flex flex-col">
								<span>{fromNow(next)}</span>
								<span class="text-muted-foreground text-xs">
									{next.toLocaleString(undefined, { dateStyle: 'medium', timeStyle: 'short' })}
								</span>
							</div>
						{:else}
							<span class="text-muted-foreground">—</span>
						{/if}
					</Table.Cell>
					<Table.Cell class="max-w-sm">
						<div class="flex items-center gap-2">
							<Badge variant="outline">{j.task.config.method}</Badge>
							<span class="truncate" title={j.task.config.url}>{j.task.config.url}</span>
						</div>
					</Table.Cell>
					<Table.Cell class="text-muted-foreground font-mono text-xs" title={j.job}>
						{j.job.slice(0, 8)}
					</Table.Cell>
					<Table.Cell>
						<Badge variant={j.running ? 'default' : 'secondary'}>
							{j.running ? 'Running' : 'Stopped'}
						</Badge>
					</Table.Cell>
					<Table.Cell class="text-right whitespace-nowrap">
						<a href="/history?job={j.job}" class={buttonVariants({ variant: 'ghost', size: 'sm' })}>
							<HistoryIcon data-icon="inline-start" />
							History
						</a>
						<AlertDialog.Root>
							<AlertDialog.Trigger class={buttonVariants({ variant: 'ghost', size: 'sm' })}>
								<CircleStopIcon data-icon="inline-start" />
								Stop
							</AlertDialog.Trigger>
							<AlertDialog.Content>
								<AlertDialog.Header>
									<AlertDialog.Title>Stop this job?</AlertDialog.Title>
									<AlertDialog.Description>
										<code class="font-mono">{j.cron}</code>
										{j.task.config.method}
										{j.task.config.url} will be stopped and removed. This cannot be undone.
									</AlertDialog.Description>
								</AlertDialog.Header>
								<AlertDialog.Footer>
									<AlertDialog.Cancel>Cancel</AlertDialog.Cancel>
									<AlertDialog.Action variant="destructive" onclick={() => onstop(j.job)}>
										Stop job
									</AlertDialog.Action>
								</AlertDialog.Footer>
							</AlertDialog.Content>
						</AlertDialog.Root>
					</Table.Cell>
				</Table.Row>
			{/each}
		</Table.Body>
	</Table.Root>

	<Pager count={jobs.length} perPage={PER_PAGE} bind:page />
</div>
