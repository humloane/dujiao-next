<template>
  <div v-if="hasMeta" class="my-5 space-y-4 rounded-2xl border bg-secondary/30 p-4">
    <div class="flex flex-wrap gap-1.5">
      <span v-if="countryLabel" class="rounded-full border bg-card px-2.5 py-1 text-xs font-semibold text-muted-foreground">
        {{ t('productMeta.country') }} · {{ countryLabel }}
      </span>
      <span v-if="ipTypeLabel" class="rounded-full border bg-card px-2.5 py-1 text-xs font-semibold text-muted-foreground">
        {{ t('productMeta.ipType') }} · {{ ipTypeLabel }}
      </span>
      <span v-if="lineTypeLabel" class="rounded-full border bg-card px-2.5 py-1 text-xs font-semibold text-muted-foreground">
        {{ t('productMeta.lineType') }} · {{ lineTypeLabel }}
      </span>
      <span
        v-for="scenario in scenarioLabels"
        :key="scenario"
        class="rounded-full bg-primary/10 px-2.5 py-1 text-xs font-semibold text-primary"
      >
        {{ scenario }}
      </span>
    </div>

    <div v-if="scenarioLabels.length" class="text-sm">
      <div class="mb-1 text-xs font-bold uppercase tracking-wider text-muted-foreground">{{ t('productMeta.scenarios') }}</div>
      <p class="leading-6 text-muted-foreground">{{ scenarioLabels.join(' · ') }}</p>
    </div>

    <div class="grid gap-2 text-xs leading-5 text-muted-foreground md:grid-cols-3">
      <div class="rounded-xl border bg-card p-3">{{ t('productMeta.compare.datacenter') }}</div>
      <div class="rounded-xl border bg-card p-3">{{ t('productMeta.compare.shared') }}</div>
      <div class="rounded-xl border bg-card p-3">{{ t('productMeta.compare.dedicated') }}</div>
    </div>

    <details class="group rounded-xl border bg-card px-3 py-2">
      <summary class="cursor-pointer list-none text-sm font-bold">{{ t('productMeta.faqTitle') }}</summary>
      <div class="mt-2 space-y-2 text-sm leading-6 text-muted-foreground">
        <p><strong class="text-foreground">{{ t('productMeta.faqQ1') }}</strong><br />{{ t('productMeta.faqA1') }}</p>
        <p><strong class="text-foreground">{{ t('productMeta.faqQ2') }}</strong><br />{{ t('productMeta.faqA2') }}</p>
        <p><strong class="text-foreground">{{ t('productMeta.faqQ3') }}</strong><br />{{ t('productMeta.faqA3') }}</p>
      </div>
    </details>
  </div>
</template>

<script setup lang="ts">
import { computed } from 'vue'
import { useI18n } from 'vue-i18n'

const props = defineProps<{ product: any }>()
const { t, locale } = useI18n()

const scenarioLabels = computed<string[]>(() => {
  const values = Array.isArray(props.product?.scenario) ? props.product.scenario : []
  return values
    .map((value: unknown) => String(value || '').trim())
    .filter(Boolean)
    .map((value: string) => t(`productFilters.scenarios.${value}`))
})

const ipTypeLabel = computed(() => {
  const value = String(props.product?.ip_type || '').trim()
  return value ? t(`productFilters.ipTypes.${value}`) : ''
})

const lineTypeLabel = computed(() => {
  const value = String(props.product?.line_type || '').trim()
  return value ? t(`productFilters.lineTypes.${value}`) : ''
})

const countryLabel = computed(() => {
  const code = String(props.product?.country || '').trim().toUpperCase()
  if (!code) return ''
  try {
    const DisplayNames = (Intl as unknown as { DisplayNames?: new (locales: string[], options: { type: string }) => { of: (value: string) => string | undefined } }).DisplayNames
    return DisplayNames ? (new DisplayNames([locale.value], { type: 'region' }).of(code) || code) : code
  } catch {
    return code
  }
})

const hasMeta = computed(() => Boolean(
  countryLabel.value || ipTypeLabel.value || lineTypeLabel.value || scenarioLabels.value.length > 0,
))
</script>
