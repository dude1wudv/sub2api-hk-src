import type { PriceCurrency } from './api'

export function formatQuota(value: number | null | undefined, digits = 2): string {
  if (value === null || value === undefined || !Number.isFinite(value)) {
    return '0 单位'
  }
  if (value > 0 && value < 0.01) {
    return `${value.toFixed(6).replace(/0+$/, '')} 单位`
  }
  return `${Number(value.toFixed(digits))} 单位`
}

export function formatPricePerMillion(value: number | null | undefined, currency: PriceCurrency): string {
  if (value === null || value === undefined || !Number.isFinite(value)) {
    return '—'
  }
  return `${currency === 'CNY' ? '¥' : '$'}${value}`
}

export function formatCountdown(totalSeconds: number): string {
  const safe = Math.max(0, Math.floor(totalSeconds))
  const hours = Math.floor(safe / 3600)
  const minutes = Math.floor((safe % 3600) / 60)
  const seconds = safe % 60
  return [hours, minutes, seconds].map((n) => String(n).padStart(2, '0')).join(':')
}

export function maskAPIKey(key: string): string {
  if (!key) return '暂未开放'
  if (key.length <= 18) return key
  return `${key.slice(0, 14)}••••••••${key.slice(-6)}`
}

export function computePercent(used: number, limit: number | null | undefined): number {
  if (!limit || limit <= 0 || !Number.isFinite(used)) return 0
  return Math.min(100, Math.max(0, (used / limit) * 100))
}

export type IntegrationClient = 'claude_code' | 'codex' | 'python' | 'curl' | 'cherry'

export function buildIntegrationSnippet(
  client: IntegrationClient,
  origin: string,
  apiKey: string,
  sampleModel: string
): string {
  const base = origin.replace(/\/+$/, '')
  const key = apiKey || 'sk-free-请先复制上方今日公共Key'
  const model = sampleModel || 'gpt-5.4'

  switch (client) {
    case 'claude_code':
      return [
        `export ANTHROPIC_BASE_URL="${base}"`,
        `export ANTHROPIC_AUTH_TOKEN="${key}"`,
        'claude'
      ].join('\n')
    case 'codex':
      return [
        `export OPENAI_BASE_URL="${base}/v1"`,
        `export OPENAI_API_KEY="${key}"`,
        'codex'
      ].join('\n')
    case 'python':
      return [
        'from openai import OpenAI',
        '',
        'client = OpenAI(',
        `    base_url="${base}/v1",`,
        `    api_key="${key}",`,
        ')',
        '',
        'resp = client.chat.completions.create(',
        `    model="${model}",`,
        '    messages=[{"role": "user", "content": "Hello from XJTU!"}],',
        ')',
        'print(resp.choices[0].message.content)'
      ].join('\n')
    case 'curl':
      return [
        `curl ${base}/v1/chat/completions \\`,
        '  -H "Content-Type: application/json" \\',
        `  -H "Authorization: Bearer ${key}" \\`,
        `  -d '{"model":"${model}","messages":[{"role":"user","content":"你好"}]}'`
      ].join('\n')
    case 'cherry':
      return [
        `API 地址 (Base URL)：${base}/v1`,
        `Anthropic 协议地址：${base}`,
        `API 密钥 (Key)：${key}`,
        `示例模型：${model}`,
        '提示：公共 Key 每日北京时间 00:00 自动轮换，次日如遇 401 请回到本页复制新 Key。'
      ].join('\n')
  }
}
