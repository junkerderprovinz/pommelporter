import { defineConfig } from 'vite';
import react from '@vitejs/plugin-react';

// Builds into ui/dist, which ui/embed.go embeds into every installer. Stable
// asset names keep the committed dist free of churn between builds.
export default defineConfig({
  plugins: [react()],
  base: './',
  build: {
    outDir: 'dist',
    emptyOutDir: true,
    rollupOptions: {
      output: {
        entryFileNames: 'assets/app.js',
        chunkFileNames: 'assets/[name].js',
        assetFileNames: 'assets/[name][extname]',
      },
    },
  },
});
