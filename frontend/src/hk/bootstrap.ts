import { initColorMode } from '@/composables/useColorMode'
import { initMotionPreferences } from '@/composables/useMotionPreferences'
import '@/styles/design-system.css'
import '@/styles/patrick.css'
import '@/styles/portal.css'

/** Fork presentation bootstrap. Upstream theme and iOS viewport setup stay out of main.ts. */
export function initHkFrontend() {
  initColorMode()
  initMotionPreferences()
}
