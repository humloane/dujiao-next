<template>
  <div v-if="isValid" class="lp min-h-screen">
    <!-- Standalone top bar: landing pages render outside the site shell -->
    <header class="lp-topbar">
      <div class="container flex h-16 items-center justify-between gap-3">
        <router-link to="/" class="flex min-w-0 items-center gap-2 text-base font-bold">
          <img v-if="brandLogo" :src="brandLogo" :alt="brandName" class="h-7 max-w-[120px] object-contain" />
          <span class="min-w-0 truncate">{{ brandName }}</span>
        </router-link>
        <div class="flex shrink-0 items-center gap-2">
          <router-link to="/" class="hidden text-sm font-semibold text-[color:var(--lp-ink-muted)] sm:inline-flex">
            {{ tx('nav.home') }}
          </router-link>
          <router-link to="/products" class="hidden text-sm font-semibold text-[color:var(--lp-ink-muted)] sm:inline-flex">
            {{ tx('nav.products') }}
          </router-link>
          <a
            :href="telegramUrl || '/about'"
            :target="telegramUrl ? '_blank' : undefined"
            rel="noopener noreferrer"
            class="lp-btn lp-btn--ghost h-11 px-4 text-sm"
            :aria-label="tx('cta.consult')"
            @click="onTelegramClick('topbar')"
          >
            <Send class="h-4 w-4" aria-hidden="true" />
            <span class="hidden sm:inline">{{ tx('cta.consult') }}</span>
          </a>
        </div>
      </div>
    </header>

    <!-- ============ 1. HERO ============ -->
    <section class="relative">
      <div class="lp-hero-glow" aria-hidden="true"></div>
      <div class="container relative pb-12 pt-10 md:pb-16 md:pt-16">
        <div class="grid items-center gap-10 lg:grid-cols-[minmax(0,55fr)_minmax(0,45fr)] lg:gap-12">
          <div class="min-w-0">
            <span class="lp-eyebrow">
              <span class="h-1.5 w-1.5 rounded-full" style="background-image: var(--lp-gradient)"></span>
              {{ text('eyebrow') }}
            </span>
            <h1 class="lp-h1">{{ text('title') }}</h1>
            <p class="lp-lead">{{ text('subtitle') }}</p>

            <div class="mt-8 flex flex-col gap-3 sm:flex-row sm:flex-wrap">
              <a
                :href="telegramUrl || '/about'"
                :target="telegramUrl ? '_blank' : undefined"
                rel="noopener noreferrer"
                class="lp-btn lp-btn--primary lp-btn--block-mobile"
                @click="onTelegramClick('hero')"
              >
                <Send class="h-4 w-4" aria-hidden="true" />
                {{ tx('cta.consult') }}
              </a>
              <router-link to="/products" class="lp-btn lp-btn--ghost lp-btn--block-mobile">
                {{ tx('cta.viewProducts') }}
                <ArrowRight class="h-4 w-4" aria-hidden="true" />
              </router-link>
            </div>

            <ul class="mt-6 grid gap-2 sm:grid-cols-2">
              <li v-for="item in trust" :key="item" class="flex items-start gap-2 text-[15px] leading-6 text-[color:var(--lp-ink-muted)] md:text-base">
                <Check class="mt-0.5 h-4 w-4 shrink-0" style="color: var(--lp-cyan)" aria-hidden="true" />
                <span class="min-w-0">{{ item }}</span>
              </li>
            </ul>
          </div>

          <!-- Visual anchor: pure CSS/SVG so it never depends on an external asset -->
          <div class="min-w-0 lg:pl-4">
            <div class="lp-visual">
              <svg class="lp-visual__map" viewBox="0 0 320 200" role="img" :aria-label="text('eyebrow')">
                <defs>
                  <linearGradient id="lpLine" x1="0" y1="0" x2="1" y2="1">
                    <stop offset="0%" stop-color="#22D3EE" />
                    <stop offset="55%" stop-color="#3B82F6" />
                    <stop offset="100%" stop-color="#8B5CF6" />
                  </linearGradient>
                </defs>
                <g stroke="url(#lpLine)" stroke-width="1.2" fill="none" opacity="0.85">
                  <path d="M52 132 C 92 96, 128 84, 168 88" />
                  <path d="M168 88 C 206 92, 244 108, 268 138" />
                  <path d="M92 58 C 130 66, 150 74, 168 88" />
                  <path d="M168 88 C 196 66, 224 54, 250 50" />
                </g>
                <g fill="#F8FAFC">
                  <circle cx="52" cy="132" r="3.4" />
                  <circle cx="168" cy="88" r="4.2" fill="#22D3EE" />
                  <circle cx="268" cy="138" r="3.4" />
                  <circle cx="92" cy="58" r="3" />
                  <circle cx="250" cy="50" r="3" />
                </g>
                <g fill="#94A3B8" font-size="8" font-family="Inter, sans-serif">
                  <text x="40" y="148">US</text>
                  <text x="160" y="76">EU</text>
                  <text x="258" y="154">APAC</text>
                  <text x="80" y="50">UK</text>
                  <text x="240" y="42">JP</text>
                </g>
              </svg>

              <div class="lp-visual__card">
                <div class="flex items-center justify-between gap-3">
                  <span class="text-[13px] font-semibold uppercase tracking-[0.14em] text-[color:var(--lp-ink-muted)]">EGRESS</span>
                  <span class="lp-badge" style="border-color: rgba(34, 211, 238, 0.45); color: #67e8f9">
                    <span class="h-1.5 w-1.5 rounded-full" style="background: var(--lp-cyan)"></span>
                    LIVE
                  </span>
                </div>
                <p class="mt-3 font-mono text-lg font-bold tracking-tight">203.0.113.24</p>
                <dl class="mt-3 grid grid-cols-2 gap-x-3 gap-y-2 text-[13px] text-[color:var(--lp-ink-muted)]">
                  <div><dt class="opacity-70">ASN</dt><dd class="font-semibold text-[color:var(--lp-ink)]">AS7922</dd></div>
                  <div><dt class="opacity-70">TYPE</dt><dd class="font-semibold text-[color:var(--lp-ink)]">Residential</dd></div>
                  <div><dt class="opacity-70">LATENCY</dt><dd class="font-semibold text-[color:var(--lp-ink)]">142 ms</dd></div>
                  <div><dt class="opacity-70">LOSS</dt><dd class="font-semibold text-[color:var(--lp-ink)]">0.0 %</dd></div>
                </dl>
                <div class="mt-3 h-1.5 overflow-hidden rounded-full" style="background: rgba(148, 163, 184, 0.18)">
                  <div class="h-full w-[78%] rounded-full" style="background-image: var(--lp-gradient)"></div>
                </div>
              </div>
            </div>
          </div>
        </div>
      </div>
    </section>


    <!-- ============ 2. TRUST STRIP ============ -->
    <section class="lp-section lp-section--flush">
      <div class="container">
        <div class="lp-trust">
          <div v-for="item in trustStrip" :key="item" class="lp-trust__item">
            <Check class="h-4 w-4 shrink-0" style="color: var(--lp-cyan)" aria-hidden="true" />
            <span>{{ item }}</span>
          </div>
        </div>
      </div>
    </section>

    <!-- ============ 3. PAINS ============ -->
    <section class="lp-section">
      <div class="container">
        <h2 class="lp-h2">{{ tx('sections.pains') }}</h2>
        <div class="mt-6 grid gap-3 md:mt-8 md:grid-cols-2 md:gap-4">
          <div v-for="(item, i) in list('pains')" :key="item" class="lp-card lp-card--soft flex gap-3">
            <span class="mt-2 h-2 w-2 shrink-0 rounded-full bg-[#f87171]" aria-hidden="true"></span>
            <div class="min-w-0">
              <p class="text-[13px] font-bold uppercase tracking-[0.14em] text-[color:var(--lp-ink-muted)]">{{ pad(i) }}</p>
              <p class="mt-1 text-base leading-7">{{ item }}</p>
            </div>
          </div>
        </div>
      </div>
    </section>

    <!-- ============ 4. SOLUTIONS ============ -->
    <section class="lp-section lp-section--tint">
      <div class="container">
        <h2 class="lp-h2">{{ tx('sections.solutions') }}</h2>
        <div class="mt-6 grid gap-3 md:mt-8 md:grid-cols-2 md:gap-4 lg:grid-cols-3">
          <article v-for="(item, i) in list('solutions')" :key="item" class="lp-card">
            <span class="lp-step">{{ pad(i) }}</span>
            <p class="mt-3 text-base leading-7">{{ item }}</p>
          </article>
        </div>
      </div>
    </section>

    <!-- ============ 5. PROCESS ============ -->
    <section class="lp-section">
      <div class="container">
        <h2 class="lp-h2">{{ tx('sections.process') }}</h2>
        <ol class="mt-6 grid gap-3 md:mt-8 md:grid-cols-2 md:gap-4 lg:grid-cols-4">
          <li v-for="(item, i) in process" :key="item" class="lp-card lp-card--soft">
            <div class="flex items-center gap-2">
              <span class="lp-step">{{ pad(i) }}</span>
              <span class="text-[13px] font-semibold uppercase tracking-[0.14em] text-[color:var(--lp-ink-muted)]">
                {{ tx('sections.process') }}
              </span>
            </div>
            <p class="mt-3 text-base leading-7">{{ item }}</p>
          </li>
        </ol>
      </div>
    </section>

    <!-- ============ 6. SPECS ============ -->
    <section class="lp-section lp-section--tint">
      <div class="container">
        <h2 class="lp-h2">{{ tx('sections.specs') }}</h2>
        <dl class="mt-6 grid gap-2 md:mt-8 md:grid-cols-2 md:gap-3">
          <div v-for="row in specRows" :key="row.label" class="lp-card flex flex-col gap-1 py-3 md:flex-row md:items-baseline md:justify-between md:gap-4">
            <dt class="text-[13px] font-semibold uppercase tracking-[0.12em] text-[color:var(--lp-ink-muted)]">{{ row.label }}</dt>
            <dd class="min-w-0 text-base font-semibold leading-6 md:text-right">{{ row.value }}</dd>
          </div>
        </dl>
        <p class="mt-4 text-[13px] leading-6 text-[color:var(--lp-ink-muted)]">{{ tx('labels.note') }}</p>
      </div>
    </section>

    <!-- ============ 7. COMPARE ============ -->
    <section class="lp-section">
      <div class="container">
        <h2 class="lp-h2">{{ tx('sections.compare') }}</h2>
        <p class="lp-lead">{{ compare.intro }}</p>
        <div class="mt-6 grid gap-3 md:mt-8 md:grid-cols-3 md:gap-4">
          <article v-for="(item, i) in compare.items" :key="item.name" class="lp-card" :class="{ 'lp-card--accent': i === 1 }">
            <h3 class="text-lg font-bold">{{ item.name }}</h3>
            <p class="mt-2 text-base leading-7 text-[color:var(--lp-ink-muted)]">{{ item.detail }}</p>
            <p class="mt-3 border-t pt-3 text-[13px] font-semibold uppercase tracking-[0.12em] text-[color:var(--lp-ink-muted)]">
              {{ tx('labels.compareFit') }}
            </p>
            <p class="mt-1 text-base">{{ item.fit }}</p>
          </article>
        </div>
      </div>
    </section>

    <!-- ============ 8. SCENARIOS ============ -->
    <section class="lp-section lp-section--tint">
      <div class="container">
        <h2 class="lp-h2">{{ tx('sections.scenarios') }}</h2>
        <div class="mt-6 grid gap-3 md:mt-8 md:grid-cols-2 md:gap-4 lg:grid-cols-3">
          <div v-for="item in list('scenarios')" :key="item" class="lp-card lp-card--soft flex items-start gap-3">
            <Check class="mt-0.5 h-5 w-5 shrink-0" style="color: var(--lp-cyan)" aria-hidden="true" />
            <p class="min-w-0 text-base leading-7">{{ item }}</p>
          </div>
        </div>
      </div>
    </section>

    <!-- ============ 9. PRICING ============ -->
    <section class="lp-section">
      <div class="container">
        <h2 class="lp-h2">{{ tx('sections.pricing') }}</h2>
        <p class="lp-lead">{{ pricing.intro }}</p>
        <div class="mt-6 grid gap-3 md:mt-8 md:grid-cols-3 md:gap-4">
          <article v-for="(tier, i) in pricing.tiers" :key="tier.name" class="lp-card flex flex-col" :class="{ 'lp-card--accent': i === 1 }">
            <h3 class="text-lg font-bold">{{ tier.name }}</h3>
            <p class="mt-2 text-2xl font-extrabold">{{ pricing.contactQuote }}</p>
            <p class="text-[13px] text-[color:var(--lp-ink-muted)]">{{ tier.cycle }}</p>
            <ul class="mt-4 flex-1 space-y-2">
              <li v-for="feature in tier.features" :key="feature" class="flex gap-2 text-base leading-7 text-[color:var(--lp-ink-muted)]">
                <Check class="mt-1 h-4 w-4 shrink-0" style="color: var(--lp-cyan)" aria-hidden="true" />
                <span class="min-w-0">{{ feature }}</span>
              </li>
            </ul>
            <a
              :href="telegramUrl || '/about'"
              :target="telegramUrl ? '_blank' : undefined"
              rel="noopener noreferrer"
              class="lp-btn lp-btn--ghost lp-btn--block-mobile mt-5"
              @click="onTelegramClick('pricing')"
            >
              {{ tx('cta.quote') }}
            </a>
          </article>
        </div>
        <p class="mt-4 text-[13px] leading-6 text-[color:var(--lp-ink-muted)]">{{ text('pricingNote') }}</p>
      </div>
    </section>


    <!-- ============ 10. CASES ============ -->
    <section class="lp-section lp-section--tint">
      <div class="container">
        <div class="flex flex-wrap items-center gap-3">
          <h2 class="lp-h2">{{ tx('sections.cases') }}</h2>
          <span class="lp-badge">{{ tx('labels.example') }}</span>
        </div>
        <div class="mt-6 grid gap-3 md:mt-8 md:grid-cols-3 md:gap-4">
          <figure v-for="item in cases.items" :key="item.title" class="lp-card flex flex-col">
            <figcaption class="text-[13px] font-semibold uppercase tracking-[0.12em] text-[color:var(--lp-ink-muted)]">
              {{ item.title }}
            </figcaption>
            <blockquote class="mt-3 flex-1 text-base leading-7">“{{ item.quote }}”</blockquote>
            <p class="mt-3 border-t pt-3 text-[13px] text-[color:var(--lp-ink-muted)]">{{ item.meta }}</p>
          </figure>
        </div>
        <p class="mt-4 text-[13px] leading-6 text-[color:var(--lp-ink-muted)]">{{ cases.disclaimer }}</p>
      </div>
    </section>

    <!-- ============ 11. FAQ ============ -->
    <section id="faq" class="lp-section">
      <div class="container">
        <h2 class="lp-h2">{{ tx('sections.faq') }}</h2>
        <div class="mt-6 space-y-2 md:mt-8">
          <details v-for="(item, i) in faq" :key="item.q" class="lp-faq" :open="i === 0">
            <summary class="lp-faq__summary">
              <span class="min-w-0">{{ item.q }}</span>
              <ChevronDown class="lp-faq__chevron" aria-hidden="true" />
            </summary>
            <p class="lp-faq__body">{{ item.a }}</p>
          </details>
        </div>
      </div>
    </section>

    <!-- ============ 12. FINAL CTA ============ -->
    <section class="lp-final">
      <div class="container">
        <h2 class="lp-h2">{{ text('finalTitle') || tx('cta.finalFallback') }}</h2>
        <p class="lp-lead">{{ text('finalSubtitle') }}</p>
        <div class="mt-6 flex flex-col gap-3 sm:flex-row">
          <a
            :href="telegramUrl || '/about'"
            :target="telegramUrl ? '_blank' : undefined"
            rel="noopener noreferrer"
            class="lp-btn lp-btn--primary lp-btn--block-mobile"
            @click="onTelegramClick('final')"
          >
            <Send class="h-4 w-4" aria-hidden="true" />
            {{ tx('cta.consult') }}
          </a>
          <router-link to="/products" class="lp-btn lp-btn--ghost lp-btn--block-mobile">
            {{ tx('cta.viewProducts') }}
            <ArrowRight class="h-4 w-4" aria-hidden="true" />
          </router-link>
        </div>
      </div>
    </section>

    <!-- Mobile sticky CTA: replaces the bottom nav on landing pages -->
    <div class="lp-sticky-cta md:hidden">
      <a
        :href="telegramUrl || '/about'"
        :target="telegramUrl ? '_blank' : undefined"
        rel="noopener noreferrer"
        class="lp-btn lp-btn--primary flex-1"
        @click="onTelegramClick('sticky')"
      >
        <Send class="h-4 w-4" aria-hidden="true" />
        {{ tx('cta.consult') }}
      </a>
      <router-link to="/products" class="lp-btn lp-btn--ghost flex-1">
        {{ tx('cta.viewProducts') }}
      </router-link>
    </div>
  </div>

  <div v-else class="lp min-h-screen">
    <div class="container py-32 text-center">
      <h1 class="lp-h2">{{ t('common.noContent') }}</h1>
      <router-link to="/products" class="lp-btn lp-btn--primary mt-6 inline-flex">
        {{ t('nav.products') }}
      </router-link>
    </div>
  </div>
</template>



<script setup lang="ts">
import { computed } from 'vue'
import { useI18n } from 'vue-i18n'
import { useRoute } from 'vue-router'
import { ArrowRight, Check, ChevronDown, Send } from 'lucide-vue-next'
import { useAppStore } from '../stores/app'
import { getImageUrl } from '../utils/image'
import { usePageSeo } from '../composables/usePageSeo'
import { track } from '../utils/analytics'

const LANDING_SLUGS = [
  'us-residential-ip',
  'tiktok-live-ip',
  'iepl',
  'asia-ip',
  'enterprise',
] as const

const route = useRoute()
const { t, tm } = useI18n()
const appStore = useAppStore()

const slug = computed(() => String(route.params.slug || ''))
const isValid = computed(() => (LANDING_SLUGS as readonly string[]).includes(slug.value))
const pageKey = computed(() => `landing.pages.${slug.value}`)

const tx = (path: string) => String(t(`landing.${path}`) || '')
const text = (field: string) => String(t(`${pageKey.value}.${field}`) || '')
const sharedText = (path: string) => String(t(`landing.${path}`) || '')

/**
 * tm() gives us the shape (array length) while t() resolves each leaf.
 * Resolving leaves through t() avoids compiled-AST nodes leaking into the DOM
 * when vue-i18n runs in runtime-only JIT mode.
 */
const indexes = (path: string): number[] => {
  const raw = tm(path) as unknown
  return Array.isArray(raw) ? raw.map((_, i) => i) : []
}

const list = (field: string): string[] => indexes(`${pageKey.value}.${field}`)
  .map((i) => String(t(`${pageKey.value}.${field}.${i}`) || ''))
  .filter(Boolean)

const trust = computed(() => list('trust'))
const process = computed(() => indexes('landing.process').map((i) => sharedText(`process.${i}`)))

const trustStrip = computed(() => [
  ...trust.value,
  sharedText('labels.trustPayment'),
  sharedText('labels.trustTelegram'),
].filter(Boolean))

const specRows = computed(() => [
  ['region', tx('labels.specRegion')],
  ['type', tx('labels.specType')],
  ['protocol', tx('labels.specProtocol')],
  ['devices', tx('labels.specDevices')],
  ['delivery', tx('labels.specDelivery')],
  ['billing', tx('labels.specBilling')],
  ['support', tx('labels.specSupport')],
].map(([key, label]) => ({ label, value: text(`specs.${key}`) })).filter((row) => row.label && row.value))

const compare = computed(() => ({
  intro: sharedText('compare.intro'),
  items: indexes('landing.compare.items').map((i) => ({
    name: sharedText(`compare.items.${i}.name`),
    detail: sharedText(`compare.items.${i}.detail`),
    fit: sharedText(`compare.items.${i}.fit`),
  })),
}))

const pricing = computed(() => ({
  intro: sharedText('pricing.intro'),
  contactQuote: sharedText('pricing.contactQuote'),
  tiers: indexes('landing.pricing.tiers').map((i) => ({
    name: sharedText(`pricing.tiers.${i}.name`),
    cycle: sharedText(`pricing.tiers.${i}.cycle`),
    features: indexes(`landing.pricing.tiers.${i}.features`).map((j) => sharedText(`pricing.tiers.${i}.features.${j}`)),
  })),
}))

const cases = computed(() => ({
  disclaimer: sharedText('cases.disclaimer'),
  items: indexes('landing.cases.items').map((i) => ({
    title: sharedText(`cases.items.${i}.title`),
    quote: sharedText(`cases.items.${i}.quote`),
    meta: sharedText(`cases.items.${i}.meta`),
  })),
}))

const faq = computed(() => indexes(`${pageKey.value}.faq`).map((i) => ({
  q: String(t(`${pageKey.value}.faq.${i}.q`) || ''),
  a: String(t(`${pageKey.value}.faq.${i}.a`) || ''),
})).filter((item) => item.q && item.a))

const telegramUrl = computed(() => String(appStore.config?.contact?.telegram || '').trim())
const brandName = computed(() => String(appStore.config?.brand?.site_name || '').trim() || 'Dujiao-Next')
const brandLogo = computed(() => {
  const raw = String(appStore.config?.brand?.site_logo || '').trim()
  return raw ? getImageUrl(raw) : ''
})
const siteBase = computed(() => {
  const configured = String(appStore.config?.brand?.site_url || '').trim().replace(/\/+$/, '')
  if (configured) return configured
  return typeof window !== 'undefined' ? window.location.origin : ''
})

const pad = (index: number) => String(index + 1).padStart(2, '0')

const onTelegramClick = (section: string) => {
  track('telegram_click', { landing: slug.value, section })
}

usePageSeo({
  title: () => (isValid.value ? text('title') : ''),
  description: () => (isValid.value ? text('subtitle') : ''),
  keywords: () => (isValid.value ? text('keywords') : ''),
  image: () => String(appStore.config?.seo?.default_og_image || ''),
  canonicalPath: () => `/lp/${slug.value}`,
  type: () => 'website',
  noindex: () => !isValid.value,
  jsonLd: () => {
    if (!isValid.value) return undefined
    const title = text('title')
    const canonical = `${siteBase.value}/lp/${slug.value}`
    const graph: Record<string, unknown>[] = [
      { '@context': 'https://schema.org', '@type': 'WebPage', name: title, description: text('subtitle'), url: canonical },
      {
        '@context': 'https://schema.org',
        '@type': 'BreadcrumbList',
        itemListElement: [
          { '@type': 'ListItem', position: 1, name: String(appStore.config?.brand?.site_name || 'Dujiao-Next'), item: `${siteBase.value}/` },
          { '@type': 'ListItem', position: 2, name: title, item: canonical },
        ],
      },
    ]
    if (faq.value.length > 0) {
      graph.push({
        '@context': 'https://schema.org',
        '@type': 'FAQPage',
        mainEntity: faq.value.map((item) => ({
          '@type': 'Question',
          name: item.q,
          acceptedAnswer: { '@type': 'Answer', text: item.a },
        })),
      })
    }
    return graph
  },
})
</script>
