import { describe, it, expect } from 'vitest'
import tailwindConfig from '../../tailwind.config.js'

describe('Tailwind Theme Tokens', () => {
  it('should have cyber color palette', () => {
    const colors = tailwindConfig.theme.extend.colors
    expect(colors.cyber).toBeDefined()
    expect(colors.cyber.black).toBe('#0a0e0f')
    expect(colors.cyber.green[500]).toBe('#00ff9f')
  })

  it('should have neon-glow shadow variants', () => {
    const shadows = tailwindConfig.theme.extend.boxShadow
    expect(shadows['neon-glow']).toContain('0 0')
    expect(shadows['neon-glow-lg']).toContain('0 0')
  })
})
