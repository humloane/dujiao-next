<template>
  <div id="app" class="min-h-screen bg-background text-foreground flex flex-col">
    <div v-if="!appStore.configSettled" class="min-h-screen bg-background text-foreground">
      <div class="h-14 border-b border-border bg-muted/40"></div>
      <div class="mx-auto w-full max-w-[1180px] px-4 py-8 sm:px-6">
        <div class="mx-auto mb-7 h-4 w-32 animate-pulse rounded-full bg-muted"></div>
        <div class="min-h-[240px] h-[min(48vh,420px)] animate-pulse rounded-2xl bg-muted"></div>
        <div class="mt-6 grid gap-4 sm:grid-cols-3">
          <div class="h-[150px] animate-pulse rounded-xl bg-muted"></div>
          <div class="h-[150px] animate-pulse rounded-xl bg-muted"></div>
          <div class="h-[150px] animate-pulse rounded-xl bg-muted"></div>
        </div>
      </div>
    </div>
    <template v-else>
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
      <!-- 预留视口高度：异步路由内容到达前，避免 Footer 位于顶部后下移造成 CLS -->
      <main class="flex-1 min-h-[calc(100vh-5rem)]" :class="isResellerConsole ? '' : 'pb-14 md:pb-0'">
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

    <ChatwootFab />
    <Toast />
    <ConfirmDialog />
    </template>
  </div>
</template>

<script setup lang="ts">
import { computed, defineAsyncComponent, onMounted, onUnmounted } from 'vue'
import { useRoute } from 'vue-router'
import { useAppStore } from './stores/app'
import { getActiveTemplate } from './templates/registry'
import Navbar from './components/Navbar.vue'
import Footer from './components/Footer.vue'
import ChatwootFab from './components/ChatwootFab.vue'
import Toast from './components/Toast.vue'
import ConfirmDialog from './components/ConfirmDialog.vue'
import ErrorBoundary from './components/ErrorBoundary.vue'
import BackToTop from './components/BackToTop.vue'
import MobileBottomNav from './components/MobileBottomNav.vue'
import { cancelChatwootRetry, openChatwoot } from './utils/chatwoot'
import { useChatwootEntryVisible } from './composables/useChatwootEntry'

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
const chatwootEntryVisible = useChatwootEntryVisible()

// v-html 商品内容/运营位里可以用 <a href="#chatwoot" data-chatwoot> 打开客服。
const handleChatwootTrigger = (event: MouseEvent) => {
  const target = event.target as Element | null
  if (!target || typeof target.closest !== 'function') return
  const trigger = target.closest('a[href="#chatwoot"], [data-chatwoot]')
  if (!trigger) return
  event.preventDefault()
  if (!chatwootEntryVisible.value) return
  openChatwoot()
}

onMounted(() => {
  document.addEventListener('click', handleChatwootTrigger)
})

onUnmounted(() => {
  document.removeEventListener('click', handleChatwootTrigger)
  cancelChatwootRetry()
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
   推荐同时在 Chatwoot 初始化参数里设置 position: 'right'；这里只做移动端兜底。 */
@media (max-width: 640px) {
  #chatwoot-container {
    bottom: calc(env(safe-area-inset-bottom, 0px) + 72px) !important;
  }
  .woot-widget-bubble {
    transform: scale(0.88);
    transform-origin: bottom right;
  }
}
/* 原生 launcher 只保留一个 CSS 兜底：自定义入口接管或当前页面不应有客服入口时隐藏圆球。
   面板（.woot-widget-holder）不受影响；关闭按钮（.woot--close）不隐藏。 */
html.chatwoot-custom-entry .woot-widget-bubble:not(.woot--close) {
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
</style>
