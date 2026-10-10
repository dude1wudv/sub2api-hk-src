import { readFileSync } from 'node:fs'
import { dirname, resolve } from 'node:path'
import { fileURLToPath } from 'node:url'

import { describe, expect, it } from 'vitest'

const dir = dirname(fileURLToPath(import.meta.url))
const sidebarSource = readFileSync(resolve(dir, '../AppSidebar.vue'), 'utf8')
const homeViewSource = readFileSync(resolve(dir, '../../../views/HomeView.vue'), 'utf8')
const keyUsageViewSource = readFileSync(resolve(dir, '../../../views/KeyUsageView.vue'), 'utf8')
const authLayoutSource = readFileSync(resolve(dir, '../AuthLayout.vue'), 'utf8')

describe('Patrick public brand and admin site_logo sanitization', () => {
  it('keeps configured site_logo scoped to the admin sidebar and sanitizes it', () => {
    expect(sidebarSource).toContain("import { sanitizeUrl } from '@/utils/url'")
    expect(sidebarSource).toContain('sanitizeUrl(appStore.siteLogo')
    expect(sidebarSource).toContain('allowRelative: true')
    expect(sidebarSource).toContain('allowDataUrl: true')
  })

  it.each([
    ['HomeView', homeViewSource],
    ['KeyUsageView', keyUsageViewSource],
    ['AuthLayout', authLayoutSource],
  ])('%s uses the fixed lowercase Patrick brand instead of site_logo', (_name, source) => {
    expect(source).toContain('PatrickBrand')
    expect(source.toLowerCase()).toContain('patrickapi')
    expect(source).not.toContain('siteLogo')
    expect(source).not.toContain('site_logo')
  })
})
