import cronstrue from 'cronstrue';

/** "30 9 * * 1-5" → { ok: true, text: "At 09:30, Monday through Friday" }; parse errors come back with ok: false. */
export function describeCron(expr: string): { ok: boolean; text: string } {
	try {
		return { ok: true, text: cronstrue.toString(expr, { use24HourTimeFormat: true }) };
	} catch (err) {
		// cronstrue throws plain strings prefixed with "Error: "
		return { ok: false, text: String(err).replace(/^Error: /, '') };
	}
}

/** Server timezone from an RFC 3339 timestamp: "…Z" → "UTC", "…+07:00" → "UTC+07:00". */
export const serverZone = (ts: string) => {
	const off = ts.match(/(Z|[+-]\d\d:\d\d)$/)?.[1];
	return off && (off === 'Z' ? 'UTC' : `UTC${off}`);
};

const rtf = new Intl.RelativeTimeFormat('en', { numeric: 'auto' });
const UNITS = [['day', 86400], ['hour', 3600], ['minute', 60]] as const;

/** "in 5 minutes", "tomorrow", "in 30 seconds" */
export function fromNow(date: Date, now = Date.now()) {
	const s = Math.round((date.getTime() - now) / 1000);
	for (const [unit, sec] of UNITS) if (Math.abs(s) >= sec) return rtf.format(Math.round(s / sec), unit);
	return rtf.format(s, 'second');
}
