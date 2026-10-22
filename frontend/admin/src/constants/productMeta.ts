/** 商品结构化字段选项，与后端 productdomain 常量保持一致。
 *  只保留实际在售地区，避免后台出现没有库存的选项。 */
export const ADMIN_COUNTRY_CODES = [
  'US', 'GB', 'DE', 'JP', 'TW', 'PH', 'SHARED',
] as const

export const ADMIN_SCENARIOS = ['live', 'ecommerce', 'store-group', 'ads', 'enterprise', 'test'] as const
export const ADMIN_IP_TYPES = ['residential', 'datacenter', 'isp', 'shared'] as const
export const ADMIN_LINE_TYPES = ['direct', 'iepl', 'shared'] as const
