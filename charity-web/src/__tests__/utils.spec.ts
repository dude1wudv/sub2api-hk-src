import { describe, expect, it } from 'vitest'
import {
  buildIntegrationSnippet,
  computePercent,
  formatCountdown,
  formatPricePerMillion,
  formatQuota,
  maskAPIKey
} from '../utils'

describe('charity-web utils', () => {
  it('formats shared quota units without a currency symbol', () => {
    expect(formatQuota(100)).toBe('100 单位')
    expect(formatQuota(12.345)).toBe('12.35 单位')
    expect(formatQuota(0.0025)).toBe('0.0025 单位')
    expect(formatQuota(null)).toBe('0 单位')
  })

  it('formats countdowns into HH:MM:SS', () => {
    expect(formatCountdown(0)).toBe('00:00:00')
    expect(formatCountdown(3661)).toBe('01:01:01')
    expect(formatCountdown(-10)).toBe('00:00:00')
  })

  it('computes bounded usage percentages and handles unlimited caps', () => {
    expect(computePercent(25, 100)).toBe(25)
    expect(computePercent(120, 100)).toBe(100)
    expect(computePercent(10, null)).toBe(0)
    expect(computePercent(10, 0)).toBe(0)
  })

  it('masks public API keys safely', () => {
    expect(maskAPIKey('')).toBe('暂未开放')
    const key = ['sk-free', '123456', 'example', '567890'].join('-')
    expect(maskAPIKey(key)).toBe('sk-free-123456••••••••567890')
  })

  it('formats per-million prices and builds client snippets', () => {
    expect(formatPricePerMillion(2, 'CNY')).toBe('¥2')
    expect(formatPricePerMillion(8, 'CNY')).toBe('¥8')
    expect(formatPricePerMillion(0.04, 'CNY')).toBe('¥0.04')
    expect(formatPricePerMillion(0.0001, 'USD')).toBe('$0.0001')
    expect(formatPricePerMillion(null, 'CNY')).toBe('—')

    const claudeSnippet = buildIntegrationSnippet('claude_code', 'https://free.xjtu.shop/', 'sk-free-abc', 'claude-sonnet-4-6')
    expect(claudeSnippet).toContain('ANTHROPIC_BASE_URL="https://free.xjtu.shop"')
    expect(claudeSnippet).toContain('ANTHROPIC_AUTH_TOKEN="sk-free-abc"')

    const codexSnippet = buildIntegrationSnippet('codex', 'https://free.xjtu.shop', 'sk-free-xyz', 'gpt-5.4')
    expect(codexSnippet).toContain('OPENAI_BASE_URL="https://free.xjtu.shop/v1"')
    expect(codexSnippet).toContain('OPENAI_API_KEY="sk-free-xyz"')
  })
})
