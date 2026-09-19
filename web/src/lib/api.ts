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
	const res = await fetch(path, {
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

export type HistoryFilter = { job: string; project: string; status: string; q: string; page: number };
export type HistoryPage = { items: History[]; total: number; projects: string[] };

export const HISTORY_PER_PAGE = 20;

export function listHistory(f: HistoryFilter) {
	const params = new URLSearchParams({ page: String(f.page), limit: String(HISTORY_PER_PAGE) });
	for (const k of ['job', 'project', 'status', 'q'] as const) if (f[k]) params.set(k, f[k]);
	return call<HistoryPage>(`/v1/history?${params}`);
}
