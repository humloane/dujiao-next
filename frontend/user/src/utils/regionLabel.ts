/**
 * 生成国家/地区代码的本地化名称。
 * 浏览器不支持 Intl.DisplayNames 时直接返回代码，避免筛选器出现空文案。
 */
export const createRegionNamer = (
  locale: string,
): ((code: string) => string) => {
  let displayNames: { of: (code: string) => string | undefined } | null = null
  try {
    const DisplayNames = (Intl as unknown as {
      DisplayNames?: new (locales: string[], options: { type: string }) => { of: (code: string) => string | undefined }
    }).DisplayNames
    displayNames = DisplayNames ? new DisplayNames([locale], { type: 'region' }) : null
  } catch {
    displayNames = null
  }
  return (code: string) => {
    try {
      return displayNames?.of(code) || code
    } catch {
      // Intl.DisplayNames 只接受合法 ISO 地区码；后台允许任意 1-8 位代码，
      // 半成品或自定义代码不能让它抛异常拖垮整页渲染。
      return code
    }
  }
}
