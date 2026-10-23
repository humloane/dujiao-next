import { defineStore } from 'pinia'
import { ref } from 'vue'
import { resolveAdminBrandName } from '@/utils/brand'

/**
 * 复用 AdminLayout/Login 已请求的 /public/config，不新增网络请求。
 * 品牌名保持稳定；描述按当前语言从 seo.description 读取。
 */
export const useAdminSiteBrandStore = defineStore('adminSiteBrand', () => {
  const brandName = ref('')
  const descriptions = ref<Record<string, string>>({})

  const setPublicConfig = (payload: any) => {
    brandName.value = resolveAdminBrandName(payload)
    const raw = payload?.seo?.description
    const next: Record<string, string> = {}
    if (raw && typeof raw === 'object') {
      Object.entries(raw).forEach(([locale, value]) => {
        if (typeof value === 'string' && value.trim()) next[locale] = value.trim()
      })
    }
    descriptions.value = next
  }

  const localizedDescription = (locale: string): string => {
    return descriptions.value[locale] || descriptions.value['zh-CN'] || Object.values(descriptions.value)[0] || ''
  }

  return {
    brandName,
    descriptions,
    setPublicConfig,
    localizedDescription,
  }
})