interface ChatwootWidget {
  toggle?: (state?: 'open' | 'close') => void
  toggleBubbleVisibility?: (state: 'hide' | 'show') => void
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

const BUBBLE_SELECTOR = '.woot-widget-bubble:not(.woot--close)'
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
  const bubble = document.querySelector<HTMLElement>(BUBBLE_SELECTOR)
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
export const openChatwoot = (): boolean => {
  if (typeof window === 'undefined') return false
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

/**
 * 隐藏 Chatwoot 原生气泡。API 未就绪时返回 false，让调用方稍后再试，
 * 避免在脚本加载前就把唯一的入口隐藏掉。
 */
export const hideNativeChatwootLauncher = (): boolean => {
  if (typeof window === 'undefined' || typeof document === 'undefined') return false
  const widget = window.$chatwoot
  if (widget && typeof widget.toggleBubbleVisibility === 'function') {
    try {
      widget.toggleBubbleVisibility('hide')
      document.body.classList.add('chatwoot-native-hidden')
      return true
    } catch {
      return false
    }
  }
  if (isChatwootReady()) {
    document.body.classList.add('chatwoot-native-hidden')
    return true
  }
  return false
}
