import test from 'node:test'
import assert from 'node:assert/strict'
import {
  absoluteAssetUrl,
  buildCollectionPageJsonLd,
  buildJsonLdPayload,
  buildJsonLdScript,
  buildWebPageJsonLd,
  isServerJsonLdReusable,
  jsonLdEntities,
  normalizeSeoPath,
} from '../src/utils/seoJsonLd.ts'

test('wraps object and array JSON-LD into exactly one @graph', () => {
  const objectPayload = buildJsonLdPayload({ '@type': 'Product', name: 'A' })
  assert.ok(objectPayload)
  assert.equal(objectPayload?.['@context'], 'https://schema.org')
  assert.equal((objectPayload?.['@graph'] as unknown[]).length, 1)

  const arrayPayload = buildJsonLdPayload([
    { '@context': 'https://schema.org', '@type': 'Product', name: 'A' },
    { '@context': 'https://schema.org', '@type': 'BreadcrumbList', itemListElement: [] },
  ])
  assert.ok(arrayPayload)
  assert.equal((arrayPayload?.['@graph'] as unknown[]).length, 2)
  assert.equal(jsonLdEntities(arrayPayload as Record<string, unknown>).some((item) => '@context' in item), false)

  const graphObjectPayload = buildJsonLdPayload({
    '@context': 'https://schema.org',
    '@graph': [{ '@type': 'WebPage', name: 'A' }],
  } as unknown as Record<string, unknown>)
  assert.equal((graphObjectPayload?.['@graph'] as unknown[]).length, 1)
  assert.equal(buildJsonLdScript({ '@type': 'WebPage' })?.key, 'jsonld-page')
})

test('normalizes paths for trailing slash, URL encoding and root', () => {
  assert.equal(normalizeSeoPath('/about/'), '/about')
  assert.equal(normalizeSeoPath('/about'), '/about')
  assert.equal(normalizeSeoPath('/products/a%20b'), '/products/a b')
  assert.equal(normalizeSeoPath('/'), '/')
  assert.equal(normalizeSeoPath('products/a'), '/products/a')
})

test('only reuses server JSON-LD for the same normalized path and locale', () => {
  assert.equal(isServerJsonLdReusable('/about/', 'zh-CN', '/about', 'zh-CN'), true)
  assert.equal(isServerJsonLdReusable('/about', 'en-US', '/about', 'zh-CN'), false)
  assert.equal(isServerJsonLdReusable('/products/a%20b', 'zh-CN', '/products/a b', 'zh-CN'), true)
  assert.equal(isServerJsonLdReusable('/about', 'zh-CN', '/terms', 'zh-CN'), false)
})

test('absolute asset URLs and WebPage/CollectionPage fields are complete', () => {
  assert.equal(absoluteAssetUrl('https://cdn.example.com/a.png'), 'https://cdn.example.com/a.png')
  assert.equal(absoluteAssetUrl('/uploads/a.png', 'https://example.com'), 'https://example.com/uploads/a.png')

  const web = buildWebPageJsonLd({
    title: 'About',
    description: 'About page',
    url: 'https://example.com/about',
    locale: 'zh-CN',
    siteName: 'Site',
    siteBase: 'https://example.com',
  })
  assert.deepEqual(web.map((item) => item['@type']), ['WebPage', 'BreadcrumbList'])
  assert.equal(web[0]?.inLanguage, 'zh-CN')
  assert.equal((web[0]?.isPartOf as Record<string, unknown>)['@type'], 'WebSite')

  const collection = buildCollectionPageJsonLd({
    title: 'Blog',
    description: 'Blog list',
    url: 'https://example.com/blog',
    locale: 'en-US',
    siteName: 'Site',
    siteBase: 'https://example.com',
  })
  assert.deepEqual(collection.map((item) => item['@type']), ['CollectionPage', 'BreadcrumbList'])
})