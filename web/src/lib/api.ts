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
	// Signed out or session expired: send the user to the login page and come back afterwards.
	if (res.status === 401 && !path.startsWith('/v1/auth/')) {
		location.href = `/login?next=${encodeURIComponent(location.pathname + location.search)}`;
		throw new Error('Signed out');
	}
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

export type Me =
	| { auth: false }
	| {
			auth: true;
			username: string;
			role: string;
			can: { write_jobs: boolean; manage_users: boolean };
	  };

export const getMe = () => call<Me>('/v1/auth/me');

export const login = (username: string, password: string) =>
	call<{ token: string }>('/v1/auth/login', {
		method: 'POST',
		body: JSON.stringify({ username, password })
	});

export const logout = () => call<null>('/v1/auth/logout', { method: 'POST' });

export const ROLES = ['admin', 'editor', 'viewer'] as const;

export type User = { username: string; role: string; created_at: string };

export const listUsers = async () => (await call<User[] | null>('/v1/users')) ?? [];

export const createUser = (u: { username: string; password: string; role: string }) =>
	call<User>('/v1/users', { method: 'POST', body: JSON.stringify(u) });

export const updateUser = (username: string, patch: { role?: string; password?: string }) =>
	call<User>(`/v1/users/${encodeURIComponent(username)}`, {
		method: 'PUT',
		body: JSON.stringify(patch)
	});

export const deleteUser = (username: string) =>
	call<null>(`/v1/users/${encodeURIComponent(username)}`, { method: 'DELETE' });
