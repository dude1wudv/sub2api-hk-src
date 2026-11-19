import { describe, expect, it } from 'vitest'
import {
  buildPlaygroundBody,
  buildPlaygroundEndpoint,
  buildPlaygroundHeaders,
  buildSnippet,
  extractNonStreamResult,
  parseSseDataLine
} from '@/utils/playgroundStream'

describe('playgroundStream utilities', () => {
  it('builds OpenAI and Anthropic endpoints, headers, and bodies correctly', () => {
    expect(buildPlaygroundEndpoint('openai', 'https://api.example.com/')).toBe(
      'https://api.example.com/v1/chat/completions'
    )
    expect(buildPlaygroundEndpoint('anthropic', 'https://api.example.com')).toBe(
      'https://api.example.com/v1/messages'
    )

    expect(buildPlaygroundHeaders('openai', 'sk-test').Authorization).toBe('Bearer sk-test')
    expect(buildPlaygroundHeaders('anthropic', 'sk-ant')['x-api-key']).toBe('sk-ant')

    const oaBody = buildPlaygroundBody({
      protocol: 'openai',
      model: 'gpt-4o',
      systemPrompt: 'Be brief',
      userPrompt: 'Hello',
      temperature: 0.5,
      maxTokens: 256,
      stream: true
    })
    expect(oaBody.stream_options).toEqual({ include_usage: true })
    expect(oaBody.messages).toEqual([
      { role: 'system', content: 'Be brief' },
      { role: 'user', content: 'Hello' }
    ])

    const antBody = buildPlaygroundBody({
      protocol: 'anthropic',
      model: 'claude-sonnet-4-5',
      systemPrompt: 'Be brief',
      userPrompt: 'Hello',
      temperature: 0.2,
      maxTokens: 512,
      stream: false
    })
    expect(antBody.system).toBe('Be brief')
    expect(antBody.messages).toEqual([{ role: 'user', content: 'Hello' }])
  })

  it('parses OpenAI and Anthropic SSE chunks and usage metrics', () => {
    const oaChunk = parseSseDataLine(
      'openai',
      JSON.stringify({ choices: [{ delta: { content: 'Hi' } }] })
    )
    expect(oaChunk.textDelta).toBe('Hi')

    const oaUsage = parseSseDataLine(
      'openai',
      JSON.stringify({ choices: [], usage: { prompt_tokens: 12, completion_tokens: 8, total_tokens: 20 } })
    )
    expect(oaUsage.usage).toEqual({ inputTokens: 12, outputTokens: 8, totalTokens: 20 })
    expect(parseSseDataLine('openai', '[DONE]').done).toBe(true)

    const antChunk = parseSseDataLine(
      'anthropic',
      JSON.stringify({ type: 'content_block_delta', delta: { text: 'World' } })
    )
    expect(antChunk.textDelta).toBe('World')
  })

  it('extracts non-stream responses and generates masked snippets without leaking real keys', () => {
    const oa = extractNonStreamResult('openai', {
      choices: [{ message: { content: 'Done' } }],
      usage: { prompt_tokens: 5, completion_tokens: 3, total_tokens: 8 }
    })
    expect(oa).toEqual({ text: 'Done', usage: { inputTokens: 5, outputTokens: 3, totalTokens: 8 } })

    const curl = buildSnippet(
      'curl',
      {
        protocol: 'openai',
        model: 'gpt-4o',
        systemPrompt: '',
        userPrompt: 'Ping',
        temperature: 0.7,
        maxTokens: 128,
        stream: true
      },
      'https://gw.example.com/v1'
    )
    expect(curl).toContain('https://gw.example.com/v1/chat/completions')
    expect(curl).toContain('YOUR_API_KEY')
  })
})
