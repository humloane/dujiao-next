import { computed } from 'vue'
import { useI18n } from 'vue-i18n'
import { useAppStore } from '../stores/app'
import { usePageSeo } from './usePageSeo'
import { buildWebPageJsonLd } from '../utils/seoJsonLd'
import { joinSeoTitle } from '../utils/seoTitle'

/**
 * 条款/隐私页共享逻辑（classic + vault 双模板共用）。
 * 完整保留原 views/Legal.vue 的行为，仅抽离为 composable。
 * type 以 getter 传入以保持响应性（来源于组件 props）。
 */
export function useLegal(type: () => 'terms' | 'privacy') {
  const { t } = useI18n()
  const appStore = useAppStore()

  const siteBase = computed(() => {
    const configured = String(appStore.config?.brand?.site_url || '').trim().replace(/\/+$/, '')
    return configured || (typeof window !== 'undefined' ? window.location.origin : '')
  })
  const siteName = computed(() => String(appStore.config?.brand?.site_name || 'Dujiao-Next'))
  const siteSeoName = computed(() => {
    const localized = appStore.config?.seo?.title?.[appStore.locale]
    return String(localized || appStore.config?.brand?.site_name || '').trim()
  })
  const pageTitle = computed(() => type() === 'terms' ? t('footer.terms') : t('footer.privacy'))
  const pageDescription = computed(() => type() === 'terms' ? t('footer.termsDescription') : t('footer.privacyDescription'))
  const pagePath = computed(() => type() === 'terms' ? '/terms' : '/privacy')
  const fullPageTitle = computed(() => joinSeoTitle(pageTitle.value, siteSeoName.value, [siteName.value]))

  usePageSeo({
    title: () => pageTitle.value,
    description: () => pageDescription.value,
    canonicalPath: () => pagePath.value,
    jsonLd: () => buildWebPageJsonLd({
      title: fullPageTitle.value,
      description: pageDescription.value,
      url: `${siteBase.value}${pagePath.value}`,
      locale: appStore.locale,
      siteName: siteName.value,
      siteBase: siteBase.value,
    }),
  })

  const loading = computed(() => appStore.loading)
  const locale = computed(() => appStore.locale)

  const title = computed(() => {
    return type() === 'terms' ? t('footer.terms') : t('footer.privacy')
  })

  const content = computed(() => {
    const config = appStore.config
    if (!config?.legal) return ''

    const legal = config.legal
    const lang = locale.value

    if (type() === 'terms' && legal.terms) {
      return legal.terms[lang] || ''
    } else if (type() === 'privacy' && legal.privacy) {
      return legal.privacy[lang] || ''
    }
    return ''
  })

  return {
    loading,
    title,
    content,
  }
}
