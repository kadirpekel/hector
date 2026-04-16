import { defineConfig } from 'vite'
import react from '@vitejs/plugin-react'
import { resolve } from 'path'
import { execSync } from 'child_process'
import pkg from './package.json'

function getVersion(): string {
  // 1. Env var from Makefile (HECTOR_VERSION)
  if (process.env.HECTOR_VERSION) return process.env.HECTOR_VERSION
  // 2. Try git describe (works in dev and CI with tags)
  try {
    return execSync('git describe --tags --always --dirty', { encoding: 'utf-8' }).trim()
  } catch {
    // 3. Fall back to package.json
    return pkg.version
  }
}

// https://vite.dev/config/
export default defineConfig({
  plugins: [react()],
  base: process.env.VITE_BASE_PATH || '/',
  resolve: {
    alias: {
      '@': resolve(__dirname, 'src'),
    },
  },
  define: {
    '__APP_VERSION__': JSON.stringify(getVersion()),
  },
})
