import { ref, onMounted, onUnmounted, watch, computed } from 'vue'
import { useRoute, useRouter } from 'vue-router'
import { useAppStore } from '../stores/app'
import { postAPI } from '../api'
import { debounceAsync } from '../utils/debounce'
import { usePageSeo } from './usePageSeo'
import { buildCollectionPageJsonLd } from '../utils/seoJsonLd'
import { joinSeoTitle } from '../utils/seoTitle'

/**
 * 文章/公告列表共享逻辑（Blog/Notice，classic + vault 双模板共用）。
 * 完整保留原 views/Blog.vue 与 Notice.vue 的行为，仅抽离为 composable。
 * Blog 启用搜索；Notice 无搜索框（searchKeyword 恒为空，watch 永不触发，行为一致）。
 */
export function usePostList(
  type: 'blog' | 'notice',
  seo: {
    title: () => string
    canonicalPath: string
    description?: () => string
    keywords?: () => string
  },
) {
  const router = useRouter()
  const route = useRoute()
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
  const pageTitle = computed(() => seo.title())
  const fullPageTitle = computed(() => joinSeoTitle(pageTitle.value, siteSeoName.value, [siteName.value]))
  const pageDescription = computed(() => seo.description?.() || '')
  const pageUrl = computed(() => `${siteBase.value}${seo.canonicalPath}`)

  usePageSeo({
    title: () => pageTitle.value,
    description: () => pageDescription.value,
    keywords: seo.keywords,
    canonicalPath: () => seo.canonicalPath,
    jsonLd: () => buildCollectionPageJsonLd({
      title: fullPageTitle.value,
      description: pageDescription.value,
      url: pageUrl.value,
      locale: appStore.locale,
      siteName: siteName.value,
      siteBase: siteBase.value,
      breadcrumbName: seo.canonicalPath.replace(/^\/+/, ''),
    }),
  })

  const loading = ref(true)
  const posts = ref<any[]>([])
  const currentPage = ref(1)
  const pageSize = ref(12)
  const total = ref(0)
  const totalPages = ref(0)
  const searchKeyword = ref('')
  const categoryID = ref<number | null>(null)
  const categories = ref<Array<{ id: number; parent_id?: number; slug?: string; name?: unknown; icon?: string }>>([])

  const readCategoryFromRoute = () => {
    const parsed = Number(route.query.category_id)
    categoryID.value = Number.isFinite(parsed) && parsed > 0 ? parsed : null
  }
  readCategoryFromRoute()

  const loadCategories = async () => {
    if (type !== 'blog') return
    try {
      const response = await postAPI.categories()
      categories.value = Array.isArray(response.data.data) ? response.data.data : []
    } catch {
      // 分类失败不阻塞文章列表，也不弹错误提示。
      categories.value = []
    }
  }

  const selectCategory = (id: number | null) => {
    const next = id && id > 0 ? id : null
    const currentParam = String(route.query.category_id || '')
    if (next === categoryID.value && currentParam === (next ? String(next) : '')) return
    categoryID.value = next
    const query = { ...route.query }
    if (next) {
      query.category_id = String(next)
    } else {
      delete query.category_id
    }
    router.replace({ query })
  }

  const getLocalizedText = (jsonData: any) => {
    if (!jsonData) return ''
    const locale = appStore.locale
    return jsonData[locale] || jsonData['zh-CN'] || jsonData['en-US'] || ''
  }

  // 列表卡片优先展示文章级 SEO 文案，缺失时回退正文标题/摘要。
  const getPostTitle = (post: any) => getLocalizedText(post?.seo_meta?.title) || getLocalizedText(post?.title)
  const getPostSummary = (post: any) => getLocalizedText(post?.seo_meta?.description) || getLocalizedText(post?.summary)

  const formatDate = (dateString: string) => {
    if (!dateString) return ''
    const date = new Date(dateString)
    return date.toLocaleDateString(appStore.locale, {
      year: 'numeric',
      month: 'long',
      day: 'numeric',
    })
  }

  const loadPosts = async () => {
    loading.value = true
    try {
      const params: Record<string, any> = {
        type,
        page: currentPage.value,
        page_size: pageSize.value,
      }
      const keyword = searchKeyword.value.trim()
      if (keyword) {
        params.search = keyword
      }
      if (categoryID.value) {
        params.category_id = categoryID.value
      }
      const response = await postAPI.list(params)
      posts.value = response.data.data || []
      if (response.data.pagination) {
        total.value = response.data.pagination.total || 0
        totalPages.value = response.data.pagination.total_page || 0
      }
    } catch (error) {
      console.error('Failed to load posts:', error)
    } finally {
      loading.value = false
    }
  }

  const debouncedLoadPosts = debounceAsync(loadPosts, 300)

  const goToPost = (slug: string) => {
    router.push(`/blog/${slug}`)
  }

  const changePage = (page: number) => {
    if (page < 1 || page > totalPages.value) return
    currentPage.value = page
    debouncedLoadPosts()
    window.scrollTo({ top: 0, behavior: 'smooth' })
  }

  watch(searchKeyword, () => {
    currentPage.value = 1
    debouncedLoadPosts()
  })

  watch(categoryID, () => {
    currentPage.value = 1
    debouncedLoadPosts()
  })

  watch(() => route.query.category_id, readCategoryFromRoute)

  onMounted(() => {
    void loadCategories()
    void loadPosts()
  })

  onUnmounted(() => {
    debouncedLoadPosts.cancel()
  })

  return {
    loading,
    posts,
    currentPage,
    totalPages,
    total,
    searchKeyword,
    categoryID,
    categories,
    selectCategory,
    getLocalizedText,
    getPostTitle,
    getPostSummary,
    formatDate,
    goToPost,
    changePage,
  }
}
