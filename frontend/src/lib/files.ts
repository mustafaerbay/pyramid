/** "1.2 MB" style sizes. */
export function formatBytes(n: number): string {
	if (n < 1024) return `${n} B`;
	if (n < 1024 * 1024) return `${Math.round(n / 102.4) / 10} KB`;
	return `${Math.round(n / 104857.6) / 10} MB`;
}
