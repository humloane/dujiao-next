import test from 'node:test'
import assert from 'node:assert/strict'
import { joinSeoTitle, siteTitleCore, titleIncludesSite } from '../src/utils/seoTitle.ts'

test('siteTitleCore strips the marketing subtitle', () => {
  assert.equal(siteTitleCore('霓途 Neaixt｜全球静态住宅 IP'), '霓途 Neaixt')
  assert.equal(siteTitleCore('Dujiao-Next Local'), 'Dujiao-Next Local')
})

test('titleIncludesSite detects the brand core and ignores case/space', () => {
  assert.equal(titleIncludesSite('美国静态住宅 IP · 直连版｜霓途 Neaixt', '霓途 Neaixt｜全球静态住宅 IP'), true)
  assert.equal(titleIncludesSite('商品中心', '霓途 Neaixt｜全球静态住宅 IP'), false)
  assert.equal(titleIncludesSite('Dujiao-Next 本地测试', ' dujiao-next  本地测试 '), true)
})

test('joinSeoTitle keeps a single brand and appends the core only once', () => {
  assert.equal(
    joinSeoTitle('美国静态住宅 IP · 直连版｜霓途 Neaixt', '霓途 Neaixt｜全球静态住宅 IP'),
    '美国静态住宅 IP · 直连版｜霓途 Neaixt',
  )
  assert.equal(
    joinSeoTitle('商品中心', '霓途 Neaixt｜全球静态住宅 IP'),
    '商品中心 - 霓途 Neaixt',
  )
  assert.equal(
    joinSeoTitle('Articles', '霓途 Neaixt｜全球静态住宅 IP', ['霓途 Neaixt']),
    'Articles - 霓途 Neaixt',
  )
})
