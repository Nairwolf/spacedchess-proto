import { defineConfig } from 'vite'
import react from '@vitejs/plugin-react'

export default defineConfig({
  plugins: [react()],
  server: {
    proxy: {
      // Dev only: the Go API runs on :8080 (see README).
      '/api': 'http://localhost:8080',
    },
  },
})
