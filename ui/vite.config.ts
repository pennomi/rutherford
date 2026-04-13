import tailwindcss from '@tailwindcss/vite';
import { sveltekit } from '@sveltejs/kit/vite';
import { defineConfig } from 'vite';

const backendPort = process.env.RUTHERFORD_PORT ?? '8080';
const backendOrigin = `http://localhost:${backendPort}`;

export default defineConfig({
  plugins: [tailwindcss(), sveltekit()],
  server: {
    proxy: {
      '/api': backendOrigin,
      '/ws': { target: backendOrigin, ws: true }
    }
  }
});
