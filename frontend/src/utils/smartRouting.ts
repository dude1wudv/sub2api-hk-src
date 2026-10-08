import type { Group } from '@/types'

const platforms = new Set(['anthropic', 'openai', 'grok', 'kimi', 'zhipu', 'deepseek', 'minimax', 'stepfun'])

export function isSmartRoutingGroup(group: Pick<Group, 'platform' | 'status'>): boolean {
  return group.status === 'active' && platforms.has(group.platform)
}
