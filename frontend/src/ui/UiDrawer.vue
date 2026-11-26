<template>
  <Teleport to="body">
    <Transition name="atelier-drawer">
      <div
        v-if="open"
        class="fixed inset-0 z-[90] flex justify-end"
        role="dialog"
        aria-modal="true"
        :aria-label="title"
      >
        <div
          class="atelier-drawer-backdrop fixed inset-0 bg-[var(--atelier-scrim)] transition-opacity"
          @click="closeOnBackdrop && $emit('close')"
        />
        <aside
          ref="panelRef"
          :class="[
            'atelier-drawer-panel relative z-10 flex h-full w-full flex-col border-l border-[var(--atelier-border-strong)] bg-[var(--atelier-surface)] shadow-[var(--atelier-shadow-drawer)]',
            widthClass
          ]"
        >
          <header class="flex h-[56px] shrink-0 items-center justify-between gap-3 border-b border-[var(--atelier-border)] px-6">
            <div class="min-w-0">
              <slot name="header">
                <h2 class="truncate text-base font-semibold tracking-tight text-[var(--atelier-ink)]">
                  {{ title }}
                </h2>
                <p v-if="subtitle" class="truncate text-xs text-[var(--atelier-ink-muted)]">
                  {{ subtitle }}
                </p>
              </slot>
            </div>
            <button
              type="button"
              class="inline-flex h-8 w-8 items-center justify-center rounded-lg text-[var(--atelier-ink-secondary)] transition-colors hover:bg-[var(--atelier-hover)] hover:text-[var(--atelier-ink)]"
              aria-label="Close"
              @click="$emit('close')"
            >
              <svg class="h-4 w-4" viewBox="0 0 16 16" fill="none" stroke="currentColor" stroke-width="1.75">
                <path d="m4 4 8 8M12 4l-8 8" stroke-linecap="round" />
              </svg>
            </button>
          </header>

          <div class="flex-1 overflow-y-auto px-6 py-5">
            <slot />
          </div>

          <footer
            v-if="$slots.footer"
            class="flex shrink-0 items-center justify-end gap-2.5 border-t border-[var(--atelier-border)] bg-[var(--atelier-surface)] px-6 py-3.5"
          >
            <slot name="footer" />
          </footer>
        </aside>
      </div>
    </Transition>
  </Teleport>
</template>

<script setup lang="ts">
import { computed, onBeforeUnmount, provide, ref, watch } from 'vue'
import { overlayHostKey } from '@/composables/overlayHost'
import { useScrollLock } from '@/composables/useScrollLock'

const props = withDefaults(
  defineProps<{
    open: boolean
    title?: string
    subtitle?: string
    width?: 'md' | 'lg' | 'xl'
    closeOnBackdrop?: boolean
    closeOnEscape?: boolean
  }>(),
  {
    title: '',
    subtitle: '',
    width: 'lg',
    closeOnBackdrop: true,
    closeOnEscape: true
  }
)

const emit = defineEmits<{
  close: []
}>()

const panelRef = ref<HTMLElement | null>(null)
provide(overlayHostKey, panelRef)
useScrollLock(computed(() => props.open))

const widthClass = computed(() => {
  switch (props.width) {
    case 'md':
      return 'max-w-[440px]'
    case 'xl':
      return 'max-w-[640px]'
    default:
      return 'max-w-[540px]'
  }
})

function onKeydown(e: KeyboardEvent) {
  if (props.open && props.closeOnEscape && e.key === 'Escape') {
    emit('close')
  }
}

watch(
  () => props.open,
  (isOpen) => {
    if (typeof window === 'undefined') return
    if (isOpen) window.addEventListener('keydown', onKeydown)
    else window.removeEventListener('keydown', onKeydown)
  },
  { immediate: true }
)

onBeforeUnmount(() => {
  if (typeof window !== 'undefined') {
    window.removeEventListener('keydown', onKeydown)
  }
})
</script>

<style scoped>
.atelier-drawer-enter-active,
.atelier-drawer-leave-active {
  transition: opacity 180ms cubic-bezier(0.2, 0.8, 0.2, 1);
}
.atelier-drawer-enter-active .atelier-drawer-panel,
.atelier-drawer-leave-active .atelier-drawer-panel {
  transition: transform 220ms cubic-bezier(0.2, 0.8, 0.2, 1);
}
.atelier-drawer-enter-from,
.atelier-drawer-leave-to {
  opacity: 0;
}
.atelier-drawer-enter-from .atelier-drawer-panel,
.atelier-drawer-leave-to .atelier-drawer-panel {
  transform: translateX(100%);
}
</style>
