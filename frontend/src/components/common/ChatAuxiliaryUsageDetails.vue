<script setup lang="ts">
import { computed } from 'vue'
import { useI18n } from 'vue-i18n'
import type { ChatAuxiliaryUsage } from '@/types'

defineProps<{ usage?: ChatAuxiliaryUsage | null }>()
const { locale } = useI18n()
const zh = computed(() => locale.value.startsWith('zh'))
const copy = (cn: string, en: string) => zh.value ? cn : en
const label = (kind: string) => ({ search: copy('文字搜索', 'Text search'), image_search: copy('图片搜索', 'Image search'), fetch: copy('读取链接', 'Read URL'), asr: copy('语音识别', 'Speech recognition'), tts: copy('语音合成', 'Speech synthesis'), ocr: 'OCR', native_files: copy('原生文档解析', 'Native document parsing') }[kind] || kind)
const unit = (value: string) => ({ call: copy('次', 'call'), page: copy('页', 'page'), second: copy('秒', 'second'), character: copy('字符', 'character') }[value] || value)
</script>

<template>
  <section v-if="usage" class="auxiliary-usage" data-testid="chat-auxiliary-usage">
    <h4>{{ copy('辅助费用 · 站内 USD', 'Auxiliary charges · site USD') }}</h4>
    <p>{{ copy('人民币刊例数值1:1记账，无倍率；不是汇率换算。', 'CNY list-price numbers booked 1:1 without multipliers; not a currency conversion.') }}</p>
    <table>
      <thead><tr><th>{{ copy('项目', 'Item') }}</th><th>{{ copy('数量', 'Quantity') }}</th><th>{{ copy('单价', 'Unit price') }}</th><th>{{ copy('辅助费', 'Charge') }}</th></tr></thead>
      <tbody><tr v-for="(item, index) in usage.items" :key="`${item.kind}-${index}`">
        <td>{{ label(item.kind) }}</td>
        <td>{{ item.quantity === null ? copy('未公开', 'Unreported') : `${item.quantity} ${unit(item.unit)}` }}</td>
        <td>{{ item.unit_price }} / {{ item.price_quantity }} {{ unit(item.unit) }}</td>
        <td>${{ item.charged_cost }}</td>
      </tr></tbody>
    </table>
    <p v-if="usage.items.some(item => item.kind === 'native_files')">{{ copy('解析由平台承担，引用token另计；未公开页数不代表上游免费。', 'Parsing is platform-funded; reference tokens are billed separately. Unreported pages do not mean free upstream parsing.') }}</p>
    <p v-if="usage.items.some(item => item.metering_status === 'incomplete')">{{ copy('未确认完整成功；无法确认的部分费用由平台承担。', 'Completion was not confirmed; unverified partial costs are platform-funded.') }}</p>
    <p v-if="usage.items.some(item => item.quantity_source === 'request_text')">{{ copy('TTS按提交文本计费字符计量，汉字计2，其余Unicode字符计1。', 'TTS uses submitted text: Han characters count as 2, other Unicode code points as 1.') }}</p>
    <small>{{ usage.price_basis }} · {{ copy('混合账单中的倍率仅适用于模型Token。', 'For mixed bills, the multiplier applies only to model tokens.') }}</small>
  </section>
</template>

<style scoped>
.auxiliary-usage { margin-block: 12px; font-size: 12px; line-height: 1.5; overflow-wrap: anywhere; }
h4 { font-weight: 600; margin-bottom: 4px; }
p, small { margin-block: 6px; opacity: .8; }
table { width: 100%; border-collapse: collapse; margin-block: 8px; }
th, td { padding: 4px; text-align: left; vertical-align: top; border-bottom: 1px solid currentColor; }
th:last-child, td:last-child { text-align: right; font-variant-numeric: tabular-nums; }
</style>
