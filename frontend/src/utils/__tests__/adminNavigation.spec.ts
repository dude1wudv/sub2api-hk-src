import { describe, expect, it } from 'vitest'
import { adminNavigation, visibleAdminNavigation } from '../adminNavigation'

describe('admin navigation preservation', () => {
  it('preserves all 24 built-in routes without duplicate entries', () => {
    const paths = adminNavigation.flatMap(group => group.items.map(item => item.path))
    expect(paths).toHaveLength(24)
    expect(new Set(paths).size).toBe(paths.length)
    expect(paths).toContain('/admin/orders')
    expect(paths).toContain('/admin/prompt-audit')
  })
  it('retains the existing simple-mode exclusions', () => {
    const paths = visibleAdminNavigation(true, () => true).flatMap(group => group.items.map(item => item.path))
    expect(paths).toContain('/admin/accounts')
    expect(paths).toContain('/admin/groups')
    expect(paths).toContain('/admin/usage')
    expect(paths).not.toContain('/admin/users')
    expect(paths).not.toContain('/admin/orders/plans')
    expect(paths).not.toContain('/admin/channels/monitor')
  })
  it('applies parent-equivalent flags to every child route', () => {
    const paths = visibleAdminNavigation(false, flag => !['riskControl', 'adminPayment', 'ops'].includes(flag)).flatMap(g => g.items.map(i => i.path))
    expect(paths.some(path => path.startsWith('/admin/orders'))).toBe(false)
    expect(paths).not.toContain('/admin/prompt-audit')
    expect(paths).not.toContain('/admin/risk-control')
    expect(paths).not.toContain('/admin/ops')
    expect(paths).toContain('/admin/usage')
  })
})
