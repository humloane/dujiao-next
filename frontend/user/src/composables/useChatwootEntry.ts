import { computed, watch, type ComputedRef } from 'vue'
import { useRoute } from 'vue-router'
import { useChatwootState } from './useChatwootState.ts'

export const isChatwootEntryPathVisible = (path: string): boolean => {
  if (path.startsWith('/checkout') || path.startsWith('/pay')) return false
  return true
}

// 整个应用共用一个可见性来源和一个 <html> class watcher：
// 各处只读取同一个 computed，卸载某个页面不会误删另一个页面设置的 class。
let sharedVisible: ComputedRef<boolean> | null = null

export const useChatwootEntryVisible = () => {
  if (sharedVisible) return sharedVisible
  const route = useRoute()
  const { ready } = useChatwootState()
  const visible = computed(() => {
    if (!ready.value) return false
    return isChatwootEntryPathVisible(route.path)
  })

  // 原生 launcher 在「自定义入口接管」或「当前页面明确不应有客服入口」时隐藏：
  // - 正常页/落地页：ready 后自定义入口出现，隐藏原生圆球，避免两个入口重叠；
  // - checkout/pay：从路由进入起就隐藏原生圆球，保证 0 入口；
  // - ready 前：不隐藏，原生圆球作为短暂兜底，自定义入口尚未出现。
  const shouldHideNativeLauncher = computed(() =>
    !isChatwootEntryPathVisible(route.path) || visible.value,
  )

  watch(shouldHideNativeLauncher, (hidden) => {
    if (typeof document === 'undefined') return
    document.documentElement.classList.toggle('chatwoot-custom-entry', hidden)
  }, { immediate: true })

  sharedVisible = visible
  return visible
}