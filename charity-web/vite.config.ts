import { defineConfig } from 'vite'
import vue from '@vitejs/plugin-vue'
import { resolve } from 'node:path'
import { writeFileSync } from 'node:fs'

const outDir = resolve(__dirname, '../backend/internal/web/charity-dist')

export default defineConfig({
  plugins: [
    vue(),
    {
      name: 'preserve-embed-keep',
      closeBundle() {
        writeFileSync(resolve(outDir, '.keep'), '# Placeholder file for Go embed (//go:embed all:charity-dist)\n')
      }
    }
  ],
  base: '/',
  build: {
    outDir,
    emptyOutDir: true,
    sourcemap: false
  },
  server: {
    port: 5175,
    proxy: {
      '/api': {
        target: 'http://127.0.0.1:8080',
        changeOrigin: true
      },
      '/v1': {
        target: 'http://127.0.0.1:8080',
        changeOrigin: true
      }
    }
  }
})
