export type PlaygroundProtocol = 'openai' | 'anthropic'

export interface PlaygroundRequestOptions {
  protocol: PlaygroundProtocol
  model: string
  systemPrompt: string
  userPrompt: string
  temperature: number
  maxTokens: number
  stream: boolean
}

export interface PlaygroundUsage {
  inputTokens: number
  outputTokens: number
  totalTokens: number
}

export interface PlaygroundSseChunk {
  textDelta: string
  usage?: Partial<PlaygroundUsage>
  done?: boolean
  rawEvent?: unknown
}

export function buildPlaygroundEndpoint(protocol: PlaygroundProtocol, origin = ''): string {
  const base = origin.replace(/\/+$/, '')
  return protocol === 'anthropic' ? `${base}/v1/messages` : `${base}/v1/chat/completions`
}

export function buildPlaygroundHeaders(protocol: PlaygroundProtocol, apiKey: string): Record<string, string> {
  const trimmed = apiKey.trim()
  if (protocol === 'anthropic') {
    return {
      'Content-Type': 'application/json',
      'x-api-key': trimmed,
      'anthropic-version': '2023-06-01'
    }
  }
  return {
    'Content-Type': 'application/json',
    Authorization: `Bearer ${trimmed}`
  }
}

export function buildPlaygroundBody(options: PlaygroundRequestOptions): Record<string, unknown> {
  const model = options.model.trim()
  const system = options.systemPrompt.trim()
  const user = options.userPrompt.trim()

  if (options.protocol === 'anthropic') {
    const body: Record<string, unknown> = {
      model,
      max_tokens: options.maxTokens,
      temperature: options.temperature,
      stream: options.stream,
      messages: [{ role: 'user', content: user }]
    }
    if (system) body.system = system
    return body
  }

  const messages: Array<{ role: string; content: string }> = []
  if (system) messages.push({ role: 'system', content: system })
  messages.push({ role: 'user', content: user })

  const body: Record<string, unknown> = {
    model,
    messages,
    temperature: options.temperature,
    max_tokens: options.maxTokens,
    stream: options.stream
  }
  if (options.stream) {
    body.stream_options = { include_usage: true }
  }
  return body
}

export function parseSseDataLine(protocol: PlaygroundProtocol, data: string): PlaygroundSseChunk {
  const trimmed = data.trim()
  if (!trimmed) return { textDelta: '' }
  if (trimmed === '[DONE]') return { textDelta: '', done: true }

  let parsed: any
  try {
    parsed = JSON.parse(trimmed)
  } catch {
    return { textDelta: '' }
  }

  if (protocol === 'anthropic') {
    let textDelta = ''
    if (parsed?.type === 'content_block_delta' && typeof parsed?.delta?.text === 'string') {
      textDelta = parsed.delta.text
    }
    const usage: Partial<PlaygroundUsage> = {}
    if (typeof parsed?.message?.usage?.input_tokens === 'number') {
      usage.inputTokens = parsed.message.usage.input_tokens
    }
    if (typeof parsed?.usage?.input_tokens === 'number') {
      usage.inputTokens = parsed.usage.input_tokens
    }
    if (typeof parsed?.usage?.output_tokens === 'number') {
      usage.outputTokens = parsed.usage.output_tokens
    }
    return {
      textDelta,
      usage: Object.keys(usage).length ? usage : undefined,
      done: parsed?.type === 'message_stop',
      rawEvent: parsed
    }
  }

  const choice = parsed?.choices?.[0]
  const deltaContent =
    typeof choice?.delta?.content === 'string'
      ? choice.delta.content
      : typeof choice?.delta?.reasoning_content === 'string'
        ? choice.delta.reasoning_content
        : ''

  let usage: Partial<PlaygroundUsage> | undefined
  if (parsed?.usage) {
    const inputTokens = Number(parsed.usage.prompt_tokens ?? parsed.usage.input_tokens ?? 0)
    const outputTokens = Number(parsed.usage.completion_tokens ?? parsed.usage.output_tokens ?? 0)
    const totalTokens = Number(parsed.usage.total_tokens ?? inputTokens + outputTokens)
    usage = { inputTokens, outputTokens, totalTokens }
  }

  return {
    textDelta: deltaContent,
    usage,
    done: Boolean(choice?.finish_reason),
    rawEvent: parsed
  }
}

export function extractNonStreamResult(protocol: PlaygroundProtocol, payload: any): {
  text: string
  usage: PlaygroundUsage
} {
  if (protocol === 'anthropic') {
    const blocks = Array.isArray(payload?.content) ? payload.content : []
    const text = blocks
      .filter((b: any) => b?.type === 'text' && typeof b?.text === 'string')
      .map((b: any) => b.text)
      .join('')
    const inputTokens = Number(payload?.usage?.input_tokens ?? 0)
    const outputTokens = Number(payload?.usage?.output_tokens ?? 0)
    return {
      text,
      usage: { inputTokens, outputTokens, totalTokens: inputTokens + outputTokens }
    }
  }

  const content = payload?.choices?.[0]?.message?.content
  const text = typeof content === 'string' ? content : ''
  const inputTokens = Number(payload?.usage?.prompt_tokens ?? payload?.usage?.input_tokens ?? 0)
  const outputTokens = Number(payload?.usage?.completion_tokens ?? payload?.usage?.output_tokens ?? 0)
  const totalTokens = Number(payload?.usage?.total_tokens ?? inputTokens + outputTokens)
  return {
    text,
    usage: { inputTokens, outputTokens, totalTokens }
  }
}

export function buildSnippet(
  lang: 'curl' | 'python' | 'node',
  options: PlaygroundRequestOptions,
  baseUrl: string,
  maskedKeyPlaceholder = 'YOUR_API_KEY'
): string {
  const cleanOrigin = baseUrl.replace(/\/v1\/?$/, '').replace(/\/+$/, '')
  const endpoint = buildPlaygroundEndpoint(options.protocol, cleanOrigin)
  const body = buildPlaygroundBody(options)
  const jsonBody = JSON.stringify(body, null, 2)

  if (lang === 'curl') {
    if (options.protocol === 'anthropic') {
      return `curl -X POST "${endpoint}" \\\n  -H "Content-Type: application/json" \\\n  -H "x-api-key: ${maskedKeyPlaceholder}" \\\n  -H "anthropic-version: 2023-06-01" \\\n  -d '${jsonBody}'`
    }
    return `curl -X POST "${endpoint}" \\\n  -H "Content-Type: application/json" \\\n  -H "Authorization: Bearer ${maskedKeyPlaceholder}" \\\n  -d '${jsonBody}'`
  }

  if (lang === 'python') {
    if (options.protocol === 'anthropic') {
      return `import anthropic\n\nclient = anthropic.Anthropic(\n    api_key="${maskedKeyPlaceholder}",\n    base_url="${cleanOrigin}"\n)\n\nmessage = client.messages.create(\n    model=${JSON.stringify(options.model)},\n    max_tokens=${options.maxTokens},\n    temperature=${options.temperature},\n    ${options.systemPrompt.trim() ? `system=${JSON.stringify(options.systemPrompt.trim())},\n    ` : ''}messages=[{"role": "user", "content": ${JSON.stringify(options.userPrompt.trim())}}]\n)\nprint(message.content[0].text)`
    }
    return `from openai import OpenAI\n\nclient = OpenAI(\n    api_key="${maskedKeyPlaceholder}",\n    base_url="${cleanOrigin}/v1"\n)\n\nresponse = client.chat.completions.create(\n    model=${JSON.stringify(options.model)},\n    temperature=${options.temperature},\n    max_tokens=${options.maxTokens},\n    messages=${JSON.stringify(body.messages, null, 4)}\n)\nprint(response.choices[0].message.content)`
  }

  if (options.protocol === 'anthropic') {
    return `const res = await fetch(${JSON.stringify(endpoint)}, {\n  method: "POST",\n  headers: {\n    "Content-Type": "application/json",\n    "x-api-key": "${maskedKeyPlaceholder}",\n    "anthropic-version": "2023-06-01"\n  },\n  body: JSON.stringify(${jsonBody})\n});\nconsole.log(await res.json());`
  }
  return `const res = await fetch(${JSON.stringify(endpoint)}, {\n  method: "POST",\n  headers: {\n    "Content-Type": "application/json",\n    "Authorization": "Bearer ${maskedKeyPlaceholder}"\n  },\n  body: JSON.stringify(${jsonBody})\n});\nconsole.log(await res.json());`
}
