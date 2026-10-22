const STORAGE_KEY = 'dj-utm-attribution'
const UTM_KEYS = ['source', 'medium', 'campaign', 'content', 'term'] as const

export type UTMPayload = Partial<Record<(typeof UTM_KEYS)[number], string>>

/** 从当前 URL 捕获 utm_*，本次会话内保留，供下单归因使用。 */
export function captureUTM(): void {
  if (typeof window === 'undefined') return
  try {
    const params = new URLSearchParams(window.location.search)
    const payload: UTMPayload = {}
    UTM_KEYS.forEach((key) => {
      const value = params.get(`utm_${key}`)
      if (value && value.trim()) payload[key] = value.trim().slice(0, 200)
    })
    if (Object.keys(payload).length > 0) {
      sessionStorage.setItem(STORAGE_KEY, JSON.stringify(payload))
    }
  } catch {
    /* sessionStorage 不可用时忽略 */
  }
}

export function getUTMPayload(): UTMPayload {
  if (typeof window === 'undefined') return {}
  try {
    const raw = sessionStorage.getItem(STORAGE_KEY)
    if (!raw) return {}
    const parsed = JSON.parse(raw) as UTMPayload
    const result: UTMPayload = {}
    UTM_KEYS.forEach((key) => {
      const value = parsed[key]
      if (typeof value === 'string' && value.trim()) result[key] = value.trim().slice(0, 200)
    })
    return result
  } catch {
    return {}
  }
}
