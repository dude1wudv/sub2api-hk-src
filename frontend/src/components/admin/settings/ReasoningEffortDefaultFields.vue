<template>
  <div class="border-t border-gray-200 pt-4 dark:border-gray-700" data-testid="reasoning-default-fields">
    <div class="flex items-center justify-between gap-4">
      <div>
        <label class="text-sm font-medium text-gray-700 dark:text-gray-300">{{ t('admin.settings.gatewayForwarding.reasoningEffortDefault') }}</label>
        <p class="mt-0.5 text-xs text-gray-500 dark:text-gray-400">{{ t('admin.settings.gatewayForwarding.reasoningEffortDefaultHint') }}</p>
      </div>
      <Toggle :model-value="modelValue.enabled" @update:model-value="emit('update:modelValue', { ...modelValue, enabled: $event })" />
    </div>
    <div class="mt-3 overflow-x-auto">
      <table v-if="modelValue.rules.length" class="w-full text-left text-xs">
        <thead class="text-gray-500 dark:text-gray-400">
          <tr>
            <th class="pb-2">{{ t('admin.groups.form.platform') }}</th>
            <th class="pb-2">{{ t('admin.groups.form.reasoningEffortMatchType') }}</th>
            <th class="pb-2">{{ t('admin.groups.form.reasoningEffortModel') }}</th>
            <th class="pb-2">{{ t('usage.reasoningEffort') }}</th>
            <th></th>
          </tr>
        </thead>
        <tbody>
          <tr v-for="(rule, index) in modelValue.rules" :key="index">
            <td class="py-1 pr-2"><select class="input" :value="rule.platform" :aria-label="t('admin.groups.form.platform')" @change="updateRule(index, 'platform', $event)"><option v-for="platform in platforms" :key="platform" :value="platform">{{ platform }}</option></select></td>
            <td class="py-1 pr-2"><select class="input" :value="rule.match_type" :aria-label="t('admin.groups.form.reasoningEffortMatchType')" @change="updateRule(index, 'match_type', $event)"><option v-for="match in matchTypes" :key="match" :value="match">{{ match }}</option></select></td>
            <td class="py-1 pr-2"><input class="input min-w-44" :value="rule.model" :aria-label="t('admin.groups.form.reasoningEffortModel')" @input="updateRule(index, 'model', $event)" /></td>
            <td class="py-1 pr-2"><select class="input" :value="rule.effort" :aria-label="t('usage.reasoningEffort')" @change="updateRule(index, 'effort', $event)"><option v-for="effort in efforts" :key="effort" :value="effort">{{ effort }}</option></select></td>
            <td class="py-1"><button type="button" class="btn btn-secondary" @click="removeRule(index)">{{ t('common.delete') }}</button></td>
          </tr>
        </tbody>
      </table>
    </div>
    <button type="button" class="btn btn-secondary mt-2" data-testid="reasoning-default-add" @click="addRule">{{ t('admin.groups.form.addReasoningEffortMapping') }}</button>
  </div>
</template>

<script setup lang="ts">
import { useI18n } from 'vue-i18n'
import Toggle from '@/components/common/Toggle.vue'
import type { SystemSettings } from '@/api/admin/settings'

type Config = SystemSettings['gateway_reasoning_effort_default']
type Rule = Config['rules'][number]
const props = defineProps<{ modelValue: Config }>()
const emit = defineEmits<{ 'update:modelValue': [value: Config] }>()
const { t } = useI18n()
const platforms = ['openai', 'anthropic', 'composite']
const matchTypes = ['exact', 'prefix', 'suffix']
const efforts = ['minimal', 'low', 'medium', 'high', 'xhigh', 'max']

function updateRule(index: number, field: keyof Rule, event: Event) {
  const value = (event.target as HTMLInputElement).value
  emit('update:modelValue', { ...props.modelValue, rules: props.modelValue.rules.map((rule, i) => i === index ? { ...rule, [field]: value } : rule) })
}
function addRule() {
  emit('update:modelValue', { ...props.modelValue, rules: [...props.modelValue.rules, { platform: 'openai', match_type: 'prefix', model: 'gpt-6.1-sol', effort: 'high' }] })
}
function removeRule(index: number) {
  emit('update:modelValue', { ...props.modelValue, rules: props.modelValue.rules.filter((_, i) => i !== index) })
}
</script>
