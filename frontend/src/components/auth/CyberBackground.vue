<template>
  <div class="cyber-background" aria-hidden="true">
    <canvas ref="gridCanvas" class="grid-canvas"></canvas>

    <svg class="map-lines" viewBox="0 0 1920 1080" preserveAspectRatio="xMidYMid slice">
      <defs>
        <linearGradient id="cyberLineGradient" x1="0%" y1="0%" x2="100%" y2="0%">
          <stop offset="0%" stop-color="#2cff43" stop-opacity="0" />
          <stop offset="45%" stop-color="#2cff43" stop-opacity="0.5" />
          <stop offset="100%" stop-color="#2cff43" stop-opacity="0" />
        </linearGradient>
      </defs>

      <path d="M 120 720 C 360 600, 520 680, 760 560 S 1240 420, 1600 560" class="map-line" />
      <path d="M 0 450 C 260 380, 440 470, 680 390 S 1120 260, 1490 360 S 1850 300, 1980 240" class="map-line delayed-1" />
      <path d="M 220 180 C 500 240, 620 140, 880 210 S 1280 300, 1680 170" class="map-line delayed-2" />
      <path d="M 310 930 C 560 780, 760 880, 980 760 S 1420 720, 1720 830" class="map-line delayed-3" />

      <g class="network-nodes">
        <circle cx="650" cy="520" r="4" />
        <circle cx="790" cy="600" r="5" />
        <circle cx="960" cy="510" r="4" />
        <circle cx="1180" cy="445" r="5" />
        <circle cx="1380" cy="520" r="4" />
        <line x1="650" y1="520" x2="790" y2="600" />
        <line x1="790" y1="600" x2="960" y2="510" />
        <line x1="960" y1="510" x2="1180" y2="445" />
        <line x1="1180" y1="445" x2="1380" y2="520" />
      </g>
    </svg>

    <div class="code-panel left-panel">
      <span>// SUB2API GATEWAY</span>
      <span>GET /v1/endpoint</span>
      <span>Authorization: Bearer ********</span>
      <span>200 OK</span>
    </div>

    <div class="data-streams">
      <div v-for="stream in streams" :key="stream.left" class="data-stream" :style="stream.style">
        <span v-for="char in stream.text" :key="char">{{ char }}</span>
      </div>
    </div>
  </div>
</template>

<script setup lang="ts">
import { ref, onMounted, onUnmounted } from 'vue'

const gridCanvas = ref<HTMLCanvasElement | null>(null)
let animationFrame: number | null = null
let gridOffset = 0

const streamWords = ['0101', 'API', 'JSON', '200', 'KEY', 'AUTH', 'POST', 'SSE']
const streams = Array.from({ length: 18 }, (_, index) => {
  const left = (index * 7 + 3) % 100
  return {
    left,
    text: streamWords[index % streamWords.length],
    style: {
      left: `${left}%`,
      animationDuration: `${9 + (index % 6) * 2}s`,
      animationDelay: `-${index % 8}s`
    }
  }
})

function drawGrid() {
  if (!gridCanvas.value) return

  const canvas = gridCanvas.value
  const ctx = canvas.getContext('2d')
  if (!ctx) return

  const dpr = window.devicePixelRatio || 1
  const width = window.innerWidth
  const height = window.innerHeight

  canvas.width = width * dpr
  canvas.height = height * dpr
  canvas.style.width = `${width}px`
  canvas.style.height = `${height}px`
  ctx.setTransform(dpr, 0, 0, dpr, 0, 0)

  gridOffset = (gridOffset + 0.22) % 56
  ctx.clearRect(0, 0, width, height)
  ctx.lineWidth = 1
  ctx.strokeStyle = 'rgba(44, 255, 67, 0.06)'

  for (let x = -56 + gridOffset; x < width; x += 56) {
    ctx.beginPath()
    ctx.moveTo(x, 0)
    ctx.lineTo(x, height)
    ctx.stroke()
  }

  for (let y = -56 + gridOffset; y < height; y += 56) {
    ctx.beginPath()
    ctx.moveTo(0, y)
    ctx.lineTo(width, y)
    ctx.stroke()
  }

  animationFrame = requestAnimationFrame(drawGrid)
}

onMounted(() => {
  drawGrid()
})

onUnmounted(() => {
  if (animationFrame) {
    cancelAnimationFrame(animationFrame)
  }
})
</script>

<style scoped>
.cyber-background {
  position: fixed;
  inset: 0;
  z-index: 0;
  overflow: hidden;
  background:
    linear-gradient(90deg, rgba(44, 255, 67, 0.04) 1px, transparent 1px),
    linear-gradient(0deg, rgba(44, 255, 67, 0.025) 1px, transparent 1px),
    linear-gradient(120deg, #020403 0%, #050806 52%, #010201 100%);
  background-size: 96px 96px, 96px 96px, auto;
}

.cyber-background::before {
  position: absolute;
  inset: 0;
  content: '';
  background:
    repeating-linear-gradient(100deg, transparent 0 18px, rgba(44, 255, 67, 0.025) 18px 19px, transparent 19px 42px),
    linear-gradient(90deg, rgba(0, 0, 0, 0.82) 0%, transparent 44%, rgba(0, 0, 0, 0.88) 100%);
}

.cyber-background::after {
  position: absolute;
  inset: 0;
  content: '';
  box-shadow: inset 0 0 220px rgba(0, 0, 0, 0.92);
}

.grid-canvas,
.map-lines,
.data-streams,
.code-panel {
  position: absolute;
  inset: 0;
}

.grid-canvas {
  opacity: 0.9;
}

.map-lines {
  opacity: 0.55;
}

.map-line {
  fill: none;
  stroke: url(#cyberLineGradient);
  stroke-width: 1.4;
  stroke-dasharray: 12 18;
  animation: dash 26s linear infinite;
}

.map-line.delayed-1 {
  animation-delay: -7s;
}

.map-line.delayed-2 {
  animation-delay: -14s;
}

.map-line.delayed-3 {
  animation-delay: -20s;
}

.network-nodes circle {
  fill: #2cff43;
  filter: drop-shadow(0 0 8px rgba(44, 255, 67, 0.7));
}

.network-nodes line {
  stroke: rgba(44, 255, 67, 0.22);
  stroke-width: 1;
}

.code-panel {
  left: 45%;
  top: 8%;
  display: flex;
  width: 310px;
  height: fit-content;
  flex-direction: column;
  gap: 9px;
  font-family: ui-monospace, SFMono-Regular, Menlo, Monaco, Consolas, monospace;
  font-size: 12px;
  color: rgba(44, 255, 67, 0.42);
  text-shadow: 0 0 10px rgba(44, 255, 67, 0.18);
}

.code-panel span:last-child {
  color: rgba(44, 255, 67, 0.85);
}

.data-streams {
  pointer-events: none;
}

.data-stream {
  position: absolute;
  top: -120px;
  display: flex;
  flex-direction: column;
  gap: 7px;
  font-family: ui-monospace, SFMono-Regular, Menlo, Monaco, Consolas, monospace;
  font-size: 12px;
  letter-spacing: 0;
  color: rgba(44, 255, 67, 0.22);
  text-shadow: 0 0 8px rgba(44, 255, 67, 0.28);
  animation: stream-fall linear infinite;
}

@keyframes dash {
  to {
    stroke-dashoffset: -220;
  }
}

@keyframes stream-fall {
  from {
    transform: translateY(-120px);
    opacity: 0;
  }
  14%,
  82% {
    opacity: 1;
  }
  to {
    transform: translateY(calc(100vh + 120px));
    opacity: 0;
  }
}

@media (max-width: 1023px) {
  .code-panel {
    display: none;
  }
}

@media (prefers-reduced-motion: reduce) {
  .map-line,
  .data-stream {
    animation: none;
  }
}
</style>
