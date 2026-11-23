import createClient from 'openapi-fetch';
import type { paths } from './schema';

// Base URL of the Go backend. Override with VITE_API_URL in .env.
export const baseUrl = import.meta.env.VITE_API_URL ?? 'http://localhost:8080';

// A fully typed client. Every path, param, body, and response is checked
// against src/lib/api/schema.d.ts, which is generated from openapi/openapi.yaml
// (the same contract the Go backend generates its types from).
// credentials: 'include' sends the session cookie (also cross-origin in
// local dev, where the API runs on another port).
export const api = createClient<paths>({ baseUrl, credentials: 'include' });

// A session that ran out mid-use: send the person to sign in, then back.
api.use({
	onResponse({ request, response }) {
		if (response.status !== 401 || typeof window === 'undefined') return;
		const path = new URL(request.url).pathname;
		if (path.endsWith('/auth/login') || path.endsWith('/auth/me')) return;
		if (['/login', '/forgot', '/reset'].includes(window.location.pathname)) return;
		const next = window.location.pathname + window.location.search;
		window.location.href = `/login?next=${encodeURIComponent(next)}`;
	}
});
