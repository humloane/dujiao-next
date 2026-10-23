import test from 'node:test'
import assert from 'node:assert/strict'
import { isChatwootReady, openChatwoot } from '../src/utils/chatwoot.ts'

test('chatwoot helpers are safe when the widget is not loaded', () => {
  assert.equal(isChatwootReady(), false)
  assert.equal(openChatwoot(), false)
})
