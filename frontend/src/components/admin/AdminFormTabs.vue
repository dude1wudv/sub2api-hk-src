<template>
  <div class="admin-form-tabs" role="tablist" :aria-label="zh ? '编辑分类' : 'Editor sections'">
    <button v-for="(tab, index) in tabs" :id="`${id}-${tab.id}-tab`" :key="tab.id" type="button" role="tab" :aria-selected="modelValue === tab.id" :aria-controls="`${id}-${tab.id}-panel`" :tabindex="modelValue === tab.id ? 0 : -1" @click="$emit('update:modelValue', tab.id)" @keydown="onKey($event, index)">{{ zh ? tab.zh : tab.en }}</button>
  </div>
</template>
<script setup lang="ts">
import { computed, nextTick } from 'vue'
import { useI18n } from 'vue-i18n'
import type { AdminFormTab } from '@/composables/useAdminFormTabs'
const props = defineProps<{ modelValue: string; id: string; tabs: AdminFormTab[] }>()
const emit = defineEmits<{ 'update:modelValue': [value: string] }>()
const { locale } = useI18n()
const zh = computed(() => locale?.value?.startsWith('zh'))
function onKey(event: KeyboardEvent, index: number) {
  if (!['ArrowLeft', 'ArrowRight', 'Home', 'End'].includes(event.key)) return
  event.preventDefault()
  const next = event.key === 'Home' ? 0 : event.key === 'End' ? props.tabs.length - 1 : (index + (event.key === 'ArrowRight' ? 1 : -1) + props.tabs.length) % props.tabs.length
  const tab = props.tabs[next]
  emit('update:modelValue', tab.id)
  void nextTick(() => document.getElementById(`${props.id}-${tab.id}-tab`)?.focus())
}
</script>
