<template>
  <BaseDialog :show="accountId !== null" :title="detail?.name || (zh ? '账号详情' : 'Account details')" presentation="drawer" @close="$emit('close')">
    <div v-if="loading" class="py-12 text-center" role="status">{{ t('common.loading') }}</div>
    <div v-else-if="error" role="alert"><p>{{ error }}</p><button class="btn btn-secondary mt-4" @click="load">{{ t('common.refresh') }}</button></div>
    <div v-else-if="detail" class="admin-account-details">
      <div class="flex items-center gap-3 text-sm"><span class="font-mono text-gray-500">#{{ detail.id }}</span><PlatformTypeBadge :platform="detail.platform" :type="detail.type" /><span>{{ t(`admin.accounts.status.${detail.status}`, detail.status) }}</span></div>
      <section><h4>{{ zh ? '基本信息' : 'General' }}</h4><dl>
        <dt>{{ zh ? '分组' : 'Groups' }}</dt><dd>{{ detail.groups?.map(g => g.name).join(' · ') || detail.group_ids?.map(id => `#${id}`).join(' · ') || '—' }}</dd>
        <dt>{{ zh ? '代理' : 'Proxy' }}</dt><dd>{{ detail.proxy?.name || (detail.proxy_id ? `#${detail.proxy_id}` : '—') }}</dd>
        <dt>{{ zh ? '有效期' : 'Expires' }}</dt><dd>{{ detail.expires_at ? date(new Date(detail.expires_at * 1000).toISOString()) : '—' }}</dd>
        <dt>{{ zh ? '创建时间' : 'Created' }}</dt><dd>{{ date(detail.created_at) }}</dd>
        <dt>{{ zh ? '最后使用' : 'Last used' }}</dt><dd>{{ date(detail.last_used_at) }}</dd>
        <dt>{{ zh ? '备注' : 'Notes' }}</dt><dd class="whitespace-pre-wrap">{{ detail.notes || '—' }}</dd>
      </dl></section>
      <section><h4>{{ zh ? '用量与额度' : 'Usage & quotas' }}</h4><AccountUsageCell :account="detail" />
        <dl class="mt-4"><template v-for="row in quotas" :key="row.label"><dt>{{ row.label }}</dt><dd>{{ row.value }}</dd></template>
          <dt>{{ zh ? '计费倍率' : 'Billing multiplier' }}</dt><dd>{{ detail.rate_multiplier ?? 1 }}×</dd>
          <dt>{{ zh ? '窗口重置' : 'Window reset' }}</dt><dd>{{ date(detail.session_window_end) }}</dd>
        </dl>
        <button class="btn btn-secondary mt-4" @click="$emit('stats', detail)">{{ t('admin.accounts.usageStatistics') }}</button>
      </section>
      <section><h4>{{ zh ? '调度状态' : 'Scheduling' }}</h4><dl>
        <dt>{{ zh ? '可调度' : 'Schedulable' }}</dt><dd>{{ detail.schedulable ? t('common.yes') : t('common.no') }}</dd>
        <dt>{{ zh ? '优先级' : 'Priority' }}</dt><dd>{{ detail.priority }}</dd>
        <dt>{{ zh ? '并发数' : 'Concurrency' }}</dt><dd>{{ detail.current_concurrency ?? '—' }} / {{ detail.concurrency }}</dd>
        <dt>{{ zh ? '限流恢复' : 'Rate limit reset' }}</dt><dd>{{ date(detail.rate_limit_reset_at) }}</dd>
        <dt>{{ zh ? '临时暂停' : 'Temporary pause' }}</dt><dd>{{ date(detail.temp_unschedulable_until) }}<p v-if="detail.temp_unschedulable_reason">{{ detail.temp_unschedulable_reason }}</p></dd>
        <template v-for="score in detail.scheduler_scores || []" :key="score.group_id ?? score.group_name"><dt>{{ score.group_name || `#${score.group_id}` }}</dt><dd>{{ score.base_score }}<span v-if="score.sticky_weighted_enabled"> / {{ score.sticky_score_infinity ? '∞' : score.sticky_score }}</span></dd></template>
      </dl><p v-if="detail.error_message" class="mt-4 text-red-700" role="status">{{ detail.error_message }}</p></section>
    </div>
    <template #footer><div class="flex justify-between gap-3"><button class="btn btn-secondary" @click="$emit('close')">{{ t('common.close') }}</button><button v-if="detail && !loading && !error" class="btn btn-primary" @click="$emit('edit', detail)">{{ t('admin.accounts.editAccount') }}</button></div></template>
  </BaseDialog>
</template>
<script setup lang="ts">
import { computed, ref, watch, onBeforeUnmount } from 'vue'
import { useI18n } from 'vue-i18n'
import { adminAPI } from '@/api/admin'
import type { Account } from '@/types'
import BaseDialog from '@/components/common/BaseDialog.vue'
import PlatformTypeBadge from '@/components/common/PlatformTypeBadge.vue'
import AccountUsageCell from './AccountUsageCell.vue'
const props = defineProps<{ accountId: number | null }>()
defineEmits<{ close: []; edit: [account: Account]; stats: [account: Account] }>()
const { t, locale } = useI18n()
const zh = computed(() => locale?.value?.startsWith('zh'))
const detail = ref<Account | null>(null), loading = ref(false), error = ref('')
let request = 0
async function load() {
  const version = ++request, id = props.accountId
  detail.value = null; error.value = ''; loading.value = id !== null
  if (id === null) return
  try { const result = await adminAPI.accounts.getById(id); if (version === request) detail.value = result }
  catch { if (version === request) error.value = zh.value ? '无法加载账号详情，请重试。' : 'Could not load account details. Please retry.' }
  finally { if (version === request) loading.value = false }
}
watch(() => props.accountId, load, { immediate: true })
onBeforeUnmount(() => { request++ })
const date = (value?: string | null) => value ? new Date(value).toLocaleString(locale?.value === 'zh' ? 'zh-CN' : 'en-US') : '—'
const quotas = computed(() => {
  if (!detail.value) return []
  const a = detail.value
  return [
    { label: zh.value ? '总额度' : 'Total quota', used: a.quota_used, limit: a.quota_limit },
    { label: zh.value ? '每日额度' : 'Daily quota', used: a.quota_daily_used, limit: a.quota_daily_limit },
    { label: zh.value ? '每周额度' : 'Weekly quota', used: a.quota_weekly_used, limit: a.quota_weekly_limit },
    { label: zh.value ? '当前窗口' : 'Current window', used: a.current_window_cost, limit: a.window_cost_limit }
  ].filter(row => row.used != null || row.limit != null).map(row => ({ label: row.label, value: `$${(row.used ?? 0).toFixed(4)} / ${row.limit ? '$' + row.limit.toFixed(2) : '—'}` }))
})
</script>
