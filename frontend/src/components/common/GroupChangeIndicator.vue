<template>
  <details v-if="group?.is_new" class="group-new-details" @click.stop @keydown.stop>
    <summary class="group-new" :aria-label="text('新分组，查看上线详情', 'New group, show launch details')">NEW</summary>
    <span class="group-rate-detail">{{ text('上线时间', 'Launched') }}: {{ launchDate || '—' }}<template v-if="group.subscription_type !== 'subscription' && group.launch_rate != null"><br />{{ text('首发倍率', 'Launch rate') }}: {{ group.launch_rate }}×</template></span>
  </details>
  <details v-if="change" class="group-rate-change" @click.stop @keydown.stop>
    <summary :aria-label="summaryLabel">{{ change.new_rate > change.old_rate ? '↑' : '↓' }} {{ delta }}%</summary>
    <span class="group-rate-detail">{{ text('原倍率', 'Previous rate') }} {{ change.old_rate }}× → {{ change.new_rate }}×<br />{{ formattedDate }}</span>
  </details>
</template>
<script setup lang="ts">
import { computed } from 'vue'
import { useI18n } from 'vue-i18n'
const props = defineProps<{ group?: { is_new?: boolean; launched_at?: string | null; launch_rate?: number | null; recent_rate_change?: { old_rate: number; new_rate: number; changed_at: string } | null; subscription_type?: string } | null }>()
const { locale } = useI18n()
const text = (zh: string, en: string) => locale.value.startsWith('zh') ? zh : en
const change = computed(() => props.group?.subscription_type === 'subscription' ? null : props.group?.recent_rate_change)
const delta = computed(() => change.value && change.value.old_rate > 0 ? Math.abs((change.value.new_rate / change.value.old_rate - 1) * 100).toFixed(1).replace(/\.0$/, '') : '—')
const formattedDate = computed(() => change.value ? new Date(change.value.changed_at).toLocaleString(locale.value) : '')
const launchDate = computed(() => props.group?.launched_at ? new Date(props.group.launched_at).toLocaleString(locale.value) : '')
const summaryLabel = computed(() => change.value ? text(`倍率从 ${change.value.old_rate} 变为 ${change.value.new_rate}，查看时间`, `Rate changed from ${change.value.old_rate} to ${change.value.new_rate}, show time`) : '')
</script>
<style scoped>
.group-new-details { display:inline-block; position:relative; } .group-new { display:inline-block; border-radius:4px; padding:2px 5px; background:#f4dfc3; color:#7a431f; font-size:10px; font-weight:700; letter-spacing:.04em; }
.group-rate-change { display:inline-block; position:relative; color:#935221; font-size:12px; } summary { cursor:pointer; list-style:none; text-decoration:underline dotted; text-underline-offset:3px; } summary::-webkit-details-marker { display:none; } .group-rate-detail { position:absolute; z-index:20; inset-block-start:calc(100% + 5px); inset-inline-end:0; width:max-content; max-width:230px; padding:9px 11px; border:1px solid rgb(var(--line)); border-radius:8px; color:rgb(var(--ink)); background:rgb(var(--surface)); box-shadow:0 4px 15px #0002; line-height:1.5; }
</style>
