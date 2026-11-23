// Runs async work on many items, at most `limit` at a time, keeping order.

export type Outcome<T> = { ok: true; value: T } | { ok: false; error: unknown };

export async function runLimited<I, T>(items: readonly I[], limit: number, fn: (item: I, index: number) => Promise<T>): Promise<Outcome<T>[]> {
	const out: Outcome<T>[] = new Array(items.length);
	let next = 0;
	const worker = async () => {
		while (next < items.length) {
			const i = next++;
			try {
				out[i] = { ok: true, value: await fn(items[i], i) };
			} catch (error) {
				out[i] = { ok: false, error };
			}
		}
	};
	await Promise.all(Array.from({ length: Math.min(limit, items.length) }, worker));
	return out;
}
