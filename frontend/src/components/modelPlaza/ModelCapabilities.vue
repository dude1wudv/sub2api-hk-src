<template>
  <details v-if="hasData" class="model-capabilities">
    <summary>{{ text('模型能力', 'Model capabilities') }}</summary>
    <div class="model-capabilities-body">
      <p v-if="model.description">{{ model.description }}</p>
      <dl>
        <template v-if="model.context_window != null"><dt>{{ text('上下文', 'Context') }}</dt><dd>{{ number(model.context_window) }} tokens</dd></template>
        <template v-if="model.max_output_tokens != null"><dt>{{ text('最大输出', 'Max output') }}</dt><dd>{{ number(model.max_output_tokens) }} tokens</dd></template>
        <template v-if="model.input_modalities?.length"><dt>{{ text('输入', 'Input') }}</dt><dd>{{ model.input_modalities.join(', ') }}</dd></template>
        <template v-if="model.output_modalities?.length"><dt>{{ text('输出', 'Output') }}</dt><dd>{{ model.output_modalities.join(', ') }}</dd></template>
        <template v-if="model.reasoning != null"><dt>{{ text('推理', 'Reasoning') }}</dt><dd>{{ yesNo(model.reasoning) }}</dd></template>
        <template v-if="model.tool_call != null"><dt>{{ text('工具调用', 'Tool use') }}</dt><dd>{{ yesNo(model.tool_call) }}</dd></template>
        <template v-if="model.family"><dt>{{ text('系列', 'Family') }}</dt><dd>{{ model.family }}</dd></template>
        <template v-if="model.release_date"><dt>{{ text('发布日期', 'Released') }}</dt><dd>{{ model.release_date }}</dd></template>
      </dl>
      <small v-if="model.metadata_source">{{ text('资料来源', 'Source') }}: {{ model.metadata_source }}<template v-if="model.metadata_match"> · {{ model.metadata_match }}</template></small>
    </div>
  </details>
</template>
<script setup lang="ts">
import { computed } from 'vue'
import { useI18n } from 'vue-i18n'
import type { PlazaModel } from '@/api/modelPlaza'
const props = defineProps<{ model: PlazaModel }>()
const { locale } = useI18n(); const text = (zh: string, en: string) => locale.value.startsWith('zh') ? zh : en
const hasData = computed(() => ['description','context_window','max_output_tokens','input_modalities','output_modalities','reasoning','tool_call','family','release_date'].some(key => props.model[key as keyof PlazaModel] != null))
const number = (value: number) => new Intl.NumberFormat(locale.value).format(value)
const yesNo = (value: boolean) => value ? text('支持', 'Supported') : text('不支持', 'Not supported')
</script>
<style scoped>
.model-capabilities { font-size:12px; color:rgb(var(--ink-2, 80 80 80)); } summary { cursor:pointer; text-decoration:underline dotted; text-underline-offset:3px; } .model-capabilities-body { margin-top:8px; min-width:180px; white-space:normal; } p { margin:0 0 8px; } dl { display:grid; grid-template-columns:auto 1fr; gap:3px 10px; } dt,small { color:rgb(var(--ink-3, 110 110 110)); } dd { margin:0; } small { display:block; margin-top:8px; }
</style>
