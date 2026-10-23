import { ref, onMounted, onUnmounted, computed, watch } from 'vue'
import { useRoute } from 'vue-router'
import { useI18n } from 'vue-i18n'
import { useAppStore } from '../stores/app'
import { postAPI } from '../api'
import { useLocalized } from './useProduct'
import { usePageSeo } from './usePageSeo'
import { absoluteAssetUrl } from '../utils/seoJsonLd'

/**
 * 文章/公告详情共享逻辑（classic + vault 双模板共用）。
 * 完整保留原 views/BlogDetail.vue 的行为，仅抽离为 composable。
 */
export function useBlogDetail() {
  const route = useRoute()
  const { t } = useI18n()
  const appStore = useAppStore()
  const { formatPrice } = useLocalized()

  const loading = ref(true)
  const post = ref<any>(null)
  let postRequest: AbortController | null = null
  const relatedProducts = computed<any[]>(() => post.value?.related_products || [])

  const getLocalizedText = (jsonData: any) => {
    if (!jsonData) return ''
    const locale = appStore.locale
    return jsonData[locale] || jsonData['zh-CN'] || jsonData['en-US'] || ''
  }

  const postSeoMeta = computed(() => post.value?.seo_meta || {})
  const seoTitle = computed(() => post.value
    ? (getLocalizedText(postSeoMeta.value.title) || getLocalizedText(post.value.title))
    : '')
  const seoDescription = computed(() => post.value
    ? (getLocalizedText(postSeoMeta.value.description) || getLocalizedText(post.value.summary))
    : '')
  const seoKeywords = computed(() => getLocalizedText(postSeoMeta.value.keywords))

  const formatDate = (dateString: string) => {
    if (!dateString) return ''
    const date = new Date(dateString)
    return date.toLocaleDateString(appStore.locale, {
      year: 'numeric',
      month: 'long',
      day: 'numeric',
    })
  }

  usePageSeo({
    title: () => seoTitle.value,
    description: () => seoDescription.value,
    keywords: () => seoKeywords.value,
    image: () => post.value?.thumbnail || '',
    canonicalPath: () => `/blog/${(route.params.slug as string) || ''}`,
    type: () => 'article',
    noindex: () => !loading.value && !post.value,
    jsonLd: () => {
      if (!post.value) return undefined
      const base = String(appStore.config?.brand?.site_url || '').replace(/\/+$/, '')
        || (typeof window !== 'undefined' ? window.location.origin : '')
      const canonical = `${base}/blog/${post.value.slug || route.params.slug || ''}`
      const title = seoTitle.value
      const siteName = String(appStore.config?.brand?.site_name || 'Dujiao-Next')
      const published = post.value.published_at || post.value.created_at
      const article: Record<string, unknown> = {
        '@type': 'Article',
        headline: title,
        description: seoDescription.value || '',
        url: canonical,
        mainEntityOfPage: canonical,
        datePublished: published || undefined,
        dateModified: published || undefined,
        author: { '@type': 'Organization', name: siteName },
        publisher: { '@type': 'Organization', name: siteName },
      }
      if (post.value.thumbnail) article.image = [absoluteAssetUrl(post.value.thumbnail)]
      return [
        article,
        {
          '@type': 'BreadcrumbList',
          itemListElement: [
            { '@type': 'ListItem', position: 1, name: siteName, item: base + '/' },
            { '@type': 'ListItem', position: 2, name: title, item: canonical },
          ],
        },
      ]
    },
  })

  const backLink = computed(() => {
    if (!post.value) return '/blog'
    return post.value.type === 'notice' ? '/notice' : '/blog'
  })

  const backText = computed(() => {
    if (!post.value) return t('blogDetail.backToBlog')
    return post.value.type === 'notice' ? t('blogDetail.backToNotice') : t('blogDetail.backToBlog')
  })

  const loadPost = async () => {
    postRequest?.abort()
    const controller = new AbortController()
    postRequest = controller
    loading.value = true
    post.value = null
    try {
      const slug = route.params.slug as string
      const response = await postAPI.detail(slug, { signal: controller.signal })
      if (controller.signal.aborted) return
      post.value = response.data.data || null
    } catch (error: any) {
      if (controller.signal.aborted || error?.name === 'AbortError') return
      console.error('Failed to load post:', error)
      post.value = null
    } finally {
      if (postRequest === controller) {
        postRequest = null
        loading.value = false
      }
    }
  }

  onMounted(() => {
    loadPost()
  })

  watch(
    () => route.params.slug,
    (slug, previous) => {
      if (!slug || slug === previous) return
      void loadPost()
    },
  )

  onUnmounted(() => {
    postRequest?.abort()
    postRequest = null
  })

  return {
    loading,
    post,
    relatedProducts,
    getLocalizedText,
    formatDate,
    formatPrice,
    backLink,
    backText,
  }
}
