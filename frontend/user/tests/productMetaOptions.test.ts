import test from 'node:test'
import assert from 'node:assert/strict'
import {
  PRODUCT_COUNTRY_CODES,
  PRODUCT_IP_TYPES,
  normalizeOptionCodes,
  resolveMetaOptionCodes,
  type ProductMetaOptions,
} from '../src/constants/productMeta.ts'

const loadedOptions = (overrides: Partial<ProductMetaOptions> = {}): ProductMetaOptions => ({
  loaded: true,
  countries: [],
  ipTypes: [],
  lineTypes: [],
  ...overrides,
})

test('option codes are trimmed and deduplicated', () => {
  assert.deepEqual(normalizeOptionCodes([' SG ', 'SG', '', null, 'TH']), ['SG', 'TH'])
  assert.deepEqual(normalizeOptionCodes('sg'), [])
})

test('unloaded country options stay empty instead of showing presets', () => {
  assert.deepEqual(
    resolveMetaOptionCodes(undefined, 'countries', PRODUCT_COUNTRY_CODES),
    [],
  )
  assert.deepEqual(
    resolveMetaOptionCodes({ loaded: false, countries: ['SG'], ipTypes: [], lineTypes: [] }, 'countries', PRODUCT_COUNTRY_CODES),
    [],
  )
})

test('loaded options come from product data, not the constants', () => {
  const options = loadedOptions({ countries: ['SG', 'TH'] })
  assert.deepEqual(
    resolveMetaOptionCodes(options, 'countries', PRODUCT_COUNTRY_CODES),
    ['SG', 'TH'],
  )
})

test('empty structured data stays empty for data-driven groups', () => {
  const options = loadedOptions({ countries: [] })
  assert.deepEqual(
    resolveMetaOptionCodes(options, 'countries', PRODUCT_COUNTRY_CODES),
    [],
  )
  assert.deepEqual(
    resolveMetaOptionCodes(options, 'ipTypes', ['residential', 'datacenter']),
    [],
  )
  assert.deepEqual(
    resolveMetaOptionCodes(options, 'lineTypes', ['direct', 'iepl']),
    [],
  )
})

test('selected value survives even when no product matches it', () => {
  const options = loadedOptions({ countries: ['SG'] })
  assert.deepEqual(
    resolveMetaOptionCodes(options, 'countries', PRODUCT_COUNTRY_CODES, 'JP'),
    ['JP', 'SG'],
  )
})

test('legacy SHARED placeholder is never exposed as a country', () => {
  const options = loadedOptions({ countries: ['SG', 'SHARED'] })
  assert.deepEqual(
    resolveMetaOptionCodes(options, 'countries', PRODUCT_COUNTRY_CODES),
    ['SG'],
  )
})

test('known values keep the canonical order, new values follow alphabetically', () => {
  const options = loadedOptions({ countries: ['TH', 'JP', 'SG', 'US'] })
  assert.deepEqual(
    resolveMetaOptionCodes(options, 'countries', PRODUCT_COUNTRY_CODES),
    ['US', 'JP', 'SG', 'TH'],
  )
})

test('option codes are normalised per group', () => {
  const options = loadedOptions({
    countries: [' sg ', 'TH'],
    ipTypes: ['Residential'],
  })
  assert.deepEqual(
    resolveMetaOptionCodes(options, 'countries', PRODUCT_COUNTRY_CODES),
    ['SG', 'TH'],
  )
  assert.deepEqual(
    resolveMetaOptionCodes(options, 'ipTypes', PRODUCT_IP_TYPES),
    ['residential'],
  )
})

test('selected value is normalised before being appended', () => {
  const options = loadedOptions({ countries: ['SG'] })
  assert.deepEqual(
    resolveMetaOptionCodes(options, 'countries', PRODUCT_COUNTRY_CODES, 'nz'),
    ['NZ', 'SG'],
  )
})
