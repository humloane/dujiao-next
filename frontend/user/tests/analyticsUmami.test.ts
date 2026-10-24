import test from 'node:test'
import assert from 'node:assert/strict'

type UmamiCall = { name: string; data: Record<string, unknown> }

const fakeWindow: {
  umami?: { track: (name: string, data?: Record<string, unknown>) => void }
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
