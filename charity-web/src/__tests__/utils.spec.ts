import { describe, expect, it } from 'vitest'
import {
  buildIntegrationSnippet,
  computePercent,
  formatCountdown,
  formatPricePerMillion,
  formatUSD,
  maskAPIKey
} from '../utils'

describe('charity-web utils', () => {
  it('formats USD values cleanly including micro-costs', () => {
    expect(formatUSD(100)).toBe('100.00')
    expect(formatUSD(12.345)).toBe('12.35')
    expect(formatUSD(0.0025)).toBe('0.0025')
    expect(formatUSD(null)).toBe('0.00')
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
    expect(maskAPIKey('sk-free-1234567890abcdef1234567890')).toBe('sk-free-123456••••••••567890')
  })

  it('formats per-million prices and builds client snippets', () => {
    expect(formatPricePerMillion(2.5)).toBe('$2.50')
    expect(formatPricePerMillion(0.15)).toBe('$0.150')
    expect(formatPricePerMillion(null)).toBe('—')

    const claudeSnippet = buildIntegrationSnippet('claude_code', 'https://free.xjtu.shop/', 'sk-free-abc', 'claude-sonnet-4-6')
    expect(claudeSnippet).toContain('ANTHROPIC_BASE_URL="https://free.xjtu.shop"')
    expect(claudeSnippet).toContain('ANTHROPIC_AUTH_TOKEN="sk-free-abc"')

    const codexSnippet = buildIntegrationSnippet('codex', 'https://free.xjtu.shop', 'sk-free-xyz', 'gpt-5.4')
    expect(codexSnippet).toContain('OPENAI_BASE_URL="https://free.xjtu.shop/v1"')
    expect(codexSnippet).toContain('OPENAI_API_KEY="sk-free-xyz"')
  })
})
