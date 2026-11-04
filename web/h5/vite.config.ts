import { defineConfig } from 'vite'
import vue from '@vitejs/plugin-vue'
import tailwindcss from '@tailwindcss/vite'
import { fileURLToPath, URL } from 'node:url'

const backendOrigin = 'http://localhost:8080'

export default defineConfig({
  plugins: [vue(), tailwindcss()],
  resolve: {
    alias: {
      '@': fileURLToPath(new URL('./src', import.meta.url))
    }
  },
  server: {
    port: 5173,
    proxy: {
      // Same-origin proxy through Nginx. Rewrite Origin so the identity CSRF
      // check still matches ALLOWED_ORIGIN=http://localhost:8080 while the
      // browser itself runs on http://localhost:5173.
      '/api': {
        target: backendOrigin,
        changeOrigin: true,
        configure(proxy: any) {
          proxy.on('proxyReq', (request: any) => {
            request.setHeader('Origin', backendOrigin)
          })
        }
      }
    }
  }
})
