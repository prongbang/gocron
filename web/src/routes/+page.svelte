<script lang="ts">
	import { onMount } from 'svelte';
	import * as Card from '$lib/components/ui/card';
	import * as Table from '$lib/components/ui/table';
	import * as Empty from '$lib/components/ui/empty';
	import * as Alert from '$lib/components/ui/alert';
	import * as AlertDialog from '$lib/components/ui/alert-dialog';
	import { Button, buttonVariants } from '$lib/components/ui/button';
	import { Badge } from '$lib/components/ui/badge';
	import { Skeleton } from '$lib/components/ui/skeleton';
	import RefreshCwIcon from '@lucide/svelte/icons/refresh-cw';
	import CircleStopIcon from '@lucide/svelte/icons/circle-stop';
	import CalendarClockIcon from '@lucide/svelte/icons/calendar-clock';
	import TriangleAlertIcon from '@lucide/svelte/icons/triangle-alert';
	import { toast } from 'svelte-sonner';
	import CreateJobDialog from '$lib/components/create-job-dialog.svelte';
	import { cn } from '$lib/utils';
	import { API_URL, listJobs, stopJob, type Job } from '$lib/api';

	let jobs = $state<Job[]>([]);
	let loading = $state(true);
	let error = $state('');

	async function load() {
		loading = true;
		try {
			jobs = (await listJobs()).sort((a, b) => a.job.localeCompare(b.job));
			error = '';
		} catch (err) {
			error = (err as Error).message;
		} finally {
			loading = false;
		}
	}

	async function stop(job: string) {
		try {
			await stopJob(job);
			toast.success('Job stopped', { description: job });
			await load();
		} catch (err) {
			toast.error('Stop failed', { description: (err as Error).message });
		}
	}

	onMount(load);
</script>

<main class="mx-auto flex max-w-6xl flex-col gap-6 p-4 sm:p-8">
	<header class="flex flex-wrap items-center justify-between gap-4">
		<div class="flex flex-col gap-1">
			<h1 class="text-2xl font-semibold tracking-tight">gocron</h1>
			<p class="text-muted-foreground text-sm">
				Scheduler API <code class="font-mono">{API_URL}</code>
			</p>
		</div>
		<div class="flex gap-2">
			<Button variant="outline" onclick={load} disabled={loading}>
				<RefreshCwIcon data-icon="inline-start" class={cn(loading && 'animate-spin')} />
				Refresh
			</Button>
			<CreateJobDialog oncreated={load} />
		</div>
	</header>

	{#if error}
		<Alert.Root variant="destructive">
			<TriangleAlertIcon />
			<Alert.Title>Cannot reach the scheduler API</Alert.Title>
			<Alert.Description>
				{error} — is gocron running with <code>GOCRON_API=true</code>?
			</Alert.Description>
		</Alert.Root>
	{/if}

	<Card.Root>
		<Card.Header>
			<Card.Title>Jobs</Card.Title>
			<Card.Description>{jobs.length} scheduled</Card.Description>
		</Card.Header>
		<Card.Content>
			{#if loading && !jobs.length}
				<div class="flex flex-col gap-3">
					{#each [1, 2, 3] as i (i)}<Skeleton class="h-10 w-full" />{/each}
				</div>
			{:else if !jobs.length}
				<Empty.Root>
					<Empty.Header>
						<Empty.Media variant="icon"><CalendarClockIcon /></Empty.Media>
						<Empty.Title>No jobs yet</Empty.Title>
						<Empty.Description>Create a job to call an endpoint on a schedule.</Empty.Description>
					</Empty.Header>
				</Empty.Root>
			{:else}
				<Table.Root>
					<Table.Header>
						<Table.Row>
							<Table.Head>Cron</Table.Head>
							<Table.Head>Request</Table.Head>
							<Table.Head>Job</Table.Head>
							<Table.Head>Status</Table.Head>
							<Table.Head class="text-right">Action</Table.Head>
						</Table.Row>
					</Table.Header>
					<Table.Body>
						{#each jobs as j (j.job)}
							<Table.Row>
								<Table.Cell class="font-mono">{j.cron}</Table.Cell>
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
								<Table.Cell class="text-right">
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
												<AlertDialog.Action variant="destructive" onclick={() => stop(j.job)}>
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
			{/if}
		</Card.Content>
	</Card.Root>
</main>
