import type { components } from '$lib/api/schema';

export type Role = components['schemas']['SystemRole'];

const RANK: Record<Role, number> = { viewer: 0, member: 1, manager: 2, admin: 3 };

export const ROLE_OPTIONS: { value: Role; label: string; hint: string }[] = [
	{ value: 'admin', label: 'Admin', hint: 'Everything, including accounts' },
	{ value: 'manager', label: 'Manager', hint: 'Everything but accounts' },
	{ value: 'member', label: 'Member', hint: "Can't change people or approve leave" },
	{ value: 'viewer', label: 'Viewer', hint: 'Read-only' }
];

export function roleAtLeast(actual: Role | undefined, needed: Role): boolean {
	return actual !== undefined && RANK[actual] >= RANK[needed];
}

/** Where to go after signing in: only paths on this site, never back to /login. */
export function safeNext(next: string | null): string {
	if (!next || !next.startsWith('/') || next.startsWith('//') || next.startsWith('/login')) return '/';
	return next;
}
