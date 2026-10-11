<template>
  <Teleport v-if="container" :to="container">
    <span
      v-show="geometry.visible"
      class="active-indicator"
      :class="{ 'active-indicator--settled': settled && !reducedMotion }"
      :style="{ width: `${geometry.width}px`, height: `${geometry.height}px`, transform: `translate3d(${geometry.x}px, ${geometry.y}px, 0)` }"
      aria-hidden="true"
      data-active-indicator
    ><span :key="pulse" class="active-indicator__ink" :class="{ 'active-indicator__ink--arriving': pulse > 0 && !reducedMotion }" /></span>
  </Teleport>
</template>

<script setup lang="ts">
import { ref, watch } from 'vue'
import { useActiveIndicator } from '@/composables/useActiveIndicator'
import { useReducedMotion } from '@/composables/useReducedMotion'
const props = withDefaults(defineProps<{ container: HTMLElement | null; activeKey: string | number | null; selector?: string }>(), { selector: '[data-active-indicator-item]' })
const { geometry } = useActiveIndicator(() => props.container, () => props.activeKey, () => props.selector)
const reducedMotion = useReducedMotion()
const settled = ref(false)
const pulse = ref(0)
watch(geometry, (next, previous) => {
  settled.value = previous.visible && next.visible
  if (settled.value && next.key !== previous.key && !reducedMotion.value) pulse.value++
})
</script>

<style scoped>
.active-indicator { position: absolute; top: 0; left: 0; z-index: 0; display: block; pointer-events: none; border-radius: var(--active-indicator-radius, 12px); transform-origin: top left; }
.active-indicator--settled { transition: transform var(--motion-overlay, 260ms) var(--motion-ease, ease), width var(--motion-overlay, 260ms) var(--motion-ease, ease), height var(--motion-overlay, 260ms) var(--motion-ease, ease); }
.active-indicator__ink { display: block; width: 100%; height: 100%; border-radius: inherit; background: var(--active-indicator-fill, rgb(var(--brand-moss) / .12)); box-shadow: inset 0 0 0 1px var(--active-indicator-edge, rgb(var(--brand-moss) / .1)); transform-origin: center; }
.active-indicator__ink--arriving { animation: ink-settle var(--motion-page, 360ms) var(--motion-ease, ease) both; }
@keyframes ink-settle { 0% { transform: scale(1.04,.92); border-radius: 18px 10px 18px 10px; } 65% { transform: scale(.99,1.02); } 100% { transform: scale(1); border-radius: inherit; } }
[data-motion='reduced'] .active-indicator { transition: none; }
[data-motion='reduced'] .active-indicator__ink { animation: none; transform: none; }
@media (prefers-reduced-motion: reduce) { .active-indicator { transition: none; } .active-indicator__ink { animation: none; transform: none; } }
@media (forced-colors: active) { .active-indicator__ink { background: Canvas; border: 1px solid Highlight; box-shadow: none; } }
</style>
