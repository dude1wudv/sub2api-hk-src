import { existsSync, readdirSync, readFileSync } from 'node:fs'
import { join } from 'node:path'
import { describe, expect, it } from 'vitest'
const repoRoot = join(process.cwd(), '..')
const manifest = JSON.parse(readFileSync(join(repoRoot, 'hk/upstream-removed.json'), 'utf8')) as {
  removed: Array<{ path: string }>
}

function filesUnder(directory: string): string[] {
  if (!existsSync(directory)) return []
  const found: string[] = []
  for (const entry of readdirSync(directory, { withFileTypes: true })) {
    const fullPath = join(directory, entry.name)
    if (entry.isDirectory()) found.push(...filesUnder(fullPath))
    else found.push(fullPath)
  }
  return found
}

describe('removed upstream paths stay absent', () => {
  it('lists the paths a sync must not restore', () => {
    expect(manifest.removed.length).toBeGreaterThan(10)
  })

  it.each(manifest.removed.map(entry => entry.path))('%s is not in the tree', (path) => {
    if (path.endsWith('/**')) {
      const directory = join(repoRoot, path.slice(0, -3))
      expect(filesUnder(directory), `${path} still has files`).toEqual([])
      return
    }
    expect(existsSync(join(repoRoot, path)), path).toBe(false)
  })
})
