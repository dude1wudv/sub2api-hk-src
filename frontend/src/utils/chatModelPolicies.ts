import type { ChatModelPolicies, ChatModelPolicy } from '@/types'

const flags = [
  'image_input', 'document_input', 'native_deepseek_extensions',
  'native_files', 'native_search', 'native_read_url',
] as const
const allowedFields = new Set<string>(['transport', ...flags])

export function parseChatModelPolicies(text: string): ChatModelPolicies {
  const parsed: unknown = JSON.parse(text)
  if (!parsed || typeof parsed !== 'object' || Array.isArray(parsed)) {
    throw new Error('Expected an object keyed by exact public model names')
  }
  const entries: [string, unknown][] = Object.entries(parsed)
  if (entries.length > 1000) throw new Error('Too many model policies')
  return Object.fromEntries(entries.map(([model, value]) => {
    if (!model || model !== model.trim() || new TextEncoder().encode(model).length > 256 ||
      ['*', '?', String.fromCharCode(92), String.fromCharCode(13), String.fromCharCode(10)].some(c => model.includes(c))) {
      throw new Error('Expected an exact public model name')
    }
    if (!value || typeof value !== 'object' || Array.isArray(value)) throw new Error('Invalid policy')
    const policy: Record<string, unknown> = Object.fromEntries(Object.entries(value))
    if (Object.keys(policy).some(key => !allowedFields.has(key))) {
      throw new Error('Unknown policy field')
    }
    const transport = policy.transport
    if (transport !== 'chat_completions' && transport !== 'messages' && transport !== 'responses') {
      throw new Error('Invalid transport')
    }
    for (const flag of flags) {
      if (policy[flag] !== undefined && typeof policy[flag] !== 'boolean') throw new Error('Flags must be boolean')
    }
    const result: ChatModelPolicy = {
      transport,
      image_input: policy.image_input === true,
      document_input: policy.document_input === true,
      native_deepseek_extensions: policy.native_deepseek_extensions === true,
      native_files: policy.native_files === true,
      native_search: policy.native_search === true,
      native_read_url: policy.native_read_url === true,
    }
    if ((result.native_deepseek_extensions || result.native_files || result.native_search || result.native_read_url) && result.transport !== 'messages') {
      throw new Error('Native DeepSeek capabilities require Messages transport')
    }
    if (result.document_input && !result.native_files) throw new Error('Documents require native Files')
    return [model, result]
  }))
}
