<template>
  <div class="border-t border-[var(--atelier-border)] pt-4" data-testid="reasoning-default-fields">
    <div class="flex items-center justify-between gap-4">
      <div>
        <label class="text-sm font-medium text-[var(--atelier-ink)]">{{ t('admin.settings.gatewayForwarding.reasoningEffortDefault') }}</label>
        <p class="mt-0.5 text-xs text-[var(--atelier-ink-secondary)]">{{ t('admin.settings.gatewayForwarding.reasoningEffortDefaultHint') }}</p>
      </div>
      <Toggle :model-value="modelValue.enabled" @update:model-value="emit('update:modelValue', { ...modelValue, enabled: $event })" />
    </div>
    <div class="mt-3 overflow-x-auto">
      <div v-if="mobile && modelValue.rules.length" class="mobile-records">
        <dl v-for="(rule, index) in modelValue.rules" :key="index" class="mobile-record">
          <div class="mobile-record-field">
            <dt>{{ t('admin.groups.form.platform') }}</dt>
            <dd>
              <UiSelect
                :model-value="rule.platform"
                :options="platformOptions"
                :aria-label="t('admin.groups.form.platform')"
                size="sm"
                @update:model-value="updateRuleValue(index, 'platform', String($event ?? ''))"
              />
            </dd>
          </div>
          <div class="mobile-record-field">
            <dt>{{ t('admin.groups.form.reasoningEffortMatchType') }}</dt>
            <dd>
              <UiSelect
                :model-value="rule.match_type"
                :options="matchTypeOptions"
                :aria-label="t('admin.groups.form.reasoningEffortMatchType')"
                size="sm"
                @update:model-value="updateRuleValue(index, 'match_type', String($event ?? ''))"
              />
            </dd>
          </div>
          <div class="mobile-record-field">
            <dt>{{ t('admin.groups.form.reasoningEffortModel') }}</dt>
            <dd>
              <input class="atelier-input min-w-44" :value="rule.model" :aria-label="t('admin.groups.form.reasoningEffortModel')" @input="updateRule(index, 'model', $event)" />
            </dd>
          </div>
          <div class="mobile-record-field">
            <dt>{{ t('usage.reasoningEffort') }}</dt>
            <dd>
              <UiSelect
                :model-value="rule.effort"
                :options="effortOptions"
                :aria-label="t('usage.reasoningEffort')"
                size="sm"
                @update:model-value="updateRuleValue(index, 'effort', String($event ?? ''))"
              />
            </dd>
          </div>
          <div class="mobile-record-field">
            <dt>{{ t('common.actions') }}</dt>
            <dd>
              <UiButton variant="secondary" size="sm" @click="removeRule(index)">{{ t('common.delete') }}</UiButton>
            </dd>
          </div>
        </dl>
      </div>
      <table v-else-if="modelValue.rules.length" class="atelier-table text-xs">
        <thead>
          <tr>
            <th>{{ t('admin.groups.form.platform') }}</th>
            <th>{{ t('admin.groups.form.reasoningEffortMatchType') }}</th>
            <th>{{ t('admin.groups.form.reasoningEffortModel') }}</th>
            <th>{{ t('usage.reasoningEffort') }}</th>
            <th></th>
          </tr>
        </thead>
        <tbody>
          <tr v-for="(rule, index) in modelValue.rules" :key="index">
            <td class="min-w-32">
              <UiSelect
                :model-value="rule.platform"
                :options="platformOptions"
                :aria-label="t('admin.groups.form.platform')"
                size="sm"
                @update:model-value="updateRuleValue(index, 'platform', String($event ?? ''))"
              />
            </td>
            <td class="min-w-28">
              <UiSelect
                :model-value="rule.match_type"
                :options="matchTypeOptions"
                :aria-label="t('admin.groups.form.reasoningEffortMatchType')"
                size="sm"
                @update:model-value="updateRuleValue(index, 'match_type', String($event ?? ''))"
              />
            </td>
            <td>
              <input class="atelier-input min-w-44" :value="rule.model" :aria-label="t('admin.groups.form.reasoningEffortModel')" @input="updateRule(index, 'model', $event)" />
            </td>
            <td class="min-w-28">
              <UiSelect
                :model-value="rule.effort"
                :options="effortOptions"
                :aria-label="t('usage.reasoningEffort')"
                size="sm"
                @update:model-value="updateRuleValue(index, 'effort', String($event ?? ''))"
              />
            </td>
            <td>
              <UiButton variant="secondary" size="sm" @click="removeRule(index)">{{ t('common.delete') }}</UiButton>
            </td>
          </tr>
        </tbody>
      </table>
    </div>
    <UiButton variant="secondary" size="sm" class="mt-2.5" data-testid="reasoning-default-add" @click="addRule">
      {{ t('admin.groups.form.addReasoningEffortMapping') }}
    </UiButton>
  </div>
</template>

<script setup lang="ts">
import { useMediaQuery } from '@vueuse/core'
import { MOBILE_CONTENT_QUERY } from '@/utils/responsive'
import { useI18n } from 'vue-i18n'
import Toggle from '@/components/common/Toggle.vue'
import { UiButton, UiSelect, type UiSelectOption } from '@/ui'
import type { SystemSettings } from '@/api/admin/settings'

type Config = SystemSettings['gateway_reasoning_effort_default']
type Rule = Config['rules'][number]
const props = defineProps<{ modelValue: Config }>()
const emit = defineEmits<{ 'update:modelValue': [value: Config] }>()
const { t } = useI18n()
const mobile = useMediaQuery(MOBILE_CONTENT_QUERY)
const platforms = ['openai', 'anthropic', 'composite']
const matchTypes = ['exact', 'prefix', 'suffix']
const efforts = ['minimal', 'low', 'medium', 'high', 'xhigh', 'max']

const platformOptions: UiSelectOption[] = platforms.map((p) => ({ value: p, label: p }))
const matchTypeOptions: UiSelectOption[] = matchTypes.map((m) => ({ value: m, label: m }))
const effortOptions: UiSelectOption[] = efforts.map((e) => ({ value: e, label: e }))

function updateRuleValue(index: number, field: keyof Rule, value: string) {
  emit('update:modelValue', { ...props.modelValue, rules: props.modelValue.rules.map((rule, i) => i === index ? { ...rule, [field]: value } : rule) })
}
function updateRule(index: number, field: keyof Rule, event: Event) {
  const value = (event.target as HTMLInputElement).value
  updateRuleValue(index, field, value)
}
function addRule() {
  emit('update:modelValue', { ...props.modelValue, rules: [...props.modelValue.rules, { platform: 'openai', match_type: 'prefix', model: 'gpt-6.1-sol', effort: 'high' }] })
}
function removeRule(index: number) {
  emit('update:modelValue', { ...props.modelValue, rules: props.modelValue.rules.filter((_, i) => i !== index) })
}
</script>
