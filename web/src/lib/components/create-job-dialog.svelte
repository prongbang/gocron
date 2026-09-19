<script lang="ts">
	import * as Dialog from '$lib/components/ui/dialog';
	import * as Field from '$lib/components/ui/field';
	import * as Select from '$lib/components/ui/select';
	import { Button, buttonVariants } from '$lib/components/ui/button';
	import { Input } from '$lib/components/ui/input';
	import { Textarea } from '$lib/components/ui/textarea';
	import { Spinner } from '$lib/components/ui/spinner';
	import PlusIcon from '@lucide/svelte/icons/plus';
	import { toast } from 'svelte-sonner';
	import { createJob } from '$lib/api';

	let { oncreated }: { oncreated: () => void } = $props();

	const METHODS = ['GET', 'POST', 'PUT', 'PATCH', 'DELETE'];

	let open = $state(false);
	let saving = $state(false);
	let cron = $state('*/1 * * * *');
	let method = $state('POST');
	let url = $state('');
	let header = $state('');
	let body = $state('');
	let errors = $state<Record<string, string>>({});

	function parseJSON(key: string, text: string) {
		if (!text.trim()) return undefined;
		try {
			return JSON.parse(text);
		} catch {
			errors[key] = 'Invalid JSON';
		}
	}

	async function submit(e: SubmitEvent) {
		e.preventDefault();
		errors = {};
		if (!cron.trim()) errors.cron = 'Required';
		if (!url.trim()) errors.url = 'Required';
		const h = parseJSON('header', header);
		const b = parseJSON('body', body);
		if (Object.keys(errors).length) return;

		saving = true;
		try {
			const { job } = await createJob({
				cron: cron.trim(),
				task: { type: 'api', config: { url: url.trim(), method, header: h, body: b } }
			});
			toast.success('Job created', { description: job });
			open = false;
			url = header = body = '';
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
		<PlusIcon data-icon="inline-start" />
		New job
	</Dialog.Trigger>
	<Dialog.Content class="sm:max-w-lg">
		<form onsubmit={submit} class="flex flex-col gap-6">
			<Dialog.Header>
				<Dialog.Title>New job</Dialog.Title>
				<Dialog.Description>Call an HTTP endpoint on a cron schedule.</Dialog.Description>
			</Dialog.Header>

			<Field.Group>
				<Field.Field data-invalid={errors.cron ? true : undefined}>
					<Field.Label for="cron">Cron</Field.Label>
					<Input id="cron" class="font-mono" bind:value={cron} aria-invalid={!!errors.cron} />
					<Field.Description>
						e.g. <code>0 0 * * *</code> — check it on
						<a href="https://crontab.guru/" target="_blank" rel="noreferrer">crontab.guru</a>
					</Field.Description>
					{#if errors.cron}<Field.Error>{errors.cron}</Field.Error>{/if}
				</Field.Field>

				<div class="flex gap-3">
					<Field.Field class="w-32 shrink-0">
						<Field.Label for="method">Method</Field.Label>
						<Select.Root type="single" bind:value={method}>
							<Select.Trigger id="method" class="w-full">{method}</Select.Trigger>
							<Select.Content>
								<Select.Group>
									{#each METHODS as m (m)}
										<Select.Item value={m}>{m}</Select.Item>
									{/each}
								</Select.Group>
							</Select.Content>
						</Select.Root>
					</Field.Field>
					<Field.Field data-invalid={errors.url ? true : undefined}>
						<Field.Label for="url">URL</Field.Label>
						<Input
							id="url"
							type="url"
							placeholder="http://localhost/notify"
							bind:value={url}
							aria-invalid={!!errors.url}
						/>
						{#if errors.url}<Field.Error>{errors.url}</Field.Error>{/if}
					</Field.Field>
				</div>

				<Field.Field data-invalid={errors.header ? true : undefined}>
					<Field.Label for="header">Headers (JSON)</Field.Label>
					<Textarea
						id="header"
						class="font-mono"
						placeholder={'{"X-API-KEY": "ABC"}'}
						bind:value={header}
						aria-invalid={!!errors.header}
					/>
					{#if errors.header}<Field.Error>{errors.header}</Field.Error>{/if}
				</Field.Field>

				<Field.Field data-invalid={errors.body ? true : undefined}>
					<Field.Label for="body">Body (JSON)</Field.Label>
					<Textarea
						id="body"
						class="font-mono"
						placeholder={'{"data": "Hi"}'}
						bind:value={body}
						aria-invalid={!!errors.body}
					/>
					{#if errors.body}<Field.Error>{errors.body}</Field.Error>{/if}
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
