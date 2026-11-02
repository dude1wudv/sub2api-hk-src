const labels: Record<string, string> = {
  openai: 'OpenAI', anthropic: 'Anthropic', gemini: 'Gemini', antigravity: 'Antigravity', grok: 'Grok',
  deepseek: 'DeepSeek', typesafe: 'TypeSafe', ollama: 'Ollama', opencode: 'OpenCode',
}
export function platformLabel(platform: string): string { return labels[platform.toLowerCase()] || platform }
