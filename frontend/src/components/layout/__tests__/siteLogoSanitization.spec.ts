import { readFileSync } from 'node:fs'
import { dirname, resolve } from 'node:path'
import { fileURLToPath } from 'node:url'

import { describe, expect, it } from 'vitest'

const dir = dirname(fileURLToPath(import.meta.url))
const adminShellSource = readFileSync(resolve(dir, '../AdminShell.vue'), 'utf8')
const homeViewSource = readFileSync(resolve(dir, '../../../views/HomeView.vue'), 'utf8')
const keyUsageViewSource = readFileSync(resolve(dir, '../../../views/KeyUsageView.vue'), 'utf8')
const authLayoutSource = readFileSync(resolve(dir, '../AuthLayout.vue'), 'utf8')

describe('Patrick public and workspace brand', () => {
  it.each([
    ['AdminShell', adminShellSource],
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
