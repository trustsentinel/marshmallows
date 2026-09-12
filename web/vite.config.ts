import { defineConfig } from 'vite'
import react from '@vitejs/plugin-react'

// Dev server runs on :8080 so the browser origin matches mm-auth's default
// WebAuthn RP origin (http://localhost:8080).
export default defineConfig({
  plugins: [react()],
  server: { port: 8080, host: true },
  preview: { port: 8080, host: true },
})
