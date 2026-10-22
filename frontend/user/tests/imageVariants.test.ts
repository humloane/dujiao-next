import test from 'node:test'
import assert from 'node:assert/strict'
import { getBannerSrcSet, getImageVariantSrcSet } from '../src/utils/image.ts'

test('local uploads get responsive WebP variants', () => {
  const srcset = getImageVariantSrcSet('/uploads/products/us-ip.png')
  assert.match(srcset, /\/uploads\/products\/us-ip-480\.webp 480w/)
  assert.match(srcset, /\/uploads\/products\/us-ip-800\.webp 800w/)
  assert.match(srcset, /\/uploads\/products\/us-ip-1200\.webp 1200w/)
})

test('banner srcset includes the 1800w variant', () => {
  assert.match(getBannerSrcSet('/uploads/banner.jpg'), /banner-1800\.webp 1800w/)
})

test('external and non-upload images do not get variants', () => {
  assert.equal(getImageVariantSrcSet('https://cdn.example.com/a.png'), '')
  assert.equal(getImageVariantSrcSet('/og-default.png'), '')
  assert.equal(getImageVariantSrcSet(''), '')
})
