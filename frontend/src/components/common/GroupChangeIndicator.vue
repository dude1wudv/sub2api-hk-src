<template>
  <details v-if="group?.is_new" class="group-new-details" @click.stop @keydown.stop>
    <summary class="group-new" :aria-label="text('新分组，查看上线详情', 'New group, show launch details')">NEW</summary>
    <span class="group-rate-detail">{{ text('上线时间', 'Launched') }}: {{ launchDate || '—' }}<template v-if="group.subscription_type !== 'subscription' && group.launch_rate != null"><br />{{ text('首发倍率', 'Launch rate') }}: {{ group.launch_rate }}×</template></span>
  </details>
  <details v-if="change" class="group-rate-change" :class="isDrop ? 'is-drop' : 'is-rise'" @click.stop @keydown.stop>
    <summary :aria-label="summaryLabel">{{ isDrop ? '↓' : '↑' }} {{ delta }}%</summary>
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
const isDrop = computed(() => Boolean(change.value && change.value.new_rate < change.value.old_rate))
const delta = computed(() => change.value && change.value.old_rate > 0 ? Math.abs((change.value.new_rate / change.value.old_rate - 1) * 100).toFixed(1).replace(/\.0$/, '') : '—')
const formattedDate = computed(() => change.value ? new Date(change.value.changed_at).toLocaleString(locale.value) : '')
const launchDate = computed(() => props.group?.launched_at ? new Date(props.group.launched_at).toLocaleString(locale.value) : '')
const summaryLabel = computed(() => change.value ? text(`倍率从 ${change.value.old_rate} 变为 ${change.value.new_rate}，查看时间`, `Rate changed from ${change.value.old_rate} to ${change.value.new_rate}, show time`) : '')
</script>
<style scoped>
.group-new-details { display:inline-block; position:relative; }
.group-new {
  display:inline-flex;
  align-items:center;
  border-radius:999px;
  padding:2px 7px;
  background:#dbeafe;
  color:#1d4ed8;
  border:1px solid #93c5fd;
  font-size:10.5px;
  font-weight:700;
  letter-spacing:.04em;
  line-height:1.3;
  cursor:pointer;
  list-style:none;
}
.group-rate-change { display:inline-block; position:relative; font-size:11.5px; font-variant-numeric:tabular-nums; }
.group-rate-change > summary {
  display:inline-flex;
  align-items:center;
  gap:2px;
  padding:2px 7px;
  border-radius:999px;
  font-weight:700;
  line-height:1.3;
  cursor:pointer;
  list-style:none;
  text-decoration:none;
}
.group-rate-change.is-drop > summary {
  background:#dcfce7;
  color:#047857;
  border:1px solid #6ee7b7;
}
.group-rate-change.is-rise > summary {
  background:#fef3c7;
  color:#b45309;
  border:1px solid #fcd34d;
}
:global(.dark) .group-new {
  background:rgba(37, 99, 235, 0.24);
  color:#93c5fd;
  border-color:rgba(96, 165, 250, 0.45);
}
:global(.dark) .group-rate-change.is-drop > summary {
  background:rgba(16, 185, 129, 0.2);
  color:#6ee7b7;
  border-color:rgba(52, 211, 153, 0.45);
}
:global(.dark) .group-rate-change.is-rise > summary {
  background:rgba(245, 158, 11, 0.2);
  color:#fcd34d;
  border-color:rgba(251, 191, 36, 0.45);
}
summary::-webkit-details-marker { display:none; }
.group-rate-detail {
  position:absolute;
  z-index:20;
  inset-block-start:calc(100% + 5px);
  inset-inline-end:0;
  width:max-content;
  max-width:230px;
  padding:9px 11px;
  border:1px solid rgb(var(--line));
  border-radius:8px;
  color:rgb(var(--ink));
  background:rgb(var(--surface-elevated, var(--surface)));
  box-shadow:0 8px 24px rgb(15 23 42 / 0.16);
  line-height:1.5;
  text-align:left;
}
</style>
