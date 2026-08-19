import { fileURLToPath, URL } from 'node:url';

import preact from '@preact/preset-vite';
import { defineConfig } from 'vite';

export default defineConfig({
  plugins: [preact()],
  build: {
    outDir: fileURLToPath(new URL('../internal/webui/dist', import.meta.url)),
    emptyOutDir: true,
  },
  server: {
    host: '127.0.0.1',
    port: 5173,
    proxy: {
      '/api': {
        target: 'http://127.0.0.1:8080',
        // The API compares Origin with Host for state-changing requests.
        changeOrigin: false,
      },
      '/healthz': {
        target: 'http://127.0.0.1:8080',
        changeOrigin: false,
      },
      '/uploads': {
        target: 'http://127.0.0.1:8080',
        changeOrigin: false,
      },
    },
  },
});
