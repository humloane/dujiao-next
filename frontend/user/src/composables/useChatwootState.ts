import { ref } from 'vue'
import { isChatwootReady } from '../utils/chatwoot.ts'

const CHATWOOT_READY_TIMEOUT_MS = 10000

// 模块级单例：整个应用只允许一个轮询。
const ready = ref(isChatwootReady())
let timer: ReturnType<typeof setInterval> | null = null
let startedAt = 0

const stop = () => {
  if (timer !== null) {
    clearInterval(timer)
    timer = null
  }
}

const ensureChatwootPolling = () => {
  if (ready.value || timer !== null || typeof window === 'undefined') return
  startedAt = Date.now()
  timer = setInterval(() => {
    if (Date.now() - startedAt >= CHATWOOT_READY_TIMEOUT_MS) {
      stop()
      return
    }
    if (isChatwootReady()) {
      ready.value = true
      stop()
    }
  }, 300)
}

export const useChatwootState = () => {
  ensureChatwootPolling()
  return { ready }
}