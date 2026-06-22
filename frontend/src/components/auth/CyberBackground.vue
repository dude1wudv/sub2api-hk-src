<template>
  <div class="cyber-background">
    <!-- Animated grid canvas -->
    <canvas ref="gridCanvas" class="grid-canvas"></canvas>

    <!-- SVG map lines -->
    <svg class="map-lines" viewBox="0 0 1920 1080" preserveAspectRatio="xMidYMid slice">
      <defs>
        <linearGradient id="lineGradient" x1="0%" y1="0%" x2="100%" y2="100%">
          <stop offset="0%" stop-color="#00d9ff" stop-opacity="0" />
          <stop offset="50%" stop-color="#00d9ff" stop-opacity="0.6" />
          <stop offset="100%" stop-color="#00d9ff" stop-opacity="0" />
        </linearGradient>
      </defs>

      <!-- Diagonal connection lines -->
      <path d="M 0 400 Q 400 300, 800 400 T 1600 400" class="map-line" />
      <path d="M 200 100 Q 600 200, 1000 100 T 1800 100" class="map-line delayed-1" />
      <path d="M 100 700 Q 500 800, 900 700 T 1700 700" class="map-line delayed-2" />
      <path d="M 300 900 Q 700 800, 1100 900 T 1900 900" class="map-line delayed-3" />

      <!-- Vertical connections -->
      <line x1="400" y1="0" x2="400" y2="1080" class="map-line pulse-1" />
      <line x1="960" y1="0" x2="960" y2="1080" class="map-line pulse-2" />
      <line x1="1520" y1="0" x2="1520" y2="1080" class="map-line pulse-3" />
    </svg>

    <!-- Data streams -->
    <div class="data-streams">
      <div v-for="i in 12" :key="i" class="data-stream" :style="getStreamStyle(i)">
        <span class="data-char">{{ getRandomChar() }}</span>
      </div>
    </div>

    <!-- Corner glows -->
    <div class="corner-glow top-left"></div>
    <div class="corner-glow top-right"></div>
    <div class="corner-glow bottom-left"></div>
    <div class="corner-glow bottom-right"></div>
  </div>
</template>

<script setup lang="ts">
import { ref, onMounted, onUnmounted } from 'vue'

const gridCanvas = ref<HTMLCanvasElement | null>(null)
let animationFrame: number | null = null
let gridOffset = 0

const getRandomChar = () => {
  const chars = '01アイウエオカキクサシスセソタチツテト'
  return chars[Math.floor(Math.random() * chars.length)]
}

const getStreamStyle = (index: number) => {
  const left = (index * 8 + Math.random() * 10) % 100
  const duration = 8 + Math.random() * 12
  const delay = Math.random() * 8
  return {
    left: `${left}%`,
    animationDuration: `${duration}s`,
    animationDelay: `${delay}s`
  }
}

const drawGrid = () => {
  if (!gridCanvas.value) return

  const canvas = gridCanvas.value
  const ctx = canvas.getContext('2d')
  if (!ctx) return

  // Set canvas size to window size
  canvas.width = window.innerWidth
  canvas.height = window.innerHeight

  const gridSize = 40
  gridOffset = (gridOffset + 0.5) % gridSize

  ctx.clearRect(0, 0, canvas.width, canvas.height)
  ctx.strokeStyle = 'rgba(0, 217, 255, 0.15)'
  ctx.lineWidth = 1

  // Vertical lines
  for (let x = -gridSize + gridOffset; x < canvas.width; x += gridSize) {
    ctx.beginPath()
    ctx.moveTo(x, 0)
    ctx.lineTo(x, canvas.height)
    ctx.stroke()
  }

  // Horizontal lines
  for (let y = -gridSize + gridOffset; y < canvas.height; y += gridSize) {
    ctx.beginPath()
    ctx.moveTo(0, y)
    ctx.lineTo(canvas.width, y)
    ctx.stroke()
  }

  // Add perspective effect nodes
  const nodes = 8
  for (let i = 0; i < nodes; i++) {
    const x = (canvas.width / nodes) * i + gridOffset * 2
    const y = canvas.height / 2 + Math.sin((Date.now() / 1000 + i) * 0.5) * 50

    ctx.fillStyle = 'rgba(0, 217, 255, 0.6)'
    ctx.beginPath()
    ctx.arc(x, y, 3, 0, Math.PI * 2)
    ctx.fill()

    // Pulsing glow
    ctx.fillStyle = `rgba(0, 217, 255, ${0.2 * Math.sin(Date.now() / 500 + i)})`
    ctx.beginPath()
    ctx.arc(x, y, 8, 0, Math.PI * 2)
    ctx.fill()
  }

  animationFrame = requestAnimationFrame(drawGrid)
}

const handleResize = () => {
  if (gridCanvas.value) {
    gridCanvas.value.width = window.innerWidth
    gridCanvas.value.height = window.innerHeight
  }
}

onMounted(() => {
  drawGrid()
  window.addEventListener('resize', handleResize)
})

onUnmounted(() => {
  if (animationFrame) {
    cancelAnimationFrame(animationFrame)
  }
  window.removeEventListener('resize', handleResize)
})
</script>

<style scoped>
.cyber-background {
  position: fixed;
  top: 0;
  left: 0;
  width: 100%;
  height: 100%;
  background: linear-gradient(135deg, #0a0e1a 0%, #1a1f3a 50%, #0f1419 100%);
  overflow: hidden;
  z-index: 0;
}

.grid-canvas {
  position: absolute;
  top: 0;
  left: 0;
  width: 100%;
  height: 100%;
  opacity: 1;
}

.map-lines {
  position: absolute;
  top: 0;
  left: 0;
  width: 100%;
  height: 100%;
  opacity: 0.4;
}

.map-line {
  fill: none;
  stroke: url(#lineGradient);
  stroke-width: 2;
  stroke-dasharray: 10 5;
  animation: dash 20s linear infinite;
}

.map-line.delayed-1 {
  animation-delay: -5s;
}

.map-line.delayed-2 {
  animation-delay: -10s;
}

.map-line.delayed-3 {
  animation-delay: -15s;
}

.map-line.pulse-1 {
  stroke: rgba(0, 217, 255, 0.3);
  stroke-width: 1;
  animation: pulse 3s ease-in-out infinite;
}

.map-line.pulse-2 {
  animation: pulse 3s ease-in-out infinite;
  animation-delay: 1s;
}

.map-line.pulse-3 {
  animation: pulse 3s ease-in-out infinite;
  animation-delay: 2s;
}

@keyframes dash {
  to {
    stroke-dashoffset: -100;
  }
}

@keyframes pulse {
  0%, 100% {
    opacity: 0.2;
  }
  50% {
    opacity: 0.8;
  }
}

.data-streams {
  position: absolute;
  top: 0;
  left: 0;
  width: 100%;
  height: 100%;
  pointer-events: none;
}

.data-stream {
  position: absolute;
  top: -50px;
  font-family: 'Courier New', monospace;
  font-size: 14px;
  color: rgba(0, 217, 255, 0.6);
  animation: stream-fall linear infinite;
  text-shadow: 0 0 8px rgba(0, 217, 255, 0.8);
}

@keyframes stream-fall {
  from {
    transform: translateY(-50px);
    opacity: 0;
  }
  10% {
    opacity: 1;
  }
  90% {
    opacity: 1;
  }
  to {
    transform: translateY(calc(100vh + 50px));
    opacity: 0;
  }
}

.data-char {
  display: inline-block;
  animation: char-flicker 0.5s infinite;
}

@keyframes char-flicker {
  0%, 100% {
    opacity: 1;
  }
  50% {
    opacity: 0.3;
  }
}

.corner-glow {
  position: absolute;
  width: 400px;
  height: 400px;
  border-radius: 50%;
  pointer-events: none;
  filter: blur(60px);
  animation: glow-pulse 4s ease-in-out infinite;
}

.corner-glow.top-left {
  top: -200px;
  left: -200px;
  background: radial-gradient(circle, rgba(0, 217, 255, 0.3) 0%, transparent 70%);
}

.corner-glow.top-right {
  top: -200px;
  right: -200px;
  background: radial-gradient(circle, rgba(138, 43, 226, 0.25) 0%, transparent 70%);
  animation-delay: 1s;
}

.corner-glow.bottom-left {
  bottom: -200px;
  left: -200px;
  background: radial-gradient(circle, rgba(255, 0, 128, 0.2) 0%, transparent 70%);
  animation-delay: 2s;
}

.corner-glow.bottom-right {
  bottom: -200px;
  right: -200px;
  background: radial-gradient(circle, rgba(0, 217, 255, 0.25) 0%, transparent 70%);
  animation-delay: 3s;
}

@keyframes glow-pulse {
  0%, 100% {
    opacity: 0.4;
    transform: scale(1);
  }
  50% {
    opacity: 0.7;
    transform: scale(1.1);
  }
}
</style>
