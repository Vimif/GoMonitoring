import { defineConfig } from 'vite';
import { resolve } from 'path';

export default defineConfig({
  root: '.',
  base: '/static/',

  build: {
    outDir: '../static/dist',
    emptyOutDir: true,

    rollupOptions: {
      input: {
        // Entry points pour chaque page
        main: resolve(__dirname, 'src/main.js'),
        dashboard: resolve(__dirname, 'src/dashboard.js'),
        machine: resolve(__dirname, 'src/machine.js'),
        terminal: resolve(__dirname, 'src/terminal.js'),
      },

      output: {
        // Noms de fichiers optimisÃ©s
        entryFileNames: 'js/[name].[hash].js',
        chunkFileNames: 'js/[name].[hash].js',
        assetFileNames: 'assets/[name].[hash].[ext]',

        // Code splitting pour optimisation
        manualChunks: {
          'vendor': ['chart.js'],
        }
      }
    },

    // Minification
    minify: 'terser',
    terserOptions: {
      compress: {
        drop_console: true,
        drop_debugger: true
      }
    },

    // Source maps pour debugging (dÃ©sactiver en prod)
    sourcemap: false,

    // Optimisations
    target: 'es2020',
    cssCodeSplit: true,
    assetsInlineLimit: 4096, // 4kb
  },

  // Configuration serveur dev
  server: {
    port: 5173,
    strictPort: true,
    proxy: {
      '/api': 'http://localhost:8080',
      '/ws': {
        target: 'ws://localhost:8080',
        ws: true
      }
    }
  },

  // Optimisations des dÃ©pendances
  optimizeDeps: {
    include: ['chart.js']
  }
});
