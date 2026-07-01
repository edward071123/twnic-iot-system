import { defineConfig, loadEnv } from 'vite'
import vue from '@vitejs/plugin-vue'

export default defineConfig(({ mode }) => {
  const env = loadEnv(mode, process.cwd(), '')
  const proxyTarget = env.VITE_PROXY_TARGET || 'http://localhost:3002'

  const proxyOptions = {
    target: proxyTarget,
    changeOrigin: true,
  }

  return {
    plugins: [vue()],
    server: {
      proxy: {
        '/api': proxyOptions,
        '/sensor': proxyOptions,
        '/sensors': proxyOptions,
        '/rooms': proxyOptions,
        '/room': proxyOptions,
        '/people': proxyOptions,
        '/floors': proxyOptions,
        '/hospitals': proxyOptions,
        '/temp_calibration': proxyOptions,
      },
    },
  }
})
