import { cp, mkdir, readFile, readdir, rm, stat } from 'node:fs/promises'
import path from 'node:path'
import { fileURLToPath } from 'node:url'
import { spawnSync } from 'node:child_process'

const frontend = path.resolve(path.dirname(fileURLToPath(import.meta.url)), '..')
const chat = path.resolve(frontend, '../chat-demo')
const dev = process.argv.includes('--dev')
const destination = path.resolve(frontend, dev ? 'public/chat-app' : '../backend/internal/web/dist/chat-app')

function run(args) {
  // pnpm's Windows shim requires cmd.exe; arguments here are fixed, never user input.
  const result = spawnSync('pnpm', args, { cwd: chat, stdio: 'inherit', shell: process.platform === 'win32' })
  if (result.error) throw result.error
  if (result.status !== 0) throw new Error(`chat-demo: pnpm ${args.join(' ')} failed (${result.status})`)
}

export async function validateBundle(output) {
  const html = await readFile(path.join(output, 'index.html'), 'utf8')
  if (!html.includes('/chat-app/')) throw new Error('chat-demo bundle must use /chat-app/ publicPath')
  if (!(await stat(path.join(output, 'LICENSE')).catch(() => null))?.isFile()) throw new Error('chat-demo bundle is missing LICENSE')
  if (!(await readdir(path.join(output, 'static/js'))).some(name => name.endsWith('.js'))) throw new Error('chat-demo bundle is missing JavaScript')
  for (const match of html.matchAll(/(?:src|href)=["'](\/chat-app\/[^"']+)["']/g)) {
    const name = decodeURIComponent(match[1].split(/[?#]/)[0].slice('/chat-app/'.length))
    const target = path.resolve(output, name)
    if (!target.startsWith(path.resolve(output) + path.sep) || !(await stat(target).catch(() => null))?.isFile()) {
      throw new Error(`chat-demo bundle is missing referenced resource: ${name}`)
    }
  }
}

async function build() {
  if (Number(process.versions.node.split('.')[0]) !== 24) throw new Error('Chat build requires Node.js 24')
  for (const file of ['package.json', 'pnpm-lock.yaml', 'src/main.tsx', 'LICENSE']) {
    if (!(await stat(path.join(chat, file)).catch(() => null))?.isFile()) {
      throw new Error(`chat-demo is not initialized: missing ${file}; run git submodule update --init -- chat-demo`)
    }
  }
  const version = spawnSync('pnpm', ['--version'], { encoding: 'utf8', shell: process.platform === 'win32' })
  if (version.status !== 0 || !/^9\./.test(version.stdout.trim())) throw new Error('Chat build requires pnpm 9')
  run(['install', '--frozen-lockfile'])
  run(['run', 'build'])
  const output = path.join(chat, 'dist')
  await validateBundle(output)
  // Only replace our generated directory; never remove the main Vite output.
  await mkdir(path.dirname(destination), { recursive: true })
  await rm(destination, { recursive: true, force: true })
  await cp(output, destination, { recursive: true, force: true, errorOnExist: false })
  if (process.argv.includes('--serve')) {
    const { createServer } = await import('vite')
    const guard = {
      name: 'chat-app-resources',
      configureServer(server) {
        server.middlewares.use(async (request, response, next) => {
          const url = new URL(request.url, 'http://localhost')
          if (url.pathname !== '/chat-app' && !url.pathname.startsWith('/chat-app/')) return next()
          if (url.pathname === '/chat-app') {
            response.writeHead(308, { Location: '/chat-app/' })
            return response.end()
          }
          let file
          try { file = decodeURIComponent(url.pathname.slice('/chat-app/'.length)) || 'index.html' } catch { file = '..' }
          const target = path.resolve(destination, file)
          const inside = target.startsWith(destination + path.sep)
          if (!inside || !(await stat(target).catch(() => null))?.isFile()) {
            response.writeHead(404, { 'Content-Type': 'text/plain', 'X-Frame-Options': 'SAMEORIGIN' })
            return response.end('Chat resource not found; rebuild with pnpm dev')
          }
          next()
        })
      },
    }
    const server = await createServer({ root: frontend, plugins: [guard] })
    await server.listen()
    server.printUrls()
  }
}

if (process.argv[1] && path.relative(path.resolve(process.argv[1]), fileURLToPath(import.meta.url)) === '') {
  build().catch(error => {
    console.error(`Chat bundle failed: ${error.message}`)
    process.exitCode = 1
  })
}