/** 商品结构化筛选项。取值与后端 productdomain 常量保持一致。
 *  只保留实际在售地区，避免给用户一堆没有库存的选项。 */
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
