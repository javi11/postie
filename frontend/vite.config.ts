import { sveltekit } from "@sveltejs/kit/vite";
import tailwindcss from "@tailwindcss/vite";
import { defineConfig } from "vite";

// https://vitejs.dev/config/
export default defineConfig({
	plugins: [sveltekit(), tailwindcss()],
	resolve: {
		alias: {
			$lib: "./src/lib",
		},
	},
	server: {
		port: 5173,
		strictPort: true,
		proxy: {
			"/api": {
				ws: process.env.VITE_WS_PROXY === "true",
				target: "http://localhost:8080",
				changeOrigin: true,
			},
		},
	},
});
