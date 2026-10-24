declare global {
  interface Window {
    dataLayer?: unknown[]
    gtag?: (...args: unknown[]) => void
    clarity?: (...args: unknown[]) => void
    umami?: { track: (name: string, data?: Record<string, unknown>) => void }
  }
}

let initialized = false

const env = (key: string): string => {
  const value = (import.meta.env as Record<string, unknown>)[key]
  return typeof value === 'string' ? value.trim() : ''
}

const appendScript = (src: string, async = true) => {
  if (typeof document === 'undefined') return
  if (document.querySelector(`script[src="${src}"]`)) return
  const script = document.createElement('script')
  script.src = src
  script.async = async
  document.head.appendChild(script)
}

/** 初始化 GA4 / Microsoft Clarity；未配置 ID 时完全静默，不产生请求。 */
export function initAnalytics(): void {
  if (initialized || typeof window === 'undefined') return
  initialized = true

  const ga4Id = env('VITE_GA4_ID')
  if (ga4Id) {
    window.dataLayer = window.dataLayer || []
    window.gtag = function gtag(...args: unknown[]) {
      window.dataLayer?.push(args)
    }
    window.gtag('js', new Date())
    window.gtag('config', ga4Id, { anonymize_ip: true })
    appendScript(`https://www.googletagmanager.com/gtag/js?id=${encodeURIComponent(ga4Id)}`)
  }

  const clarityId = env('VITE_CLARITY_ID')
  if (clarityId) {
    window.clarity = window.clarity || function clarity(...args: unknown[]) {
      ;(window.clarity as unknown as { q?: unknown[][] }).q =
        (window.clarity as unknown as { q?: unknown[][] }).q || []
      ;(window.clarity as unknown as { q: unknown[][] }).q.push(args)
    }
    appendScript(`https://www.clarity.ms/tag/${encodeURIComponent(clarityId)}`)
  }
}

/** 统一事件入口；未加载任何统计脚本时是空操作。 */
export function track(name: string, params?: Record<string, unknown>): void {
  if (typeof window === 'undefined') return
  if (window.gtag) window.gtag('event', name, params || {})
  if (window.clarity) window.clarity('event', name)
  if (window.umami && typeof window.umami.track === 'function') {
    try {
      window.umami.track(name, params || {})
    } catch {
      // 统计脚本异常不能影响页面交互；未加载 Umami 时本来就无操作。
    }
  }
}

/** Google Ads 购买转化；未加载 gtag 时静默。只在订单支付成功后调用一次。 */
export function trackGoogleAdsConversion(params: {
  sendTo: string
  value?: number
  currency?: string
  transactionId?: string
}): void {
  if (typeof window === 'undefined' || typeof window.gtag !== 'function') return
  window.gtag('event', 'conversion', {
    send_to: params.sendTo,
    value: params.value ?? 0,
    currency: params.currency || 'CNY',
    transaction_id: params.transactionId || '',
  })
}

export function initTelegramClickTracking(): void {
  if (typeof document === 'undefined') return
  document.addEventListener(
    'click',
    (event) => {
      const target = event.target as HTMLElement | null
      const link = target?.closest?.('a[href*="t.me"], a[href*="telegram.me"], a[href*="telegram.org"]') as HTMLAnchorElement | null
      if (!link) return
      track('telegram_click', {
        link_url: link.href,
        page_path: window.location.pathname,
      })
    },
    { passive: true },
  )
}
