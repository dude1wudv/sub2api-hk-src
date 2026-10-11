<template>
  <component
    :is="as"
    ref="surface"
    v-bind="$attrs"
    class="ink-surface"
    :type="as === 'button' ? ($attrs.type || 'button') : undefined"
    data-ink-surface
    :data-ink-interactive="interactive || magnetic"
    :data-ink-lift="interactive && as !== 'button' && as !== 'a'"
  >
    <span class="ink-surface__wash" aria-hidden="true"><span class="ink-surface__liquid" /></span>
    <span v-if="magnetic" class="ink-surface__magnet"><slot /></span>
    <slot v-else />
  </component>
</template>

<script setup lang="ts">
import { ref } from 'vue'
import { usePointerSurface } from '@/composables/usePointerSurface'
defineOptions({ inheritAttrs: false })
const props = withDefaults(defineProps<{ as?: string; interactive?: boolean; magnetic?: boolean }>(), { as: 'div', interactive: false, magnetic: false })
const surface = ref<HTMLElement | null>(null)
usePointerSurface(surface, {
  enabled: () => props.interactive || props.magnetic,
  magnetic: () => props.magnetic,
  lift: () => props.interactive && props.as !== 'button' && props.as !== 'a',
})
</script>

<style scoped>
.ink-surface { position: relative; isolation: isolate; }
.ink-surface__wash { position: absolute; inset: 0; z-index: -1; overflow: clip; contain: paint; border-radius: inherit; pointer-events: none; opacity: 0; transition: opacity var(--motion-menu, 180ms) var(--motion-ease, ease); }
.ink-surface__liquid { position: absolute; inset: 0; border-radius: inherit; pointer-events: none; background: radial-gradient(260px 180px at var(--ink-pointer-x, 50%) var(--ink-pointer-y, 50%), rgb(var(--brand-moss) / .2), transparent 76%), radial-gradient(160px 240px at var(--ink-pointer-x, 50%) var(--ink-pointer-y, 50%), rgb(var(--brand-copper) / .12), transparent 80%); transform: scale(.985); transition: transform var(--motion-menu, 180ms) var(--motion-ease, ease); }
.ink-surface[data-pointer-active='true'] > .ink-surface__wash { opacity: 1; }
.ink-surface[data-pointer-active='true'] { --ink-3: var(--ink-2); }
.ink-surface[data-pointer-active='true'] > .ink-surface__wash > .ink-surface__liquid { transform: translateY(var(--ink-lift, 0px)) scale(1.02); }
.ink-surface__magnet { display: inline-flex; align-items: center; justify-content: inherit; gap: inherit; transform: translate(var(--ink-magnet-x, 0px), var(--ink-magnet-y, 0px)); transition: transform var(--motion-fast, 160ms) var(--motion-ease, ease); pointer-events: none; }
[data-motion='reduced'] .ink-surface { translate: none; transition: none; }
[data-motion='reduced'] .ink-surface__magnet { transform: none; transition: none; }
[data-motion='reduced'] .ink-surface__wash { display: none; }
@media (prefers-reduced-motion: reduce), (hover: none), (pointer: coarse), (max-width: 767px) {
  .ink-surface { translate: none; transition: none; }
  .ink-surface__magnet { transform: none; transition: none; }
  .ink-surface__wash { display: none; }
}
</style>
