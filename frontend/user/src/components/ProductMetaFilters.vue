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
        <optgroup v-if="countryGroups.presets.length" :label="t('productFilters.countrySectionCommon')">
          <option v-for="code in countryGroups.presets" :key="code" :value="code">{{ countryLabel(code) }}</option>
        </optgroup>
        <optgroup v-if="countryGroups.extras.length" :label="t('productFilters.countrySectionOther')">
          <option v-for="code in countryGroups.extras" :key="code" :value="code">{{ countryLabel(code) }}</option>
        </optgroup>
        <optgroup v-if="countryGroups.hasShared" :label="t('productFilters.countrySectionShared')">
          <option value="SHARED">{{ countryLabel('SHARED') }}</option>
        </optgroup>
      </select>

      <select
        :value="modelValue.scenario"
        class="h-11 min-w-[132px] flex-none rounded-xl border border-border bg-background px-3 text-base font-medium text-foreground outline-none focus:border-primary"
        :aria-label="t('productFilters.scenario')"
        @change="update('scenario', ($event.target as HTMLSelectElement).value)"
      >
        <option value="">{{ t('productFilters.allScenarios') }}</option>
        <option v-for="value in scenarioCodes" :key="value" :value="value">
          {{ scenarioLabel(value) }}
        </option>
      </select>

      <select
        :value="modelValue.ipType"
        class="h-11 min-w-[132px] flex-none rounded-xl border border-border bg-background px-3 text-base font-medium text-foreground outline-none focus:border-primary"
        :aria-label="t('productFilters.ipType')"
        @change="update('ipType', ($event.target as HTMLSelectElement).value)"
      >
        <option value="">{{ t('productFilters.allIpTypes') }}</option>
        <option v-for="value in ipTypeCodes" :key="value" :value="value">
          {{ ipTypeLabel(value) }}
        </option>
      </select>

      <select
        :value="modelValue.lineType"
        class="h-11 min-w-[132px] flex-none rounded-xl border border-border bg-background px-3 text-base font-medium text-foreground outline-none focus:border-primary"
        :aria-label="t('productFilters.lineType')"
        @change="update('lineType', ($event.target as HTMLSelectElement).value)"
      >
        <option value="">{{ t('productFilters.allLineTypes') }}</option>
        <option v-for="value in lineTypeCodes" :key="value" :value="value">
          {{ lineTypeLabel(value) }}
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
      :meta-options="metaOptions"
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
  resolveMetaOptionCodes,
  type ProductMetaFilter,
  type ProductMetaFilterKey,
  type ProductMetaOptions,
} from '../constants/productMeta'
import { createRegionNamer } from '../utils/regionLabel'

const props = defineProps<{
  modelValue: ProductMetaFilter
  /** 父组件的真实结果总数；没有就不展示数字。 */
  total?: number | null
  /** 后端按在售商品聚合的筛选取值；未加载时回落到常量。 */
  metaOptions?: ProductMetaOptions
}>()

const emit = defineEmits<{
  'update:modelValue': [value: ProductMetaFilter]
  reset: []
}>()

const { t, te, locale } = useI18n()

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

const countryLabel = computed(() => createRegionNamer(locale.value, t('productFilters.shared')))

const COUNTRY_PRESETS: string[] = PRODUCT_COUNTRY_CODES.filter((code) => code !== 'SHARED')

const countryCodes = computed(() => resolveMetaOptionCodes(
  props.metaOptions,
  'countries',
  PRODUCT_COUNTRY_CODES,
  props.modelValue.country,
))

// 国家分组：常用地区按预设顺序，其余按字母序，SHARED 单独一组。
const countryGroups = computed(() => {
  const available = new Set(countryCodes.value)
  return {
    presets: COUNTRY_PRESETS.filter((code) => available.has(code)),
    extras: countryCodes.value.filter((code) => code !== 'SHARED' && !COUNTRY_PRESETS.includes(code)),
    hasShared: available.has('SHARED'),
  }
})

const scenarioCodes = computed(() => resolveMetaOptionCodes(
  props.metaOptions,
  'scenarios',
  PRODUCT_SCENARIOS,
  props.modelValue.scenario,
))

const ipTypeCodes = computed(() => resolveMetaOptionCodes(
  props.metaOptions,
  'ipTypes',
  PRODUCT_IP_TYPES,
  props.modelValue.ipType,
))

const lineTypeCodes = computed(() => resolveMetaOptionCodes(
  props.metaOptions,
  'lineTypes',
  PRODUCT_LINE_TYPES,
  props.modelValue.lineType,
))

const localizedLabel = (group: 'scenarios' | 'ipTypes' | 'lineTypes', value: string): string => {
  const key = `productFilters.${group}.${value}`
  return te(key) ? t(key) : value
}

const scenarioLabel = (value: string) => localizedLabel('scenarios', value)
const ipTypeLabel = (value: string) => localizedLabel('ipTypes', value)
const lineTypeLabel = (value: string) => localizedLabel('lineTypes', value)

const valueLabel = (key: ProductMetaFilterKey, value: string): string => {
  switch (key) {
    case 'country':
      return countryLabel.value(value)
    case 'scenario':
      return scenarioLabel(value)
    case 'ipType':
      return ipTypeLabel(value)
    case 'lineType':
      return lineTypeLabel(value)
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

