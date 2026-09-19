import tailwindcss from '@tailwindcss/vite';
import adapter from '@sveltejs/adapter-static';
import { sveltekit } from '@sveltejs/kit/vite';
import { defineConfig } from 'vite';

declare const process: { env: Record<string, string | undefined> }; // avoids pulling in @types/node

export default defineConfig({
	plugins: [
		tailwindcss(),
		sveltekit({
			compilerOptions: {
				// Force runes mode for the project, except for libraries. Can be removed in svelte 6.
				runes: ({ filename }) => filename.split(/[/\\]/).includes('node_modules') ? undefined : true
			},

			// Static SPA embedded into the Go binary (see web.go); the Go server falls back to index.html.
			adapter: adapter({ fallback: 'index.html' })
		})
	],
	server: {
		// In production the UI is served by gocron itself, so the API is same-origin.
		proxy: { '/v1': process.env.GOCRON_API ?? 'http://localhost:8000' }
	}
});
