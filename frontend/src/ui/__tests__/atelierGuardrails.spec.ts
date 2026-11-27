import { readdirSync, readFileSync, statSync } from 'node:fs'
import { join, relative, resolve } from 'node:path'
import { describe, expect, it } from 'vitest'

const SRC_ROOT = resolve(__dirname, '../..')

function collectVueFiles(dir: string, acc: string[] = []): string[] {
  for (const entry of readdirSync(dir)) {
    if (entry === '__tests__' || entry === 'node_modules') continue
    const full = join(dir, entry)
    const st = statSync(full)
    if (st.isDirectory()) collectVueFiles(full, acc)
    else if (entry.endsWith('.vue')) acc.push(full)
  }
  return acc
}

function collectSourceFiles(dir: string, acc: string[] = []): string[] {
  for (const entry of readdirSync(dir)) {
    if (entry === '__tests__' || entry === 'node_modules') continue
    const full = join(dir, entry)
    const st = statSync(full)
    if (st.isDirectory()) collectSourceFiles(full, acc)
    else if (entry.endsWith('.vue') || entry.endsWith('.ts')) acc.push(full)
  }
  return acc
}

describe('Atelier UI rewrite guardrails', () => {
  it('contains zero native <select> elements across all Vue templates', () => {
    const offenders: string[] = []
    for (const file of collectVueFiles(SRC_ROOT)) {
      const content = readFileSync(file, 'utf8')
      if (/<select[\s>]/.test(content)) {
        offenders.push(relative(SRC_ROOT, file))
      }
    }
    expect(offenders).toEqual([])
  })

  it('contains zero native <input type="range"> elements across all Vue templates', () => {
    const offenders: string[] = []
    for (const file of collectVueFiles(SRC_ROOT)) {
      const content = readFileSync(file, 'utf8')
      if (/type=["']range["']/i.test(content)) {
        offenders.push(relative(SRC_ROOT, file))
      }
    }
    expect(offenders).toEqual([])
  })

  it('does not import removed legacy stylesheets anywhere in app source', () => {
    const forbidden = [
      '@/styles/portal.css',
      '@/styles/design-system.css',
      '@/styles/patrick.css',
      '@/styles/admin-portal.css',
      '@/styles/user-shell.css',
      '@/styles/patrick-home.css',
      '@/styles/patrick-auth.css'
    ]
    const offenders: string[] = []
    for (const file of collectSourceFiles(SRC_ROOT)) {
      const content = readFileSync(file, 'utf8')
      for (const target of forbidden) {
        if (content.includes(target)) {
          offenders.push(`${relative(SRC_ROOT, file)} -> ${target}`)
        }
      }
    }
    expect(offenders).toEqual([])
  })
})
