import test from 'node:test'
import assert from 'node:assert/strict'
import { mock } from 'node:test'

const fakeWindow: { $chatwoot?: unknown; chatwootSDK?: unknown } = {}
const fakeDocument = {
  querySelector: () => null,
  createElement: () => ({ innerHTML: '', content: { firstChild: null } }),
}
Object.defineProperty(globalThis, 'window', { value: fakeWindow, configurable: true })
Object.defineProperty(globalThis, 'document', { value: fakeDocument, configurable: true })

const stateModule = await import('../src/composables/useChatwootState.ts')
const entryModule = await import('../src/composables/useChatwootEntry.ts')

test('chatwoot polling covers ready at 0, 9.9, 10.1, 12 and 15 seconds', async (t) => {
  mock.timers.enable({ apis: ['setInterval', 'setTimeout', 'Date'], now: 0 })
  t.after(() => {
    mock.timers.reset()
    delete fakeWindow.$chatwoot
    delete fakeWindow.chatwootSDK
  })

  const { ready } = stateModule.useChatwootState()
  const ensure = () => {
    fakeWindow.$chatwoot = undefined
    stateModule.ensureChatwootPolling()
  }

  // 0s: 第一次轮询就可用。
  ensure()
  fakeWindow.$chatwoot = {}
  mock.timers.tick(300)
  assert.equal(ready.value, true)

  // 9.9s: 截止前成功。
  ready.value = false
  mock.timers.setTime(0)
  ensure()
  mock.timers.tick(9600)
  fakeWindow.$chatwoot = {}
  mock.timers.tick(300)
  assert.equal(ready.value, true)

  // 10.1s: 截止后即使刚好可用，也不再打开自定义入口。
  ready.value = false
  mock.timers.setTime(0)
  ensure()
  mock.timers.tick(9900)
  fakeWindow.$chatwoot = {}
  mock.timers.tick(300)
  assert.equal(ready.value, false)

  // 12s / 15s: 同样保持不可用，原生气泡由 App 自己保留。
  for (const delay of [12_000, 15_000]) {
    ready.value = false
    mock.timers.setTime(0)
    ensure()
    mock.timers.tick(delay)
    fakeWindow.$chatwoot = {}
    mock.timers.tick(300)
    assert.equal(ready.value, false)
  }
})

test('chatwoot entry visibility excludes checkout, pay and landing pages', () => {
  assert.equal(entryModule.isChatwootEntryPathVisible('/checkout'), false)
  assert.equal(entryModule.isChatwootEntryPathVisible('/pay/1'), false)
  assert.equal(entryModule.isChatwootEntryPathVisible('/lp/x', 'landing-page'), false)
  assert.equal(entryModule.isChatwootEntryPathVisible('/'), true)
  assert.equal(entryModule.isChatwootEntryPathVisible('/products/x'), true)
})

test('unavailable chatwoot stays not ready', async () => {
  const { isChatwootReady } = await import('../src/utils/chatwoot.ts')
  delete fakeWindow.$chatwoot
  delete fakeWindow.chatwootSDK
  assert.equal(isChatwootReady(), false)
})