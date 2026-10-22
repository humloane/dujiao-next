import test from 'node:test'
import assert from 'node:assert/strict'
import {
  PRODUCT_COUNTRY_CODES,
  PRODUCT_IP_TYPES,
  PRODUCT_SCENARIOS,
  normalizeOptionCodes,
  resolveMetaOptionCodes,
  type ProductMetaOptions,
} from '../src/constants/productMeta.ts'

const loadedOptions = (overrides: Partial<ProductMetaOptions> = {}): ProductMetaOptions => ({
  loaded: true,
  countries: [],
  scenarios: [],
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
    resolveMetaOptionCodes({ loaded: false, countries: ['SG'], scenarios: [], ipTypes: [], lineTypes: [] }, 'countries', PRODUCT_COUNTRY_CODES),
    [],
  )
  assert.deepEqual(
    resolveMetaOptionCodes(undefined, 'scenarios', PRODUCT_SCENARIOS),
    [...PRODUCT_SCENARIOS],
  )
})

test('loaded options come from product data, not the constants', () => {
  const options = loadedOptions({ countries: ['SG', 'TH'], scenarios: ['live'] })
  assert.deepEqual(
    resolveMetaOptionCodes(options, 'countries', PRODUCT_COUNTRY_CODES),
    ['SG', 'TH'],
  )
  assert.deepEqual(
    resolveMetaOptionCodes(options, 'scenarios', PRODUCT_SCENARIOS),
    ['live'],
  )
})

test('empty country data stays empty; other groups keep their fallback', () => {
  const options = loadedOptions({ countries: [], scenarios: [] })
  assert.deepEqual(
    resolveMetaOptionCodes(options, 'countries', PRODUCT_COUNTRY_CODES),
    [],
  )
  assert.deepEqual(
    resolveMetaOptionCodes(options, 'ipTypes', ['residential', 'datacenter']),
    ['residential', 'datacenter'],
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
    scenarios: ['Live'],
    ipTypes: ['Residential'],
  })
  assert.deepEqual(
    resolveMetaOptionCodes(options, 'countries', PRODUCT_COUNTRY_CODES),
    ['SG', 'TH'],
  )
  assert.deepEqual(
    resolveMetaOptionCodes(options, 'scenarios', PRODUCT_SCENARIOS),
    ['live'],
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
