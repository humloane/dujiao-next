<template>
  <div class="mb-5">
    <!-- Phones: a single trigger opens a searchable bottom sheet (no native selects). -->
    <div class="flex items-stretch gap-2 md:hidden">
      <button
        type="button"
        class="inline-flex h-11 min-w-0 flex-1 items-center justify-between gap-2 rounded-xl border border-border bg-card px-4 text-base font-semibold text-foreground"
        :aria-expanded="sheetOpen"
        aria-haspopup="dialog"
        @click="openSheet"
      >
        <span class="flex min-w-0 items-center gap-2">
          <SlidersHorizontal class="h-4 w-4 shrink-0 text-muted-foreground" aria-hidden="true" />
          <span class="truncate">{{ t('productFilters.openFilters') }}</span>
        </span>
        <span
          v-if="activeCount > 0"
          class="shrink-0 rounded-full bg-primary/10 px-2 py-0.5 text-[13px] font-semibold text-primary"
        >
          {{ t('productFilters.activeCount', { count: activeCount }) }}
        </span>
        <ChevronDown v-else class="h-4 w-4 shrink-0 text-muted-foreground" aria-hidden="true" />
      </button>
      <button
        v-if="hasProductMetaFilter(modelValue)"
        type="button"
        class="inline-flex h-11 w-11 shrink-0 items-center justify-center rounded-xl border border-border bg-card text-muted-foreground"
        :aria-label="t('productFilters.reset')"
        @click="clear"
      >
        <X class="h-4 w-4" />
      </button>
    </div>

    <!-- Tablet / desktop: inline selects, 44px tall and 16px text. -->
    <div class="hidden flex-wrap items-center gap-3 rounded-2xl border bg-card/60 p-3 md:flex">
      <span class="shrink-0 text-xs font-bold uppercase tracking-wider text-muted-foreground">
        {{ t('productFilters.title') }}
      </span>

      <select
        :value="modelValue.country"
        class="h-11 min-w-[132px] flex-none rounded-xl border border-border bg-background px-3 text-base font-medium text-foreground outline-none focus:border-primary"
        :aria-label="t('productFilters.country')"
        @change="update('country', ($event.target as HTMLSelectElement).value)"
      >
        <option value="">{{ t('productFilters.allCountries') }}</option>
        <option v-for="code in PRODUCT_COUNTRY_CODES" :key="code" :value="code">{{ countryLabel(code) }}</option>
      </select>

      <select
        :value="modelValue.scenario"
        class="h-11 min-w-[132px] flex-none rounded-xl border border-border bg-background px-3 text-base font-medium text-foreground outline-none focus:border-primary"
        :aria-label="t('productFilters.scenario')"
        @change="update('scenario', ($event.target as HTMLSelectElement).value)"
      >
        <option value="">{{ t('productFilters.allScenarios') }}</option>
        <option v-for="value in PRODUCT_SCENARIOS" :key="value" :value="value">
          {{ t(`productFilters.scenarios.${value}`) }}
        </option>
      </select>

      <select
        :value="modelValue.ipType"
        class="h-11 min-w-[132px] flex-none rounded-xl border border-border bg-background px-3 text-base font-medium text-foreground outline-none focus:border-primary"
        :aria-label="t('productFilters.ipType')"
        @change="update('ipType', ($event.target as HTMLSelectElement).value)"
      >
        <option value="">{{ t('productFilters.allIpTypes') }}</option>
        <option v-for="value in PRODUCT_IP_TYPES" :key="value" :value="value">
          {{ t(`productFilters.ipTypes.${value}`) }}
        </option>
      </select>

      <select
        :value="modelValue.lineType"
        class="h-11 min-w-[132px] flex-none rounded-xl border border-border bg-background px-3 text-base font-medium text-foreground outline-none focus:border-primary"
        :aria-label="t('productFilters.lineType')"
        @change="update('lineType', ($event.target as HTMLSelectElement).value)"
      >
        <option value="">{{ t('productFilters.allLineTypes') }}</option>
        <option v-for="value in PRODUCT_LINE_TYPES" :key="value" :value="value">
          {{ t(`productFilters.lineTypes.${value}`) }}
        </option>
      </select>

      <select
        :value="modelValue.sort"
        class="h-11 min-w-[128px] flex-none rounded-xl border border-border bg-background px-3 text-base font-medium text-foreground outline-none focus:border-primary"
        :aria-label="t('productFilters.sort')"
        @change="update('sort', ($event.target as HTMLSelectElement).value)"
      >
        <option v-for="value in PRODUCT_SORTS" :key="value" :value="value">
          {{ t(`productFilters.sorts.${value}`) }}
        </option>
      </select>

      <button
        v-if="hasProductMetaFilter(modelValue)"
        type="button"
        class="ml-auto inline-flex h-11 items-center gap-1 rounded-xl px-3 text-base font-semibold text-muted-foreground transition-colors hover:bg-secondary hover:text-foreground"
        @click="clear"
      >
        <X class="h-4 w-4" />
        {{ t('products.clearFilters') }}
      </button>
    </div>
  </div>
    <!-- Bottom sheet (phones only) -->
    <Teleport to="body">
      <div v-if="sheetOpen" class="fixed inset-0 z-[80] md:hidden">
        <div class="absolute inset-0 bg-black/60" @click="closeSheet"></div>
        <div
          ref="sheetRef"
          class="absolute inset-x-0 bottom-0 max-h-[85vh] overflow-y-auto overscroll-contain rounded-t-2xl border-t border-border bg-card shadow-2xl"
          role="dialog"
          aria-modal="true"
          :aria-label="t('productFilters.sheetTitle')"
          tabindex="-1"
        >
          <div class="sticky top-0 z-10 border-b border-border bg-card px-4 pb-3 pt-3">
            <div class="flex items-center justify-between gap-3">
              <span class="text-lg font-bold">{{ t('productFilters.sheetTitle') }}</span>
              <button
                type="button"
                class="inline-flex h-11 w-11 items-center justify-center rounded-xl text-muted-foreground hover:bg-secondary"
                :aria-label="t('productFilters.close')"
                @click="closeSheet"
              >
                <X class="h-5 w-5" />
              </button>
            </div>
            <div class="relative mt-2">
              <Search class="pointer-events-none absolute left-3 top-1/2 h-4 w-4 -translate-y-1/2 text-muted-foreground" aria-hidden="true" />
              <input
                ref="searchRef"
                v-model="search"
                type="search"
                class="h-11 w-full rounded-xl border border-border bg-background pl-9 pr-3 text-base text-foreground outline-none focus:border-primary"
                :placeholder="t('productFilters.search')"
                :aria-label="t('productFilters.search')"
              />
            </div>
          </div>

          <div class="px-4 pb-2">
            <div v-for="group in filteredGroups" :key="group.key" class="border-b border-border/70 py-3 last:border-b-0">
              <p class="mb-2 text-[13px] font-bold uppercase tracking-[0.12em] text-muted-foreground">{{ group.title }}</p>
              <div class="grid gap-1">
                <button
                  v-for="option in group.options"
                  :key="`${group.key}-${option.value || 'all'}`"
                  type="button"
                  class="flex min-h-12 w-full items-center justify-between gap-3 rounded-xl px-3 text-left text-base"
                  :class="option.active ? 'bg-primary/10 font-semibold text-primary' : 'text-foreground hover:bg-secondary'"
                  @click="selectOption(group.key, option.value)"
                >
                  <span class="min-w-0">{{ option.label }}</span>
                  <Check v-if="option.active" class="h-4 w-4 shrink-0" aria-hidden="true" />
                </button>
              </div>
            </div>
            <p v-if="filteredGroups.length === 0" class="py-8 text-center text-base text-muted-foreground">
              {{ t('productFilters.noResults') }}
            </p>
          </div>

          <div class="sticky bottom-0 flex gap-2 border-t border-border bg-card px-4 py-3 pb-[calc(0.75rem+env(safe-area-inset-bottom))]">
            <button
              type="button"
              class="inline-flex h-12 flex-1 items-center justify-center rounded-xl border border-border text-base font-semibold text-muted-foreground"
              @click="clear"
            >
              {{ t('productFilters.reset') }}
            </button>
            <button
              type="button"
              class="inline-flex h-12 flex-[1.4] items-center justify-center rounded-xl bg-primary text-base font-bold text-primary-foreground"
              @click="closeSheet"
            >
              {{ t('productFilters.apply') }}
            </button>
          </div>
        </div>
      </div>
    </Teleport>
</template>

<script setup lang="ts">
import { computed, nextTick, onBeforeUnmount, ref, watch } from 'vue'
import { useI18n } from 'vue-i18n'
import { Check, ChevronDown, Search, SlidersHorizontal, X } from 'lucide-vue-next'
import {
  PRODUCT_COUNTRY_CODES,
  PRODUCT_IP_TYPES,
  PRODUCT_LINE_TYPES,
  PRODUCT_SCENARIOS,
  PRODUCT_SORTS,
  emptyProductMetaFilter,
  hasProductMetaFilter,
  type ProductMetaFilter,
} from '../constants/productMeta'

const props = defineProps<{ modelValue: ProductMetaFilter }>()
const emit = defineEmits<{ 'update:modelValue': [value: ProductMetaFilter] }>()

const { t, locale } = useI18n()

const sheetOpen = ref(false)
const search = ref('')
const sheetRef = ref<HTMLElement | null>(null)
const searchRef = ref<HTMLInputElement | null>(null)
let previousOverflow = ''

const regionNames = computed<{ of: (code: string) => string | undefined } | null>(() => {
  try {
    const DisplayNames = (Intl as unknown as { DisplayNames?: new (locales: string[], options: { type: string }) => { of: (code: string) => string | undefined } }).DisplayNames
    return DisplayNames ? new DisplayNames([locale.value], { type: 'region' }) : null
  } catch {
    return null
  }
})

const countryLabel = (code: string) => {
  if (code === 'SHARED') return t('productFilters.shared')
  return regionNames.value?.of(code) || code
}

const activeCount = computed(() => {
  const { country, scenario, ipType, lineType } = props.modelValue
  return [country, scenario, ipType, lineType].filter(Boolean).length
})

type GroupKey = 'country' | 'scenario' | 'ipType' | 'lineType' | 'sort'

const groups = computed<Array<{ key: GroupKey; title: string; options: Array<{ value: string; label: string; active: boolean }> }>>(() => [
  {
    key: 'country',
    title: t('productFilters.country'),
    options: [
      { value: '', label: t('productFilters.allCountries') },
      ...PRODUCT_COUNTRY_CODES.map((code) => ({ value: code as string, label: countryLabel(code) })),
    ].map((option) => ({ ...option, active: props.modelValue.country === option.value })),
  },
  {
    key: 'scenario',
    title: t('productFilters.scenario'),
    options: [
      { value: '', label: t('productFilters.allScenarios') },
      ...PRODUCT_SCENARIOS.map((value) => ({ value, label: t(`productFilters.scenarios.${value}`) })),
    ].map((option) => ({ ...option, active: props.modelValue.scenario === option.value })),
  },
  {
    key: 'ipType',
    title: t('productFilters.ipType'),
    options: [
      { value: '', label: t('productFilters.allIpTypes') },
      ...PRODUCT_IP_TYPES.map((value) => ({ value, label: t(`productFilters.ipTypes.${value}`) })),
    ].map((option) => ({ ...option, active: props.modelValue.ipType === option.value })),
  },
  {
    key: 'lineType',
    title: t('productFilters.lineType'),
    options: [
      { value: '', label: t('productFilters.allLineTypes') },
      ...PRODUCT_LINE_TYPES.map((value) => ({ value, label: t(`productFilters.lineTypes.${value}`) })),
    ].map((option) => ({ ...option, active: props.modelValue.lineType === option.value })),
  },
  {
    key: 'sort',
    title: t('productFilters.sort'),
    options: PRODUCT_SORTS.map((value) => ({ value: value as string, label: t(`productFilters.sorts.${value}`) }))
      .map((option) => ({ ...option, active: props.modelValue.sort === option.value })),
  },
])

const filteredGroups = computed(() => {
  const term = search.value.trim().toLowerCase()
  return groups.value
    .map((group) => ({
      ...group,
      options: term
        ? group.options.filter((option) => option.label.toLowerCase().includes(term))
        : group.options,
    }))
    .filter((group) => group.options.length > 0)
})

const update = (key: keyof ProductMetaFilter, value: string) => {
  emit('update:modelValue', { ...props.modelValue, [key]: value })
}

const selectOption = (key: GroupKey, value: string) => {
  // "All" options and re-tapping the active chip both clear the group.
  const next = key !== 'sort' && props.modelValue[key] === value ? '' : value
  update(key, next)
}

const clear = () => {
  emit('update:modelValue', emptyProductMetaFilter())
}

const openSheet = async () => {
  search.value = ''
  sheetOpen.value = true
  await nextTick()
  searchRef.value?.focus()
}

const closeSheet = () => {
  sheetOpen.value = false
}

const onKeydown = (event: KeyboardEvent) => {
  if (event.key === 'Escape') {
    event.preventDefault()
    closeSheet()
    return
  }
  if (event.key !== 'Tab' || !sheetRef.value) return
  const focusable = sheetRef.value.querySelectorAll<HTMLElement>(
    'button:not([disabled]), input:not([disabled]), [href], [tabindex]:not([tabindex="-1"])',
  )
  if (focusable.length === 0) return
  const first = focusable[0] as HTMLElement
  const last = focusable[focusable.length - 1] as HTMLElement
  if (event.shiftKey && document.activeElement === first) {
    event.preventDefault()
    last.focus()
  } else if (!event.shiftKey && document.activeElement === last) {
    event.preventDefault()
    first.focus()
  }
}

watch(sheetOpen, (open) => {
  if (typeof document === 'undefined') return
  if (open) {
    previousOverflow = document.body.style.overflow
    document.body.style.overflow = 'hidden'
    document.addEventListener('keydown', onKeydown)
    return
  }
  document.body.style.overflow = previousOverflow
  document.removeEventListener('keydown', onKeydown)
})

onBeforeUnmount(() => {
  if (typeof document === 'undefined') return
  document.body.style.overflow = previousOverflow
  document.removeEventListener('keydown', onKeydown)
})
</script>

