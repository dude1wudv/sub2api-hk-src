<template>
  <div class="md-tabs-shell" :class="{ 'md-tabs-shell--bordered': bordered }">
    <component
      :is="'md-tabs'"
      ref="tabsRef"
      class="md-tabs-host"
      :active-tab-index="activeIndex"
      :aria-label="ariaLabel || undefined"
      @change="onChange"
    >
      <component
        :is="tabTag"
        v-for="item in items"
        :key="item.id"
        :id="`md-tab-${item.id}`"
        :active="item.id === modelValue || undefined"
        :disabled="item.disabled || undefined"
        :inline-icon="inlineIcon || undefined"
        @click="selectTab(item)"
      >
        <span class="md-tab-content">
          <span>{{ item.label }}</span>
          <span v-if="item.badge !== undefined && item.badge !== null && item.badge !== ''" class="md-tab-badge">
            {{ item.badge }}
          </span>
        </span>
      </component>
    </component>
  </div>
</template>

<script setup lang="ts">
import { computed, ref, watch } from 'vue'
import '@material/web/tabs/tabs.js'
import '@material/web/tabs/primary-tab.js'
import '@material/web/tabs/secondary-tab.js'

export interface MdTabItem {
  id: string
  label: string
  badge?: string | number | null
  disabled?: boolean
}

const props = withDefaults(
  defineProps<{
    modelValue: string
    items: MdTabItem[]
    variant?: 'primary' | 'secondary'
    inlineIcon?: boolean
    bordered?: boolean
    ariaLabel?: string
  }>(),
  {
    variant: 'primary',
    inlineIcon: true,
    bordered: true
  }
)

const emit = defineEmits<{
  (e: 'update:modelValue', id: string): void
  (e: 'change', id: string): void
}>()

const tabsRef = ref<(HTMLElement & { activeTabIndex?: number }) | null>(null)
const tabTag = computed(() => (props.variant === 'secondary' ? 'md-secondary-tab' : 'md-primary-tab'))
const activeIndex = computed(() => {
  const idx = props.items.findIndex((item) => item.id === props.modelValue)
  return idx >= 0 ? idx : 0
})

watch(activeIndex, (idx) => {
  if (tabsRef.value && tabsRef.value.activeTabIndex !== idx) {
    tabsRef.value.activeTabIndex = idx
  }
})

function selectTab(item: MdTabItem) {
  if (item.disabled || item.id === props.modelValue) return
  emit('update:modelValue', item.id)
  emit('change', item.id)
}

function onChange(event: Event) {
  const target = event.target as HTMLElement & { activeTabIndex?: number }
  const idx = typeof target?.activeTabIndex === 'number' ? target.activeTabIndex : -1
  const next = props.items[idx]
  if (next && !next.disabled && next.id !== props.modelValue) {
    emit('update:modelValue', next.id)
    emit('change', next.id)
  }
}
</script>

<style scoped>
.md-tabs-shell {
  width: 100%;
  overflow-x: auto;
  scrollbar-width: none;
}

.md-tabs-shell::-webkit-scrollbar {
  display: none;
}

.md-tabs-shell--bordered {
  border-bottom: 1px solid var(--md-sys-color-outline-variant);
}

.md-tabs-host {
  --md-primary-tab-container-color: transparent;
  --md-secondary-tab-container-color: transparent;
  --md-divider-color: transparent;
  --md-primary-tab-label-text-font: var(--font-sans);
  --md-secondary-tab-label-text-font: var(--font-sans);
}

.md-tab-content {
  display: inline-flex;
  align-items: center;
  gap: 6px;
  white-space: nowrap;
}

.md-tab-badge {
  display: inline-flex;
  align-items: center;
  justify-content: center;
  min-width: 18px;
  height: 18px;
  padding: 0 6px;
  border-radius: 9999px;
  font-size: 0.7rem;
  font-weight: 600;
  background: var(--md-sys-color-secondary-container);
  color: var(--md-sys-color-on-secondary-container);
}
</style>
