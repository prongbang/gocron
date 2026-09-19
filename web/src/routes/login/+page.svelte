<script lang="ts">
	import { goto } from '$app/navigation';
	import { page } from '$app/state';
	import * as Card from '$lib/components/ui/card';
	import * as Field from '$lib/components/ui/field';
	import * as Alert from '$lib/components/ui/alert';
	import { Input } from '$lib/components/ui/input';
	import { Button } from '$lib/components/ui/button';
	import { Spinner } from '$lib/components/ui/spinner';
	import TriangleAlertIcon from '@lucide/svelte/icons/triangle-alert';
	import logo from '$lib/assets/logo.svg';
	import { login } from '$lib/api';

	let username = $state('');
	let password = $state('');
	let error = $state('');
	let busy = $state(false);

	// Same-site paths only, so ?next= cannot bounce people to another site.
	const next = $derived.by(() => {
		const n = page.url.searchParams.get('next') ?? '/';
		return n.startsWith('/') && !n.startsWith('//') ? n : '/';
	});

	async function submit(e: SubmitEvent) {
		e.preventDefault();
		busy = true;
		error = '';
		try {
			await login(username.trim(), password);
			await goto(next);
		} catch (err) {
			const msg = (err as Error).message;
			error = msg === 'Unauthorized' ? 'Wrong username or password.' : msg;
		} finally {
			busy = false;
		}
	}
</script>

<main class="flex min-h-svh items-center justify-center p-4">
	<Card.Root class="w-full max-w-sm">
		<Card.Header class="text-center">
			<img src={logo} alt="" class="mx-auto size-10" />
			<Card.Title>Sign in to gocron</Card.Title>
		</Card.Header>
		<Card.Content>
			<form onsubmit={submit} class="flex flex-col gap-6">
				{#if error}
					<Alert.Root variant="destructive">
						<TriangleAlertIcon />
						<Alert.Title>{error}</Alert.Title>
					</Alert.Root>
				{/if}
				<Field.Group>
					<Field.Field>
						<Field.Label for="username">Username</Field.Label>
						<Input id="username" autocomplete="username" required bind:value={username} />
					</Field.Field>
					<Field.Field>
						<Field.Label for="password">Password</Field.Label>
						<Input
							id="password"
							type="password"
							autocomplete="current-password"
							required
							bind:value={password}
						/>
					</Field.Field>
				</Field.Group>
				<Button type="submit" disabled={busy}>
					{#if busy}<Spinner data-icon="inline-start" />{/if}
					Sign in
				</Button>
			</form>
		</Card.Content>
	</Card.Root>
</main>
