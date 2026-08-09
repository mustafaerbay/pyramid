import { defineConfig } from 'vitest/config';

// Deliberately minimal — these are plain TypeScript unit tests for pure
// logic (frontend/src/lib/*.test.ts), not Svelte component tests, so this
// config skips the sveltekit/tailwind vite plugins vite.config.ts uses for
// the app build. If component-level tests are added later (e.g. via
// @testing-library/svelte), reach for a jsdom environment + those plugins
// then rather than paying that cost for every unit test run now.
export default defineConfig({
	test: {
		environment: 'node',
		include: ['src/**/*.{test,spec}.ts']
	}
});
