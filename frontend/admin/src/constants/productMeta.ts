/** 商品结构化字段选项，与后端 productdomain 常量保持一致。
 *  这里只是「常用地区」推荐项；商品已有的其它国家/地区必须保留并可显示。 */
export const ADMIN_COUNTRY_CODES = [
  'US', 'GB', 'DE', 'JP', 'TW', 'PH', 'SHARED',
] as const

/**
 * 汇总国家/地区建议项：常用地区 + 商品里已经用过的取值 + 当前商品取值。
 * 后台允许直接填写新代码，所以这里只作为 datalist 建议，不构成取值上限。
 */
export const mergeAdminCountryCodes = (
  current?: string | null,
  used: readonly string[] = [],
): string[] => {
  const codes = [...ADMIN_COUNTRY_CODES] as string[]
  const push = (raw: string) => {
    const value = String(raw || '').trim().toUpperCase()
    if (value && !codes.includes(value)) {
      codes.push(value)
    }
  }
  used.forEach(push)
  push(current || '')
  return codes
}

export const ADMIN_SCENARIOS = ['live', 'ecommerce', 'store-group', 'ads', 'enterprise', 'test'] as const
export const ADMIN_IP_TYPES = ['residential', 'datacenter', 'isp', 'shared'] as const
export const ADMIN_LINE_TYPES = ['direct', 'iepl', 'shared'] as const
