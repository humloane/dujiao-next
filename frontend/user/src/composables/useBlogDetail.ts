import { ref, onMounted, onUnmounted, computed } from 'vue'
import { useRoute } from 'vue-router'
import { useI18n } from 'vue-i18n'
import { useAppStore } from '../stores/app'
import { postAPI } from '../api'
import { debounceAsync } from '../utils/debounce'
import { useLocalized } from './useProduct'
import { usePageSeo } from './usePageSeo'

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
  const relatedProducts = computed<any[]>(() => post.value?.related_products || [])

  const getLocalizedText = (jsonData: any) => {
    if (!jsonData) return ''
    const locale = appStore.locale
    return jsonData[locale] || jsonData['zh-CN'] || jsonData['en-US'] || ''
  }

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
    title: () => post.value ? getLocalizedText(post.value.title) : '',
    description: () => post.value ? getLocalizedText(post.value.summary) : '',
    image: () => post.value?.thumbnail || '',
    canonicalPath: () => `/blog/${(route.params.slug as string) || ''}`,
    type: () => 'article',
    jsonLd: () => {
      if (!post.value) return undefined
      const base = String(appStore.config?.brand?.site_url || '').replace(/\/+$/, '')
        || (typeof window !== 'undefined' ? window.location.origin : '')
      const canonical = `${base}/blog/${post.value.slug || route.params.slug || ''}`
      const title = getLocalizedText(post.value.title)
      const siteName = String(appStore.config?.brand?.site_name || 'Dujiao-Next')
      const published = post.value.published_at || post.value.created_at
      const article: Record<string, unknown> = {
        '@context': 'https://schema.org',
        '@type': 'Article',
        headline: title,
        description: getLocalizedText(post.value.summary) || '',
        url: canonical,
        mainEntityOfPage: canonical,
        datePublished: published || undefined,
        dateModified: post.value.updated_at || published || undefined,
        author: { '@type': 'Organization', name: siteName },
        publisher: { '@type': 'Organization', name: siteName },
      }
      if (post.value.thumbnail) article.image = [post.value.thumbnail]
      return [
        article,
        {
          '@context': 'https://schema.org',
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
    loading.value = true
    try {
      const slug = route.params.slug as string
      const response = await postAPI.detail(slug)
      post.value = response.data.data || null
    } catch (error) {
      console.error('Failed to load post:', error)
      post.value = null
    } finally {
      loading.value = false
    }
  }

  const debouncedLoadPost = debounceAsync(loadPost, 300)

  onMounted(() => {
    loadPost()
  })

  onUnmounted(() => {
    debouncedLoadPost.cancel()
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
