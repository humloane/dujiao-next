import { createApp } from 'vue'
import { createPinia } from 'pinia'
import { createHead } from '@unhead/vue/client'
import './style.css'
import App from './App.vue'
import router, { warmupCommonRoutes } from './router'
import i18n, { detectLocale, setI18nLocale, warmupLocaleMessages } from './i18n'
import { useTelegramMiniAppStore } from './stores/telegramMiniApp'
import { initTemplateOverride } from './templates/registry'
import { initAnalytics, initTelegramClickTracking } from './utils/analytics'
import { captureUTM } from './utils/utm'

// 预览用：?template=vault 持久化激活模板（站长正式切换走站点配置）
initTemplateOverride()
captureUTM()
initAnalytics()
initTelegramClickTracking()

const brandLog = (globalThis as any).console?.log?.bind(console)
brandLog?.(
  '%c Dujiao-Next %c Digital Commerce Platform ',
  'background:#0071e3;color:#fff;padding:4px 8px;border-radius:4px 0 0 4px;font-weight:bold;',
  'background:#1d1d1f;color:#f5f5f7;padding:4px 8px;border-radius:0 4px 4px 0;',
)
brandLog?.('%cGitHub → https://github.com/dujiao-next', 'color:#6e6e73;')

const app = createApp(App)
const head = createHead()
const pinia = createPinia()

app.use(pinia)
app.use(head)
app.use(router)
app.use(i18n)

// 先挂载，让首屏骨架立即交给 Vue 接管；语言包和 Telegram 初始化在后台完成，
// 不得在 mount 前 await 任何网络请求。
app.mount('#app')

void Promise.allSettled([
  useTelegramMiniAppStore(pinia).init(),
  setI18nLocale(detectLocale()),
])

void router.isReady().then(() => {
    warmupCommonRoutes()
    warmupLocaleMessages()
})
