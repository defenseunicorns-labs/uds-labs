/**
 * Copyright 2026 Defense Unicorns
 * SPDX-License-Identifier: AGPL-3.0-or-later OR LicenseRef-Defense-Unicorns-Commercial
 */

import { defineConfig } from 'vite'

export default defineConfig({
  base: '/ide-assets/',
  build: {
    outDir: '../static/ide-assets',
    emptyOutDir: true,
    rollupOptions: {
      input: { main: './src/main.js' },
      output: {
        entryFileNames: '[name].js',
        chunkFileNames: '[name]-[hash].js',
        assetFileNames: (info) => info.name === 'main.css' ? 'main.css' : '[name]-[hash][extname]',
      },
    },
  },
})
