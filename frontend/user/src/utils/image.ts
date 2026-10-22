/**
 * 获取完整的图片 URL
 * 如果传入的是相对路径（如 /uploads/xxx.png），则拼接 API 基础 URL
 * 如果传入的是完整 URL（如 http://...），则直接返回
 */
export function getImageUrl(path: string | undefined | null): string {
    if (!path) return ''

    // 如果已经是完整 URL，直接返回
    if (path.startsWith('http://') || path.startsWith('https://')) {
        return path
    }

    // 获取 API 基础 URL（Node 单测里 import.meta.env 可能不存在）
    const env = (import.meta as ImportMeta & { env?: Record<string, string | undefined> }).env
    const apiBaseUrl = env?.VITE_API_BASE_URL || ''

    // 确保路径以 / 开头
    const normalizedPath = path.startsWith('/') ? path : `/${path}`

    return `${apiBaseUrl}${normalizedPath}`
}

/**
 * 生成商品图/横幅的 srcset：只对本地 uploads 下的 jpg/png 生效。
 * 变体由 scripts/optimize-images.sh 生成（-480/-800/-1200/-1800.webp），
 * 组件在变体加载失败时会自动回退到原图。
 */
export function getImageVariantSrcSet(
  path: string | undefined | null,
  widths: number[] = [480, 800, 1200],
): string {
  if (!path) return ''
  if (/^(https?:)?\/\//i.test(path) || path.startsWith('data:')) return ''
  const normalized = path.startsWith('/') ? path : `/${path}`
  if (!/^\/uploads\/.+\.(png|jpe?g)$/i.test(normalized)) return ''
  const base = normalized.replace(/\.(png|jpe?g)$/i, '')
  return widths.map((width) => `${getImageUrl(`${base}-${width}.webp`)} ${width}w`).join(', ')
}

/** 首屏 banner 使用更大的 1800 档，避免 1440/2x 屏幕发虚。 */
export function getBannerSrcSet(path: string | undefined | null): string {
  return getImageVariantSrcSet(path, [800, 1200, 1800])
}

/**
 * 从 images 数组中获取第一张图片的完整 URL
 */
export function getFirstImageUrl(images: any): string {
    if (!images) return ''

    let imageUrl = ''

    if (Array.isArray(images)) {
        imageUrl = images[0] || ''
    } else if (images.images && Array.isArray(images.images)) {
        imageUrl = images.images[0] || ''
    }

    return getImageUrl(imageUrl)
}
