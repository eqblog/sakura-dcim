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
        configure: (proxy) => {
          proxy.on('proxyReq', (proxyReq, req) => {
            // Forward browser's original Host and protocol to backend
            if (req.headers.host) {
              proxyReq.setHeader('X-Forwarded-Host', req.headers.host);
            }
            // TLS is terminated at Vite, so tell backend the original scheme
            const isTLS = (req.socket as any).encrypted;
            proxyReq.setHeader('X-Forwarded-Proto', isTLS ? 'https' : 'http');
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
