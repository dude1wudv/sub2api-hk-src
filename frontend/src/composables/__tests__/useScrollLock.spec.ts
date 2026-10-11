import { afterEach, describe, expect, it } from 'vitest'
import { effectScope, ref, type EffectScope } from 'vue'
import { useScrollLock } from '../useScrollLock'

const scopes: EffectScope[] = []
const openLock = (open = true) => {
  const active = ref(open)
  const scope = effectScope()
  scopes.push(scope)
  scope.run(() => useScrollLock(active))
  return { active, scope }
}
afterEach(() => { scopes.splice(0).forEach(scope => scope.stop()); document.body.removeAttribute('style'); document.body.classList.remove('modal-open') })

describe('shared overlay scroll lock', () => {
  it('keeps nested independent owners locked until the last closes and restores preexisting styles', () => {
    document.body.style.overflow = 'auto'
    document.body.style.paddingRight = '7px'
    const drawer = openLock()
    const dialog = openLock()
    const hidden = openLock(false)
    hidden.scope.stop()
    drawer.active.value = false
    expect(document.body.style.overflow).toBe('hidden')
    expect(document.body.classList.contains('modal-open')).toBe(true)
    dialog.scope.stop()
    expect(document.body.style.overflow).toBe('auto')
    expect(document.body.style.paddingRight).toBe('7px')
    expect(document.body.classList.contains('modal-open')).toBe(false)
  })

  it('does not remove a lock class owned by a preexisting integration', () => {
    document.body.classList.add('modal-open')
    const dialog = openLock()
    dialog.active.value = false
    expect(document.body.classList.contains('modal-open')).toBe(true)
  })
})
