<script setup lang="ts">
import { computed, onMounted, ref } from 'vue'
import {
  buildBearerHeaders,
  buildClientTraceHeaders,
  buildImageApiUrl,
  buildTextImagePayload,
  extractImageResults,
  normalizeImageApiBase,
  type ImageOutputFormat,
  type ImageQuality,
} from '@/utils/imageWorkbench'

type WorkbenchMode = 'text' | 'image'

interface GeneratedWork {
  id: string
  src: string
  title: string
  prompt: string
  size: string
  createdAt: string
}

const requestUrlStorageKey = 'sub2api-image2-request-url'
const draftStorageKey = 'sub2api-image2-draft'
const apiKeyStorageKey = 'sub2api-image2-api-key'

const requestUrl = ref('')
const apiKey = ref('')
const mode = ref<WorkbenchMode>('text')
const prompt = ref('')
const subject = ref('')
const style = ref('clean-product')
const model = ref('gpt-image-2')
const size = ref('')
const quality = ref<ImageQuality>('high')
const outputFormat = ref<ImageOutputFormat | ''>('')
const isEmbed = ref(false)
const batchCount = ref(1)
const referenceFiles = ref<File[]>([])
const works = ref<GeneratedWork[]>([])
const selectedWorkId = ref('')
const isGenerating = ref(false)
const statusText = ref('待命')
const errorText = ref('')

const styleOptions = [
  { value: 'clean-product', label: '电商主图', text: '干净白底商品摄影，主体完整居中，边缘清晰，自然投影，不要水印、二维码、乱码文字。' },
  { value: 'lifestyle', label: '生活场景', text: '真实生活方式场景，自然光，高级质感，背景有层次但不抢主体。' },
  { value: 'social-avatar', label: '头像写真', text: '自然光人像摄影，情绪松弛，干净背景，适合头像或社媒分享。' },
  { value: 'poster', label: '海报封面', text: '强视觉焦点，移动端可读，留出标题安全区，避免复杂小字。' },
  { value: 'free', label: '纯提示词', text: '' },
]

const modelOptions = [
  { value: 'gpt-image-2', label: 'gpt-image-2' },
  { value: 'gpt-image-1', label: 'gpt-image-1' },
]

const sizeOptions = [
  { value: '', label: '接口默认' },
  { value: '1024x1024', label: '1:1 1024' },
  { value: '1024x1536', label: '2:3 1024x1536' },
  { value: '1536x1024', label: '3:2 1536x1024' },
  { value: '1024x1792', label: '4:7 1024x1792' },
  { value: '1792x1024', label: '7:4 1792x1024' },
]

const qualityOptions: Array<{ value: ImageQuality; label: string }> = [
  { value: 'low', label: '快速' },
  { value: 'medium', label: '标准' },
  { value: 'high', label: '精细' },
  { value: 'auto', label: '自动' },
]

const outputFormatOptions: Array<{ value: ImageOutputFormat | ''; label: string }> = [
  { value: '', label: '接口默认' },
  { value: 'png', label: 'PNG' },
  { value: 'jpeg', label: 'JPEG' },
  { value: 'webp', label: 'WebP' },
]

const selectedWork = computed(() => works.value.find((work) => work.id === selectedWorkId.value) || works.value[0])
const normalizedBase = computed(() => normalizeImageApiBase(requestUrl.value, window.location.origin))
const canGenerate = computed(() => Boolean(apiKey.value.trim() && prompt.value.trim() && !isGenerating.value))

function saveDraft() {
  localStorage.setItem(requestUrlStorageKey, requestUrl.value)
  localStorage.setItem(draftStorageKey, JSON.stringify({
    mode: mode.value,
    model: model.value,
    prompt: prompt.value,
    subject: subject.value,
    style: style.value,
    size: size.value,
    quality: quality.value,
    outputFormat: outputFormat.value,
    batchCount: batchCount.value,
  }))
}

function saveApiKey() {
  sessionStorage.setItem(apiKeyStorageKey, apiKey.value.trim())
}

function restoreDraft() {
  requestUrl.value = localStorage.getItem(requestUrlStorageKey) || window.location.origin
  apiKey.value = sessionStorage.getItem(apiKeyStorageKey) || ''
  try {
    const draft = JSON.parse(localStorage.getItem(draftStorageKey) || '{}')
    mode.value = draft.mode === 'image' ? 'image' : 'text'
    model.value = String(draft.model || 'gpt-image-2')
    prompt.value = String(draft.prompt || '')
    subject.value = String(draft.subject || '')
    style.value = String(draft.style || 'clean-product')
    size.value = String(draft.size || '')
    quality.value = ['low', 'medium', 'high', 'auto'].includes(draft.quality) ? draft.quality : 'high'
    outputFormat.value = ['png', 'jpeg', 'webp'].includes(draft.outputFormat) ? draft.outputFormat : ''
    batchCount.value = Math.max(1, Math.min(10, Number(draft.batchCount) || 1))
  } catch {
    mode.value = 'text'
  }
}

function adoptQueryParams() {
  const params = new URLSearchParams(window.location.search)
  const base = params.get('api_base') || params.get('base_url') || params.get('request_url')
  const key = params.get('api_key') || params.get('key')
  const nextMode = params.get('mode')
  const nextModel = params.get('model')
  const nextSize = params.get('size')
  const nextQuality = params.get('quality')
  const nextFormat = params.get('format') || params.get('output_format')
  const nextPrompt = params.get('prompt')
  const nextCount = params.get('count') || params.get('n')

  isEmbed.value = params.get('embed') === '1' || params.get('embed') === 'true'
  if (base) requestUrl.value = normalizeImageApiBase(base, window.location.origin)
  if (nextMode === 'text' || nextMode === 'image') mode.value = nextMode
  if (nextModel) model.value = nextModel.slice(0, 80)
  if (nextSize) size.value = nextSize
  if (['low', 'medium', 'high', 'auto'].includes(String(nextQuality))) quality.value = nextQuality as ImageQuality
  if (['png', 'jpeg', 'webp'].includes(String(nextFormat))) outputFormat.value = nextFormat as ImageOutputFormat
  if (nextPrompt) prompt.value = nextPrompt.slice(0, 4000)
  if (nextCount) batchCount.value = Math.max(1, Math.min(10, Number(nextCount) || 1))
  if (key) {
    apiKey.value = key
    sessionStorage.setItem(apiKeyStorageKey, key)
    params.delete('api_key')
    params.delete('key')
    const nextQuery = params.toString()
    window.history.replaceState({}, '', `${window.location.pathname}${nextQuery ? `?${nextQuery}` : ''}${window.location.hash}`)
  }
}

function composePrompt() {
  const selectedStyle = styleOptions.find((item) => item.value === style.value)
  const parts = [
    prompt.value.trim(),
    subject.value.trim() ? `主体：${subject.value.trim()}` : '',
    selectedStyle?.text || '',
    '要求：画面清晰可信，主体完整，不生成水印、二维码、无关 Logo 或乱码文字。',
  ].filter(Boolean)
  return parts.join('\n')
}

function applyTemplate(nextStyle: string) {
  style.value = nextStyle
  const selectedStyle = styleOptions.find((item) => item.value === nextStyle)
  if (!prompt.value.trim() && selectedStyle?.value !== 'free') {
    prompt.value = selectedStyle?.label === '电商主图'
      ? '生成一张适合上架和投放的商品图片，突出商品本体与核心卖点。'
      : `生成一张${selectedStyle?.label || '图片'}。`
  }
  saveDraft()
}

function onReferenceChange(event: Event) {
  const input = event.target as HTMLInputElement
  referenceFiles.value = Array.from(input.files || []).filter((file) => file.type.startsWith('image/')).slice(0, 8)
}

function buildTrace(index: number) {
  const batchId = `image2-${Date.now()}`
  return {
    batchId,
    itemId: `${batchId}-${index}`,
    requestId: `${batchId}-${index}-${Math.random().toString(16).slice(2)}`,
  }
}

async function requestJson(url: string, init: RequestInit) {
  const response = await fetch(url, init)
  const text = await response.text()
  const data = text ? JSON.parse(text) : {}
  if (!response.ok) {
    throw new Error(data.error?.message || data.message || `请求失败：${response.status}`)
  }
  return data
}

async function generateOne(index: number) {
  const finalPrompt = composePrompt()
  const trace = buildTrace(index)
  const headers = {
    ...buildBearerHeaders(apiKey.value),
    ...buildClientTraceHeaders(trace),
  }

  if (mode.value === 'image') {
    if (!referenceFiles.value.length) {
      throw new Error('图生图需要先上传参考图')
    }
    const form = new FormData()
    form.append('model', model.value.trim() || 'gpt-image-2')
    form.append('prompt', finalPrompt)
    form.append('quality', quality.value)
    form.append('n', '1')
    if (size.value) form.append('size', size.value)
    if (outputFormat.value) form.append('output_format', outputFormat.value)
    const fieldName = referenceFiles.value.length > 1 ? 'image[]' : 'image'
    referenceFiles.value.forEach((file) => form.append(fieldName, file, file.name || 'reference.png'))

    return requestJson(buildImageApiUrl(normalizedBase.value, '/v1/images/edits'), {
      method: 'POST',
      headers,
      body: form,
    })
  }

  return requestJson(buildImageApiUrl(normalizedBase.value, '/v1/images/generations'), {
    method: 'POST',
    headers: {
      ...headers,
      'Content-Type': 'application/json',
    },
    body: JSON.stringify(buildTextImagePayload({
      model: model.value.trim() || 'gpt-image-2',
      prompt: finalPrompt,
      size: size.value,
      quality: quality.value,
      n: 1,
      outputFormat: outputFormat.value,
    })),
  })
}

async function generateImages() {
  errorText.value = ''
  if (!canGenerate.value) {
    errorText.value = apiKey.value.trim() ? '请先填写提示词' : '请先填写 API Key'
    return
  }

  sessionStorage.setItem(apiKeyStorageKey, apiKey.value.trim())
  saveDraft()
  isGenerating.value = true
  statusText.value = '提交中'

  try {
    const count = Math.max(1, Math.min(10, Number(batchCount.value) || 1))
    for (let index = 0; index < count; index += 1) {
      statusText.value = count > 1 ? `生成中 ${index + 1}/${count}` : '生成中'
      const payload = await generateOne(index)
      const images = extractImageResults(payload)
      if (!images.length) {
        throw new Error('接口没有返回图片字段')
      }
      const now = new Date().toLocaleString()
      const nextWorks = images.map((image, imageIndex) => ({
        id: `${Date.now()}-${index}-${imageIndex}`,
        src: image.src,
        title: `${mode.value === 'image' ? '图生图' : '文生图'} #${works.value.length + imageIndex + 1}`,
        prompt: composePrompt(),
        size: size.value || '接口默认',
        createdAt: now,
      }))
      works.value = [...nextWorks, ...works.value].slice(0, 30)
      selectedWorkId.value = nextWorks[0]?.id || selectedWorkId.value
    }
    statusText.value = '完成'
  } catch (error) {
    const message = error instanceof Error ? error.message : '生成失败'
    errorText.value = message
    statusText.value = '失败'
  } finally {
    isGenerating.value = false
  }
}

function downloadWork(work?: GeneratedWork) {
  if (!work) return
  const link = document.createElement('a')
  link.href = work.src
  link.download = `${work.title.replace(/[\\/:*?"<>|]+/g, '-')}.png`
  document.body.appendChild(link)
  link.click()
  link.remove()
}

function clearWorks() {
  works.value = []
  selectedWorkId.value = ''
}

onMounted(() => {
  restoreDraft()
  adoptQueryParams()
})
</script>

<template>
  <main class="min-h-screen bg-gray-50 text-gray-900 dark:bg-dark-900 dark:text-white">
    <section class="mx-auto flex w-full max-w-7xl flex-col gap-5 px-4 sm:px-6 lg:px-8" :class="isEmbed ? 'py-3' : 'py-5'">
      <header v-if="!isEmbed" class="flex flex-col gap-3 border-b border-gray-200 pb-4 dark:border-dark-700 md:flex-row md:items-center md:justify-between">
        <div>
          <p class="text-sm font-semibold text-primary-600 dark:text-primary-400">Sub2API</p>
          <h1 class="text-2xl font-semibold tracking-normal">图像生成</h1>
          <p class="mt-1 text-sm text-gray-500 dark:text-gray-400">可直接嵌入网站的 Sub2API 生图工具。</p>
        </div>
        <div class="flex flex-wrap items-center gap-2 text-sm">
          <span class="rounded-md border border-gray-200 px-3 py-1.5 dark:border-dark-700">模型 {{ model || 'gpt-image-2' }}</span>
          <span class="rounded-md border border-gray-200 px-3 py-1.5 dark:border-dark-700">{{ statusText }}</span>
        </div>
      </header>

      <div class="grid gap-5" :class="isEmbed ? 'lg:grid-cols-[380px_minmax(0,1fr)]' : 'lg:grid-cols-[420px_minmax(0,1fr)]'">
        <aside class="space-y-4 rounded-lg border border-gray-200 bg-white p-4 shadow-sm dark:border-dark-700 dark:bg-dark-800">
          <div v-if="isEmbed" class="space-y-1">
            <p class="text-sm font-semibold text-primary-600 dark:text-primary-400">图像生成</p>
            <p class="text-xs text-gray-500 dark:text-gray-400">输入 API Key 和提示词，结果仅在浏览器本地展示。</p>
          </div>
          <div class="space-y-2">
            <label class="text-sm font-medium" for="image2-request-url">请求网址</label>
            <input
              id="image2-request-url"
              v-model="requestUrl"
              class="w-full rounded-md border border-gray-300 bg-white px-3 py-2 text-sm outline-none focus:border-primary-500 dark:border-dark-600 dark:bg-dark-900"
              placeholder="https://sub.sunmmyapi.xyz"
              @change="saveDraft"
            >
          </div>

          <div class="space-y-2">
            <label class="text-sm font-medium" for="image2-api-key">API Key</label>
            <input
              id="image2-api-key"
              v-model="apiKey"
              class="w-full rounded-md border border-gray-300 bg-white px-3 py-2 text-sm outline-none focus:border-primary-500 dark:border-dark-600 dark:bg-dark-900"
              placeholder="sk-... 或 Bearer sk-..."
              type="password"
              autocomplete="off"
              @change="saveApiKey"
            >
          </div>

          <div class="grid grid-cols-2 gap-2">
            <button
              type="button"
              class="rounded-md border px-3 py-2 text-sm font-medium"
              :class="mode === 'text' ? 'border-primary-500 bg-primary-50 text-primary-700 dark:bg-primary-900/30 dark:text-primary-200' : 'border-gray-200 dark:border-dark-700'"
              @click="mode = 'text'; saveDraft()"
            >
              文生图
            </button>
            <button
              type="button"
              class="rounded-md border px-3 py-2 text-sm font-medium"
              :class="mode === 'image' ? 'border-primary-500 bg-primary-50 text-primary-700 dark:bg-primary-900/30 dark:text-primary-200' : 'border-gray-200 dark:border-dark-700'"
              @click="mode = 'image'; saveDraft()"
            >
              图生图
            </button>
          </div>

          <label class="space-y-2 text-sm">
            <span class="font-medium">模型</span>
            <select v-model="model" class="w-full rounded-md border border-gray-300 bg-white px-2 py-2 dark:border-dark-600 dark:bg-dark-900" @change="saveDraft">
              <option v-for="option in modelOptions" :key="option.value" :value="option.value">{{ option.label }}</option>
            </select>
          </label>

          <div class="grid grid-cols-2 gap-2">
            <button
              v-for="option in styleOptions"
              :key="option.value"
              type="button"
              class="rounded-md border px-3 py-2 text-sm"
              :class="style === option.value ? 'border-primary-500 bg-primary-50 text-primary-700 dark:bg-primary-900/30 dark:text-primary-200' : 'border-gray-200 dark:border-dark-700'"
              @click="applyTemplate(option.value)"
            >
              {{ option.label }}
            </button>
          </div>

          <div class="space-y-2">
            <label class="text-sm font-medium" for="image2-subject">主体/商品</label>
            <input
              id="image2-subject"
              v-model="subject"
              class="w-full rounded-md border border-gray-300 bg-white px-3 py-2 text-sm outline-none focus:border-primary-500 dark:border-dark-600 dark:bg-dark-900"
              placeholder="例如：折叠露营灯、人物头像、宠物"
              @input="saveDraft"
            >
          </div>

          <div v-if="mode === 'image'" class="space-y-2">
            <label class="text-sm font-medium" for="image2-reference">参考图</label>
            <input
              id="image2-reference"
              class="w-full rounded-md border border-dashed border-gray-300 bg-white px-3 py-2 text-sm dark:border-dark-600 dark:bg-dark-900"
              type="file"
              accept="image/*"
              multiple
              @change="onReferenceChange"
            >
            <p class="text-xs text-gray-500 dark:text-gray-400">最多 8 张参考图，请求字段自动使用 image 或 image[]。</p>
          </div>

          <div class="grid grid-cols-2 gap-2 sm:grid-cols-4">
            <label class="space-y-2 text-sm">
              <span class="font-medium">尺寸</span>
              <select v-model="size" class="w-full rounded-md border border-gray-300 bg-white px-2 py-2 dark:border-dark-600 dark:bg-dark-900" @change="saveDraft">
                <option v-for="option in sizeOptions" :key="option.value" :value="option.value">{{ option.label }}</option>
              </select>
            </label>
            <label class="space-y-2 text-sm">
              <span class="font-medium">质量</span>
              <select v-model="quality" class="w-full rounded-md border border-gray-300 bg-white px-2 py-2 dark:border-dark-600 dark:bg-dark-900" @change="saveDraft">
                <option v-for="option in qualityOptions" :key="option.value" :value="option.value">{{ option.label }}</option>
              </select>
            </label>
            <label class="space-y-2 text-sm">
              <span class="font-medium">格式</span>
              <select v-model="outputFormat" class="w-full rounded-md border border-gray-300 bg-white px-2 py-2 dark:border-dark-600 dark:bg-dark-900" @change="saveDraft">
                <option v-for="option in outputFormatOptions" :key="option.value" :value="option.value">{{ option.label }}</option>
              </select>
            </label>
            <label class="space-y-2 text-sm">
              <span class="font-medium">张数</span>
              <input v-model.number="batchCount" class="w-full rounded-md border border-gray-300 bg-white px-2 py-2 dark:border-dark-600 dark:bg-dark-900" max="10" min="1" type="number" @change="saveDraft">
            </label>
          </div>

          <div class="space-y-2">
            <label class="text-sm font-medium" for="image2-prompt">提示词</label>
            <textarea
              id="image2-prompt"
              v-model="prompt"
              class="min-h-40 w-full resize-y rounded-md border border-gray-300 bg-white px-3 py-2 text-sm leading-6 outline-none focus:border-primary-500 dark:border-dark-600 dark:bg-dark-900"
              placeholder="描述你要画什么"
              @input="saveDraft"
            />
          </div>

          <button
            type="button"
            class="w-full rounded-md bg-primary-600 px-4 py-3 text-sm font-semibold text-white disabled:cursor-not-allowed disabled:opacity-50"
            :disabled="!canGenerate"
            @click="generateImages"
          >
            {{ isGenerating ? '生成中...' : '生成图片' }}
          </button>

          <p v-if="errorText" class="rounded-md border border-red-200 bg-red-50 px-3 py-2 text-sm text-red-700 dark:border-red-900/60 dark:bg-red-950/30 dark:text-red-200">
            {{ errorText }}
          </p>
        </aside>

        <section class="space-y-4">
          <div class="rounded-lg border border-gray-200 bg-white p-4 shadow-sm dark:border-dark-700 dark:bg-dark-800">
            <div class="mb-3 flex items-center justify-between gap-3">
              <div>
                <h2 class="text-lg font-semibold">生成预览</h2>
                <p class="text-sm text-gray-500 dark:text-gray-400">{{ selectedWork ? selectedWork.size : '等待生成' }}</p>
              </div>
              <button
                type="button"
                class="rounded-md border border-gray-200 px-3 py-2 text-sm disabled:opacity-50 dark:border-dark-700"
                :disabled="!selectedWork"
                @click="downloadWork(selectedWork)"
              >
                下载当前
              </button>
            </div>

            <div class="flex min-h-[420px] items-center justify-center rounded-md bg-gray-100 p-3 dark:bg-dark-900">
              <img
                v-if="selectedWork"
                :src="selectedWork.src"
                :alt="selectedWork.title"
                class="max-h-[680px] max-w-full rounded-md object-contain"
              >
              <div v-else class="text-center text-sm text-gray-500 dark:text-gray-400">
                真实生成结果会显示在这里
              </div>
            </div>
          </div>

          <div class="rounded-lg border border-gray-200 bg-white p-4 shadow-sm dark:border-dark-700 dark:bg-dark-800">
            <div class="mb-3 flex items-center justify-between gap-3">
              <div>
                <h2 class="text-lg font-semibold">作品库</h2>
                <p class="text-sm text-gray-500 dark:text-gray-400">{{ works.length }} 张，本地临时保存</p>
              </div>
              <button
                type="button"
                class="rounded-md border border-gray-200 px-3 py-2 text-sm disabled:opacity-50 dark:border-dark-700"
                :disabled="!works.length"
                @click="clearWorks"
              >
                清空
              </button>
            </div>

            <div v-if="works.length" class="grid grid-cols-2 gap-3 md:grid-cols-3 xl:grid-cols-4">
              <button
                v-for="work in works"
                :key="work.id"
                type="button"
                class="group overflow-hidden rounded-md border text-left"
                :class="selectedWork?.id === work.id ? 'border-primary-500' : 'border-gray-200 dark:border-dark-700'"
                @click="selectedWorkId = work.id"
              >
                <img :src="work.src" :alt="work.title" class="aspect-square w-full object-cover">
                <span class="block space-y-1 px-2 py-2">
                  <span class="block truncate text-sm font-medium">{{ work.title }}</span>
                  <span class="block truncate text-xs text-gray-500 dark:text-gray-400">{{ work.createdAt }}</span>
                </span>
              </button>
            </div>
            <p v-else class="rounded-md bg-gray-50 px-3 py-6 text-center text-sm text-gray-500 dark:bg-dark-900 dark:text-gray-400">
              暂无作品
            </p>
          </div>
        </section>
      </div>
    </section>
  </main>
</template>
