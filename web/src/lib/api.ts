import { env } from '$env/dynamic/public';

export const API_URL = (env.PUBLIC_GOCRON_API || 'http://localhost:8000').replace(/\/$/, '');

export type Job = {
	job: string;
	project?: string;
	cron: string;
	running: boolean;
	next_run?: string;
	task: {
		type: string;
		config: { url: string; method: string; header?: unknown; body?: unknown };
	};
};

export type NewJob = Omit<Job, 'job' | 'running' | 'next_run'>;

async function call<T>(path: string, init?: RequestInit): Promise<T> {
	const res = await fetch(API_URL + path, {
		...init,
		headers: { 'Content-Type': 'application/json' }
	});
	// fiber.ErrBadRequest replies with plain text, core.* replies with JSON
	const text = await res.text();
	let json: { message?: string; data?: T } = {};
	try {
		json = JSON.parse(text);
	} catch {
		json = { message: text };
	}
	if (!res.ok) throw new Error(json.message || res.statusText);
	return json.data as T;
}

export const listJobs = async () => (await call<Job[] | null>('/v1/scheduler')) ?? [];

export const createJob = (job: NewJob) =>
	call<{ job: string }>('/v1/scheduler', { method: 'POST', body: JSON.stringify(job) });

export const stopJob = (job: string) =>
	call<{ job: string }>('/v1/scheduler/stop', { method: 'POST', body: JSON.stringify({ job }) });

export type History = {
	job: string;
	project: string;
	cron: string;
	method: string;
	url: string;
	status: number;
	response?: string;
	started_at: string;
	duration_ms: number;
};

export const listHistory = async (job = '', limit = 500) =>
	(await call<History[] | null>(`/v1/history?${new URLSearchParams({ job, limit: String(limit) })}`)) ?? [];
