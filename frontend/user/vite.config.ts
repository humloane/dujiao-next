import { fileURLToPath, URL } from 'node:url'
import { dirname, resolve } from 'node:path'
import { defineConfig } from 'vite'
import vue from '@vitejs/plugin-vue'
import VueI18nPlugin from '@intlify/unplugin-vue-i18n/vite'

const cfAsyncModuleScriptPlugin = () => ({
  name: 'cfasync-module-script',
  transformIndexHtml(html: string) {
    return html.replace(
      /<script\s+type="module"(?![^>]*data-cfasync)/g,
      '<script data-cfasync="false" type="module"',
    )
  },
})

const asyncMainStylePlugin = () => ({
  name: 'async-main-style',
  enforce: 'post' as const,
  transformIndexHtml: {
    order: 'post' as const,
    handler(html: string) {
      const fallbackLinks: string[] = []
      let first = true
      const output = html.replace(/<link\b[^>]*\brel="stylesheet"[^>]*>/gi, (tag) => {
        const href = tag.match(/\bhref="([^"]+)"/)?.[1]
        if (!href) return tag
        const crossorigin = /\bcrossorigin(?:\s|=|>)/.test(tag) ? ' crossorigin' : ''
        const id = first ? ' id="main-style"' : ''
        fallbackLinks.push(`<link rel="stylesheet" href="${href}"${crossorigin}>`)
        first = false
        return `<link rel="stylesheet" href="${href}"${crossorigin} media="print" onload="this.media='all'"${id}>`
      })
      if (fallbackLinks.length === 0) return html
      return output.replace(
        '</head>',
        `  <noscript>${fallbackLinks.join('')}</noscript>\n</head>`,
      )
    },
  },
})

// https://vite.dev/config/
export default defineConfig(({ mode }) => ({
  plugins: [
    vue(),
    // 语言包 JSON 构建期预编译为 AST，运行时无需 message compiler（vue-i18n 走 runtime-only 构建）
    VueI18nPlugin({
      include: [resolve(dirname(fileURLToPath(import.meta.url)), 'src/i18n/locales/**')],
      dropMessageCompiler: true,
    }),
    cfAsyncModuleScriptPlugin(),
    asyncMainStylePlugin(),
  ],
  resolve: {
    alias: {
      '@': fileURLToPath(new URL('./src', import.meta.url)),
    },
  },
  // vue-i18n 9.x 需显式开启 JIT，runtime 才能解释预编译的 AST 消息（v10+ 默认开启，届时可移除）
  define: {
    __INTLIFY_JIT_COMPILATION__: true,
  },
  esbuild: mode === 'production' ? { drop: ['console', 'debugger'] } : {},
  build: {
    rollupOptions: {
      output: {
        manualChunks: {
          'vendor-qrcode': ['qrcode'],
          'vendor-vue-i18n': ['vue-i18n'],
        },
      },
    },
  },
  server: {
    host: '0.0.0.0', // 监听所有网络接口
    port: 5173,
    strictPort: true,
    // 允许通过分销商子域名(*.dujiao-next.test)访问 dev server，否则 Vite 5.4+ 会拦截非 localhost 的 Host
    allowedHosts: ['.dujiao.test'],
    proxy: {
      // changeOrigin 必须为 false：保留原始子域名 Host，后端才能据此解析分销商租户
      '/api': {
        target: 'http://localhost:8080',
        changeOrigin: false,
      },
      '/uploads': {
        target: 'http://localhost:8080',
        changeOrigin: false,
      },
      '/sitemap.xml': {
        target: 'http://localhost:8080',
        changeOrigin: false,
      },
      '/robots.txt': {
        target: 'http://localhost:8080',
        changeOrigin: false,
      },
    }
  },
}))
