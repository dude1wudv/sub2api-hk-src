import { readFileSync } from 'node:fs'
import { dirname, resolve } from 'node:path'
import { fileURLToPath } from 'node:url'
import { describe, expect, it } from 'vitest'
import { adminNavigation, visibleAdminNavigation } from '@/utils/adminNavigation'
import en from '@/i18n/locales/en'
import zh from '@/i18n/locales/zh'

const here = dirname(fileURLToPath(import.meta.url))
const read = (path: string) => readFileSync(resolve(here, path), 'utf8')

describe('Prompt Audit integration surface', () => {
  it('registers an admin and risk-control guarded route', () => {
    const router = read('../../../router/index.ts')
    expect(router).toContain("path: '/admin/prompt-audit'")
    const route = router.slice(router.indexOf("path: '/admin/prompt-audit'"), router.indexOf("path: '/admin/usage'"))
    expect(route).toContain('requiresAuth: true')
    expect(route).toContain('requiresAdmin: true')
    expect(route).toContain('requiresRiskControl: true')
  })

  it('keeps content moderation and prompt audit in the security group behind their feature flag', () => {
    const security = adminNavigation.find(group => group.id === 'security')!
    expect(security.items).toEqual(expect.arrayContaining([
      expect.objectContaining({ path: '/admin/risk-control', flag: 'riskControl' }),
      expect.objectContaining({ path: '/admin/prompt-audit', flag: 'riskControl' }),
    ]))
    const visible = visibleAdminNavigation(false, flag => flag !== 'riskControl').flatMap(group => group.items)
    expect(visible.some(item => item.path === '/admin/risk-control')).toBe(false)
    expect(visible.some(item => item.path === '/admin/prompt-audit')).toBe(false)
  })

  it('keeps Prompt Audit locale trees symmetric and all operational controls named', () => {
    expect(Object.keys(zh.admin.promptAudit)).toEqual(Object.keys(en.admin.promptAudit))
    expect(zh.nav.securityAudit).toBeTruthy()
    expect(en.nav.securityAudit).toBeTruthy()
    const endpoint = read('../components/EndpointPool.vue')
    const events = read('../components/EventWorkspace.vue')
    expect(endpoint).toContain('aria-label')
    expect(events).toContain('aria-label')
    expect(events).toContain('overflow-x-auto')
    expect(events).toContain('sm:grid-cols-2')
  })
})
