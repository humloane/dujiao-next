import test from 'node:test'
import assert from 'node:assert/strict'

type UmamiCall = { name: string; data: Record<string, unknown> }

const fakeWindow: {
  umami?: { track: (name: string, data?: Record<string, unknown>) => void }
  gtag?: (...args: unknown[]) => void
  setTimeout?: (handler: () => void, timeout?: number) => number
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

test('trackGoogleAdsConversion awaits the gtag event callback', async () => {
  let eventData: Record<string, unknown> | null = null
  fakeWindow.setTimeout = () => 0
  fakeWindow.gtag = (...args) => {
    eventData = args[2] as Record<string, unknown>
    const callback = eventData.event_callback
    if (typeof callback === 'function') callback()
  }

  await analytics.trackGoogleAdsConversion({
    sendTo: 'AW-18534567890/abc123',
    value: 19.9,
    currency: 'CNY',
    transactionId: 'DJ202610240001',
  })

  assert.ok(eventData)
  const { event_callback, ...payload } = eventData
  assert.equal(typeof event_callback, 'function')
  assert.deepEqual(payload, {
    send_to: 'AW-18534567890/abc123',
    value: 19.9,
    currency: 'CNY',
    transaction_id: 'DJ202610240001',
  })

  delete fakeWindow.gtag
})

test('trackGoogleAdsConversion falls back to its 800ms timeout', async () => {
  let timeoutMs = 0
  fakeWindow.gtag = () => {}
  fakeWindow.setTimeout = (handler, timeout) => {
    timeoutMs = timeout || 0
    handler()
    return 0
  }

  await analytics.trackGoogleAdsConversion({ sendTo: 'AW-18534567890/abc123' })

  assert.equal(timeoutMs, 800)
  delete fakeWindow.gtag
})

test('trackGoogleAdsConversion resolves without gtag or when gtag throws', async () => {
  delete fakeWindow.gtag
  assert.equal(await analytics.trackGoogleAdsConversion({
    sendTo: 'AW-18534567890/abc123',
  }), undefined)

  fakeWindow.gtag = () => { throw new Error('gtag failed') }
  assert.equal(await analytics.trackGoogleAdsConversion({
    sendTo: 'AW-18534567890/abc123',
  }), undefined)
  delete fakeWindow.gtag
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
