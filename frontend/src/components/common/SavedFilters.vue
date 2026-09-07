<template>
  <div ref="rootRef" class="relative inline-flex">
    <button
      type="button"
      class="inline-flex h-8 items-center gap-1.5 rounded-md border border-gray-200 bg-white px-2.5 text-xs font-medium text-gray-600 transition-colors hover:bg-gray-50 focus:outline-none focus-visible:ring-2 focus-visible:ring-primary-500/40 dark:border-dark-700 dark:bg-dark-900 dark:text-dark-200 dark:hover:bg-dark-800"
      :aria-expanded="open"
      aria-haspopup="dialog"
      :aria-label="t('console.table.savedFilters')"
      @click="toggle"
    >
      <Icon name="filter" size="sm" aria-hidden="true" />
      {{ t('console.table.savedFilters') }}
    </button>
    <div
      v-if="open"
      class="absolute right-0 z-40 mt-1 w-64 rounded-lg border border-gray-200 bg-white p-2 shadow-lg dark:border-dark-700 dark:bg-dark-900"
      role="dialog"
      :aria-label="t('console.table.savedFilters')"
      @keydown.esc.prevent="close"
    >
      <div class="flex gap-1.5">
        <input
          v-model="name"
          class="input h-8 min-w-0 flex-1 px-2 text-xs"
          :placeholder="t('console.table.filterNamePlaceholder')"
          @keydown.enter.prevent="save"
        />
        <button type="button" class="btn btn-primary h-8 px-2 text-xs" :disabled="!name.trim()" @click="save">
          {{ t('console.table.saveFilter') }}
        </button>
      </div>
      <div v-if="items.length" class="mt-2 max-h-52 space-y-0.5 overflow-y-auto border-t border-gray-100 pt-2 dark:border-dark-700">
        <div v-for="item in items" :key="item.id" class="group flex min-h-8 items-center gap-1 rounded px-1 hover:bg-gray-50 dark:hover:bg-dark-800">
          <button
            type="button"
            class="min-w-0 flex-1 truncate px-1 text-left text-sm text-gray-700 focus:outline-none focus-visible:ring-2 focus-visible:ring-primary-500/40 dark:text-dark-200"
            @click="apply(item.id)"
          >
            {{ item.name }}
          </button>
          <button
            type="button"
            class="inline-flex h-6 w-6 items-center justify-center rounded text-gray-400 opacity-0 transition-opacity hover:bg-gray-200 hover:text-gray-700 focus:opacity-100 focus:outline-none focus-visible:ring-2 focus-visible:ring-primary-500/40 group-hover:opacity-100 dark:hover:bg-dark-700 dark:hover:text-dark-100"
            :aria-label="t('common.remove')"
            @click.stop="remove(item.id)"
          >
            <Icon name="x" size="xs" aria-hidden="true" />
          </button>
        </div>
      </div>
      <p v-else class="px-1 py-3 text-xs text-gray-500 dark:text-dark-400">{{ t('console.table.noSavedFilters') }}</p>
    </div>
  </div>
</template>

<script setup lang="ts">
import { onMounted, onUnmounted, ref } from 'vue'
import { useI18n } from 'vue-i18n'
import Icon from '@/components/icons/Icon.vue'
import type { SavedTableFilter } from '@/composables/useTablePreferences'

interface Props {
  items: SavedTableFilter[]
}

const props = defineProps<Props>()
const emit = defineEmits<{ save: [name: string]; apply: [id: string]; remove: [id: string] }>()
const { t } = useI18n()
const rootRef = ref<HTMLElement | null>(null)
const open = ref(false)
const name = ref('')

const close = () => {
  open.value = false
}

const toggle = () => {
  open.value = !open.value
}

const save = () => {
  const nextName = name.value.trim()
  if (!nextName) return
  emit('save', nextName)
  name.value = ''
}

const apply = (id: string) => {
  emit('apply', id)
  close()
}

const remove = (id: string) => {
  emit('remove', id)
}

const handleDocumentPointerDown = (event: PointerEvent) => {
  if (!rootRef.value?.contains(event.target as Node)) close()
}

onMounted(() => document.addEventListener('pointerdown', handleDocumentPointerDown))
onUnmounted(() => document.removeEventListener('pointerdown', handleDocumentPointerDown))
</script>
