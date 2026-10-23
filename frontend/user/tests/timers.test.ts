import test from 'node:test'
import assert from 'node:assert/strict'
import { mock } from 'node:test'
import { createCountdownTimer } from '../src/utils/timers.ts'

test('countdown timer stops after the owning component unmounts', (t) => {
  mock.timers.enable({ apis: ['setInterval'] })
  t.after(() => mock.timers.reset())

  let ticks = 0
  const timer = createCountdownTimer(() => { ticks += 1 }, 1000)
  timer.start()
  assert.equal(timer.isActive(), true)

  mock.timers.tick(2000)
  assert.equal(ticks, 2)

  // 对应 onUnmounted 中的清理：停止后不应再触发任何 tick。
  timer.stop()
  assert.equal(timer.isActive(), false)
  mock.timers.tick(5000)
  assert.equal(ticks, 2)
})

test('starting a countdown twice does not leave the old interval running', (t) => {
  mock.timers.enable({ apis: ['setInterval'] })
  t.after(() => mock.timers.reset())

  let ticks = 0
  const timer = createCountdownTimer(() => { ticks += 1 }, 1000)
  timer.start()
  timer.start()

  mock.timers.tick(1000)
  assert.equal(ticks, 1)

  timer.stop()
})