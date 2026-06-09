import { describe, expect, it } from 'vitest'
import {
  buildBearerHeaders,
  buildClientTraceHeaders,
  buildImageApiUrl,
  buildTextImagePayload,
  extractImageResults,
  normalizeImageApiBase,
} from '../imageWorkbench'

describe('imageWorkbench', () => {
  it('normalizes empty and v1 request URLs', () => {
    expect(normalizeImageApiBase('', 'https://sub.example.com')).toBe('https://sub.example.com')
    expect(normalizeImageApiBase('https://sub.example.com/v1/', 'https://fallback.example.com')).toBe('https://sub.example.com')
    expect(normalizeImageApiBase('not a url', 'https://fallback.example.com')).toBe('https://fallback.example.com')
  })

  it('builds OpenAI-compatible image endpoint URLs without duplicating v1', () => {
    expect(buildImageApiUrl('https://sub.example.com/v1', '/v1/images/generations')).toBe(
      'https://sub.example.com/v1/images/generations',
    )
    expect(buildImageApiUrl('https://sub.example.com/api', '/v1/images/edits')).toBe(
      'https://sub.example.com/api/v1/images/edits',
    )
  })

  it('builds bearer headers from raw key or prefixed key', () => {
    expect(buildBearerHeaders('sk-test')).toEqual({ Authorization: 'Bearer sk-test' })
    expect(buildBearerHeaders('Bearer sk-test')).toEqual({ Authorization: 'Bearer sk-test' })
    expect(() => buildBearerHeaders('')).toThrow('API Key')
  })

  it('builds compact text image payload', () => {
    expect(buildTextImagePayload({
      model: 'gpt-image-2',
      prompt: 'draw a product',
      size: '1024x1024',
      quality: 'high',
    })).toEqual({
      model: 'gpt-image-2',
      prompt: 'draw a product',
      n: 1,
      quality: 'high',
      size: '1024x1024',
    })
  })

  it('builds ascii trace headers and drops empty fields', () => {
    expect(buildClientTraceHeaders({
      batchId: 'batch-1',
      itemId: 'item-1',
      requestId: 'req-1',
      width: 1024,
      height: 768,
    })).toMatchObject({
      'X-TK-Client-Batch-Id': 'batch-1',
      'X-TK-Client-Item-Id': 'item-1',
      'X-TK-Client-Request-Id': 'req-1',
      'X-TK-Target-Width': '1024',
      'X-TK-Target-Height': '768',
    })
  })

  it('extracts urls and base64 images from common response shapes', () => {
    const results = extractImageResults({
      data: [{ b64_json: 'aGVsbG8=' }],
      output: [{ result: 'https://cdn.example.com/image.png' }],
      message: 'see https://cdn.example.com/other.webp',
    })

    expect(results).toEqual([
      { src: 'data:image/png;base64,aGVsbG8=', kind: 'b64' },
      { src: 'https://cdn.example.com/image.png', kind: 'url' },
      { src: 'https://cdn.example.com/other.webp', kind: 'url' },
    ])
  })
})
