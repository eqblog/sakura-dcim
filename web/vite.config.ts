import { defineConfig, type PluginOption } from 'vite'
import react from '@vitejs/plugin-react'

// TLS for dev — optional dependency, gracefully skipped if not installed
let basicSsl: (() => PluginOption) | undefined
try {
  basicSsl = (await import('@vitejs/plugin-basic-ssl')).default
} catch {
  // not installed — dev server will run on plain HTTP
}

// https://vite.dev/config/
export default defineConfig({
  plugins: [react(), basicSsl?.()].filter(Boolean) as PluginOption[],
  server: {
    proxy: {
      '/api': {
        target: `http://127.0.0.1:${process.env.BACKEND_PORT || 8080}`,
        changeOrigin: true,
        ws: true,
        // Forward original host so backend can build correct public URLs
        headers: {
          'X-Forwarded-Host': '',  // placeholder, overridden by configure()
        },
        configure: (proxy) => {
          proxy.on('proxyReq', (proxyReq, req) => {
            // Pass the browser's original Host to backend
            if (req.headers.host) {
              proxyReq.setHeader('X-Forwarded-Host', req.headers.host);
            }
          });
        },
      },
    },
  },
  build: {
    target: 'esnext',
  },
  optimizeDeps: {
    esbuildOptions: {
      target: 'esnext',
    },
  },
})
