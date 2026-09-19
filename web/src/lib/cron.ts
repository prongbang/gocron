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
