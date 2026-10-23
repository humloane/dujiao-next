import { ref, watch } from 'vue'

const TEA_THEME_KEY = 'dujiao_theme'

type Theme = 'light' | 'dark'

// 首屏内联脚本已按 localStorage / 系统偏好写好 html.dark，这里只复用结果，
// 避免应用启动时再次判断主题造成骨架与应用配色不一致。
const initial: Theme = document.documentElement.classList.contains('dark') ? 'dark' : 'light'
const theme = ref<Theme>(initial)

const applyTheme = (newTheme: Theme) => {
    const root = document.documentElement
    if (newTheme === 'dark') {
        root.classList.add('dark')
    } else {
        root.classList.remove('dark')
    }
    localStorage.setItem(TEA_THEME_KEY, newTheme)
}

// Watch for changes and apply
watch(theme, (newVal) => {
    applyTheme(newVal)
}, { immediate: true })

export const useTheme = () => {
    const toggleTheme = () => {
        theme.value = theme.value === 'dark' ? 'light' : 'dark'
    }

    return {
        theme,
        toggleTheme
    }
}
