import { describe, expect, it } from 'vitest'
import { brandInitials, resolveAdminBrandName } from './brand'

describe('admin brand helpers', () => {
  it('prefers explicit brand names and localized SEO title fallback', () => {
    expect(resolveAdminBrandName({ brand: { name: 'Acme' } })).toBe('Acme')
    expect(resolveAdminBrandName({ brand: { site_name: 'Site' } })).toBe('Site')
    expect(resolveAdminBrandName({ seo: { title: { 'en-US': 'Acme Global' } } }, 'en-US')).toBe('Acme Global')
    expect(resolveAdminBrandName({})).toBe('')
  })

  it('builds collapsed initials from the configured brand', () => {
    expect(brandInitials('Dujiao-Next')).toBe('DN')
    expect(brandInitials('霓途 Neaixt')).toBe('霓N')
    expect(brandInitials('')).toBe('AD')
  })
})