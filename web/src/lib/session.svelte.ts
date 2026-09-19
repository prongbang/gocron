import { getMe, type Me } from '$lib/api';

// Who is signed in; null until /v1/auth/me answers.
export const session = $state<{ me: Me | null }>({ me: null });

export async function refreshSession() {
	session.me = await getMe();
}

/** True when auth is off, or the signed-in role may create and stop jobs. */
export const canWriteJobs = () =>
	session.me?.auth === false || (session.me?.auth === true && session.me.can.write_jobs);
