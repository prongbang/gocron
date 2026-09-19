<script lang="ts">
	import './layout.css';
	import favicon from '$lib/assets/favicon.svg';
	import { ModeWatcher } from 'mode-watcher';
	import { page } from '$app/state';
	import { Toaster } from '$lib/components/ui/sonner';
	import { buttonVariants } from '$lib/components/ui/button';

	let { children } = $props();

	const links = [
		{ href: '/', label: 'Jobs' },
		{ href: '/history', label: 'History' }
	];
</script>

<svelte:head>
	<link rel="icon" href={favicon} />
	<title>gocron</title>
</svelte:head>

<ModeWatcher />
<Toaster richColors />
<nav class="border-b">
	<div class="mx-auto flex max-w-6xl items-center gap-4 px-4 py-3 sm:px-8">
		<span class="font-semibold tracking-tight">gocron</span>
		<div class="flex gap-1">
			{#each links as l (l.href)}
				{@const active = page.url.pathname === l.href}
				<a
					href={l.href}
					aria-current={active ? 'page' : undefined}
					class={buttonVariants({ variant: active ? 'secondary' : 'ghost', size: 'sm' })}
				>
					{l.label}
				</a>
			{/each}
		</div>
	</div>
</nav>
{@render children()}
