import createClient from 'openapi-fetch';
import type { paths } from './schema';

// Base URL of the Go backend. Override with VITE_API_URL in .env.
const baseUrl = import.meta.env.VITE_API_URL ?? 'http://localhost:8080';

// A fully typed client. Every path, param, body, and response is checked
// against src/lib/api/schema.d.ts, which is generated from openapi/openapi.yaml
// (the same contract the Go backend generates its types from).
export const api = createClient<paths>({ baseUrl });
