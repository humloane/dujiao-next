import { computed, reactive, ref, watch } from 'vue'
import { useRouter, useRoute } from 'vue-router'
import { productAPI, categoryAPI } from '../api'
import { buildCategoryGroups, createCategoryMap, normalizeCategoryParentId, type PublicCategory } from '../utils/category'
import { debounceAsync } from '../utils/debounce'
import {
  emptyProductMetaFilter,
  emptyProductMetaOptions,
  hasProductMetaFilter,
  normalizeOptionCodes,
  type ProductMetaFilter,
  type ProductMetaOptions,
} from '../constants/productMeta'

export interface UseProductListOptions {
  pageSize?: number
  homeRouteName?: string
  categoryRouteName?: string
  /** 是否拉取后端聚合的筛选取值；只有真正渲染筛选器的页面需要。 */
  withMetaOptions?: boolean
}

export function useProductList(options: UseProductListOptions = {}) {
  const {
    pageSize: defaultPageSize = 20,
    homeRouteName = 'home',
    categoryRouteName = 'category-products',
    withMetaOptions = false,
  } = options

  const router = useRouter()
  const route = useRoute()

  const loading = ref(true)
  const products = ref<any[]>([])
  const totalCount = ref(0)
  // 后端按在售商品聚合的筛选取值；请求失败时回落到常量。
  const metaOptions = reactive<ProductMetaOptions>(emptyProductMetaOptions())
  const categories = ref<PublicCategory[]>([])
  const selectedCategory = ref<number | null>(null)
  const searchQuery = ref('')
  const currentPage = ref(1)
  const pageSize = ref(defaultPageSize)
  const totalPages = ref(0)
  const showFilterDrawer = ref(false)
  const expandedParentIds = ref<number[]>([])
  const metaFilter = reactive<ProductMetaFilter>(emptyProductMetaFilter())

  const categoryGroups = computed(() => buildCategoryGroups(categories.value))
  const categoryMap = computed(() => createCategoryMap(categories.value))
  const selectedCategoryData = computed(() => {
    if (!selectedCategory.value) return undefined
    return categoryMap.value.get(selectedCategory.value)
  })
  const hasMetaFilters = computed(() => hasProductMetaFilter(metaFilter))

  let initializing = true
  let productsAbort: AbortController | null = null

  const isParentExpanded = (categoryId: number) => {
    return expandedParentIds.value.includes(categoryId)
  }

  const expandParentCategory = (categoryId: number) => {
    if (!categoryId || isParentExpanded(categoryId)) return
    expandedParentIds.value = [...expandedParentIds.value, categoryId]
  }

  const toggleParentCategory = (categoryId: number) => {
    if (isParentExpanded(categoryId)) {
      expandedParentIds.value = expandedParentIds.value.filter((id) => id !== categoryId)
      return
    }
    expandParentCategory(categoryId)
  }

  const getParentToggleButtonClass = (categoryId: number) => {
    return isParentExpanded(categoryId)
      ? 'bg-primary text-primary-foreground border-transparent'
      : 'bg-card/70 text-muted-foreground hover:text-foreground'
  }

  const syncExpandedCategoryState = () => {
    if (!selectedCategory.value) return

    const matched = categoryMap.value.get(selectedCategory.value)
    if (!matched) return

    const parentId = normalizeCategoryParentId(matched.parent_id)
    if (parentId > 0) {
      expandParentCategory(parentId)
      return
    }

    const selectedGroup = categoryGroups.value.find((group) => group.id === matched.id)
    if (selectedGroup?.children.length) {
      expandParentCategory(selectedGroup.id)
    }
  }

  const selectCategory = (categoryId: number | null, closeDrawer = false) => {
    selectedCategory.value = categoryId
    if (closeDrawer) {
      showFilterDrawer.value = false
    }
  }

  const loadProducts = async () => {
    // 快速连续切换筛选时，旧请求必须被取消，否则慢响应会覆盖新结果。
    productsAbort?.abort()
    const controller = new AbortController()
    productsAbort = controller
    loading.value = true
    try {
      const params: any = {
        page: currentPage.value,
        page_size: pageSize.value,
      }
      if (selectedCategory.value) {
        params.category_id = selectedCategory.value
      }
      const keyword = searchQuery.value.trim()
      if (keyword) {
        params.search = keyword
      }
      if (metaFilter.country) params.country = metaFilter.country
      if (metaFilter.scenario) params.scenario = metaFilter.scenario
      if (metaFilter.ipType) params.ip_type = metaFilter.ipType
      if (metaFilter.lineType) params.line = metaFilter.lineType
      if (metaFilter.sort && metaFilter.sort !== 'recommended') params.sort = metaFilter.sort
      const response = await productAPI.list(params, { signal: controller.signal })
      if (controller.signal.aborted) return
      products.value = response.data.data || []
      totalCount.value = response.data.pagination?.total || 0
      if (response.data.pagination) {
        totalPages.value = response.data.pagination.total_page || 0
      }
    } catch (error: any) {
      // 主动取消不是错误：既不打日志，也不改动已有列表。
      if (error?.name === 'AbortError') return
      console.error('Failed to load products:', error)
    } finally {
      if (productsAbort === controller) {
        productsAbort = null
        loading.value = false
      }
    }
  }

  const loadCategories = async () => {
    try {
      const response = await categoryAPI.list()
      categories.value = response.data.data || []
    } catch (error) {
      console.error('Failed to load categories:', error)
    }
  }

  // 筛选取值来自后端商品数据：新增国家/地区不需要改前端常量。
  const loadMetaOptions = async () => {
    try {
      const response = await productAPI.filterOptions()
      const data = response.data.data || {}
      metaOptions.countries = normalizeOptionCodes(data.countries)
      metaOptions.scenarios = normalizeOptionCodes(data.scenarios)
      metaOptions.ipTypes = normalizeOptionCodes(data.ip_types)
      metaOptions.lineTypes = normalizeOptionCodes(data.line_types)
      metaOptions.loaded = true
    } catch (error) {
      console.error('Failed to load product filter options:', error)
    }
  }

  const debouncedLoadProducts = debounceAsync(loadProducts, 300)

  const changePage = (page: number) => {
    if (page < 1 || page > totalPages.value) return
    currentPage.value = page
    debouncedLoadProducts()
    window.scrollTo({ top: 0, behavior: 'smooth' })
  }

  const clearSearch = () => {
    if (!searchQuery.value) return
    searchQuery.value = ''
    currentPage.value = 1
    debouncedLoadProducts()
  }

  const setMetaFilter = (value: ProductMetaFilter) => {
    // 筛选是离散点击而非连续输入：立即请求，不再等 300ms 防抖。
    debouncedLoadProducts.cancel()
    Object.assign(metaFilter, value)
    currentPage.value = 1
    void loadProducts()
    if (!initializing) {
      syncMetaFilterToRoute()
    }
  }

  const clearMetaFilters = () => {
    if (!hasMetaFilters.value) return
    debouncedLoadProducts.cancel()
    Object.assign(metaFilter, emptyProductMetaFilter())
    currentPage.value = 1
    void loadProducts()
    if (!initializing) {
      syncMetaFilterToRoute()
    }
  }

  const syncMetaFilterToRoute = () => {
    const query = { ...route.query }
    const mapping: Array<[keyof ProductMetaFilter, string]> = [
      ['country', 'country'],
      ['scenario', 'scenario'],
      ['ipType', 'ip_type'],
      ['lineType', 'line'],
      ['sort', 'sort'],
    ]
    mapping.forEach(([key, queryKey]) => {
      const value = String(metaFilter[key] || '').trim()
      if (!value || (key === 'sort' && value === 'recommended')) {
        delete query[queryKey]
      } else {
        query[queryKey] = value
      }
    })
    router.replace({ query })
  }

  const applyInitialMetaFilter = () => {
    const read = (key: string) => {
      const value = route.query[key]
      return typeof value === 'string' ? value.trim() : ''
    }
    Object.assign(metaFilter, {
      country: read('country').toUpperCase(),
      scenario: read('scenario'),
      ipType: read('ip_type'),
      lineType: read('line'),
      sort: read('sort') || 'recommended',
    })
  }

  const onSearch = () => {
    currentPage.value = 1
    debouncedLoadProducts()
  }

  const syncSelectedCategoryFromRoute = () => {
    if (route.name !== categoryRouteName) {
      if (selectedCategory.value !== null) {
        selectedCategory.value = null
      }
      return false
    }

    const slugParam = route.params.slug as string | undefined
    if (!slugParam || categories.value.length === 0) return false

    const matched = categories.value.find((category) => category.slug === slugParam)
    if (!matched) return false

    if (selectedCategory.value !== matched.id) {
      selectedCategory.value = matched.id
    }

    return true
  }

  watch(selectedCategory, () => {
    if (initializing) return
    currentPage.value = 1
    syncExpandedCategoryState()
    debouncedLoadProducts()

    if (selectedCategory.value) {
      const matched = categories.value.find((category) => category.id === selectedCategory.value)
      if (matched?.slug && route.params.slug !== matched.slug) {
        router.replace({ name: categoryRouteName, params: { slug: matched.slug }, query: route.query })
      }
    } else if (route.name === categoryRouteName) {
      router.replace({ name: homeRouteName, query: route.query })
    }
  })

  watch(searchQuery, () => {
    if (initializing) return
    currentPage.value = 1
    debouncedLoadProducts()
  })

  watch(
    () => route.params.slug,
    () => {
      if (initializing) return
      if (categories.value.length === 0) return
      syncSelectedCategoryFromRoute()
    },
  )

  const initialize = async () => {
    applyInitialMetaFilter()
    const pending: Promise<unknown>[] = [loadCategories()]
    if (withMetaOptions) {
      pending.push(loadMetaOptions())
    }
    await Promise.all(pending)
    if (syncSelectedCategoryFromRoute()) {
      syncExpandedCategoryState()
    }
    await loadProducts()
    initializing = false
  }

  const cleanup = () => {
    debouncedLoadProducts.cancel()
    productsAbort?.abort()
    productsAbort = null
  }

  return {
    loading,
    products,
    totalCount,
    metaOptions,
    categories,
    selectedCategory,
    searchQuery,
    currentPage,
    pageSize,
    totalPages,
    showFilterDrawer,
    expandedParentIds,
    categoryGroups,
    categoryMap,
    selectedCategoryData,
    metaFilter,
    hasMetaFilters,
    setMetaFilter,
    clearMetaFilters,
    isParentExpanded,
    toggleParentCategory,
    getParentToggleButtonClass,
    selectCategory,
    loadProducts,
    loadCategories,
    changePage,
    clearSearch,
    onSearch,
    syncSelectedCategoryFromRoute,
    syncExpandedCategoryState,
    initialize,
    cleanup,
  }
}
