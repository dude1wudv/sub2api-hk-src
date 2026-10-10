import fs from 'node:fs'
import path from 'node:path'
import { fileURLToPath } from 'node:url'
import { execFileSync } from 'node:child_process'

const root = path.resolve(path.dirname(fileURLToPath(import.meta.url)), '../..')
const out = path.join(root, 'implement-stage/admin/FUNCTION_COVERAGE.json')
const base = 'a92691325d9f68cb1ecfdec13a59cad2f9cabb9a'
const tracked = new Set(execFileSync('git', ['ls-tree', '-r', '--name-only', base], { cwd: root, encoding: 'utf8' }).trim().split('\n'))
const read = file => execFileSync('git', ['show', `${base}:${file}`], { cwd: root, encoding: 'utf8', maxBuffer: 16 * 1024 * 1024 })
const router = read('frontend/src/router/index.ts')
const pages = [...router.matchAll(/component:\s*\(\)\s*=>\s*import\('(@\/[^']+\.vue)'\)/g)]
  .map(m => [null, [...router.slice(0, m.index).matchAll(/path:\s*'([^']+)'/g)].at(-1)?.[1], m[1]])
  .filter(m => m[1]?.startsWith('/admin') || m[1]?.startsWith('/custom/'))
const visited = new Set()
const files = {}
function inventory(file) {
  if (visited.has(file)) return
  visited.add(file)
  let source
  try { source = read(file) } catch { return }
  const template = source.split('<script')[0]
  const models = [...template.matchAll(/v-model(?:[\w:.-]*)?="([^"]+)"/g)].map(m => m[1])
  const actions = [...template.matchAll(/@([\w:.-]+)="([^"]+)"/g)].map(m => ({ event: m[1], handler: m[2] }))
  const conditions = [...template.matchAll(/v-(?:if|else-if|show)="([^"]+)"/g)].map(m => m[1])
  const components = [...source.matchAll(/(?:from\s*|import\s*)['"](@\/[^'"]+\.vue|\.[^'"]+\.vue)['"]/g)]
    .map(m => m[1].startsWith('@/') ? `frontend/src/${m[1].slice(2)}` : path.posix.normalize(path.posix.join(path.posix.dirname(file), m[1])))
  const barrels = [...source.matchAll(/from\s*['"](@\/components\/[^'".]+)['"]/g)].map(m => `frontend/src/${m[1].slice(2)}/index.ts`)
  components.push(...barrels.filter(file => tracked.has(file)))
  files[file] = { models, actions, conditions, components }
  components.forEach(inventory)
}
if (!process.argv.includes('--check')) pages.forEach(m => inventory(`frontend/src/${m[2].slice(2)}`))
if (process.argv.includes('--check')) {
  const baseline = JSON.parse(fs.readFileSync(out, 'utf8'))
  const losses = []
  for (const [file, item] of Object.entries(baseline.files)) {
    const source = fs.readFileSync(path.join(root, file), 'utf8').split('<script')[0]
    const values = [...source.matchAll(/v-model(?:[\w:.-]*)?="([^"]+)"/g)].map(m => m[1])
    for (const model of new Set(item.models)) if (values.filter(v => v === model).length < item.models.filter(v => v === model).length) losses.push({ file, model })
  }
  console.log(JSON.stringify({ pages: baseline.pages.length, files: Object.keys(baseline.files).length, missingBindings: losses }, null, 2))
  process.exitCode = losses.length ? 1 : 0
} else {
  fs.writeFileSync(out, JSON.stringify({ base, note: '字段、事件、条件及组件路径基线。逐页沿用这些绑定与业务处理；重排标签不移除业务字段。此清单不是独立正确性证明。', pages: pages.map(m => ({ route: m[1], file: `frontend/src/${m[2].slice(2)}`, destination: '管理端新外壳；原业务页和请求契约保留' })), files }, null, 2) + '\n')
  console.log(`${pages.length} pages; ${Object.keys(files).length} components inventoried`)
}
