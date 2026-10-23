import { onMounted, onUnmounted, ref } from 'vue'
import { isChatwootReady } from '../utils/chatwoot'

/**
 * 轮询等待 Chatwoot 真正可用，避免在脚本未加载/加载失败/后台关闭时
 * 显示点了没反应的客服入口。
 *
 * 超过 timeoutMs 后停止轮询并保持 false；组件卸载时清理定时器。
 */
export const useChatwootReady = (timeoutMs = 10000, intervalMs = 300) => {
  const ready = ref(isChatwootReady())
  let timer: ReturnType<typeof setInterval> | null = null

  const stopPolling = () => {
    if (timer !== null) {
      clearInterval(timer)
      timer = null
    }
  }

  onMounted(() => {
    if (ready.value) return

    const startedAt = Date.now()
    timer = setInterval(() => {
      if (isChatwootReady()) {
        ready.value = true
        stopPolling()
        return
      }
      if (Date.now() - startedAt >= timeoutMs) {
        stopPolling()
      }
    }, intervalMs)
  })

  onUnmounted(stopPolling)

  return { ready }
}