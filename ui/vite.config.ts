import { defineConfig } from 'vitest/config';
import react from '@vitejs/plugin-react';

// Builds into ui/dist, which ui/embed.go embeds into every installer. Stable
// asset names keep the committed dist free of churn between builds.
export default defineConfig({
  plugins: [react()],
  base: './',
  // The GlimStone copy imports @glimstone/core as the package does, and
  // tsconfig.json points that name at the copy, so a port stays a plain copy.
  resolve: { tsconfigPaths: true },
  // Vitest reads every stylesheet as empty, and the tests that hold the look to
  // GlimStone need index.css and the copied tokens as they are written.
  test: { css: { include: /\/(index|tokens)\.css/ } },
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
