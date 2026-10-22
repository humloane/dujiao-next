<template>
  <Teleport to="body">
    <div v-if="mounted" class="mfd-root" :class="{ 'is-active': active }">
      <div class="mfd-overlay" @click="requestClose"></div>

      <section
        ref="panelRef"
        class="mfd-panel"
        role="dialog"
        aria-modal="true"
        aria-labelledby="mfd-title"
        tabindex="-1"
      >
        <header class="mfd-head">
          <div class="mfd-head-text">
            <h2 id="mfd-title" class="mfd-title">{{ t('productFilters.sheetTitle') }}</h2>
            <p v-if="typeof total === 'number'" class="mfd-count" aria-live="polite">
              {{ t('productFilters.resultCount', { count: total }) }}
            </p>
          </div>
          <button
            ref="closeBtnRef"
            type="button"
            class="mfd-close"
            :aria-label="t('productFilters.close')"
            @click="requestClose"
          >
            <X class="h-5 w-5" />
          </button>
        </header>

        <div class="mfd-body">
          <div v-for="group in renderedGroups" :key="group.key" class="mfd-group">
            <p class="mfd-group-title">{{ group.title }}</p>

            <input
              v-if="group.searchable"
              v-model="searchTerms[group.key]"
              type="search"
              class="mfd-search"
              :placeholder="t('productFilters.search')"
              :aria-label="`${t('productFilters.search')} · ${group.title}`"
            />

            <div role="listbox" :aria-label="group.title" class="mfd-options">
              <template v-for="(row, index) in group.rows" :key="`${group.key}-${index}`">
                <p v-if="row.type === 'section'" class="mfd-section">{{ row.label }}</p>
                <button
                  v-else
                  type="button"
                  role="option"
                  :aria-selected="row.active"
                  class="mfd-option"
                  :class="{ 'is-selected': row.active }"
                  @click="select(group.key, row.value)"
                >
                  <span class="mfd-option-label">{{ row.label }}</span>
                  <Check v-if="row.active" class="mfd-check" aria-hidden="true" />
                </button>
              </template>
              <p v-if="group.optionCount === 0" class="mfd-empty">
                {{ t('productFilters.noResults') }}
              </p>
            </div>
          </div>
        </div>

        <footer class="mfd-foot">
          <button type="button" class="mfd-reset" @click="resetAll">
            {{ t('productFilters.resetAll') }}
          </button>
        </footer>
      </section>
    </div>
  </Teleport>
</template>


<script setup lang="ts">
import { computed, nextTick, onBeforeUnmount, reactive, ref, watch } from 'vue'
import { useI18n } from 'vue-i18n'
import { Check, X } from 'lucide-vue-next'
import {
  PRODUCT_COUNTRY_CODES,
  PRODUCT_IP_TYPES,
  PRODUCT_LINE_TYPES,
  PRODUCT_SCENARIOS,
  PRODUCT_SORTS,
  resolveMetaOptionCodes,
  type ProductMetaFilter,
  type ProductMetaOptions,
} from '../constants/productMeta'
import { createRegionNamer } from '../utils/regionLabel'

const props = defineProps<{
  open: boolean
  modelValue: ProductMetaFilter
  /** 当前结果总数；父组件拿不到真实值时传 null，抽屉里就不显示数字。 */
  total?: number | null
  /** 后端按在售商品聚合的筛选取值；未加载时回落到常量。 */
  metaOptions?: ProductMetaOptions
}>()

const emit = defineEmits<{
  'update:open': [value: boolean]
  'update:modelValue': [value: ProductMetaFilter]
  reset: []
}>()

const { t, te, locale } = useI18n()

const CLOSE_DURATION = 180

const mounted = ref(false)
const active = ref(false)
const panelRef = ref<HTMLElement | null>(null)
const closeBtnRef = ref<HTMLButtonElement | null>(null)
const searchTerms = reactive<Record<string, string>>({})
let closeTimer: ReturnType<typeof setTimeout> | null = null
let enterFrame: number | null = null
let lastFocused: HTMLElement | null = null
let savedScrollY = 0
let savedOverflow = ''
let savedPaddingRight = ''

const countryLabel = computed(() => createRegionNamer(locale.value))

type GroupKey = 'country' | 'scenario' | 'ipType' | 'lineType' | 'sort'

interface GroupOption {
  value: string
  label: string
  /** 分组小标题：国家/地区里「常用 / 其他 / 共享」分开显示。 */
  section?: string
}

interface Group {
  key: GroupKey
  title: string
  options: GroupOption[]
}

// 用非联合类型，避免模板里 v-if/v-else 无法收窄字段。
interface RenderRow {
  type: 'section' | 'option'
  label: string
  value: string
  active: boolean
}

const COUNTRY_PRESETS: string[] = [...PRODUCT_COUNTRY_CODES]

const countryOptionRows = computed<GroupOption[]>(() => {
  const common = t('productFilters.countrySectionCommon')
  const other = t('productFilters.countrySectionOther')
  const rows: GroupOption[] = [{ value: '', label: t('productFilters.allCountries') }]
  const available = resolveMetaOptionCodes(
    props.metaOptions,
    'countries',
    PRODUCT_COUNTRY_CODES,
    props.modelValue.country,
  )
  const codes = new Set(available)
  COUNTRY_PRESETS.forEach((code) => {
    if (codes.has(code)) rows.push({ value: code, label: countryLabel.value(code), section: common })
  })
  available
    .filter((code) => !COUNTRY_PRESETS.includes(code))
    .forEach((code) => rows.push({ value: code, label: countryLabel.value(code), section: other }))
  return rows
})

const localizedLabel = (group: 'scenarios' | 'ipTypes' | 'lineTypes', value: string): string => {
  const key = `productFilters.${group}.${value}`
  return te(key) ? t(key) : value
}

const groups = computed<Group[]>(() => [
  { key: 'country', title: t('productFilters.country'), options: countryOptionRows.value },
  {
    key: 'scenario',
    title: t('productFilters.scenario'),
    options: [
      { value: '', label: t('productFilters.allScenarios') },
      ...resolveMetaOptionCodes(props.metaOptions, 'scenarios', PRODUCT_SCENARIOS, props.modelValue.scenario)
        .map((value) => ({ value, label: localizedLabel('scenarios', value) })),
    ],
  },
  {
    key: 'ipType',
    title: t('productFilters.ipType'),
    options: [
      { value: '', label: t('productFilters.allIpTypes') },
      ...resolveMetaOptionCodes(props.metaOptions, 'ipTypes', PRODUCT_IP_TYPES, props.modelValue.ipType)
        .map((value) => ({ value, label: localizedLabel('ipTypes', value) })),
    ],
  },
  {
    key: 'lineType',
    title: t('productFilters.lineType'),
    options: [
      { value: '', label: t('productFilters.allLineTypes') },
      ...resolveMetaOptionCodes(props.metaOptions, 'lineTypes', PRODUCT_LINE_TYPES, props.modelValue.lineType)
        .map((value) => ({ value, label: localizedLabel('lineTypes', value) })),
    ],
  },
  {
    key: 'sort',
    title: t('productFilters.sort'),
    options: PRODUCT_SORTS.map((value) => ({ value: value as string, label: t(`productFilters.sorts.${value}`) })),
  },
])

const isSelected = (key: GroupKey, value: string) => {
  if (key === 'sort') return props.modelValue.sort === value || (!props.modelValue.sort && value === 'recommended')
  return props.modelValue[key] === value
}

const visibleOptions = (group: Group): GroupOption[] => {
  const term = String(searchTerms[group.key] || '').trim().toLowerCase()
  if (!term) return group.options
  return group.options.filter((option) => option.label.toLowerCase().includes(term))
}

// 把选项展开成「小标题 + 选项」的行序列，搜索时只保留命中的选项。
const renderedGroups = computed(() => groups.value.map((group) => {
  const rows: RenderRow[] = []
  let lastSection = ''
  visibleOptions(group).forEach((option) => {
    if (option.section && option.section !== lastSection) {
      rows.push({ type: 'section', label: option.section, value: '', active: false })
      lastSection = option.section
    }
    rows.push({
      type: 'option',
      label: option.label,
      value: option.value,
      active: isSelected(group.key, option.value),
    })
  })
  return {
    key: group.key,
    title: group.title,
    optionCount: rows.filter((row) => row.type === 'option').length,
    searchable: group.options.length > 8,
    rows,
  }
}))

const select = (key: GroupKey, value: string) => {
  // 「全部」行在筛选组里等价于清空该字段。
  emit('update:modelValue', { ...props.modelValue, [key]: value })
  emit('update:open', false)
}

const resetAll = () => {
  emit('reset')
  emit('update:open', false)
}

const requestClose = () => {
  emit('update:open', false)
}

const FOCUSABLE = 'a[href], button:not([disabled]), input:not([disabled]), [tabindex]:not([tabindex="-1"])'

const onKeydown = (event: KeyboardEvent) => {
  if (!active.value) return
  if (event.key === 'Escape') {
    event.preventDefault()
    requestClose()
    return
  }
  if (event.key !== 'Tab' || !panelRef.value) return
  const items = Array.from(panelRef.value.querySelectorAll<HTMLElement>(FOCUSABLE))
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

// 打开时记录滚动位置并锁 body；关闭时恢复原位置，避免页面跳动。
const lockScroll = () => {
  savedScrollY = window.scrollY
  savedOverflow = document.body.style.overflow
  savedPaddingRight = document.body.style.paddingRight
  const scrollbar = window.innerWidth - document.documentElement.clientWidth
  document.body.style.overflow = 'hidden'
  if (scrollbar > 0) {
    document.body.style.paddingRight = `${scrollbar}px`
  }
}

const unlockScroll = () => {
  document.body.style.overflow = savedOverflow
  document.body.style.paddingRight = savedPaddingRight
  window.scrollTo(0, savedScrollY)
}

const enter = async () => {
  if (closeTimer) {
    clearTimeout(closeTimer)
    closeTimer = null
  }
  lastFocused = document.activeElement instanceof HTMLElement ? document.activeElement : null
  Object.keys(searchTerms).forEach((key) => { searchTerms[key] = '' })
  mounted.value = true
  lockScroll()
  document.addEventListener('keydown', onKeydown)
  await nextTick()
  // 等待挂载期间用户可能已经关闭（例如连点触发器），这时不要再补入场动画。
  if (!props.open) return
  // 下一帧再置为 active，保证 translateX 过渡能跑起来；关闭时要能取消，
  // 否则「打开后立刻关闭」会在关闭动画中重新播放入场。
  cancelEnterFrame()
  enterFrame = requestAnimationFrame(() => {
    enterFrame = null
    if (mounted.value && props.open) active.value = true
  })
  closeBtnRef.value?.focus()
}

const cancelEnterFrame = () => {
  if (enterFrame !== null) {
    cancelAnimationFrame(enterFrame)
    enterFrame = null
  }
}

const leave = () => {
  cancelEnterFrame()
  active.value = false
  document.removeEventListener('keydown', onKeydown)
  unlockScroll()
  closeTimer = setTimeout(() => {
    mounted.value = false
    closeTimer = null
  }, CLOSE_DURATION)
  lastFocused?.focus?.()
  lastFocused = null
}

watch(
  () => props.open,
  (open) => {
    if (open) {
      void enter()
      return
    }
    if (mounted.value) leave()
  },
  { immediate: true },
)

onBeforeUnmount(() => {
  if (closeTimer) clearTimeout(closeTimer)
  cancelEnterFrame()
  document.removeEventListener('keydown', onKeydown)
  if (mounted.value) unlockScroll()
})
</script>

<style scoped>
.mfd-root {
  position: fixed;
  inset: 0;
  z-index: 80;
}

.mfd-overlay {
  position: absolute;
  inset: 0;
  background: rgba(2, 6, 23, 0.55);
  opacity: 0;
  transition: opacity 180ms ease;
}

.mfd-panel {
  position: absolute;
  top: 12px;
  right: 12px;
  z-index: 90;
  display: flex;
  flex-direction: column;
  width: min(92vw, 520px);
  height: calc(100dvh - 24px);
  border: 1px solid rgba(255, 255, 255, 0.14);
  border-radius: 24px;
  background: rgba(15, 23, 42, 0.55);
  background-image: linear-gradient(180deg, rgba(255, 255, 255, 0.12), transparent 30%);
  -webkit-backdrop-filter: blur(24px) saturate(140%);
  backdrop-filter: blur(24px) saturate(140%);
  box-shadow: 0 24px 80px -32px rgba(2, 6, 23, 0.9);
  color: #f8fafc;
  transform: translateX(100%);
  transition: transform 220ms cubic-bezier(0.22, 1, 0.36, 1);
}

/* 不支持 backdrop-filter 时退回不透明面板，保证可读性 */
@supports not ((backdrop-filter: blur(1px)) or (-webkit-backdrop-filter: blur(1px))) {
  .mfd-panel {
    background: #0f1730;
    background-image: linear-gradient(180deg, rgba(255, 255, 255, 0.06), transparent 30%);
  }
}

.mfd-root.is-active .mfd-overlay {
  opacity: 1;
}

.mfd-root.is-active .mfd-panel {
  transform: translateX(0);
}

.mfd-head {
  display: flex;
  align-items: flex-start;
  justify-content: space-between;
  gap: 12px;
  border-bottom: 1px solid rgba(255, 255, 255, 0.1);
  padding: 16px 16px 12px;
}

.mfd-head-text {
  min-width: 0;
}

.mfd-title {
  font-size: 18px;
  font-weight: 700;
  line-height: 1.3;
}

.mfd-count {
  margin-top: 2px;
  font-size: 13px;
  color: rgba(248, 250, 252, 0.66);
}

.mfd-close {
  display: inline-flex;
  width: 44px;
  height: 44px;
  flex-shrink: 0;
  align-items: center;
  justify-content: center;
  border-radius: 12px;
  color: rgba(248, 250, 252, 0.75);
  transition: background-color 160ms ease, color 160ms ease;
}

.mfd-close:hover {
  background: rgba(255, 255, 255, 0.1);
  color: #f8fafc;
}

.mfd-body {
  flex: 1;
  overflow-y: auto;
  overscroll-behavior: contain;
  padding: 8px 16px 24px;
}

.mfd-group {
  padding-top: 12px;
}

.mfd-group-title {
  margin-bottom: 6px;
  font-size: 13px;
  font-weight: 700;
  letter-spacing: 0.08em;
  text-transform: uppercase;
  color: rgba(248, 250, 252, 0.55);
}

.mfd-search {
  width: 100%;
  height: 44px;
  margin-bottom: 6px;
  border-radius: 12px;
  border: 1px solid rgba(255, 255, 255, 0.14);
  background: rgba(2, 6, 23, 0.35);
  padding: 0 12px;
  font-size: 16px;
  color: #f8fafc;
}

.mfd-search::placeholder {
  color: rgba(248, 250, 252, 0.45);
}

.mfd-options {
  display: grid;
  gap: 2px;
}

.mfd-section {
  margin-top: 10px;
  margin-bottom: 2px;
  padding-inline: 16px;
  font-size: 12px;
  font-weight: 600;
  letter-spacing: 0.06em;
  color: rgba(248, 250, 252, 0.42);
}

.mfd-section:first-child {
  margin-top: 0;
}

.mfd-option {
  display: flex;
  min-height: 52px;
  align-items: center;
  justify-content: space-between;
  gap: 12px;
  border-radius: 12px;
  padding: 0 16px;
  font-size: 16px;
  color: rgba(248, 250, 252, 0.9);
  transition: background-color 140ms ease, color 140ms ease;
}

.mfd-option:hover {
  background: rgba(255, 255, 255, 0.08);
}

.mfd-option.is-selected {
  background: rgba(59, 130, 246, 0.22);
  color: #ffffff;
  font-weight: 600;
}

.mfd-option-label {
  min-width: 0;
}

.mfd-check {
  width: 20px;
  height: 20px;
  flex-shrink: 0;
  color: #22d3ee;
}

.mfd-empty {
  padding: 12px 16px;
  font-size: 15px;
  color: rgba(248, 250, 252, 0.55);
}

.mfd-foot {
  border-top: 1px solid rgba(255, 255, 255, 0.1);
  padding: 12px 16px calc(12px + env(safe-area-inset-bottom, 0px));
}

.mfd-reset {
  display: inline-flex;
  width: 100%;
  min-height: 48px;
  align-items: center;
  justify-content: center;
  border-radius: 12px;
  border: 1px solid rgba(255, 255, 255, 0.18);
  font-size: 16px;
  font-weight: 600;
  color: rgba(248, 250, 252, 0.86);
}

.mfd-reset:hover {
  background: rgba(255, 255, 255, 0.08);
}

@media (prefers-reduced-motion: reduce) {
  .mfd-overlay,
  .mfd-panel {
    transition-duration: 0ms;
  }
}
</style>
