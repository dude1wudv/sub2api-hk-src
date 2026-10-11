import { onScopeDispose, toValue, watch, type MaybeRefOrGetter } from 'vue'

const owners = new Set<symbol>()
let original: { overflow: string; paddingRight: string; hadClass: boolean } | undefined

/** A drawer and a dialog may be open together; only the last owner restores scrolling. */
export function useScrollLock(active: MaybeRefOrGetter<boolean>) {
  const owner = Symbol('scroll-lock')
  function release() {
    if (!owners.delete(owner) || owners.size || !original) return
    document.body.style.overflow = original.overflow
    document.body.style.paddingRight = original.paddingRight
    document.body.classList.toggle('modal-open', original.hadClass)
    original = undefined
  }
  watch(() => toValue(active), locked => {
    if (typeof document === 'undefined') return
    if (!locked) { release(); return }
    if (owners.has(owner)) return
    if (!owners.size) {
      const body = document.body
      original = { overflow: body.style.overflow, paddingRight: body.style.paddingRight, hadClass: body.classList.contains('modal-open') }
      const clientWidth = document.documentElement.clientWidth
      const scrollbar = clientWidth > 0 ? Math.max(0, window.innerWidth - clientWidth) : 0
      if (scrollbar) body.style.paddingRight = `${(parseFloat(getComputedStyle(body).paddingRight) || 0) + scrollbar}px`
      body.style.overflow = 'hidden'
      body.classList.add('modal-open')
    }
    owners.add(owner)
  }, { immediate: true, flush: 'sync' })
  onScopeDispose(release)
}
