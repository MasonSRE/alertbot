import { defineConfig } from 'vite'
import react from '@vitejs/plugin-react'
import path from 'path'

export default defineConfig({
  plugins: [react()],
  resolve: {
    alias: {
      '@': path.resolve(__dirname, './src'),
    },
  },
  server: {
    host: '0.0.0.0',  // Allow access from outside container
    port: 3000,
    proxy: {
      '/api': {
        target: process.env.NODE_ENV === 'production' 
          ? 'http://alertbot:8080'  // Docker container name
          : 'http://localhost:8080', // Local development
        changeOrigin: true,
      },
      '/health': {
        target: process.env.NODE_ENV === 'production' 
          ? 'http://alertbot:8080'  // Docker container name
          : 'http://localhost:8080', // Local development
        changeOrigin: true,
      },
      '/metrics': {
        target: process.env.NODE_ENV === 'production' 
          ? 'http://alertbot:8080'  // Docker container name
          : 'http://localhost:8080', // Local development
        changeOrigin: true,
      },
    },
  },
})