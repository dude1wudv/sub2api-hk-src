import { initColorMode } from '@/composables/useColorMode'
import { initMotionPreferences } from '@/composables/useMotionPreferences'
import '@/styles/atelier/index.css'

/** Atelier presentation bootstrap — completely decoupled from legacy portal/design-system CSS. */
export function initHkFrontend() {
  initColorMode()
  initMotionPreferences()
}
