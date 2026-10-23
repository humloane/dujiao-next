import { computed } from 'vue'
import { useRoute } from 'vue-router'
import { useChatwootState } from './useChatwootState.ts'

export const isChatwootEntryPathVisible = (path: string, routeName?: unknown): boolean => {
  if (path.startsWith('/checkout') || path.startsWith('/pay')) return false
  if (routeName === 'landing-page') return false
  return true
}

export const useChatwootEntryVisible = () => {
  const route = useRoute()
  const { ready } = useChatwootState()
  return computed(() => {
    if (!ready.value) return false
    return isChatwootEntryPathVisible(route.path, route.name)
  })
}