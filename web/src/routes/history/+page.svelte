<script lang="ts">
	import { goto } from '$app/navigation';
	import { page as appPage } from '$app/state';
	import * as Card from '$lib/components/ui/card';
	import * as Table from '$lib/components/ui/table';
	import * as Empty from '$lib/components/ui/empty';
	import * as Alert from '$lib/components/ui/alert';
	import * as Select from '$lib/components/ui/select';
	import * as ToggleGroup from '$lib/components/ui/toggle-group';
	import { Button } from '$lib/components/ui/button';
	import { Input } from '$lib/components/ui/input';
	import { Badge } from '$lib/components/ui/badge';
	import { Skeleton } from '$lib/components/ui/skeleton';
	import RefreshCwIcon from '@lucide/svelte/icons/refresh-cw';
	import HistoryIcon from '@lucide/svelte/icons/history';
	import TriangleAlertIcon from '@lucide/svelte/icons/triangle-alert';
	import XIcon from '@lucide/svelte/icons/x';
	import Pager from '$lib/components/pager.svelte';
	import { cn } from '$lib/utils';
	import { fromNow } from '$lib/cron';
	import { HISTORY_PER_PAGE, listHistory, type History, type HistoryFilter } from '$lib/api';

	const ok = (h: History) => h.status >= 200 && h.status < 300;

	// Filters live in the URL so a filtered view can be shared or bookmarked.
	const filter: HistoryFilter = $derived.by(() => {
		const p = appPage.url.searchParams;
		return {
			job: p.get('job') ?? '',
			project: p.get('project') ?? '',
			status: p.get('status') ?? '',
			q: p.get('q') ?? '',
			page: Math.max(1, Number(p.get('page')) || 1)
		};
	});
	const filtered = $derived(!!(filter.job || filter.project || filter.status || filter.q));

	/** Update URL params; empty values are dropped and any filter change goes back to page 1. */
	function set(patch: Partial<Record<keyof HistoryFilter, string | number>>) {
		const p = new URLSearchParams(appPage.url.searchParams);
		if (!('page' in patch)) p.delete('page');
		for (const [k, v] of Object.entries(patch)) {
			if (v && !(k === 'page' && v === 1)) p.set(k, String(v));
			else p.delete(k);
		}
		goto(`?${p}`, { replaceState: true, keepFocus: true, noScroll: true });
	}

	let items = $state<History[]>([]);
	let total = $state(0);
	let projects = $state<string[]>([]);
	let loading = $state(true);
	let error = $state('');
	let loaded = $state(false); // Pager must not clamp ?page= against total=0 before the first response
	let seq = 0;

	async function load(f: HistoryFilter, silent = false) {
		const mine = ++seq;
		if (!silent) loading = true;
		try {
			const res = await listHistory(f);
			if (mine !== seq) return; // a newer filter's response wins
			({ items, total, projects } = res);
			error = '';
			loaded = true;
		} catch (err) {
			if (mine === seq) error = (err as Error).message;
		} finally {
			if (mine === seq) loading = false;
		}
	}

	$effect(() => {
		const f = filter;
		load(f);
		const id = setInterval(() => load(f, true), 30_000);
		return () => clearInterval(id);
	});

	// Search box: keep typing snappy, push to the URL after a short pause.
	let search = $state('');
	const urlQ = $derived(filter.q);
	$effect(() => {
		search = urlQ;
	});
	let timer: ReturnType<typeof setTimeout>;
	function onsearch() {
		clearTimeout(timer);
		timer = setTimeout(() => set({ q: search.trim() }), 300);
	}
</script>

<main class="mx-auto flex max-w-6xl flex-col gap-6 p-4 sm:p-8">
	<header class="flex flex-wrap items-center justify-between gap-4">
		<div class="flex flex-col gap-1">
			<h1 class="text-2xl font-semibold tracking-tight">History</h1>
			<p class="text-muted-foreground text-sm">Every job run, kept for 7 days</p>
		</div>
		<Button variant="outline" onclick={() => load(filter)} disabled={loading}>
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
				{total} {filtered ? 'matching' : ''} runs · newest first
				{#if filter.job}· job <code class="font-mono">{filter.job.slice(0, 8)}</code>{/if}
			</Card.Description>
			{#if filtered}
				<Card.Action>
					<Button variant="ghost" size="sm" onclick={() => goto('/history', { noScroll: true })}>
						<XIcon data-icon="inline-start" />
						Clear filters
					</Button>
				</Card.Action>
			{/if}
		</Card.Header>
		<Card.Content class="flex flex-col gap-4">
			<div class="flex flex-wrap items-center gap-2">
				<Input
					type="search"
					class="min-w-48 flex-1"
					placeholder="Search URL, job, status, response…"
					aria-label="Search history"
					bind:value={search}
					oninput={onsearch}
				/>
				<Select.Root
					type="single"
					value={filter.project || 'all'}
					onValueChange={(v) => set({ project: v === 'all' ? '' : v })}
				>
					<Select.Trigger class="w-44" aria-label="Project">
						{filter.project || 'All projects'}
					</Select.Trigger>
					<Select.Content>
						<Select.Group>
							<Select.Item value="all">All projects</Select.Item>
							{#each projects as p (p)}
								<Select.Item value={p}>{p}</Select.Item>
							{/each}
						</Select.Group>
					</Select.Content>
				</Select.Root>
				<ToggleGroup.Root
					type="single"
					variant="outline"
					value={filter.status || 'all'}
					onValueChange={(v) => v && set({ status: v === 'all' ? '' : v })}
				>
					<ToggleGroup.Item value="all">All</ToggleGroup.Item>
					<ToggleGroup.Item value="ok">Success</ToggleGroup.Item>
					<ToggleGroup.Item value="failed">Failed</ToggleGroup.Item>
				</ToggleGroup.Root>
			</div>

			{#if loading && !items.length}
				{#each [1, 2, 3] as i (i)}<Skeleton class="h-10 w-full" />{/each}
			{:else if !items.length}
				<Empty.Root>
					<Empty.Header>
						<Empty.Media variant="icon"><HistoryIcon /></Empty.Media>
						<Empty.Title>{filtered ? 'No matching runs' : 'No runs yet'}</Empty.Title>
						<Empty.Description>
							{filtered ? 'Try a different search or filter.' : 'Runs appear here each time a job fires.'}
						</Empty.Description>
					</Empty.Header>
				</Empty.Root>
			{:else}
				<Table.Root>
					<Table.Header>
						<Table.Row>
							<Table.Head>Project</Table.Head>
							<Table.Head>Time</Table.Head>
							<Table.Head>Status</Table.Head>
							<Table.Head>Request</Table.Head>
							<Table.Head>Job</Table.Head>
							<Table.Head class="text-right">Duration</Table.Head>
						</Table.Row>
					</Table.Header>
					<Table.Body>
						{#each items as h (h.started_at + h.job)}
							{@const at = new Date(h.started_at)}
							<Table.Row>
								<Table.Cell>
									{#if h.project}
										<button class="hover:underline" onclick={() => set({ project: h.project })}>
											{h.project}
										</button>
									{:else}
										<span class="text-muted-foreground">—</span>
									{/if}
								</Table.Cell>
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
								<Table.Cell class="font-mono text-xs">
									<button
										class="text-muted-foreground hover:underline"
										title={h.job}
										onclick={() => set({ job: h.job })}
									>
										{h.job.slice(0, 8)}
									</button>
								</Table.Cell>
								<Table.Cell class="text-right tabular-nums">{h.duration_ms} ms</Table.Cell>
							</Table.Row>
						{/each}
					</Table.Body>
				</Table.Root>
			{/if}

			<!-- Mounted even when the page is empty so an out-of-range ?page= gets clamped. -->
			{#if loaded}
				<Pager
					count={total}
					perPage={HISTORY_PER_PAGE}
					bind:page={() => filter.page, (v) => set({ page: v })}
				/>
			{/if}
		</Card.Content>
	</Card.Root>
</main>
