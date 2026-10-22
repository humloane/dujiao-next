/** 商品结构化筛选项。取值与后端 productdomain 常量保持一致。
 *  实际可用取值以后端 /public/product-filter-options 返回的商品数据为准，
 *  这里的常量只用于「接口未返回/失败」时的兜底与排序基准。 */
export const PRODUCT_COUNTRY_CODES = [
  'US', 'GB', 'DE', 'JP', 'TW', 'PH', 'SHARED',
] as const

export const PRODUCT_SCENARIOS = [
  'live', 'ecommerce', 'store-group', 'ads', 'enterprise', 'test',
] as const

export const PRODUCT_IP_TYPES = ['residential', 'datacenter', 'isp', 'shared'] as const
export const PRODUCT_LINE_TYPES = ['direct', 'iepl', 'shared'] as const
export const PRODUCT_SORTS = ['recommended', 'price_asc', 'price_desc', 'newest'] as const

export type ProductMetaFilter = {
  country: string
  scenario: string
  ipType: string
  lineType: string
  sort: string
}

/** 筛选抽屉与 chips 使用的字段顺序。 */
export const PRODUCT_META_FILTER_KEYS = ['country', 'scenario', 'ipType', 'lineType'] as const

export type ProductMetaFilterKey = (typeof PRODUCT_META_FILTER_KEYS)[number]

/**
 * 后端按在售商品聚合出的筛选取值。
 * loaded=false 表示尚未拿到或请求失败，此时前端回落到常量列表。
 */
export interface ProductMetaOptions {
  loaded: boolean
  countries: string[]
  scenarios: string[]
  ipTypes: string[]
  lineTypes: string[]
}

export const emptyProductMetaOptions = (): ProductMetaOptions => ({
  loaded: false,
  countries: [],
  scenarios: [],
  ipTypes: [],
  lineTypes: [],
})

/** 过滤空值并去重；可选 transform 用于统一大小写。 */
export const normalizeOptionCodes = (value: unknown, transform?: (code: string) => string): string[] => {
  if (!Array.isArray(value)) return []
  const seen = new Set<string>()
  const result: string[] = []
  value.forEach((item) => {
    const raw = String(item ?? '').trim()
    if (!raw) return
    const code = transform ? transform(raw) : raw
    if (!code || seen.has(code)) return
    seen.add(code)
    result.push(code)
  })
  return result
}

/** 国家/地区统一大写；场景、IP 类型、线路类型是固定小写枚举。 */
const optionTransform = (
  key: 'countries' | 'scenarios' | 'ipTypes' | 'lineTypes',
): ((code: string) => string) => key === 'countries'
  ? (code) => code.toUpperCase()
  : (code) => code.toLowerCase()

/**
 * 解析某个筛选组的选项：
 * - 接口成功且商品里确实有数据时，完全使用商品数据，新增取值不需要改前端；
 * - 商品还没有填这些字段（新站）或请求失败时，回落到常量，避免筛选器整体空掉；
 * - 取值统一清洗大小写，避免历史数据里的 us / Residential 变成第二份选项；
 * - URL 上带的选择即使没有商品命中也要保留，否则下拉框无法回显。
 */
export const resolveMetaOptionCodes = (
  options: ProductMetaOptions | undefined,
  key: 'countries' | 'scenarios' | 'ipTypes' | 'lineTypes',
  fallback: readonly string[],
  selected = '',
): string[] => {
  const transform = optionTransform(key)
  const dynamic = normalizeOptionCodes(options?.[key], transform)
  const source = options?.loaded === true && dynamic.length > 0
    ? dynamic
    : normalizeOptionCodes(fallback, transform)
  const result = [...source]
  const current = normalizeOptionCodes([selected], transform)[0] || ''
  if (current && !result.includes(current)) result.push(current)

  // 常量里的标准顺序优先，动态新增项按字母序排在后面。
  const rank = new Map<string, number>(
    normalizeOptionCodes(fallback, transform).map((code, index) => [code, index]),
  )
  result.sort((a, b) => {
    const rankA = rank.has(a) ? (rank.get(a) as number) : Number.MAX_SAFE_INTEGER
    const rankB = rank.has(b) ? (rank.get(b) as number) : Number.MAX_SAFE_INTEGER
    if (rankA !== rankB) return rankA - rankB
    return a.localeCompare(b)
  })
  return result
}

export const emptyProductMetaFilter = (): ProductMetaFilter => ({
  country: '',
  scenario: '',
  ipType: '',
  lineType: '',
  sort: 'recommended',
})

export const hasProductMetaFilter = (filter: ProductMetaFilter): boolean => {
  return Boolean(filter.country || filter.scenario || filter.ipType || filter.lineType)
}
