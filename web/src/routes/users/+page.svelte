<script lang="ts">
	import { onMount } from 'svelte';
	import * as Card from '$lib/components/ui/card';
	import * as Table from '$lib/components/ui/table';
	import * as Select from '$lib/components/ui/select';
	import * as AlertDialog from '$lib/components/ui/alert-dialog';
	import * as Alert from '$lib/components/ui/alert';
	import { Badge } from '$lib/components/ui/badge';
	import { buttonVariants } from '$lib/components/ui/button';
	import Trash2Icon from '@lucide/svelte/icons/trash-2';
	import TriangleAlertIcon from '@lucide/svelte/icons/triangle-alert';
	import { toast } from 'svelte-sonner';
	import CreateUserDialog from '$lib/components/create-user-dialog.svelte';
	import { ROLES, deleteUser, listUsers, updateUser, type User } from '$lib/api';
	import { session } from '$lib/session.svelte';

	let users = $state<User[]>([]);
	let error = $state('');
	const me = $derived(session.me?.auth ? session.me.username : '');

	async function load() {
		try {
			users = (await listUsers()).sort((a, b) => a.username.localeCompare(b.username));
			error = '';
		} catch (err) {
			error = (err as Error).message;
		}
	}

	async function run(action: () => Promise<unknown>, done: string) {
		try {
			await action();
			toast.success(done);
		} catch (err) {
			toast.error((err as Error).message);
		}
		await load();
	}

	onMount(load);
</script>

<main class="mx-auto flex max-w-6xl flex-col gap-6 p-4 sm:p-8">
	<header class="flex flex-wrap items-center justify-between gap-4">
		<div class="flex flex-col gap-1">
			<h1 class="text-2xl font-semibold tracking-tight">Users</h1>
			<p class="text-muted-foreground text-sm">{users.length} users</p>
		</div>
		<CreateUserDialog oncreated={load} />
	</header>

	{#if error}
		<Alert.Root variant="destructive">
			<TriangleAlertIcon />
			<Alert.Title>Cannot load users</Alert.Title>
			<Alert.Description>{error}</Alert.Description>
		</Alert.Root>
	{/if}

	<Card.Root>
		<Card.Content>
			<Table.Root>
				<Table.Header>
					<Table.Row>
						<Table.Head>Username</Table.Head>
						<Table.Head>Role</Table.Head>
						<Table.Head>Created</Table.Head>
						<Table.Head class="text-right">Action</Table.Head>
					</Table.Row>
				</Table.Header>
				<Table.Body>
					{#each users as u (u.username)}
						<Table.Row>
							<Table.Cell>
								<div class="flex items-center gap-2">
									{u.username}
									{#if u.username === me}<Badge variant="outline">you</Badge>{/if}
								</div>
							</Table.Cell>
							<Table.Cell>
								<Select.Root
									type="single"
									value={u.role}
									disabled={u.username === me}
									onValueChange={(role) =>
										run(() => updateUser(u.username, { role }), `${u.username} is now ${role}`)}
								>
									<Select.Trigger class="w-32" aria-label="Role of {u.username}">{u.role}</Select.Trigger>
									<Select.Content>
										<Select.Group>
											{#each ROLES as r (r)}<Select.Item value={r}>{r}</Select.Item>{/each}
										</Select.Group>
									</Select.Content>
								</Select.Root>
							</Table.Cell>
							<Table.Cell class="text-muted-foreground">
								{new Date(u.created_at).toLocaleDateString(undefined, { dateStyle: 'medium' })}
							</Table.Cell>
							<Table.Cell class="text-right">
								{#if u.username !== me}
									<AlertDialog.Root>
										<AlertDialog.Trigger class={buttonVariants({ variant: 'ghost', size: 'sm' })}>
											<Trash2Icon data-icon="inline-start" />
											Delete
										</AlertDialog.Trigger>
										<AlertDialog.Content>
											<AlertDialog.Header>
												<AlertDialog.Title>Delete {u.username}?</AlertDialog.Title>
												<AlertDialog.Description>
													They are signed out and can no longer sign in. Jobs they created keep running.
												</AlertDialog.Description>
											</AlertDialog.Header>
											<AlertDialog.Footer>
												<AlertDialog.Cancel>Cancel</AlertDialog.Cancel>
												<AlertDialog.Action
													variant="destructive"
													onclick={() => run(() => deleteUser(u.username), `${u.username} deleted`)}
												>
													Delete user
												</AlertDialog.Action>
											</AlertDialog.Footer>
										</AlertDialog.Content>
									</AlertDialog.Root>
								{/if}
							</Table.Cell>
						</Table.Row>
					{/each}
				</Table.Body>
			</Table.Root>
		</Card.Content>
	</Card.Root>
</main>
