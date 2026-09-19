<script lang="ts">
	import { page as appPage } from '$app/state';
	import * as Card from '$lib/components/ui/card';
	import * as Table from '$lib/components/ui/table';
	import * as Empty from '$lib/components/ui/empty';
	import * as Alert from '$lib/components/ui/alert';
	import * as ToggleGroup from '$lib/components/ui/toggle-group';
	import { Button, buttonVariants } from '$lib/components/ui/button';
	import { Badge } from '$lib/components/ui/badge';
	import { Skeleton } from '$lib/components/ui/skeleton';
	import RefreshCwIcon from '@lucide/svelte/icons/refresh-cw';
	import HistoryIcon from '@lucide/svelte/icons/history';
	import TriangleAlertIcon from '@lucide/svelte/icons/triangle-alert';
	import XIcon from '@lucide/svelte/icons/x';
	import Pager from '$lib/components/pager.svelte';
	import { cn } from '$lib/utils';
	import { fromNow } from '$lib/cron';
	import { listHistory, type History } from '$lib/api';

	const PER_PAGE = 20;
	const ok = (h: History) => h.status >= 200 && h.status < 300;

	const job = $derived(appPage.url.searchParams.get('job') ?? '');
	let runs = $state<History[]>([]);
	let loading = $state(true);
	let error = $state('');
	let status = $state<'all' | 'ok' | 'failed'>('all');
	let page = $state(1);

	const filtered = $derived(
		status === 'all' ? runs : runs.filter((h) => (status === 'ok') === ok(h))
	);
	const rows = $derived(filtered.slice((page - 1) * PER_PAGE, page * PER_PAGE));
	const failed = $derived(runs.filter((h) => !ok(h)).length);

	async function load(silent = false) {
		if (!silent) loading = true;
		try {
			runs = await listHistory(job);
			error = '';
		} catch (err) {
			error = (err as Error).message;
		} finally {
			loading = false;
		}
	}

	// Reload whenever the ?job= filter changes, and every 30s in the background.
	$effect(() => {
		job;
		page = 1;
		load();
		const id = setInterval(() => load(true), 30_000);
		return () => clearInterval(id);
	});
</script>

<main class="mx-auto flex max-w-6xl flex-col gap-6 p-4 sm:p-8">
	<header class="flex flex-wrap items-center justify-between gap-4">
		<div class="flex flex-col gap-1">
			<h1 class="text-2xl font-semibold tracking-tight">History</h1>
			<p class="text-muted-foreground text-sm">
				{runs.length} runs · {failed} failed · kept for 7 days
			</p>
		</div>
		<Button variant="outline" onclick={() => load()} disabled={loading}>
			<RefreshCwIcon data-icon="inline-start" class={cn(loading && 'animate-spin')} />
			Refresh
		</Button>
	</header>

	{#if error}
		<Alert.Root variant="destructive">
			<TriangleAlertIcon />
			<Alert.Title>Cannot load history</Alert.Title>
			<Alert.Description>{error}</Alert.Description>
		</Alert.Root>
	{/if}

	<Card.Root>
		<Card.Header>
			<Card.Title>Runs</Card.Title>
			<Card.Description>
				{#if job}
					Job <code class="font-mono">{job.slice(0, 8)}</code>
				{:else}
					All jobs, newest first
				{/if}
			</Card.Description>
			<Card.Action class="flex items-center gap-2">
				{#if job}
					<a href="/history" class={buttonVariants({ variant: 'ghost', size: 'sm' })}>
						<XIcon data-icon="inline-start" />
						All jobs
					</a>
				{/if}
				<ToggleGroup.Root
					type="single"
					variant="outline"
					size="sm"
					bind:value={
						() => status,
						(v) => {
							if (v) status = v as typeof status;
							page = 1;
						}
					}
				>
					<ToggleGroup.Item value="all">All</ToggleGroup.Item>
					<ToggleGroup.Item value="ok">Success</ToggleGroup.Item>
					<ToggleGroup.Item value="failed">Failed</ToggleGroup.Item>
				</ToggleGroup.Root>
			</Card.Action>
		</Card.Header>
		<Card.Content class="flex flex-col gap-4">
			{#if loading && !runs.length}
				{#each [1, 2, 3] as i (i)}<Skeleton class="h-10 w-full" />{/each}
			{:else if !filtered.length}
				<Empty.Root>
					<Empty.Header>
						<Empty.Media variant="icon"><HistoryIcon /></Empty.Media>
						<Empty.Title>{runs.length ? 'No matching runs' : 'No runs yet'}</Empty.Title>
						<Empty.Description>Runs appear here each time a job fires.</Empty.Description>
					</Empty.Header>
				</Empty.Root>
			{:else}
				<Table.Root>
					<Table.Header>
						<Table.Row>
							<Table.Head>Time</Table.Head>
							<Table.Head>Status</Table.Head>
							<Table.Head>Request</Table.Head>
							<Table.Head>Project</Table.Head>
							<Table.Head>Job</Table.Head>
							<Table.Head class="text-right">Duration</Table.Head>
						</Table.Row>
					</Table.Header>
					<Table.Body>
						{#each rows as h (h.started_at + h.job)}
							{@const at = new Date(h.started_at)}
							<Table.Row>
								<Table.Cell>
									<div class="flex flex-col">
										<span>{fromNow(at)}</span>
										<span class="text-muted-foreground text-xs">
											{at.toLocaleString(undefined, { dateStyle: 'medium', timeStyle: 'medium' })}
										</span>
									</div>
								</Table.Cell>
								<Table.Cell class="max-w-56">
									<div class="flex flex-col items-start gap-1">
										<Badge variant={ok(h) ? 'secondary' : 'destructive'}>{h.status}</Badge>
										{#if !ok(h) && h.response}
											<span class="text-muted-foreground truncate text-xs" title={h.response}>
												{h.response}
											</span>
										{/if}
									</div>
								</Table.Cell>
								<Table.Cell class="max-w-sm">
									<div class="flex items-center gap-2">
										<Badge variant="outline">{h.method}</Badge>
										<span class="truncate" title={h.url}>{h.url}</span>
									</div>
								</Table.Cell>
								<Table.Cell>{h.project || '—'}</Table.Cell>
								<Table.Cell class="font-mono text-xs">
									<a href="/history?job={h.job}" class="text-muted-foreground hover:underline" title={h.job}>
										{h.job.slice(0, 8)}
									</a>
								</Table.Cell>
								<Table.Cell class="text-right tabular-nums">{h.duration_ms} ms</Table.Cell>
							</Table.Row>
						{/each}
					</Table.Body>
				</Table.Root>
				<Pager count={filtered.length} perPage={PER_PAGE} bind:page />
			{/if}
		</Card.Content>
	</Card.Root>
</main>
