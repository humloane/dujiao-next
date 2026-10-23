/** 站点名里可能带副标题（如「霓途 Neaixt｜全球静态住宅 IP」），取第一段作为品牌核心。 */
const BRAND_SEPARATOR_PATTERN = /[|｜·•—–:：]/

export const siteTitleCore = (site: string): string => {
  const value = String(site || '').trim()
  if (!value) return ''
  return (value.split(BRAND_SEPARATOR_PATTERN)[0] || '').trim()
}

const normalizeTitle = (value: string): string => String(value || '').replace(/\s+/g, ' ').trim().toLowerCase()

/** 页面级标题是否已经包含站点名或品牌核心，避免重复追加品牌。 */
export const titleIncludesSite = (page: string, site: string): boolean => {
  const title = normalizeTitle(page)
  const fullSite = normalizeTitle(site)
  if (!title || !fullSite) return false
  if (title.includes(fullSite)) return true
  const core = normalizeTitle(siteTitleCore(site))
  return core.length >= 2 && title.includes(core)
}

/**
 * 拼接页面标题与站点名：
 * - 页面标题已含站点名/品牌核心时原样返回，品牌只出现一次；
 * - 否则追加品牌核心（而不是完整站点副标题），避免标题过长。
 */
export const joinSeoTitle = (page: string, site: string, aliases: string[] = []): string => {
  const pageTitle = String(page || '').trim()
  const siteName = String(site || '').trim()
  if (!pageTitle) return siteName
  if (!siteName) return pageTitle
  const candidates = [siteName, ...aliases].map((item) => String(item || '').trim()).filter(Boolean)
  if (candidates.some((candidate) => titleIncludesSite(pageTitle, candidate))) return pageTitle
  return `${pageTitle} - ${siteTitleCore(siteName) || siteName}`
}
