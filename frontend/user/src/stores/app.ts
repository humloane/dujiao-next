import { defineStore } from 'pinia'
import { computed, ref } from 'vue'
import { configAPI } from '../api'
import { applyCustomScripts } from '../utils/customScripts'
import { getImageUrl } from '../utils/image'
import { getLocalizedText } from '../utils/resellerSiteConfig'
import { detectLocale, setI18nLocale } from '../i18n'
import { useHead } from '@unhead/vue'

export const useAppStore = defineStore('app', () => {
    // 与 vue-i18n 复用同一套语言检测逻辑，避免首次访问时
    // UI 语言（vue-i18n）与商品多语言取值（appStore.locale）不一致
    const locale = ref(detectLocale())
    const config = ref<any>(null)
    const loading = ref(false)
    // 配置请求已结束（成功、失败或套用本地快照），供首屏骨架判断何时接管。
    const configSettled = ref(false)

    const CONFIG_CACHE_KEY = 'dj-app-config-snapshot'
    const CONFIG_CACHE_TTL_MS = 10 * 60 * 1000
    const CONFIG_CACHE_VERSION = 1

    type ConfigSnapshot = {
        data: any
        savedAt: number
        version: number
    }

    const readConfigSnapshot = (): ConfigSnapshot | null => {
        if (typeof localStorage === 'undefined') return null
        try {
            const raw = localStorage.getItem(CONFIG_CACHE_KEY)
            if (!raw) return null
            const parsed = JSON.parse(raw) as ConfigSnapshot
            if (!parsed || parsed.version !== CONFIG_CACHE_VERSION || !parsed.data) return null
            const age = Date.now() - Number(parsed.savedAt)
            if (!Number.isFinite(age) || age < 0 || age > CONFIG_CACHE_TTL_MS) return null
            return parsed
        } catch {
            return null
        }
    }

    const writeConfigSnapshot = (data: any) => {
        if (typeof localStorage === 'undefined') return
        try {
            localStorage.setItem(CONFIG_CACHE_KEY, JSON.stringify({
                data,
                savedAt: Date.now(),
                version: CONFIG_CACHE_VERSION,
            } satisfies ConfigSnapshot))
        } catch {
            // 存储配额不足或隐私模式禁用 localStorage 时忽略缓存。
        }
    }

    const applyConfigSnapshot = (): boolean => {
        if (config.value) {
            configSettled.value = true
            return true
        }
        const snapshot = readConfigSnapshot()
        if (!snapshot) return false
        config.value = snapshot.data
        if (snapshot.data?.server_time) {
            serverTimeOffset.value = snapshot.data.server_time - snapshot.savedAt
        }
        configSettled.value = true
        applySEO()
        applyCustomScripts(snapshot.data?.scripts)
        return true
    }
    // 服务器与客户端的时间偏移量（毫秒），serverTime = clientTime + offset
    const serverTimeOffset = ref(0)
    const siteIconHref = computed(() => {
        const siteIcon = String(config.value?.brand?.site_icon || '').trim()
        return siteIcon ? getImageUrl(siteIcon) : '/dj.svg'
    })
    const isResellerTenant = computed(() => {
        return String(config.value?.tenant?.mode || '').trim().toLowerCase() === 'reseller'
    })
    const canAccessResellerConsole = computed(() => !!config.value && !isResellerTenant.value)

    // 设置语言：同时驱动 vue-i18n（内部按需加载语言包）
    const setLocale = (newLocale: string) => {
        locale.value = newLocale
        localStorage.setItem('locale', newLocale)
        void setI18nLocale(newLocale)
    }

    // 全局 head 默认值（标题/描述/keywords/favicon/html lang）。
    // canonical / og / twitter 等页面级别字段交由各页面通过 usePageSeo 接管，
    // 避免与页面级 useHead 冲突或产生重复标签。
    useHead({
        htmlAttrs: { lang: computed(() => locale.value) },
        title: () => {
            const seo = config.value?.seo
            const lang = locale.value
            const localized = getLocalizedText(seo?.title, lang)
            if (localized) return String(localized).trim() || undefined
            const siteName = String(config.value?.brand?.site_name || '').trim()
            return siteName || undefined
        },
        link: () => [{ key: 'favicon', rel: 'icon', href: siteIconHref.value }],
        meta: () => {
            const seo = config.value?.seo
            if (!seo) return []
            const lang = locale.value
            const tags: Array<{ name?: string; property?: string; content: string }> = []
            const keywords = getLocalizedText(seo.keywords, lang)
            if (keywords) {
                tags.push({ name: 'keywords', content: keywords })
            }
            const description = getLocalizedText(seo.description, lang)
            if (description) {
                tags.push({ name: 'description', content: description })
            }
            return tags
        }
    })

    // 更新SEO信息 (向后兼容的方法)
    const applySEO = () => {
        // 由于 useHead 已经是响应式的，这里不再需要显式调用
        // 留空函数以防其他组件出错
    }

    // 加载全局配置
    // 同一时间只允许一个非强制请求：路由守卫和页面 onMounted 都会调用，
    // 共享同一个 Promise，避免首屏发出两份 /public/config。
    let configRequest: Promise<void> | null = null

    const loadConfig = (force = false): Promise<void> => {
        if (config.value && !force) {
            configSettled.value = true
            applySEO()
            applyCustomScripts(config.value?.scripts)
            return Promise.resolve()
        }
        if (!force && configRequest) {
            return configRequest
        }
        if (!force) loading.value = true

        const request = (async () => {
            try {
                const requestTime = Date.now()
                const response = await configAPI.get()
                const nextConfig = response.data.data
                // 快照已经提供给首屏使用时，只有内容真的变化才替换响应式对象。
                const configChanged = !config.value || JSON.stringify(config.value) !== JSON.stringify(nextConfig)
                if (configChanged) {
                    config.value = nextConfig
                }
                if (nextConfig?.server_time) {
                    const responseTime = Date.now()
                    const roundTripTime = responseTime - requestTime
                    const estimatedServerNow = nextConfig.server_time + roundTripTime / 2
                    serverTimeOffset.value = estimatedServerNow - responseTime
                }
                writeConfigSnapshot(nextConfig)
                applySEO()
                applyCustomScripts(nextConfig?.scripts)
                // Print version to console (dev only)
                if (import.meta.env.DEV && nextConfig?.app_version) {
                    console.log(
                        '%c Version %c ' + nextConfig.app_version + ' %c',
                        'background:#34c759;color:#fff;padding:2px 6px;border-radius:4px 0 0 4px;font-weight:bold;',
                        'background:#1d1d1f;color:#f5f5f7;padding:2px 6px;border-radius:0 4px 4px 0;',
                        'background:transparent;',
                    )
                }
            } catch (error) {
                console.error('Failed to load config:', error)
            } finally {
                if (!force) loading.value = false
                configSettled.value = true
            }
        })()

        if (!force) {
            let wrapped: Promise<void>
            wrapped = request.finally(() => {
                if (configRequest === wrapped) configRequest = null
            })
            configRequest = wrapped
            return wrapped
        }
        return request
    }

    // 获取校正后的服务器当前时间（毫秒时间戳）
    const getServerTime = () => Date.now() + serverTimeOffset.value

    // 获取校正后的服务器当前 Date 对象
    const getServerDate = () => new Date(getServerTime())

    return {
        locale,
        config,
        loading,
        configSettled,
        serverTimeOffset,
        isResellerTenant,
        canAccessResellerConsole,
        setLocale,
        loadConfig,
        applyConfigSnapshot,
        applySEO,
        getServerTime,
        getServerDate,
    }
})
