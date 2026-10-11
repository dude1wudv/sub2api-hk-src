<script setup lang="ts">
import { computed } from 'vue'
import { useI18n } from 'vue-i18n'
import type { UsageLog } from '@/types'
import Icon from '@/components/icons/Icon.vue'
import {
  isTokenUsage, usageBillingLabel, usageBillingMode,
  formatUsageNumber, formatUsageRate, formatUsagePercent,
  usageCacheRate, usageOutputStageRate,
} from '@/utils/portalUsage'

defineProps<{ row: UsageLog }>()
const emit = defineEmits<{ detail: [row: UsageLog] }>()
const { locale } = useI18n()
const zh = computed(() => locale.value.startsWith('zh'))
const copy = (cn: string, en: string) => zh.value ? cn : en
function compact(value: number) {
  return value >= 1000000 ? `${(value / 1000000).toFixed(1)}M` : value >= 1000 ? `${(value / 1000).toFixed(1)}K` : formatUsageNumber(value)
}
</script>

<template>
  <div v-if="isTokenUsage(row)" class="usage-token-cell">
    <div class="usage-token-counts">
      <span :title="`${copy('输入', 'Input')} ${formatUsageNumber(row.input_tokens)} Token`"><Icon name="arrowDown" size="sm" class="usage-input-icon" aria-hidden="true" /> {{ formatUsageNumber(row.input_tokens) }}</span>
      <span :title="`${copy('输出', 'Output')} ${formatUsageNumber(row.output_tokens)} Token`"><Icon name="arrowUp" size="sm" class="usage-output-icon" aria-hidden="true" /> {{ formatUsageNumber(row.output_tokens) }}</span>
    </div>
    <div v-if="row.cache_read_tokens > 0 || row.cache_creation_tokens > 0" class="usage-cache">
      <span v-if="row.cache_read_tokens > 0" :title="`${copy('缓存读取', 'Cache read')} ${formatUsageNumber(row.cache_read_tokens)} Token`" class="usage-cache-read"><Icon name="inbox" size="sm" aria-hidden="true" />{{ compact(row.cache_read_tokens) }}</span>
      <span v-if="row.cache_creation_tokens > 0" :title="`${copy('缓存写入', 'Cache write')} ${formatUsageNumber(row.cache_creation_tokens)} Token`" class="usage-cache-write"><Icon name="database" size="sm" aria-hidden="true" />{{ compact(row.cache_creation_tokens) }}<i v-if="row.cache_creation_1h_tokens > 0">1h</i><i v-if="row.cache_ttl_overridden" :title="copy('缓存 TTL 已被覆盖', 'Cache TTL overridden')">R</i></span>
    </div>
    <small v-if="row.image_input_tokens > 0">{{ copy('图像输入', 'Image input') }} {{ formatUsageNumber(row.image_input_tokens) }} Token</small>
    <small v-if="row.image_output_tokens > 0">{{ copy('图像输出', 'Image output') }} {{ formatUsageNumber(row.image_output_tokens) }} Token</small>
    <div class="usage-token-rates">
      <span :title="copy('缓存命中率 = 缓存读取 / (输入 + 缓存写入 + 缓存读取)', 'Cache hit rate = cache read / (input + cache write + cache read)')">{{ copy('缓存', 'Cache') }} {{ formatUsagePercent(usageCacheRate(row)) }}</span>
      <span :title="copy('输出阶段速率 = 输出 Token / (总耗时 − 首字耗时)', 'Output stage rate = output tokens / (duration - first token)')">{{ copy('输出阶段', 'Output stage') }} {{ formatUsageRate(usageOutputStageRate(row)) }}</span>
    </div>
    <small v-if="usageBillingMode(row) === 'per_request'">{{ copy('按次计费', 'Billed per request') }}</small>
  </div>
  <div v-else-if="usageBillingMode(row) === 'per_request' && !row.image_count" class="usage-media">
    <strong>{{ copy('1 次请求', '1 request') }}</strong>
    <small>{{ copy('按次计费', 'Billed per request') }}</small>
    <button type="button" class="usage-detail-link" @click="emit('detail', row)">{{ copy('查看计费详情', 'Billing details') }}</button>
  </div>
  <div v-else class="usage-media">
    <strong>{{ usageBillingMode(row) === 'video' ? copy('视频计费', 'Video billing') : row.image_count > 0 || usageBillingMode(row) === 'image' ? `${formatUsageNumber(row.image_count)} ${copy('张图片', 'images')}` : usageBillingLabel(row, !zh) }}</strong>
    <small v-if="row.image_size && usageBillingMode(row) !== 'video'">{{ row.image_size }}</small>
    <button type="button" class="usage-detail-link" @click="emit('detail', row)">{{ copy('查看计费详情', 'Billing details') }}</button>
  </div>
</template>

<style scoped>
small { display:block; margin-top:5px; color:#929088; font-size:11px; line-height:1.45; }
.usage-detail-link { display:inline-block; margin-top:7px; color:#929088; font-size:11px; text-decoration:underline; }
.usage-token-counts { display:flex; gap:12px; font-size:13px; font-variant-numeric:tabular-nums; }
.usage-token-counts > span, .usage-cache > span { display:inline-flex; align-items:center; gap:4px; }
.usage-input-icon { color:#059669; }
.usage-output-icon { color:#8b5cf6; }
.usage-cache-read { color:#0284c7; }
.usage-cache-write { color:#d97706; }
.usage-cache { display:flex; gap:12px; margin-top:5px; color:#77746d; }
.usage-cache i { margin-left:5px; padding:1px 3px; border:1px solid #dedbd1; border-radius:3px; font-size:9px; font-style:normal; }
.usage-token-rates { display:flex; flex-wrap:wrap; gap:5px; margin-top:8px; font-size:10px; }
.usage-token-rates span { padding:2px 5px; border:1px solid #bfdbfe; border-radius:4px; background:#eff6ff; color:#1d4ed8; }
.usage-token-rates span + span { border-color:#a5e3e8; background:#ecfeff; color:#0e7490; }
.usage-media strong { font-size:12px; font-weight:500; }
</style>
