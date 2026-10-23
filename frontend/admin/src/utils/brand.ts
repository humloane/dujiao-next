type LocalizedText = Record<string, string | undefined> | string | undefined | null

const pickLocalized = (value: LocalizedText, locale?: string): string => {
  if (typeof value === 'string') return value.trim()
  if (!value || typeof value !== 'object') return ''
  const preferred = locale ? value[locale] : ''
  if (typeof preferred === 'string' && preferred.trim()) return preferred.trim()
  for (const candidate of Object.values(value)) {
    if (typeof candidate === 'string' && candidate.trim()) return candidate.trim()
  }
  return ''
}

/**
 * 后台品牌名只从公开配置读取（不额外请求）：
 * brand.name / brand.site_name → 当前语言 seo.title → 首个非空 seo.title。
 */
export const resolveAdminBrandName = (payload: any, locale?: string): string => {
  const brand = payload?.brand || {}
  const explicit = typeof brand.name === 'string' ? brand.name.trim() : ''
  const siteName = typeof brand.site_name === 'string' ? brand.site_name.trim() : ''
  const seoTitle = pickLocalized(payload?.seo?.title, locale)
  return explicit || siteName || seoTitle || ''
}

/** 折叠侧栏使用品牌首字母/词首缩写，空品牌回退到 fallback。 */
export const brandInitials = (name: string | null | undefined, fallback = 'AD'): string => {
  const value = String(name || '').trim()
  if (!value) return fallback
  const words = value.split(/[\s|｜·•—–:：_-]+/).filter(Boolean)
  if (words.length >= 2) {
    return words.slice(0, 2).map((word) => Array.from(word)[0] || '').join('').toUpperCase()
  }
  return Array.from(value).slice(0, 2).join('').toUpperCase() || fallback
}