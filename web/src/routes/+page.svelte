<script lang="ts">
	import { onMount } from 'svelte';
	import * as Card from '$lib/components/ui/card';
	import * as Empty from '$lib/components/ui/empty';
	import * as Alert from '$lib/components/ui/alert';
	import { Button } from '$lib/components/ui/button';
	import { Skeleton } from '$lib/components/ui/skeleton';
	import RefreshCwIcon from '@lucide/svelte/icons/refresh-cw';
	import CalendarClockIcon from '@lucide/svelte/icons/calendar-clock';
	import TriangleAlertIcon from '@lucide/svelte/icons/triangle-alert';
	import FolderIcon from '@lucide/svelte/icons/folder';
	import { toast } from 'svelte-sonner';
	import CreateJobDialog from '$lib/components/create-job-dialog.svelte';
	import JobTable from '$lib/components/job-table.svelte';
	import { cn } from '$lib/utils';
	import { API_URL, listJobs, stopJob, type Job } from '$lib/api';

	let jobs = $state<Job[]>([]);
	let loading = $state(true);
	let error = $state('');

	const NO_PROJECT = 'No project';
	// Named projects A→Z, jobs without a project last.
	const groups = $derived(
		Object.entries(Object.groupBy(jobs, (j) => j.project?.trim() || NO_PROJECT)).sort(
			([a], [b]) => +(a === NO_PROJECT) - +(b === NO_PROJECT) || a.localeCompare(b)
		) as [string, Job[]][]
	);
	const projects = $derived(groups.map(([p]) => p).filter((p) => p !== NO_PROJECT));

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
				{jobs.length} jobs · {projects.length} projects ·
				<code class="font-mono">{API_URL}</code>
			</p>
		</div>
		<div class="flex gap-2">
			<Button variant="outline" onclick={load} disabled={loading}>
				<RefreshCwIcon data-icon="inline-start" class={cn(loading && 'animate-spin')} />
				Refresh
			</Button>
			<CreateJobDialog {projects} oncreated={load} />
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

	{#if loading && !jobs.length}
		<Card.Root>
			<Card.Content class="flex flex-col gap-3">
				{#each [1, 2, 3] as i (i)}<Skeleton class="h-10 w-full" />{/each}
			</Card.Content>
		</Card.Root>
	{:else if !jobs.length}
		<Card.Root>
			<Card.Content>
				<Empty.Root>
					<Empty.Header>
						<Empty.Media variant="icon"><CalendarClockIcon /></Empty.Media>
						<Empty.Title>No jobs yet</Empty.Title>
						<Empty.Description>Create a job to call an endpoint on a schedule.</Empty.Description>
					</Empty.Header>
				</Empty.Root>
			</Card.Content>
		</Card.Root>
	{:else}
		{#each groups as [project, list] (project)}
			<Card.Root>
				<Card.Header>
					<Card.Title class="flex items-center gap-2">
						<FolderIcon class="text-muted-foreground size-4" />
						{project}
					</Card.Title>
					<Card.Description>{list.length} {list.length === 1 ? 'job' : 'jobs'}</Card.Description>
				</Card.Header>
				<Card.Content>
					<JobTable jobs={list} onstop={stop} />
				</Card.Content>
			</Card.Root>
		{/each}
	{/if}
</main>
