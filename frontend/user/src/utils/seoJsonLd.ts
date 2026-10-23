export type JsonLdEntity = Record<string, unknown>

const isRecord = (value: unknown): value is Record<string, unknown> =>
  Boolean(value) && typeof value === 'object' && !Array.isArray(value)

/** 去尾斜杠、解码 URL、保留根路径，供服务端 data-path 和客户端 pathname 对齐。 */
export const normalizeSeoPath = (path: string | null | undefined): string => {
  let value = String(path || '').trim()
  try {
    value = decodeURIComponent(value)
  } catch {
    // 非法百分号编码保持原样，后续匹配失败时由客户端补一份。
  }
  value = value.replace(/\/+$/, '')
  if (value === '') return '/'
  return value.startsWith('/') ? value : `/${value}`
}

/** 把相对资源路径转成绝对 URL；已是 http(s) 的原样返回。 */
export const absoluteAssetUrl = (
  path: string | null | undefined,
  origin = typeof window !== 'undefined' ? window.location.origin : '',
): string => {
  const value = String(path || '').trim()
  if (!value) return ''
  if (/^https?:\/\//i.test(value)) return value
  const base = String(origin || '').replace(/\/+$/, '')
  if (!base) return value
  return `${base}${value.startsWith('/') ? value : `/${value}`}`
}

type WebPageLdInput = {
  title: string
  description?: string
  url: string
  locale: string
  siteName: string
  siteBase: string
  /** CollectionPage 面包屑第二项；缺省用 title，与服务端路径段命名对齐时传入。 */
  breadcrumbName?: string
}

export const buildWebPageJsonLd = (input: WebPageLdInput): JsonLdEntity[] => {
  const siteURL = `${String(input.siteBase || '').replace(/\/+$/, '')}/`
  return [
    {
      '@type': 'WebPage',
      name: input.title,
      description: input.description || '',
      url: input.url,
      inLanguage: input.locale,
      isPartOf: {
        '@type': 'WebSite',
        name: input.siteName,
        url: siteURL,
      },
    },
    {
      '@type': 'BreadcrumbList',
      itemListElement: [
        { '@type': 'ListItem', position: 1, name: input.siteName, item: siteURL },
        { '@type': 'ListItem', position: 2, name: input.title, item: input.url },
      ],
    },
  ]
}

export const buildCollectionPageJsonLd = (input: WebPageLdInput): JsonLdEntity[] => {
  const siteURL = `${String(input.siteBase || '').replace(/\/+$/, '')}/`
  return [
    {
      '@type': 'CollectionPage',
      name: input.title,
      description: input.description || '',
      url: input.url,
      inLanguage: input.locale,
      isPartOf: {
        '@type': 'WebSite',
        name: input.siteName,
        url: siteURL,
      },
    },
    {
      '@type': 'BreadcrumbList',
      itemListElement: [
        { '@type': 'ListItem', position: 1, name: input.siteName, item: siteURL },
        { '@type': 'ListItem', position: 2, name: input.breadcrumbName || input.title, item: input.url },
      ],
    },
  ]
}

/**
 * 统一 JSON-LD 实体：
 * - 数组直接作为实体列表；
 * - 已经包成 { @context, @graph } 的对象取 @graph；
 * - 单个对象作为一个实体；
 * - 去掉实体自身的 @context，最终由 buildJsonLdPayload 统一输出一份。
 */
export const jsonLdEntities = (
  value: Record<string, unknown> | Record<string, unknown>[] | null | undefined,
): JsonLdEntity[] => {
  if (!value) return []
  const source = Array.isArray(value)
    ? value
    : isRecord(value) && Array.isArray(value['@graph'])
      ? value['@graph']
      : [value]

  return source.filter(isRecord).map((item) => {
    const copy = { ...item }
    delete copy['@context']
    return copy
  })
}

export const buildJsonLdPayload = (
  value: Record<string, unknown> | Record<string, unknown>[] | null | undefined,
): Record<string, unknown> | null => {
  const items = jsonLdEntities(value)
  if (items.length === 0) return null
  return {
    '@context': 'https://schema.org',
    '@graph': items,
  }
}

export const buildJsonLdScript = (
  value: Record<string, unknown> | Record<string, unknown>[] | null | undefined,
): { key: string; type: string; innerHTML: string } | null => {
  const payload = buildJsonLdPayload(value)
  if (!payload) return null
  return {
    key: 'jsonld-page',
    type: 'application/ld+json',
    innerHTML: JSON.stringify(payload),
  }
}

/** 服务端节点只有在路径和当前语言都一致时才可复用。 */
export const isServerJsonLdReusable = (
  nodePath: string | null | undefined,
  nodeLocale: string | null | undefined,
  path: string,
  locale: string,
): boolean => normalizeSeoPath(nodePath) === normalizeSeoPath(path) && String(nodeLocale || '') === String(locale || '')