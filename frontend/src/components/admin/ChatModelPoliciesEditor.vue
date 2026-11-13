<template>
  <section class="space-y-2 rounded-lg border border-gray-200 p-4 dark:border-dark-600">
    <label class="block space-y-2">
      <span class="text-sm font-medium text-gray-900 dark:text-gray-100">{{ t('admin.groups.chatPolicies.title') }}</span>
      <span class="block text-xs text-gray-500 dark:text-gray-400">{{ t('admin.groups.chatPolicies.hint') }}</span>
      <textarea
        :value="draft"
        :aria-invalid="Boolean(error)"
        :aria-label="t('admin.groups.chatPolicies.title')"
        rows="8"
        spellcheck="false"
        class="input w-full font-mono text-xs"
        data-testid="chat-model-policies"
        @input="onInput"
      />
    </label>
    <p v-if="error" class="text-xs text-red-600 dark:text-red-400" role="alert">{{ error }}</p>
    <details class="text-xs text-gray-500 dark:text-gray-400">
      <summary class="cursor-pointer">{{ t('admin.groups.chatPolicies.example') }}</summary>
      <pre class="mt-2 overflow-x-auto whitespace-pre-wrap">{{ example }}</pre>
    </details>
  </section>
</template>

<script setup lang="ts">
import { ref, toRaw, watch } from 'vue'
import { useI18n } from 'vue-i18n'
import type { ChatModelPolicies } from '@/types'
import { parseChatModelPolicies } from '@/utils/chatModelPolicies'

const props = defineProps<{ modelValue: ChatModelPolicies }>()
const emit = defineEmits<{ 'update:modelValue': [value: ChatModelPolicies] }>()
const { t } = useI18n()
const draft = ref('{}')
const error = ref('')
let published: ChatModelPolicies | undefined
const example = JSON.stringify({
  'public-model-name': {
    transport: 'chat_completions', image_input: false, document_input: false,
    native_deepseek_extensions: false, native_files: false, native_search: false, native_read_url: false,
  },
}, null, 2)

watch(() => props.modelValue, value => {
  if (toRaw(value) === published) return
  draft.value = JSON.stringify(value ?? {}, null, 2)
  error.value = ''
}, { immediate: true })

function validate(): boolean {
  try {
    const value = parseChatModelPolicies(draft.value)
    published = value
    emit('update:modelValue', value)
    error.value = ''
    return true
  } catch {
    error.value = t('admin.groups.chatPolicies.invalid')
    return false
  }
}

function onInput(event: Event) {
  const target = event.target
  if (!(target instanceof HTMLTextAreaElement)) return
  draft.value = target.value
  validate()
}

defineExpose({ validate })
</script>
