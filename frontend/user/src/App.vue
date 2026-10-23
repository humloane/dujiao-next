<template>
  <div id="app" class="min-h-screen bg-background text-foreground flex flex-col">
    <!-- 落地页是独立投放页：不套站点外壳，避免深色页面与站点浅色主题混搭 -->
    <ErrorBoundary v-if="isLandingPage">
      <RouterView v-slot="{ Component }">
        <Transition name="page-fade" mode="out-in">
          <component :is="Component" />
        </Transition>
      </RouterView>
    </ErrorBoundary>

    <!-- vault 模板：自带顶栏/页脚的外壳包裹页面（控制台仍走下方分支） -->
    <VaultLayout v-else-if="isVault && !isResellerConsole">
      <ErrorBoundary>
        <RouterView v-slot="{ Component }">
          <Transition name="page-fade" mode="out-in">
            <component :is="Component" />
          </Transition>
        </RouterView>
      </ErrorBoundary>
    </VaultLayout>

    <!-- classic 模板 / 分销控制台（保持原有结构不变） -->
    <template v-else>
      <Navbar v-if="!isResellerConsole" />
      <main class="flex-1" :class="isResellerConsole ? '' : 'pb-14 md:pb-0'">
        <ErrorBoundary>
          <RouterView v-slot="{ Component }">
            <Transition name="page-fade" mode="out-in">
              <component :is="Component" />
            </Transition>
          </RouterView>
        </ErrorBoundary>
      </main>
      <Footer v-if="!isResellerConsole" />
      <BackToTop v-if="!isResellerConsole" />
      <MobileBottomNav v-if="!isResellerConsole" />
    </template>

    <Loading :loading="appStore.loading" />
    <ChatwootFab />
    <Toast />
    <ConfirmDialog />
  </div>
</template>

<script setup lang="ts">
import { computed, defineAsyncComponent, onMounted, onUnmounted, watch } from 'vue'
import { useRoute } from 'vue-router'
import { useAppStore } from './stores/app'
import { getActiveTemplate } from './templates/registry'
import Navbar from './components/Navbar.vue'
import Footer from './components/Footer.vue'
import ChatwootFab from './components/ChatwootFab.vue'
import Loading from './components/Loading.vue'
import Toast from './components/Toast.vue'
import ConfirmDialog from './components/ConfirmDialog.vue'
import ErrorBoundary from './components/ErrorBoundary.vue'
import BackToTop from './components/BackToTop.vue'
import MobileBottomNav from './components/MobileBottomNav.vue'
import { hideNativeChatwootLauncher, openChatwoot } from './utils/chatwoot'

// vault 外壳按需加载，classic 用户不会拉取其 chunk/样式
const VaultLayout = defineAsyncComponent(() => import('./templates/vault/layout/VaultLayout.vue'))

// config 由 router.beforeEach 统一加载，无需在此重复调用
const appStore = useAppStore()
const route = useRoute()
const isResellerConsole = computed(() => route.meta.resellerConsole === true)
// Landing pages carry their own sticky CTA instead of the global bottom nav.
const isLandingPage = computed(() => route.name === 'landing-page')
// getActiveTemplate 读取 appStore.config（响应式），config 加载后会重新计算
const isVault = computed(() => getActiveTemplate() === 'vault')

// 结算/支付页隐藏 Chatwoot 气泡，避免遮挡支付按钮。
const chatwootHidden = computed(() => route.path.startsWith('/checkout') || route.path.startsWith('/pay'))
watch(chatwootHidden, (hidden) => {
  document.body.classList.toggle('chatwoot-hidden', hidden)
}, { immediate: true })

// v-html 商品内容/运营位里可以用 <a href="#chatwoot" data-chatwoot> 打开客服。
const handleChatwootTrigger = (event: MouseEvent) => {
  const target = event.target as Element | null
  if (!target || typeof target.closest !== 'function') return
  const trigger = target.closest('a[href="#chatwoot"], [data-chatwoot]')
  if (!trigger) return
  event.preventDefault()
  openChatwoot()
}

// Chatwoot 脚本异步加载：确认可用后再隐藏原生气泡，避免加载前把唯一入口藏掉。
let nativeLauncherTimer: number | undefined
let nativeLauncherAttempts = 0
const hideNativeLauncherWhenReady = () => {
  if (hideNativeChatwootLauncher()) return
  nativeLauncherAttempts += 1
  if (nativeLauncherAttempts <= 30) {
    nativeLauncherTimer = window.setTimeout(hideNativeLauncherWhenReady, 500)
  }
}

onMounted(() => {
  document.addEventListener('click', handleChatwootTrigger)
  hideNativeLauncherWhenReady()
})

onUnmounted(() => {
  document.removeEventListener('click', handleChatwootTrigger)
  if (nativeLauncherTimer) window.clearTimeout(nativeLauncherTimer)
  document.body.classList.remove('chatwoot-hidden')
  document.body.classList.remove('chatwoot-native-hidden')
})
</script>

<style>
.page-fade-enter-active,
.page-fade-leave-active {
  transition: opacity 200ms ease;
}

.page-fade-enter-from,
.page-fade-leave-to {
  opacity: 0;
}

/* Chatwoot 通过 site_config.scripts 注入，气泡容器不在 Vue 模板里。
   推荐同时在 Chatwoot 初始化参数里设置 position: 'left'；这里只做移动端兜底。 */
@media (max-width: 640px) {
  #chatwoot-container {
    bottom: calc(env(safe-area-inset-bottom, 0px) + 72px) !important;
  }
  .woot-widget-bubble {
    transform: scale(0.88);
    transform-origin: bottom right;
  }
}
body.chatwoot-hidden #chatwoot-container,
body.chatwoot-hidden .woot-widget-bubble {
  display: none !important;
}

/* 自定义客服按钮固定右下角；把“回到顶部”上移，避免两个悬浮按钮重叠。 */
@media (min-width: 768px) {
  .back-to-top {
    bottom: 6rem !important;
  }
}
@media (max-width: 767px) {
  .back-to-top {
    bottom: calc(3.5rem + env(safe-area-inset-bottom, 0px) + 5rem) !important;
  }
}

/* 原生 launcher 只在 Chatwoot 确认加载后被隐藏；保留关闭按钮（woot--close）。 */
body.chatwoot-native-hidden .woot-widget-bubble:not(.woot--close) {
  display: none !important;
}
</style>
