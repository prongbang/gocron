<script lang="ts">
	import './layout.css';
	import logo from '$lib/assets/logo.svg';
	import { ModeWatcher } from 'mode-watcher';
	import { goto } from '$app/navigation';
	import { page } from '$app/state';
	import { Toaster } from '$lib/components/ui/sonner';
	import { Badge } from '$lib/components/ui/badge';
	import { Button, buttonVariants } from '$lib/components/ui/button';
	import LogOutIcon from '@lucide/svelte/icons/log-out';
	import { logout } from '$lib/api';
	import { session, refreshSession } from '$lib/session.svelte';

	let { children } = $props();

	const onLogin = $derived(page.url.pathname === '/login');
	$effect(() => {
		if (!onLogin) refreshSession().catch(() => {});
	});

	const links = $derived([
		{ href: '/', label: 'Jobs' },
		{ href: '/history', label: 'History' },
		...(session.me?.auth && session.me.can.manage_users ? [{ href: '/users', label: 'Users' }] : [])
	]);

	async function signOut() {
		await logout().catch(() => {});
		session.me = null;
		goto('/login');
	}
</script>

<svelte:head>
	<link rel="icon" href={logo} type="image/svg+xml" />
	<title>gocron</title>
</svelte:head>

<ModeWatcher />
<Toaster richColors />
{#if !onLogin}
	<nav class="border-b">
		<div class="mx-auto flex max-w-6xl items-center gap-4 px-4 py-3 sm:px-8">
			<a href="/" class="flex items-center gap-2 font-semibold tracking-tight">
				<img src={logo} alt="" class="size-6" />
				gocron
			</a>
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
			{#if session.me?.auth}
				<div class="ml-auto flex items-center gap-2">
					<span class="text-sm">{session.me.username}</span>
					<Badge variant="secondary">{session.me.role}</Badge>
					<Button variant="ghost" size="sm" onclick={signOut}>
						<LogOutIcon data-icon="inline-start" />
						Sign out
					</Button>
				</div>
			{/if}
		</div>
	</nav>
{/if}
{@render children()}
