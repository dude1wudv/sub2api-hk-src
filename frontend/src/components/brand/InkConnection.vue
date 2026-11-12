<template>
  <figure ref="viewport" class="ink-connection" :data-protocol="protocol" :data-running="running" :aria-label="text('API 协议连接示意', 'API protocol connection diagram')">
    <InkSurface interactive class="ink-connection-stage">
      <div class="ink-connection-label">
        <span class="ink-stage-badge"><i aria-hidden="true" />GATEWAY // ROUTER</span>
        <span class="ink-stage-telemetry">{{ protocol === 'openai' ? 'OPENAI · V1' : 'ANTHROPIC · V1' }}</span>
      </div>
      <svg class="ink-connection-art" viewBox="0 0 600 400" fill="none" aria-hidden="true">
        <g class="ink-grid-guides">
          <line x1="64" y1="202" x2="536" y2="202" stroke="currentColor" stroke-dasharray="3 5" />
          <circle cx="272" cy="202" r="96" stroke="currentColor" stroke-dasharray="2 6" />
          <circle cx="272" cy="202" r="68" stroke="currentColor" />
        </g>
        <path class="ink-wire" d="M112 202H218M326 202H352C374 202 374 126 396 126H432" />
        <path class="ink-wire" d="M112 202H218M326 202H352C374 202 374 278 396 278H432" />
        <path class="ink-route ink-route--ingress" d="M112 202H218" />
        <path class="ink-route ink-route--openai" d="M326 202H352C374 202 374 126 396 126H432" />
        <path class="ink-route ink-route--anthropic" d="M326 202H352C374 202 374 278 396 278H432" />
        <path class="ink-pulse ink-pulse--openai" d="M112 202H218M326 202H352C374 202 374 126 396 126H432" />
        <path class="ink-pulse ink-pulse--anthropic" d="M112 202H218M326 202H352C374 202 374 278 396 278H432" />
        <g class="ink-core">
          <rect x="218" y="148" width="108" height="108" rx="22" class="ink-core-halo" />
          <rect x="226" y="156" width="92" height="92" rx="18" class="ink-core-chip" />
          <path class="ink-p" d="M258 223V183C258 179.7 260.7 177 264 177H277.5a13.5 13.5 0 0 1 0 27H258" />
          <circle class="ink-p-dot" cx="288" cy="218" r="4.5" />
        </g>
        <circle class="ink-terminal" cx="112" cy="202" r="4" />
        <circle class="ink-terminal" cx="218" cy="202" r="4" />
        <circle class="ink-terminal" cx="326" cy="202" r="4" />
        <circle class="ink-copper-node" cx="272" cy="106" r="3.5" />
      </svg>
      <span class="ink-source">
        <span class="ink-source-box" aria-hidden="true">⌘</span>
        <strong>YOUR APP</strong>
        <small>SDK · CLI</small>
      </span>
      <div class="ink-destinations" role="group" :aria-label="text('选择 API 协议', 'Select API protocol')">
        <button type="button" :aria-pressed="protocol === 'openai'" class="ink-destination ink-destination--openai" @click="$emit('update:protocol', 'openai')">
          <span class="ink-node" aria-hidden="true" />
          <strong>OpenAI</strong>
          <span class="ink-protocol-type">/v1/chat/completions</span>
        </button>
        <button type="button" :aria-pressed="protocol === 'anthropic'" class="ink-destination ink-destination--anthropic" @click="$emit('update:protocol', 'anthropic')">
          <span class="ink-node" aria-hidden="true" />
          <strong>Anthropic</strong>
          <span class="ink-protocol-type">/v1/messages</span>
        </button>
      </div>
      <div class="ink-connection-caption">
        <span class="ink-caption-dot" aria-hidden="true" />
        <span>{{ text('点击节点切换协议路由', 'Select a node to switch protocol') }}</span>
        <code>200 OK · STREAM</code>
      </div>
    </InkSurface>
    <figcaption>{{ text('协议连接示意 · 示例不会发送请求', 'Protocol preview · no requests are sent') }}</figcaption>
  </figure>
</template>

<script setup lang="ts">
import { computed, ref } from 'vue'
import { useDocumentVisibility, useIntersectionObserver, useMediaQuery } from '@vueuse/core'
import { useI18n } from 'vue-i18n'
import InkSurface from '@/components/common/InkSurface.vue'
import { useReducedMotion } from '@/composables/useReducedMotion'
defineProps<{ protocol: 'openai' | 'anthropic' }>()
defineEmits<{ 'update:protocol': [value: 'openai' | 'anthropic'] }>()
const { locale } = useI18n()
const text = (zh: string, en: string) => locale.value.startsWith('zh') ? zh : en
const viewport = ref<HTMLElement | null>(null), inView = ref(false)
const visibility = useDocumentVisibility(), reducedMotion = useReducedMotion()
const desktopMotion = useMediaQuery('(min-width: 768px) and (hover: hover) and (pointer: fine)')
useIntersectionObserver(viewport, ([entry]) => { inView.value = entry?.isIntersecting ?? false }, { threshold: .1 })
const running = computed(() => inView.value && visibility.value === 'visible' && desktopMotion.value && !reducedMotion.value)
</script>

<style scoped>
.ink-connection { width:100%; min-width:0; margin:0; }
.ink-connection-stage {
  position:relative;
  overflow:hidden;
  width:100%;
  aspect-ratio:1.38;
  border:1px solid rgb(var(--line));
  border-radius:var(--radius-overlay, 20px);
  background-color:rgb(var(--surface));
  background-image:
    radial-gradient(ellipse at 85% 15%, rgb(var(--brand-moss) / .08), transparent 55%),
    radial-gradient(rgb(var(--line-strong) / .28) 1px, transparent 1px);
  background-size:100% 100%, 20px 20px;
  box-shadow:var(--shadow-panel);
}
.ink-connection-label {
  position:absolute;
  inset:18px 22px auto;
  z-index:2;
  display:flex;
  align-items:center;
  justify-content:space-between;
  gap:8px;
  padding-bottom:12px;
  border-bottom:1px solid rgb(var(--line) / .85);
  color:rgb(var(--ink-3));
  font-family:var(--font-mono);
  font-size:11.5px;
  letter-spacing:.03em;
}
.ink-stage-badge {
  display:inline-flex;
  align-items:center;
  gap:7px;
  color:rgb(var(--ink-2));
  font-weight:600;
}
.ink-stage-badge > i {
  width:7px;
  height:7px;
  border-radius:50%;
  background:rgb(var(--brand-moss));
  box-shadow:0 0 0 3px rgb(var(--brand-moss) / .16);
}
.ink-stage-telemetry {
  padding:2px 8px;
  border:1px solid rgb(var(--line));
  border-radius:6px;
  background:rgb(var(--surface-2));
  color:rgb(var(--ink-2));
  font-variant-numeric:tabular-nums;
}
.ink-connection-art {
  position:absolute;
  inset:8% 0 8%;
  width:100%;
  height:84%;
  overflow:visible;
  color:rgb(var(--ink));
}
.ink-grid-guides {
  color:rgb(var(--line-strong) / .35);
}
.ink-wire {
  stroke:rgb(var(--line-strong) / .45);
  stroke-width:1.75;
  stroke-linecap:round;
  stroke-linejoin:round;
}
.ink-route {
  stroke:rgb(var(--brand-moss));
  stroke-width:2.25;
  stroke-linecap:round;
  stroke-linejoin:round;
  opacity:0;
  transition:opacity var(--motion-overlay) var(--motion-ease);
}
.ink-route--ingress { opacity:1; }
[data-protocol='openai'] .ink-route--openai,
[data-protocol='anthropic'] .ink-route--anthropic {
  opacity:1;
}
.ink-pulse {
  stroke:rgb(var(--brand-moss));
  stroke-width:3;
  stroke-linecap:round;
  stroke-dasharray:18 220;
  stroke-dashoffset:238;
  opacity:0;
}
[data-running='true'][data-protocol='openai'] .ink-pulse--openai,
[data-running='true'][data-protocol='anthropic'] .ink-pulse--anthropic {
  opacity:.9;
  animation:ink-signal 2.6s linear infinite;
}
.ink-core-halo {
  fill:rgb(var(--surface-2) / .9);
  stroke:rgb(var(--line-strong) / .45);
  stroke-width:1;
}
.ink-core-chip {
  fill:rgb(var(--action-bg));
  stroke:rgb(var(--brand-moss) / .45);
  stroke-width:1.5;
}
.ink-p {
  stroke:rgb(var(--action-text));
  stroke-width:4.4;
  stroke-linecap:round;
  stroke-linejoin:round;
}
.ink-p-dot {
  fill:rgb(var(--brand-moss));
}
.ink-terminal {
  fill:rgb(var(--surface));
  stroke:rgb(var(--brand-moss));
  stroke-width:2;
}
.ink-copper-node {
  fill:rgb(var(--brand-copper));
}
.ink-source {
  position:absolute;
  top:50%;
  left:4.5%;
  transform:translateY(-50%);
  display:grid;
  justify-items:center;
  gap:5px;
  font-size:11px;
  color:rgb(var(--ink-2));
}
.ink-source-box {
  display:grid;
  place-items:center;
  width:46px;
  height:46px;
  border:1px solid rgb(var(--line-strong) / .6);
  border-radius:12px;
  background:rgb(var(--surface));
  color:rgb(var(--ink));
  font-size:20px;
  box-shadow:var(--shadow-panel);
}
.ink-source strong {
  font-family:var(--font-mono);
  font-size:11px;
  font-weight:600;
  letter-spacing:.03em;
}
.ink-source small {
  color:rgb(var(--ink-3));
  font-family:var(--font-mono);
  font-size:10px;
}
.ink-destinations {
  position:absolute;
  inset:0;
  pointer-events:none;
}
.ink-destination {
  position:absolute;
  right:4%;
  display:grid;
  grid-template-columns:8px minmax(0,1fr);
  align-items:center;
  column-gap:8px;
  width:31%;
  min-height:62px;
  padding:10px 12px;
  border:1px solid rgb(var(--line));
  border-radius:12px;
  background:rgb(var(--surface));
  box-shadow:0 1px 2px rgb(15 23 42 / .04);
  text-align:left;
  cursor:pointer;
  transition:border-color var(--motion-fast),background-color var(--motion-fast),box-shadow var(--motion-fast);
  pointer-events:auto;
}
.ink-destination:hover {
  border-color:rgb(var(--line-strong));
  box-shadow:var(--shadow-card-hover);
}
.ink-destination--openai { top:23%; }
.ink-destination--anthropic { top:61%; }
.ink-destination strong {
  color:rgb(var(--ink));
  font-size:13.5px;
  font-weight:600;
  letter-spacing:-.01em;
}
.ink-node {
  width:7px;
  height:7px;
  border-radius:50%;
  background:rgb(var(--line-strong));
}
.ink-destination[aria-pressed='true'] {
  border-color:rgb(var(--brand-moss) / .65);
  background:rgb(var(--selection-bg));
  box-shadow:0 0 0 1px rgb(var(--brand-moss) / .18), var(--shadow-panel);
}
.ink-destination[aria-pressed='true'] .ink-node {
  background:rgb(var(--brand-moss));
  box-shadow:0 0 0 3px rgb(var(--brand-moss) / .16);
}
.ink-protocol-type {
  grid-column:1 / -1;
  margin-top:3px;
  color:rgb(var(--ink-3));
  font-family:var(--font-mono);
  font-size:10.5px;
  overflow-wrap:anywhere;
}
.ink-connection-caption {
  position:absolute;
  inset:auto 22px 16px;
  z-index:2;
  display:flex;
  gap:8px;
  align-items:center;
  padding-top:10px;
  border-top:1px solid rgb(var(--line) / .85);
  font-size:12px;
  color:rgb(var(--ink-3));
}
.ink-connection-caption > code {
  margin-left:auto;
  color:rgb(var(--text-ok));
  font-family:var(--font-mono);
  font-size:11px;
  font-weight:600;
  font-variant-numeric:tabular-nums;
}
.ink-caption-dot {
  width:6px;
  height:6px;
  flex-shrink:0;
  border-radius:50%;
  background:rgb(var(--brand-copper));
}
figcaption {
  margin-top:10px;
  color:rgb(var(--ink-3));
  font-size:12.5px;
  text-align:right;
}
@keyframes ink-signal {
  to { stroke-dashoffset:0; }
}
@media(max-width:767px) {
  .ink-connection-stage {
    aspect-ratio:1.25;
    border-radius:var(--radius-card, 16px);
  }
  .ink-connection-label {
    inset:12px 14px auto;
    padding-bottom:8px;
    font-size:10px;
  }
  .ink-connection-caption {
    inset:auto 14px 12px;
    padding-top:8px;
    font-size:11px;
  }
  .ink-connection-caption > code { display:none; }
  .ink-destination {
    right:3.5%;
    width:36%;
    min-height:52px;
    padding:7px 9px;
  }
  .ink-destination--openai { top:22%; }
  .ink-destination--anthropic { top:58%; }
  .ink-destination strong { font-size:12.5px; }
  .ink-protocol-type { font-size:9.5px; }
  .ink-source { left:3.5%; font-size:10px; }
  .ink-source-box { width:38px; height:38px; font-size:17px; }
  .ink-source small { display:none; }
  .ink-connection figcaption { font-size:11.5px; }
  .ink-pulse { animation:none !important; opacity:0 !important; }
}
:global([data-motion='reduced']) .ink-pulse { animation:none !important; opacity:0 !important; }
@media(prefers-reduced-motion:reduce) { .ink-pulse { animation:none !important; opacity:0 !important; } }
</style>
