import { defineConfig } from '/root/.local/share/fnm/node-versions/v25.9.0/installation/lib/node_modules/vite/dist/node/index.js'
import react from '/root/.local/share/fnm/node-versions/v25.9.0/installation/lib/node_modules/@vitejs/plugin-react/dist/index.js'

export default defineConfig({
  plugins: [react()],
  build: {
    outDir: 'dist',
    emptyOutDir: true,
  },
})
