import { computed, onMounted } from 'vue'
import { useI18n } from 'vue-i18n'
import { useAppStore } from '../stores/app'
import { usePageSeo } from './usePageSeo'
import { buildWebPageJsonLd } from '../utils/seoJsonLd'
import { joinSeoTitle } from '../utils/seoTitle'

/**
 * 关于页共享逻辑（classic + vault 双模板共用）。
 * 完整保留原 views/About.vue 的行为，仅抽离为 composable。
 */
export function useAbout() {
  const { t, locale } = useI18n()
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
  const pageTitle = computed(() => t('about.title'))
  const pageDescription = computed(() => t('about.seoDescription'))
  const fullPageTitle = computed(() => joinSeoTitle(pageTitle.value, siteSeoName.value, [siteName.value]))

  usePageSeo({
    title: () => pageTitle.value,
    description: () => pageDescription.value,
    canonicalPath: () => '/about',
    jsonLd: () => buildWebPageJsonLd({
      title: fullPageTitle.value,
      description: pageDescription.value,
      url: `${siteBase.value}/about`,
      locale: appStore.locale,
      siteName: siteName.value,
      siteBase: siteBase.value,
    }),
  })

  const aboutConfig = computed(() => appStore.config?.about || null)
  const contactConfig = computed(() => appStore.config?.contact || null)

  const resolveLocalizedText = (raw: unknown): string => {
    if (!raw || typeof raw !== 'object') {
      return ''
    }

    const record = raw as Record<string, unknown>
    const lang = String(locale.value || appStore.locale || 'zh-CN')
    const candidates = [record[lang], record['zh-CN'], record['zh-TW'], record['en-US']]

    for (const candidate of candidates) {
      if (typeof candidate === 'string' && candidate.trim() !== '') {
        return candidate.trim()
      }
    }

    return ''
  }

  const heroTitle = computed(() => resolveLocalizedText(aboutConfig.value?.hero?.title) || t('about.title'))
  const heroSubtitle = computed(() => resolveLocalizedText(aboutConfig.value?.hero?.subtitle) || t('about.subtitle'))
  const introductionText = computed(() => resolveLocalizedText(aboutConfig.value?.introduction))
  const servicesTitle = computed(() => resolveLocalizedText(aboutConfig.value?.services?.title))
  const contactTitle = computed(() => resolveLocalizedText(aboutConfig.value?.contact?.title))
  const contactText = computed(() => resolveLocalizedText(aboutConfig.value?.contact?.text))

  const serviceItems = computed(() => {
    const raw = aboutConfig.value?.services?.items
    if (!Array.isArray(raw)) {
      return []
    }

    return raw
      .map((item) => resolveLocalizedText(item))
      .filter((item) => item !== '')
  })

  const hasIntroduction = computed(() => introductionText.value !== '')
  const hasServices = computed(() => servicesTitle.value !== '' || serviceItems.value.length > 0)
  const hasContactLinks = computed(() => !!(contactConfig.value?.telegram || contactConfig.value?.whatsapp || contactConfig.value?.email || contactConfig.value?.support_url))
  const hasContact = computed(() => contactTitle.value !== '' || contactText.value !== '' || hasContactLinks.value)

  onMounted(async () => {
    if (!appStore.config) {
      await appStore.loadConfig()
    }
  })

  return {
    contactConfig,
    heroTitle,
    heroSubtitle,
    introductionText,
    servicesTitle,
    contactTitle,
    contactText,
    serviceItems,
    hasIntroduction,
    hasServices,
    hasContactLinks,
    hasContact,
  }
}
