import { defineConfig } from 'vite';
import react from '@vitejs/plugin-react';

// Builds into ui/dist, which ui/embed.go embeds into every installer. Stable
// asset names keep the committed dist free of churn between builds.
export default defineConfig({
  plugins: [react()],
  base: './',
  // The GlimStone copy imports @glimstone/core as the package does, and
  // tsconfig.json points that name at the copy, so a port stays a plain copy.
  resolve: { tsconfigPaths: true },
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
