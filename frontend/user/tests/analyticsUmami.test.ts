import test from 'node:test'
import assert from 'node:assert/strict'

type UmamiCall = { name: string; data: Record<string, unknown> }

const fakeWindow: {
  umami?: { track: (name: string, data?: Record<string, unknown>) => void }
  gtag?: (...args: unknown[]) => void
  $chatwoot?: { toggle?: (state?: 'open' | 'close') => void }
} = {}
Object.defineProperty(globalThis, 'window', { value: fakeWindow, configurable: true })

const analytics = await import('../src/utils/analytics.ts')
const chatwoot = await import('../src/utils/chatwoot.ts')

test('track forwards events to Umami with params', () => {
  const calls: UmamiCall[] = []
  fakeWindow.umami = { track: (name, data) => calls.push({ name, data: data || {} }) }

  analytics.track('sign_up', { method: 'email', locale: 'zh-CN' })

  assert.deepEqual(calls, [{
    name: 'sign_up',
    data: { method: 'email', locale: 'zh-CN' },
  }])
})

test('track stays silent without Umami and isolates script errors', () => {
  delete fakeWindow.umami
  assert.doesNotThrow(() => analytics.track('add_to_cart', { quantity: 1 }))

  fakeWindow.umami = { track: () => { throw new Error('umami failed') } }
  assert.doesNotThrow(() => analytics.track('add_to_cart', { quantity: 1 }))
})

test('trackGoogleAdsConversion sends order value and transaction id', () => {
  const calls: unknown[][] = []
  fakeWindow.gtag = (...args) => calls.push(args)

  analytics.trackGoogleAdsConversion({
    sendTo: 'AW-18534567890/abc123',
    value: 19.9,
    currency: 'CNY',
    transactionId: 'DJ202610240001',
  })

  assert.deepEqual(calls, [[
    'event',
    'conversion',
    {
      send_to: 'AW-18534567890/abc123',
      value: 19.9,
      currency: 'CNY',
      transaction_id: 'DJ202610240001',
    },
  ]])

  delete fakeWindow.gtag
})

test('trackGoogleAdsConversion is silent without gtag', () => {
  delete fakeWindow.gtag
  assert.doesNotThrow(() => analytics.trackGoogleAdsConversion({
    sendTo: 'AW-18534567890/abc123',
  }))
})

test('openChatwoot reports its source exactly once', () => {
  const calls: UmamiCall[] = []
  fakeWindow.umami = { track: (name, data) => calls.push({ name, data: data || {} }) }
  fakeWindow.$chatwoot = { toggle: () => {} }

  assert.equal(chatwoot.openChatwoot('fab'), true)
  assert.deepEqual(calls, [{
    name: 'chatwoot_click',
    data: { source: 'fab' },
  }])

  delete fakeWindow.$chatwoot
})
