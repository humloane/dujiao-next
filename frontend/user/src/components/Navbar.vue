<template>
  <nav
    class="fixed top-0 left-0 right-0 z-50 bg-card/80 border-b backdrop-blur-md transition-all"
    :class="scrolled ? 'py-2 shadow-lg' : 'py-4'"
    :style="{ transitionDuration: 'var(--ui-duration-normal)' }">
    <div class="container mx-auto px-4 flex items-center justify-between gap-4">
      <!-- Logo -->
      <router-link to="/" class="theme-wordmark group relative gap-3" :title="brandSiteName">
        <img
          v-if="brandLogo"
          :src="brandLogo"
          :alt="brandSiteName"
          class="h-8 max-w-[140px] shrink-0 object-contain md:max-w-[180px]"
        />
        <span class="theme-wordmark-text">{{ brandSiteName }}</span>
      </router-link>

      <!-- Desktop Menu -->
      <div class="hidden min-w-0 items-center space-x-1 overflow-x-auto scrollbar-hide md:flex">
        <template v-for="item in menuItems" :key="item.key">
          <Button v-if="item.type === 'route'" as-child variant="ghost" size="sm"
            class="gap-1.5 text-muted-foreground whitespace-nowrap shrink-0">
            <router-link :to="item.path" active-class="!text-primary !bg-primary/10">
              <component :is="item.icon" class="w-4 h-4 shrink-0 opacity-70" />
              <span>{{ item.label }}</span>
            </router-link>
          </Button>
          <Button v-else as-child variant="ghost" size="sm"
            class="gap-1.5 text-muted-foreground whitespace-nowrap shrink-0">
            <a :href="item.path" :target="item.target" rel="noopener noreferrer">
              <component :is="item.icon" class="w-4 h-4 shrink-0 opacity-70" />
              <span>{{ item.label }}</span>
            </a>
          </Button>
        </template>
      </div>

      <!-- Right Side Actions -->
      <div class="flex items-center shrink-0 space-x-2 lg:space-x-4">
        <!-- Cart (desktop only, mobile has bottom nav) -->
        <Button as-child variant="ghost" size="sm" class="relative hidden gap-2 text-muted-foreground md:flex">
          <router-link to="/cart">
            <ShoppingCart class="w-4 h-4 shrink-0" />
            <span class="text-xs font-medium">{{ t('navbar.cart') }}</span>
            <span v-if="cartCount > 0"
              class="absolute -top-1 -right-1 inline-flex items-center justify-center rounded-full px-1.5 py-0.5 text-[10px] font-bold leading-none min-w-[1.1rem] bg-primary text-primary-foreground"
              :class="{ 'theme-bounce-in': cartBounce }">
              {{ cartCount }}
            </span>
          </router-link>
        </Button>

        <Button v-if="!userAuthStore.isAuthenticated" as-child variant="ghost" size="sm"
          class="hidden gap-1.5 whitespace-nowrap text-muted-foreground md:inline-flex">
          <router-link to="/guest/orders">
            <ClipboardList class="w-4 h-4 shrink-0 opacity-70" />
            {{ t('navbar.guestOrders') }}
          </router-link>
        </Button>
        <Button v-if="!userAuthStore.isAuthenticated" as-child variant="ghost" size="sm"
          class="hidden gap-1.5 whitespace-nowrap text-muted-foreground md:inline-flex">
          <router-link to="/auth/login">
            <LogIn class="w-4 h-4 shrink-0 opacity-70" />
            {{ t('navbar.login') }}
          </router-link>
        </Button>
        <Button v-if="userAuthStore.isAuthenticated" as-child variant="ghost" size="sm"
          class="hidden gap-1.5 whitespace-nowrap text-muted-foreground md:inline-flex">
          <router-link to="/me">
            <User class="w-4 h-4 shrink-0 opacity-70" />
            {{ t('navbar.personalCenter') }}
          </router-link>
        </Button>
        <Button v-if="userAuthStore.isAuthenticated" variant="ghost" size="sm"
          class="hidden gap-1.5 whitespace-nowrap text-destructive hover:bg-destructive/10 hover:text-destructive md:inline-flex"
          @click="userAuthStore.logout()">
          <LogOut class="w-4 h-4 shrink-0 opacity-70" />
          {{ t('navbar.logout') }}
        </Button>
        <!-- Theme Switcher -->
        <Button variant="ghost" size="icon" class="text-muted-foreground" @click="toggleTheme">
          <Sun v-if="theme === 'dark'" class="w-4 h-4" />
          <Moon v-else class="w-4 h-4" />
        </Button>

        <!-- Language Switcher (Desktop) -->
        <Popover v-model:open="langOpen">
          <PopoverTrigger as-child>
            <Button variant="ghost" size="sm" class="hidden gap-2 text-muted-foreground md:inline-flex">
              <Languages class="w-4 h-4" />
              <span class="text-xs font-medium uppercase tracking-wider">{{ currentLocale }}</span>
            </Button>
          </PopoverTrigger>
          <PopoverContent align="end" class="w-40 p-2">
            <div class="px-2 pb-2 mb-2 border-b">
              <span class="text-xs text-muted-foreground font-mono px-2">{{ t('navbar.selectLanguage') }}</span>
            </div>
            <button v-for="lang in languages" :key="lang.code" @click="changeLanguage(lang.code)"
              class="w-full text-left px-3 py-2.5 text-sm rounded-md transition-colors flex items-center justify-between hover:bg-accent hover:text-accent-foreground"
              :class="{ 'text-primary': appStore.locale === lang.code }">
              {{ lang.name }}
              <span v-if="appStore.locale === lang.code" class="w-1.5 h-1.5 rounded-full bg-primary"></span>
            </button>
          </PopoverContent>
        </Popover>

        <!-- Mobile Menu Button (more menu, not main nav) -->
        <Button variant="ghost" size="icon" class="text-muted-foreground [&_svg]:size-5 md:hidden"
          @click="toggleMobileMenu">
          <EllipsisVertical />
        </Button>
      </div>
    </div>

  </nav>

  <!-- Teleport drawer outside nav to avoid backdrop-filter containing block bug -->
  <Teleport to="body">
    <!-- Mobile Drawer Overlay -->
    <Transition
      enter-active-class="transition duration-300 ease-out"
      enter-from-class="opacity-0"
      enter-to-class="opacity-100"
      leave-active-class="transition duration-200 ease-in"
      leave-from-class="opacity-100"
      leave-to-class="opacity-0">
      <div v-if="showMobileMenu" class="fixed inset-0 z-[60] bg-black/60 md:hidden" @click="showMobileMenu = false" style="overscroll-behavior: none;"></div>
    </Transition>

    <!-- Mobile Drawer: full-height panel containing every primary nav item -->
    <Transition
      enter-active-class="transition duration-300 ease-out"
      enter-from-class="translate-x-full"
      enter-to-class="translate-x-0"
      leave-active-class="transition duration-200 ease-in"
      leave-from-class="translate-x-0"
      leave-to-class="translate-x-full">
      <div v-if="showMobileMenu"
        ref="drawerRef"
        class="fixed right-0 top-0 bottom-0 z-[70] flex w-[88vw] max-w-[22rem] flex-col overflow-y-auto border-l bg-card md:hidden"
        role="dialog"
        aria-modal="true"
        :aria-label="t('navbar.menu')"
        style="overscroll-behavior: contain;">
        <div class="flex items-center justify-between gap-3 border-b px-4 py-3">
          <span class="min-w-0 truncate text-base font-bold">{{ brandSiteName }}</span>
          <button
            type="button"
            class="inline-flex h-11 w-11 shrink-0 items-center justify-center rounded-xl text-muted-foreground hover:bg-secondary"
            :aria-label="t('navbar.closeMenu')"
            @click="showMobileMenu = false"
          >
            <X class="h-5 w-5" />
          </button>
        </div>

        <nav class="flex-1">
          <template v-for="item in mobileDrawerItems" :key="item.key">
            <router-link
              v-if="item.type === 'route'"
              :to="item.path"
              class="flex min-h-14 items-center gap-3 border-b px-4 text-base text-foreground"
              active-class="!text-primary !font-semibold"
              @click="showMobileMenu = false"
            >
              <component :is="item.icon" class="h-5 w-5 shrink-0 opacity-70" />
              <span class="min-w-0 truncate">{{ item.label }}</span>
            </router-link>
            <a
              v-else
              :href="item.path"
              :target="item.target"
              rel="noopener noreferrer"
              class="flex min-h-14 items-center gap-3 border-b px-4 text-base text-foreground"
              @click="showMobileMenu = false"
            >
              <component :is="item.icon" class="h-5 w-5 shrink-0 opacity-70" />
              <span class="min-w-0 truncate">{{ item.label }}</span>
            </a>
          </template>

          <router-link
            v-if="!userAuthStore.isAuthenticated"
            to="/guest/orders"
            class="flex min-h-14 items-center gap-3 border-b px-4 text-base text-foreground"
            active-class="!text-primary !font-semibold"
            @click="showMobileMenu = false"
          >
            <ClipboardList class="h-5 w-5 shrink-0 opacity-70" />
            {{ t('navbar.guestOrders') }}
          </router-link>

          <a
            v-if="telegramUrl"
            :href="telegramUrl"
            target="_blank"
            rel="noopener noreferrer"
            class="flex min-h-14 items-center gap-3 border-b px-4 text-base text-foreground"
            @click="showMobileMenu = false"
          >
            <Send class="h-5 w-5 shrink-0 opacity-70" />
            {{ t('navbar.telegram') }}
          </a>

          <button
            v-if="userAuthStore.isAuthenticated"
            type="button"
            class="flex min-h-14 w-full items-center gap-3 border-b px-4 text-left text-base text-destructive"
            @click="userAuthStore.logout(); showMobileMenu = false"
          >
            <LogOut class="h-5 w-5 shrink-0 opacity-70" />
            {{ t('navbar.logout') }}
          </button>
        </nav>

        <div class="border-t px-4 py-4">
          <span class="text-[13px] font-semibold uppercase tracking-wider text-muted-foreground">{{ t('navbar.selectLanguage') }}</span>
          <div class="mt-2 grid gap-1">
            <button
              v-for="lang in languages"
              :key="lang.code"
              type="button"
              class="flex min-h-12 items-center justify-between rounded-xl px-3 text-base"
              :class="appStore.locale === lang.code ? 'font-semibold text-primary' : 'text-muted-foreground hover:text-foreground'"
              @click="changeLanguage(lang.code)"
            >
              {{ lang.name }}
              <span v-if="appStore.locale === lang.code" class="h-1.5 w-1.5 rounded-full bg-primary"></span>
            </button>
          </div>
        </div>
      </div>
    </Transition>
  </Teleport>
</template>

<script setup lang="ts">
import { ref, computed, onMounted, onUnmounted, watch, nextTick } from 'vue'
import { useI18n } from 'vue-i18n'
import { useAppStore } from '../stores/app'
import { useCartStore } from '../stores/cart'
import { useUserAuthStore } from '../stores/userAuth'
import { useTheme } from '../utils/theme'
import { getImageUrl } from '../utils/image'
import { useNavConfig } from '../composables/useNavConfig'
import {
  Sun, Moon, ShoppingCart, ClipboardList, LogIn, User, LogOut, Languages,
  EllipsisVertical, X, Send,
} from 'lucide-vue-next'
import { Button } from '@/components/ui/button'
import { Popover, PopoverContent, PopoverTrigger } from '@/components/ui/popover'

const { t } = useI18n()
const appStore = useAppStore()
const cartStore = useCartStore()
const userAuthStore = useUserAuthStore()
const { theme, toggleTheme } = useTheme()
const { primaryNavItems } = useNavConfig()

const showMobileMenu = ref(false)
const langOpen = ref(false)
const scrolled = ref(false)
const cartBounce = ref(false)
const drawerRef = ref<HTMLElement | null>(null)
let previousBodyOverflow = ''

const telegramUrl = computed(() => String(appStore.config?.contact?.telegram || '').trim())

const menuItems = primaryNavItems

// <768px: the drawer carries every primary nav item, not just the "secondary" ones.
const mobileDrawerItems = primaryNavItems

const languages = [
  { code: 'zh-CN', name: '简体中文' },
  { code: 'zh-TW', name: '繁體中文' },
  { code: 'en-US', name: 'English' },
]

const currentLocale = computed(() => {
  const lang = languages.find(l => l.code === appStore.locale)
  if (!lang) return 'CN'
  return lang.code === 'en-US' ? 'EN' : (lang.code === 'zh-CN' ? '简' : '繁')
})

const cartCount = computed(() => cartStore.totalItems)

const brandSiteName = computed(() => {
  const text = String(appStore.config?.brand?.site_name || '').trim()
  return text !== '' ? text : 'Dujiao-Next'
})

const brandLogo = computed(() => {
  const raw = String(appStore.config?.brand?.site_logo || '').trim()
  return raw ? getImageUrl(raw) : ''
})

const toggleMobileMenu = () => {
  showMobileMenu.value = !showMobileMenu.value
}

const focusableSelector = 'a[href], button:not([disabled]), input:not([disabled]), [tabindex]:not([tabindex="-1"])'

const handleDrawerKeydown = (event: KeyboardEvent) => {
  if (!showMobileMenu.value) return
  if (event.key === 'Escape') {
    event.preventDefault()
    showMobileMenu.value = false
    return
  }
  if (event.key !== 'Tab' || !drawerRef.value) return
  const items = Array.from(drawerRef.value.querySelectorAll<HTMLElement>(focusableSelector))
  if (items.length === 0) return
  const first = items[0] as HTMLElement
  const last = items[items.length - 1] as HTMLElement
  if (event.shiftKey && document.activeElement === first) {
    event.preventDefault()
    last.focus()
  } else if (!event.shiftKey && document.activeElement === last) {
    event.preventDefault()
    first.focus()
  }
}

watch(showMobileMenu, async (open) => {
  if (typeof document === 'undefined') return
  if (open) {
    previousBodyOverflow = document.body.style.overflow
    document.body.style.overflow = 'hidden'
    document.addEventListener('keydown', handleDrawerKeydown)
    await nextTick()
    drawerRef.value?.querySelector<HTMLElement>(focusableSelector)?.focus()
    return
  }
  document.body.style.overflow = previousBodyOverflow
  document.removeEventListener('keydown', handleDrawerKeydown)
})

const changeLanguage = (langCode: string) => {
  appStore.setLocale(langCode)
  langOpen.value = false
}

const handleScroll = () => {
  scrolled.value = window.scrollY > 20
}

// Cart badge bounce animation on count change
watch(cartCount, (newVal, oldVal) => {
  if (newVal > oldVal) {
    cartBounce.value = true
    setTimeout(() => { cartBounce.value = false }, 400)
  }
})

onMounted(() => {
  window.addEventListener('scroll', handleScroll, { passive: true })
})

onUnmounted(() => {
  window.removeEventListener('scroll', handleScroll)
  document.removeEventListener('keydown', handleDrawerKeydown)
  document.body.style.overflow = previousBodyOverflow
})
</script>

<style scoped>
.scrollbar-hide {
  -ms-overflow-style: none;
  scrollbar-width: none;
}
.scrollbar-hide::-webkit-scrollbar {
  display: none;
}
</style>
