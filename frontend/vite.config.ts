import tailwindcss from '@tailwindcss/vite';
import { sveltekit } from '@sveltejs/kit/vite';
import { defineConfig } from 'vite';

export default defineConfig({
  plugins: [tailwindcss(), sveltekit()],
  // --- 新增修复配置：强制跳过原生 esbuild 检测 ---
  optimizeDeps: {
    exclude: ['esbuild']
  },
  // ---------------------------------------
  server: {
    proxy: {
      '/hybridaction': {
        target: 'http://127.0.0.1:8080', 
        changeOrigin: true,
      }
    }
  }
});