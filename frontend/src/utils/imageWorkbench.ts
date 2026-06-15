export type ImageQuality = 'low' | 'medium' | 'high' | 'auto'
export type ImageOutputFormat = 'png' | 'jpeg' | 'webp'

export interface TextImagePayloadInput {
  model: string
  prompt: string
  size?: string
  quality?: ImageQuality
  n?: number
  outputFormat?: ImageOutputFormat | ''
}

export interface ImageTraceInput {
  batchId?: string
  itemId?: string
  requestId?: string
  width?: number
  height?: number
}

export interface ExtractedImageResult {
  src: string
  kind: 'url' | 'b64'
}

const imageUrlPattern = /https?:\/\/[^\s"'<>]+?\.(?:png|jpe?g|webp|gif|avif)(?:\?[^\s"'<>]*)?/gi
const dataUrlPattern = /data:image\/[a-zA-Z0-9.+-]+;base64,[A-Za-z0-9+/=\s]+/i

export function normalizeImageApiBase(value: string, fallback = ''): string {
  const raw = String(value || '').trim()
  const safeFallback = String(fallback || '').trim().replace(/\/+$/, '')
  if (!raw) return safeFallback

  try {
    const url = new URL(raw)
    url.hash = ''
    url.search = ''
    url.pathname = url.pathname.replace(/\/+$/, '')
    if (url.pathname === '/v1') {
      url.pathname = ''
    }
    return url.toString().replace(/\/+$/, '')
  } catch {
    return safeFallback
  }
}

export function buildImageApiUrl(base: string, path: string): string {
  const normalizedBase = normalizeImageApiBase(base)
  const normalizedPath = `/${String(path || '').replace(/^\/+/, '')}`
  if (!normalizedBase) return normalizedPath
  return `${normalizedBase}${normalizedPath}`
}

export function buildBearerHeaders(key: string): Record<string, string> {
  const trimmed = String(key || '').trim()
  if (!trimmed) {
    throw new Error('请先填写 API Key')
  }
  return {
    Authorization: /^bearer\s+/i.test(trimmed) ? trimmed : `Bearer ${trimmed}`,
  }
}

export function buildTextImagePayload(input: TextImagePayloadInput): Record<string, unknown> {
  const payload: Record<string, unknown> = {
    model: input.model,
    prompt: input.prompt,
    n: Math.max(1, Math.min(10, Number(input.n) || 1)),
    quality: input.quality || 'high',
  }
  if (input.size) {
    payload.size = input.size
  }
  if (input.outputFormat) {
    payload.output_format = input.outputFormat
  }
  return payload
}

export function buildClientTraceHeaders(trace: ImageTraceInput): Record<string, string> {
  const headers: Record<string, string> = {}
  const addAscii = (key: string, value: unknown) => {
    const text = String(value || '').trim()
    if (text && /^[\x20-\x7E]+$/.test(text)) {
      headers[key] = text.slice(0, 160)
    }
  }

  addAscii('X-TK-Client-Batch-Id', trace.batchId)
  addAscii('X-TK-Client-Item-Id', trace.itemId)
  addAscii('X-TK-Client-Request-Id', trace.requestId)
  if (Number(trace.width) > 0) addAscii('X-TK-Target-Width', Math.round(Number(trace.width)))
  if (Number(trace.height) > 0) addAscii('X-TK-Target-Height', Math.round(Number(trace.height)))
  addAscii('X-TK-Client-Created-At', new Date().toISOString())
  return headers
}

export function extractImageResults(payload: unknown): ExtractedImageResult[] {
  const results: ExtractedImageResult[] = []
  const seen = new Set<string>()
  const addResult = (src: string, kind: ExtractedImageResult['kind']) => {
    const clean = src.trim().replace(/\s/g, kind === 'b64' ? '' : ' ')
    if (!clean || seen.has(clean)) return
    seen.add(clean)
    results.push({ src: clean, kind })
  }

  const visit = (value: unknown) => {
    if (!value) return
    if (typeof value === 'string') {
      const text = value.trim()
      const dataUrl = text.match(dataUrlPattern)?.[0]
      if (dataUrl) {
        addResult(dataUrl.replace(/\s/g, ''), 'b64')
        return
      }
      const urls = text.match(imageUrlPattern)
      if (urls?.length) {
        urls.forEach((url) => addResult(url, 'url'))
        return
      }
      if (/^[A-Za-z0-9+/=\s]{80,}$/.test(text)) {
        addResult(`data:image/png;base64,${text.replace(/\s/g, '')}`, 'b64')
      }
      return
    }
    if (Array.isArray(value)) {
      value.forEach(visit)
      return
    }
    if (typeof value !== 'object') return

    const record = value as Record<string, unknown>
    const url = record.url || record.image_url || record.imageUrl || record.output_url || record.src
    const b64 = record.b64_json || record.b64Json || record.base64 || record.image_base64
    if (typeof url === 'string') visit(url)
    if (typeof b64 === 'string') addResult(`data:image/png;base64,${b64.replace(/\s/g, '')}`, 'b64')
    visit(record.image)
    visit(record.result)
    visit(record.content)
    visit(record.data)
    visit(record.images)
    visit(record.output)
    visit(record.message)
  }

  visit(payload)
  return results
}
