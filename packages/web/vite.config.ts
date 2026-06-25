import { defineConfig } from 'vite'
import react from '@vitejs/plugin-react'

export default defineConfig({
  base: '/_authgate/',
  plugins: [react()],
  server: {
    port: 5174,
    proxy: {
      '/_authgate/api': 'http://localhost:80',
    },
  },
  optimizeDeps: {
    include: ['recharts'],
  },
  build: {
    outDir: 'dist',
    emptyOutDir: true,
  },
  test: {
    environment: 'jsdom',
    setupFiles: './src/test/setup.ts',
  },
})
