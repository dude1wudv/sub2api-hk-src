<template>
  <figure ref="viewport" class="ink-connection" :data-protocol="protocol" :data-running="running" :aria-label="text('API 协议连接示意', 'API protocol connection diagram')">
    <InkSurface interactive class="ink-connection-stage">
      <div class="ink-connection-label"><span>01 — CONNECT</span><span>{{ text('从想法到调用', 'From idea to request') }}</span></div>
      <svg class="ink-connection-art" viewBox="0 0 600 400" fill="none" aria-hidden="true">
        <circle class="ink-orbit" cx="302" cy="204" r="135" stroke="currentColor" stroke-dasharray="2 9" />
        <path class="ink-wire" d="M100 205H210C250 205 250 108 306 108H475" />
        <path class="ink-wire" d="M100 205H210C250 205 250 300 306 300H475" />
        <path class="ink-route ink-route--openai" d="M100 205H210C250 205 250 108 306 108H475" />
        <path class="ink-route ink-route--anthropic" d="M100 205H210C250 205 250 300 306 300H475" />
        <g class="ink-float ink-float--one"><ellipse cx="324" cy="204" rx="80" ry="75" fill="currentColor" /></g>
        <g class="ink-float ink-float--two"><ellipse cx="269" cy="220" rx="38" ry="42" fill="currentColor" /></g>
        <g class="ink-float ink-float--three"><circle cx="392" cy="249" r="13" fill="currentColor" /></g>
        <path class="ink-p" d="M292 240V173c0-8 6-13 14-13h16c20 0 32 12 32 28s-12 28-32 28h-30m20-40h10c8 0 12 4 12 12s-4 12-12 12h-10" />
        <circle class="ink-p-dot" cx="345" cy="241" r="4" />
        <circle class="ink-orbit-dot" cx="310" cy="70" r="7" />
        <path class="ink-copper-arc" d="M228 335c35 22 85 28 131 7" />
      </svg>
      <span class="ink-source"><span aria-hidden="true">⌘</span><strong>YOUR APP</strong></span>
      <div class="ink-destinations" role="group" :aria-label="text('选择 API 协议', 'Select API protocol')">
        <button type="button" :aria-pressed="protocol === 'openai'" class="ink-destination ink-destination--openai" @click="$emit('update:protocol', 'openai')"><span class="ink-node" aria-hidden="true" /><strong>OpenAI</strong><span class="ink-protocol-type">/chat/completions</span></button>
        <button type="button" :aria-pressed="protocol === 'anthropic'" class="ink-destination ink-destination--anthropic" @click="$emit('update:protocol', 'anthropic')"><span class="ink-node" aria-hidden="true" /><strong>Anthropic</strong><span class="ink-protocol-type">/messages</span></button>
      </div>
      <div class="ink-connection-caption"><span class="ink-caption-dot" aria-hidden="true" />{{ text('选择协议，探索连接方式。', 'Pick a protocol. Explore the connection.') }}<span>↗</span></div>
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
.ink-connection-stage { position:relative; overflow:hidden; width:100%; aspect-ratio:1.35; border:1px solid rgb(var(--line)); border-radius:32px 32px 80px 32px; background:rgb(var(--surface-2) / .62); }
.ink-connection-label { position:absolute; inset:26px 28px auto; display:flex; align-items:center; justify-content:space-between; gap:8px; color:rgb(var(--ink-3)); font-size:13px; letter-spacing:.02em; }
.ink-connection-label > :first-child { font-family:ui-monospace,SFMono-Regular,Consolas,monospace; font-size:12px; }
.ink-connection-art { position:absolute; inset:6% 0; width:100%; height:88%; overflow:visible; color:rgb(var(--ink)); }
.ink-orbit { color:rgb(var(--brand-moss) / .3); }
.ink-wire { stroke:rgb(var(--line-strong) / .5); stroke-width:1; }
.ink-route { stroke:rgb(var(--brand-moss)); stroke-width:2; stroke-dasharray:600; stroke-dashoffset:600; opacity:0; transition:stroke-dashoffset var(--motion-page) var(--motion-ease),opacity var(--motion-overlay); }
[data-protocol='openai'] .ink-route--openai,[data-protocol='anthropic'] .ink-route--anthropic { stroke-dashoffset:0; opacity:1; }
.ink-float { transform-box:fill-box; transform-origin:center; animation:ink-drift 8s ease-in-out infinite; animation-play-state:paused; }
.ink-float--two { animation-delay:-3s; animation-direction:reverse; }
.ink-float--three { animation-delay:-5s; }
[data-running='true'] .ink-float { animation-play-state:running; }
.ink-p { stroke:rgb(var(--canvas)); stroke-width:6; stroke-linecap:round; stroke-linejoin:round; }
.ink-p-dot { fill:rgb(var(--canvas)); }
.ink-orbit-dot { fill:rgb(var(--brand-moss)); }
.ink-copper-arc { stroke:rgb(var(--brand-copper)); stroke-width:4; stroke-linecap:round; }
.ink-source { position:absolute; top:46%; left:5%; display:grid; justify-items:center; gap:9px; font-size:11px; color:rgb(var(--ink-3)); }
.ink-source > span { display:grid; place-items:center; width:44px; height:44px; border:1px solid rgb(var(--line)); border-radius:14px; background:rgb(var(--surface)); color:rgb(var(--ink)); font-size:23px; }
.ink-source strong { font-weight:500; letter-spacing:.03em; }
.ink-destinations { position:absolute; inset:0; pointer-events:none; }
.ink-destination { position:absolute; right:4%; display:grid; grid-template-columns:10px 1fr; align-items:center; column-gap:8px; width:30%; min-height:60px; padding:10px; border:1px solid rgb(var(--line)); border-radius:15px; background:rgb(var(--surface)); text-align:left; transition:border-color var(--motion-fast),background-color var(--motion-fast); pointer-events:auto; }
.ink-destination--openai { top:23%; }
.ink-destination--anthropic { top:65%; }
.ink-destination strong { color:rgb(var(--ink)); font-size:14px; font-weight:550; }
.ink-node { width:6px; height:6px; border-radius:50%; background:rgb(var(--line-strong)); }
.ink-destination[aria-pressed='true'] { border-color:rgb(var(--brand-moss) / .55); background:rgb(var(--selection-bg)); }
.ink-destination[aria-pressed='true'] .ink-node { background:rgb(var(--brand-moss)); box-shadow:0 0 0 3px rgb(var(--brand-moss) / .12); }
.ink-protocol-type { grid-column:1 / -1; margin-top:3px; color:rgb(var(--ink-3)); font-size:11px; overflow-wrap:anywhere; }
.ink-connection-caption { position:absolute; inset:auto 28px 25px; display:flex; gap:7px; align-items:center; font-size:13px; color:rgb(var(--ink-3)); }
.ink-connection-caption > :last-child { margin-left:auto; color:rgb(var(--ink)); font-size:20px; }
.ink-caption-dot { width:5px; height:5px; flex-shrink:0; border-radius:50%; background:rgb(var(--brand-copper)); }
figcaption { margin-top:13px; color:rgb(var(--ink-3)); font-size:13px; text-align:right; }
@keyframes ink-drift { 0%,100% { transform:translate(0,0) scale(1,1); } 45% { transform:translate(5px,-5px) scale(.94,1.06); } 70% { transform:translate(-3px,3px) scale(1.04,.96); } }
@media(max-width:767px) { .ink-connection-stage { border-radius:24px 24px 52px 24px; } .ink-connection-label { inset:18px 18px auto; font-size:11px; } .ink-connection-caption { inset:auto 18px 15px; font-size:11px; } .ink-destination { padding:7px; } .ink-destination strong { font-size:13px; } .ink-protocol-type { font-size:10px; } .ink-source { font-size:9px; } .ink-connection figcaption { font-size:12px; } .ink-float { animation:none; } }
:global([data-motion='reduced']) .ink-float { animation:none; }
@media(prefers-reduced-motion:reduce) { .ink-float { animation:none; } }
</style>
