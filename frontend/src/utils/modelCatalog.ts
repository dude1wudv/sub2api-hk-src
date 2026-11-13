import type { PlazaModel } from '@/api/modelPlaza'

// A transport (for example, OpenAI-compatible) is not a model vendor.
// Keep unrecognized aliases in Other rather than asserting an incorrect vendor.
const vendors = [
  { id: 'deepseek', label: 'DeepSeek', match: /deepseek/i },
  { id: 'claude', label: 'Claude', match: /claude|anthropic/i },
  { id: 'gemini', label: 'Gemini', match: /gemini|gemma|learnlm|imagen|veo/i },
  { id: 'openai', label: 'OpenAI', match: /(?:^|[/:])(?:gpt|chatgpt|o[134](?:-|$)|dall-e|sora|whisper|tts-|text-embedding|codex)/i },
  { id: 'grok', label: 'xAI', match: /grok/i },
  { id: 'minimax', label: 'MiniMax', match: /minimax|abab/i },
  { id: 'qwen', label: 'Qwen', match: /qwen|qwq/i },
  { id: 'moonshot', label: 'Moonshot', match: /kimi|moonshot/i },
  { id: 'zhipu', label: 'Z.ai', match: /glm|cogview|cogvideo/i },
  { id: 'mistral', label: 'Mistral', match: /mistral|mixtral|codestral|pixtral|voxtral|magistral/i },
  { id: 'meta', label: 'Meta', match: /llama/i },
  { id: 'bytedance', label: 'ByteDance', match: /doubao|seedance|seedream|seed-/i },
  { id: 'tencent', label: 'Tencent', match: /hunyuan/i },
  { id: 'xiaomi', label: 'Xiaomi', match: /mimo/i },
  { id: 'typesafe', label: 'TypeSafe', match: /(?:^|[/:])jev(?:[-.\s]|$)|typesafe/i },
  { id: 'cohere', label: 'Cohere', match: /cohere|command-r|c4ai/i },
] as const

export function modelVendor(model: PlazaModel): { id: string; label: string } {
  const vendor = vendors.find(vendor => vendor.match.test(model.name))
    ?? vendors.find(vendor => vendor.match.test(model.family || ''))
  return vendor ? { id: vendor.id, label: vendor.label } : { id: 'other', label: 'Other' }
}

export function matchesCatalogModel(model: PlazaModel, query: string): boolean {
  const needle = query.trim().toLowerCase()
  return !needle || [model.name, model.family, modelVendor(model).label].filter(Boolean).join(' ').toLowerCase().includes(needle)
}
