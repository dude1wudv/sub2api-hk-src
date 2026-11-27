import { BRAND_ICON_DATA } from './brandIconData'

export function resolveBrandKey(raw?: string | null): string | null {
  const value = (raw || '').trim().toLowerCase()
  if (!value) return null
  if (value in BRAND_ICON_DATA) return value
  if (value.includes('claude') || value.includes('anthropic')) return 'claude'
  if (
    value.includes('gpt') ||
    value.includes('openai') ||
    value.includes('codex') ||
    /^o[134](-|$)/.test(value) ||
    value.includes('chatgpt') ||
    value.includes('dall-e') ||
    value.includes('whisper') ||
    value.includes('tts-') ||
    value.includes('text-embedding')
  ) {
    return 'openai'
  }
  if (value.includes('gemini') || value.includes('imagen') || value.includes('veo') || value.includes('gemma')) return 'gemini'
  if (value.includes('google') || value.includes('vertex')) return 'google'
  if (value.includes('deepseek')) return 'deepseek'
  if (value.includes('grok') || value.includes('xai') || value.includes('x-ai')) return 'grok'
  if (value.includes('kimi') || value.includes('moonshot')) return 'kimi'
  if (value.includes('glm') || value.includes('zhipu') || value.includes('bigmodel')) return 'zhipu'
  if (value.includes('qwen') || value.includes('qwq') || value.includes('wanx') || value.includes('alibaba')) return 'qwen'
  if (value.includes('minimax') || value.includes('abab')) return 'minimax'
  if (value.includes('mistral') || value.includes('mixtral') || value.includes('pixtral') || value.includes('codestral')) return 'mistral'
  if (value.includes('llama') || value.includes('meta')) return 'meta'
  if (value.includes('doubao') || value.includes('seedream') || value.includes('bytedance') || value.includes('ark')) return 'doubao'
  if (value.includes('spark')) return 'spark'
  if (value.includes('hunyuan')) return 'hunyuan'
  if (value.includes('ernie') || value.includes('wenxin') || value.includes('baidu')) return 'wenxin'
  if (value.includes('sonar') || value.includes('perplexity')) return 'perplexity'
  if (value.includes('command') || value.includes('cohere')) return 'cohere'
  if (value.includes('cloudflare')) return 'cloudflare'
  if (value.includes('midjourney') || value.includes('mj-')) return 'midjourney'
  if (value.includes('suno')) return 'suno'
  if (value.includes('ollama')) return 'ollama'
  if (value.includes('openrouter')) return 'openrouter'
  if (value.includes('step') || value.includes('stepfun')) return 'stepfun'
  if (value.includes('yi-')) return 'yi'
  if (value.includes('jina')) return 'jina'
  if (value.includes('baichuan')) return 'baichuan'
  if (value.includes('bedrock')) return 'bedrock'
  if (value.includes('dify')) return 'dify'
  if (value.includes('coze')) return 'coze'
  if (value.includes('360')) return 'ai360'
  if (value.includes('cline')) return 'cline'
  return null
}
