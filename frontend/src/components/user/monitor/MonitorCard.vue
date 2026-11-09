<template>
  <article class="channel-card">
    <div class="channel-card-header">
      <button type="button" class="channel-card-identity" @click="emit('click')">
        <span class="channel-provider-icon" :class="providerTintClass"><ProviderIcon :provider="item.provider" :size="23" /></span>
        <span class="channel-card-name"><strong :title="item.name">{{ item.name }}</strong><span class="channel-card-meta"><span class="channel-provider-badge" :class="providerBadgeClass(item.provider)">{{ providerLabel(item.provider) }}</span><span class="channel-model">{{ formatMonitorModel(item.primary_model) }}</span></span></span>
      </button>
      <span class="channel-status" :class="statusBadgeClass(item.primary_status)">{{ statusLabel(item.primary_status) }}</span>
    </div>
    <div class="channel-metrics">
      <div :title="cacheHint"><span>{{ text('缓存率 · 本站', 'Cache · local usage') }}</span><strong data-testid="cache-rate">{{ cacheRate }}</strong></div>
      <div :title="availabilityLabel"><span>{{ text('可用率', 'Availability') }}</span><strong :style="availabilityStyle">{{ availabilityValue == null ? '—' : `${availabilityValue.toFixed(2)}%` }}</strong></div>
      <div :title="text('主动检测的完整响应耗时', 'Full response duration from active checks')"><span>{{ text('响应延迟', 'Response latency') }}</span><strong class="channel-latency">{{ item.primary_latency_ms == null ? '—' : `${(item.primary_latency_ms / 1000).toFixed(2)}s` }}</strong></div>
    </div>
    <p class="channel-metric-note">{{ text('缓存近 ', 'Cache: last ') }}{{ item.cache_window_minutes || 90 }}{{ text(' 分钟 · 可用率 ', ' min · Availability: ') }}{{ t(`channelStatus.windowTab.${window}`) }}</p>
    <MonitorQuotaView v-if="quotaVisible" :snapshot="item.latest_quota" class="mt-2" />
    <div class="channel-card-actions"><button v-if="qualityEnabled" type="button" class="channel-quality-button" @click="emit('quality')">{{ text('降智检测', 'Quality check') }}</button></div>
    <MonitorTimeline :buckets="item.timeline" :countdown-seconds="countdownSeconds" />
  </article>
</template>
<script setup lang="ts">
import { computed } from 'vue'
import { useI18n } from 'vue-i18n'
import type { UserMonitorView } from '@/api/channelMonitor'
import { useChannelMonitorFormat, hslForPct } from '@/composables/useChannelMonitorFormat'
import { isChannelMonitorQuotaVisible } from '@/utils/featureFlags'
import ProviderIcon from './ProviderIcon.vue'
import MonitorTimeline from './MonitorTimeline.vue'
import MonitorQuotaView from '@/components/common/MonitorQuotaView.vue'
const props = defineProps<{ item: UserMonitorView; window: '7d' | '15d' | '30d'; availabilityValue: number | null; countdownSeconds: number }>()
const emit = defineEmits<{ (event: 'click'): void; (event: 'quality'): void }>()
const { t, locale } = useI18n()
const text = (zh: string, en: string) => String(locale?.value || 'zh').startsWith('zh') ? zh : en
const { statusLabel, statusBadgeClass, providerLabel, providerBadgeClass, formatMonitorModel } = useChannelMonitorFormat()
const providerTintClass = computed(() => props.item.provider === 'openai' ? 'text-emerald-600 dark:text-emerald-300' : 'text-primary-600 dark:text-primary-300')
const quotaVisible = computed(() => isChannelMonitorQuotaVisible() && !!props.item.latest_quota)
const qualityEnabled = computed(() => !!props.item.group_id && [72, 237].includes(props.item.group_id))
const cacheRate = computed(() => props.item.cache_rate == null ? '—' : `${(props.item.cache_rate * 100).toFixed(2)}%`)
const cacheHint = computed(() => `${text('本站真实请求：', 'Actual local requests: ')}${props.item.cache_request_count ?? 0}`)
const availabilityLabel = computed(() => `${t('monitorCommon.availabilityPrefix')} · ${t(`channelStatus.windowTab.${props.window}`)}`)
const availabilityStyle = computed(() => ({ color: hslForPct(props.availabilityValue) || 'rgb(var(--ink-3))' }))
</script>
<style scoped>
.channel-card{display:flex;flex-direction:column;min-width:0;min-height:330px;padding:24px;border:1px solid rgb(var(--line));border-radius:28px;background:rgb(var(--surface));box-shadow:0 10px 32px rgba(35,43,36,.035);transition:box-shadow .2s,border-color .2s}.channel-card:hover{border-color:rgb(var(--line-strong));box-shadow:0 14px 38px rgba(35,43,36,.065)}.channel-card-header{display:flex;align-items:flex-start;justify-content:space-between;gap:12px}.channel-card-identity{display:flex;align-items:flex-start;gap:12px;min-width:0;text-align:left}.channel-provider-icon{display:grid;place-items:center;flex-shrink:0;width:44px;height:44px;border-radius:15px;background:rgba(16,185,129,.09)}.channel-card-name{display:block;min-width:0;padding-top:2px}.channel-card-name>strong{display:block;overflow:hidden;text-overflow:ellipsis;white-space:nowrap;font-size:17px;line-height:1.4;color:rgb(var(--ink));font-weight:650}.channel-card-meta{display:flex;align-items:center;flex-wrap:wrap;gap:6px;margin-top:7px}.channel-provider-badge{padding:2px 7px;border-radius:6px;font-size:11px}.channel-model{font:10px/1.5 monospace;color:rgb(var(--ink-3));overflow-wrap:anywhere}.channel-status{flex-shrink:0;padding:5px 10px;border-radius:999px;font-size:11px;font-weight:600}.channel-metrics{display:grid;grid-template-columns:repeat(3,minmax(0,1fr));gap:9px;margin-top:28px}.channel-metrics>div{display:flex;flex-direction:column;justify-content:space-between;gap:14px;min-width:0;padding:15px 12px;border:1px solid rgb(var(--line));border-radius:18px;background:rgb(var(--surface-2)/.35)}.channel-metrics span{color:rgb(var(--ink-3));font-size:10px;font-weight:500;white-space:nowrap}.channel-metrics strong{font:600 18px/1.2 ui-monospace,monospace;color:rgb(var(--ink));letter-spacing:-.6px}.channel-metrics .channel-latency{color:#168879}.channel-metric-note{margin:9px 0 0;font-size:10px;color:rgb(var(--ink-3))}.channel-card-actions{display:flex;justify-content:flex-end;min-height:36px;margin-top:13px}.channel-quality-button{padding:6px 12px;border:1px solid rgb(var(--line));border-radius:9px;background:rgb(var(--surface));color:rgb(var(--ink-2));font-size:12px;transition:background .15s,border-color .15s}.channel-quality-button:hover{background:rgb(var(--surface-2));border-color:#33b9a6}@media(max-width:420px){.channel-card{padding:18px}.channel-metrics>div{padding:13px 9px}.channel-metrics strong{font-size:16px}.channel-card-name>strong{font-size:15px}}@media(prefers-reduced-motion:reduce){.channel-card,.channel-quality-button{transition:none}}
</style>
