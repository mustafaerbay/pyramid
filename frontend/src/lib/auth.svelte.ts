import { api } from '$lib/api/client';
import type { components } from '$lib/api/schema';
import { roleAtLeast, type Role } from '$lib/roles';

type AuthMe = components['schemas']['AuthMe'];

/** The signed-in account; `checked` turns true after the first lookup. */
export const session = $state<{ me: AuthMe | null; checked: boolean }>({ me: null, checked: false });

export async function loadMe(): Promise<AuthMe | null> {
	const { data } = await api.GET('/api/v1/auth/me');
	session.me = data ?? null;
	session.checked = true;
	return session.me;
}

export async function signOut() {
	await api.POST('/api/v1/auth/logout');
	session.me = null;
}

export function can(role: Role): boolean {
	return roleAtLeast(session.me?.user.role, role);
}
