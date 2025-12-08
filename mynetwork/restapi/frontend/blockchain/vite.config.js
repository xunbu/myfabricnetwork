import { defineConfig } from 'vite'
import vue from '@vitejs/plugin-vue'

export default defineConfig({
  plugins: [vue()],
  server: {
    proxy: {
      '/valuechain': {
        target: 'http://localhost:8081',
        changeOrigin: true,
        // rewrite: (path) => path.replace(/^\/valuechain/, ''),
        // 可选配置
        // secure: false, // 如果目标服务器使用自签名证书，设置为 false
        // ws: true, // 如果需要代理 WebSocket
      }
    }
  }
})