import { defineConfig } from 'vite'
import vue from '@vitejs/plugin-vue'
import { fileURLToPath, URL } from 'node:url'

// https://vitejs.dev/config/
export default defineConfig({
  plugins: [vue()],
  resolve: {
    alias: {
      // This tells Vite that '@' maps to your 'src' directory
      '@': fileURLToPath(new URL('./src', import.meta.url))
    }
  }
})
