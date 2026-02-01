import { defineConfig } from 'vite';
import preact from '@preact/preset-vite';
import path from 'path';

export default defineConfig({
  plugins: [preact()],
  root: path.resolve(__dirname, 'src'),
  base: '/static/dist/',
  build: {
    outDir: path.resolve(__dirname, '../static/dist'),
    emptyOutDir: true,
    manifest: false,
    rollupOptions: {
      input: path.resolve(__dirname, 'src/main.jsx'),
      output: {
        entryFileNames: 'main.js',
        chunkFileNames: 'chunks/[name].js',
        assetFileNames: 'assets/[name][extname]'
      }
    }
  }
});
