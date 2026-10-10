// Dedicated synthetic preview on 3420/3421. Reuses the existing isolated preview
// server without changing its source, process, fixtures or user session.
import { readFileSync, writeFileSync, unlinkSync } from 'node:fs'
import { tmpdir } from 'node:os'
import { join, dirname } from 'node:path'
import { fileURLToPath, pathToFileURL } from 'node:url'
import { randomUUID } from 'node:crypto'
const directory = dirname(fileURLToPath(import.meta.url))
let source = readFileSync(new URL('./user-preview.mjs', import.meta.url), 'utf8')
source = source.replace('const VITE_PORT = 3410', 'const VITE_PORT = 3420').replace('const API_PORT = 3411', 'const API_PORT = 3421')
source = source.replace('const devDirectory = dirname(fileURLToPath(import.meta.url))', `const devDirectory = ${JSON.stringify(directory)}`)
source = source.replace(/from '([^']+)'/g, (_, specifier) => `from '${specifier.startsWith('node:') ? specifier : import.meta.resolve(specifier)}'`)
source = `import { adminPreview } from '${new URL('./admin-preview-fixtures.mjs', import.meta.url).href}'\n` + source
source = source.replace("  const records = filterUsage(userUsage(user), params)", `  if (localPath.startsWith('/admin/')) {
    const result = adminPreview(method, localPath, params, fixtures, publicSettings)
    if (result !== undefined) return ok(response, result)
  }
  const records = filterUsage(userUsage(user), params)`)
const runtime = join(tmpdir(), `patrick-admin-preview-${randomUUID()}.mjs`)
writeFileSync(runtime, source)
process.on('exit', () => { try { unlinkSync(runtime) } catch { /* only our temporary runtime */ } })
await import(pathToFileURL(runtime).href)
