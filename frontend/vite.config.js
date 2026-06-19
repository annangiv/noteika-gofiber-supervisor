import { defineConfig } from 'vite'
import react from '@vitejs/plugin-react'
import path from 'path'

// https://vite.dev/config/
export default defineConfig(({ mode }) => ({
  plugins: [react()],
  // Production bundle is served by Go at /static/; dev uses root with proxy below.
  base: mode === 'production' ? '/static/' : '/',
  build: {
    outDir: path.resolve(__dirname, '../static'),
    emptyOutDir: true,
  },
  optimizeDeps: {
    exclude: ['@huggingface/transformers'],
  },
  server: {
    port: 5173,
    proxy: {
      '/api': 'http://localhost:8080',
      '/auth': 'http://localhost:8080',
      '/oauth': 'http://localhost:8080',
    },
  },
}))
