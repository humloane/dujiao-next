<template>
  <div class="mb-5">
    <!-- Phones: trigger + count badge + removable chips. Native selects stay desktop-only. -->
    <div class="md:hidden">
      <div class="flex items-stretch gap-2">
        <button
          type="button"
          class="inline-flex h-11 min-w-0 flex-1 items-center justify-between gap-2 rounded-xl border border-border bg-card px-4 text-base font-semibold text-foreground"
          :aria-expanded="drawerOpen"
          aria-haspopup="dialog"
          @click="drawerOpen = true"
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
          :aria-label="t('productFilters.resetAll')"
          @click="emit('reset')"
        >
          <X class="h-4 w-4" />
        </button>
      </div>

      <ul v-if="chips.length" class="mt-2 flex flex-wrap gap-2" :aria-label="t('productFilters.selectedLabel')">
        <li v-for="chip in chips" :key="chip.key">
          <button
            type="button"
            class="inline-flex h-9 items-center gap-1.5 rounded-full border border-border bg-secondary px-3 text-sm font-medium text-foreground"
            :aria-label="`${t('productFilters.reset')} ${chip.label}`"
            @click="removeChip(chip.key)"
          >
            <span class="max-w-[10rem] truncate">{{ chip.label }}</span>
            <X class="h-3.5 w-3.5 shrink-0 opacity-70" aria-hidden="true" />
          </button>
        </li>
      </ul>
    </div>


    <!-- Tablet / desktop: unchanged inline selects -->
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
        @click="emit('reset')"
      >
        <X class="h-4 w-4" />
        {{ t('products.clearFilters') }}
      </button>
    </div>

    <MobileFilterDrawer
      v-if="isPhone"
      v-model:open="drawerOpen"
      :model-value="modelValue"
      :total="total ?? null"
      @update:model-value="emit('update:modelValue', $event)"
      @reset="emit('reset')"
    />
  </div>
</template>

<script setup lang="ts">
import { computed, onBeforeUnmount, onMounted, ref } from 'vue'
import { useI18n } from 'vue-i18n'
import { ChevronDown, SlidersHorizontal, X } from 'lucide-vue-next'
import MobileFilterDrawer from './MobileFilterDrawer.vue'
import {
  PRODUCT_COUNTRY_CODES,
  PRODUCT_IP_TYPES,
  PRODUCT_LINE_TYPES,
  PRODUCT_META_FILTER_KEYS,
  PRODUCT_SCENARIOS,
  PRODUCT_SORTS,
  hasProductMetaFilter,
  type ProductMetaFilter,
  type ProductMetaFilterKey,
} from '../constants/productMeta'

const props = defineProps<{
  modelValue: ProductMetaFilter
  /** 父组件的真实结果总数；没有就不展示数字。 */
  total?: number | null
}>()

const emit = defineEmits<{
  'update:modelValue': [value: ProductMetaFilter]
  reset: []
}>()

const { t, locale } = useI18n()

const drawerOpen = ref(false)
const isPhone = ref(false)
let mediaQuery: MediaQueryList | null = null

const syncPhone = () => {
  isPhone.value = mediaQuery?.matches ?? false
  if (!isPhone.value) drawerOpen.value = false
}

onMounted(() => {
  mediaQuery = window.matchMedia('(max-width: 767px)')
  syncPhone()
  mediaQuery.addEventListener('change', syncPhone)
})

onBeforeUnmount(() => {
  mediaQuery?.removeEventListener('change', syncPhone)
})

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

const valueLabel = (key: ProductMetaFilterKey, value: string): string => {
  switch (key) {
    case 'country':
      return countryLabel(value)
    case 'scenario':
      return t(`productFilters.scenarios.${value}`)
    case 'ipType':
      return t(`productFilters.ipTypes.${value}`)
    case 'lineType':
      return t(`productFilters.lineTypes.${value}`)
    default:
      return value
  }
}

const activeCount = computed(() => PRODUCT_META_FILTER_KEYS.filter((key) => props.modelValue[key]).length)

const chips = computed(() => PRODUCT_META_FILTER_KEYS
  .filter((key) => props.modelValue[key])
  .map((key) => ({ key, label: valueLabel(key, props.modelValue[key]) }))
  .filter((chip) => chip.label))

const update = (key: keyof ProductMetaFilter, value: string) => {
  emit('update:modelValue', { ...props.modelValue, [key]: value })
}

const removeChip = (key: ProductMetaFilterKey) => {
  update(key, '')
}
</script>

