<script lang="ts">
	import * as Dialog from '$lib/components/ui/dialog';
	import * as Field from '$lib/components/ui/field';
	import * as Select from '$lib/components/ui/select';
	import { Button, buttonVariants } from '$lib/components/ui/button';
	import { Input } from '$lib/components/ui/input';
	import { Spinner } from '$lib/components/ui/spinner';
	import UserPlusIcon from '@lucide/svelte/icons/user-plus';
	import { toast } from 'svelte-sonner';
	import { ROLES, createUser } from '$lib/api';

	let { oncreated }: { oncreated: () => void } = $props();

	let open = $state(false);
	let saving = $state(false);
	let username = $state('');
	let password = $state('');
	let role = $state('viewer');

	async function submit(e: SubmitEvent) {
		e.preventDefault();
		saving = true;
		try {
			await createUser({ username: username.trim(), password, role });
			toast.success('User created', { description: username });
			open = false;
			username = password = '';
			oncreated();
		} catch (err) {
			toast.error('Create failed', { description: (err as Error).message });
		} finally {
			saving = false;
		}
	}
</script>

<Dialog.Root bind:open>
	<Dialog.Trigger class={buttonVariants()}>
		<UserPlusIcon data-icon="inline-start" />
		New user
	</Dialog.Trigger>
	<Dialog.Content class="sm:max-w-md">
		<form onsubmit={submit} class="flex flex-col gap-6">
			<Dialog.Header>
				<Dialog.Title>New user</Dialog.Title>
				<Dialog.Description>
					Viewers can look, editors can also create and stop jobs, admins can also manage users.
				</Dialog.Description>
			</Dialog.Header>
			<Field.Group>
				<Field.Field>
					<Field.Label for="new-username">Username</Field.Label>
					<Input id="new-username" required pattern={'[a-z0-9_.\\-]{3,32}'} bind:value={username} />
					<Field.Description>3–32 characters: a–z, 0–9, _ . -</Field.Description>
				</Field.Field>
				<Field.Field>
					<Field.Label for="new-password">Password</Field.Label>
					<Input
						id="new-password"
						type="password"
						autocomplete="new-password"
						required
						minlength={8}
						maxlength={72}
						bind:value={password}
					/>
				</Field.Field>
				<Field.Field>
					<Field.Label for="new-role">Role</Field.Label>
					<Select.Root type="single" bind:value={role}>
						<Select.Trigger id="new-role" class="w-full">{role}</Select.Trigger>
						<Select.Content>
							<Select.Group>
								{#each ROLES as r (r)}<Select.Item value={r}>{r}</Select.Item>{/each}
							</Select.Group>
						</Select.Content>
					</Select.Root>
				</Field.Field>
			</Field.Group>
			<Dialog.Footer>
				<Dialog.Close class={buttonVariants({ variant: 'outline' })}>Cancel</Dialog.Close>
				<Button type="submit" disabled={saving}>
					{#if saving}<Spinner data-icon="inline-start" />{/if}
					Create
				</Button>
			</Dialog.Footer>
		</form>
	</Dialog.Content>
</Dialog.Root>
