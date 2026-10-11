import { onScopeDispose, readonly, ref } from 'vue'

export function useReducedMotion() {
  const query = typeof window === 'undefined' ? undefined : window.matchMedia('(prefers-reduced-motion: reduce)')
  const reducedMotion = ref(query?.matches ?? false)
  const update = (event: MediaQueryListEvent) => { reducedMotion.value = event.matches }
  query?.addEventListener('change', update)
  onScopeDispose(() => query?.removeEventListener('change', update))
  return readonly(reducedMotion)
}
