import { useMotionPreferences } from './useMotionPreferences'

export function useReducedMotion() {
  return useMotionPreferences().reducedMotion
}
