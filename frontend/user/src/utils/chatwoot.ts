import { track } from './analytics.ts'

interface ChatwootWidget {
  toggle?: (state?: 'open' | 'close') => void
}

interface ChatwootSDK {
  toggle?: () => void
}

declare global {
  interface Window {
    $chatwoot?: ChatwootWidget
    chatwootSDK?: ChatwootSDK
  }
}

const OPEN_TARGET_SELECTOR = '.woot-widget-bubble:not(.woot--close)'
const RETRY_INTERVAL_MS = 300
const MAX_RETRIES = 10
let retryTimer: ReturnType<typeof setTimeout> | null = null

/** Chatwoot 脚本是否已经加载到可交互状态。 */
export const isChatwootReady = (): boolean => {
  if (typeof window === 'undefined' || typeof document === 'undefined') return false
  return Boolean(
    window.$chatwoot
    || window.chatwootSDK
    || document.querySelector('#chatwoot-container, .woot-widget-bubble'),
  )
}

const tryOpen = (): boolean => {
  if (typeof window === 'undefined') return false
  const widget = window.$chatwoot
  if (widget && typeof widget.toggle === 'function') {
    try {
      widget.toggle('open')
      return true
    } catch {
      // 继续尝试其它入口
    }
  }
  const sdk = window.chatwootSDK
  if (sdk && typeof sdk.toggle === 'function') {
    try {
      sdk.toggle()
      return true
    } catch {
      // 继续尝试泡泡按钮
    }
  }
  const bubble = document.querySelector<HTMLElement>(OPEN_TARGET_SELECTOR)
  if (bubble) {
    try {
      bubble.click()
      return true
    } catch {
      return false
    }
  }
  return false
}

export const cancelChatwootRetry = () => {
  if (retryTimer) {
    clearTimeout(retryTimer)
    retryTimer = null
  }
}

/**
 * 打开 Chatwoot 面板：脚本尚未加载时最多重试 10 次（每 300ms），
 * 未配置 Chatwoot 也不会抛错。
 */
export const openChatwoot = (source = 'unknown'): boolean => {
  if (typeof window === 'undefined') return false
  // 每次用户主动打开只上报一次；后续重试不再重复 track。
  track('chatwoot_click', { source })
  cancelChatwootRetry()
  if (tryOpen()) return true

  let attempts = 0
  const retry = () => {
    retryTimer = null
    if (tryOpen()) return
    attempts += 1
    if (attempts < MAX_RETRIES) {
      retryTimer = setTimeout(retry, RETRY_INTERVAL_MS)
    }
  }
  retryTimer = setTimeout(retry, RETRY_INTERVAL_MS)
  return false
}
