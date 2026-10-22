/**
 * 生成国家/地区代码的本地化名称。
 * 浏览器不支持 Intl.DisplayNames 时直接返回代码，避免筛选器出现空文案。
 */
export const createRegionNamer = (
  locale: string,
  sharedLabel: string,
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
    if (code === 'SHARED') return sharedLabel
    return displayNames?.of(code) || code
  }
}
