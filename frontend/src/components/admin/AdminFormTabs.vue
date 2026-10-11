<template>
  <div ref="container" class="admin-form-tabs admin-form-tabs--ink" role="tablist" :aria-label="zh ? '编辑分类' : 'Editor sections'">
    <ActiveIndicator :container="container" :active-key="modelValue" />
    <button v-for="(tab, index) in tabs" :id="`${id}-${tab.id}-tab`" :key="tab.id" :data-active-indicator-item="tab.id" type="button" role="tab" :aria-selected="modelValue === tab.id" :aria-controls="`${id}-${tab.id}-panel`" :tabindex="modelValue === tab.id ? 0 : -1" @click="$emit('update:modelValue', tab.id)" @keydown="onKey($event, index)">{{ zh ? tab.zh : tab.en }}</button>
  </div>
</template>
<script setup lang="ts">
import { computed, nextTick, ref } from 'vue'
import { useI18n } from 'vue-i18n'
import type { AdminFormTab } from '@/composables/useAdminFormTabs'
import ActiveIndicator from '@/components/common/ActiveIndicator.vue'
const container = ref<HTMLElement | null>(null)
const props = defineProps<{ modelValue: string; id: string; tabs: AdminFormTab[] }>()
const emit = defineEmits<{ 'update:modelValue': [value: string] }>()
const { locale } = useI18n()
const zh = computed(() => locale?.value?.startsWith('zh'))
function onKey(event: KeyboardEvent, index: number) {
  if (!['ArrowLeft', 'ArrowRight', 'Home', 'End'].includes(event.key)) return
  if (!props.tabs.length) return
  event.preventDefault()
  const next = event.key === 'Home' ? 0 : event.key === 'End' ? props.tabs.length - 1 : (index + (event.key === 'ArrowRight' ? 1 : -1) + props.tabs.length) % props.tabs.length
  const tab = props.tabs[next]
  emit('update:modelValue', tab.id)
  void nextTick(() => document.getElementById(`${props.id}-${tab.id}-tab`)?.focus())
}
</script>
<style scoped>
.admin-form-tabs--ink { position: relative; isolation: isolate; gap: 6px; padding: 4px; border: 1px solid rgb(var(--line)); border-radius: 16px; }
.admin-form-tabs--ink > button { position: relative; z-index: 1; min-height: 44px; padding: 9px 14px; border: 0; border-radius: 12px; background: transparent; }
.admin-form-tabs--ink > button[aria-selected='true'] { border: 0; background: transparent; color: rgb(var(--ink)); }
</style>
