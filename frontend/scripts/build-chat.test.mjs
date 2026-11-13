import test from 'node:test'
import assert from 'node:assert/strict'
import { mkdtemp, mkdir, writeFile, rm } from 'node:fs/promises'
import os from 'node:os'
import path from 'node:path'
import { validateBundle } from './build-chat.mjs'

async function fixture(t) {
  const root = await mkdtemp(path.join(os.tmpdir(), 'chat-bundle-'))
  t.after(() => rm(root, { recursive: true, force: true }))
  await mkdir(path.join(root, 'static/js'), { recursive: true })
  await writeFile(path.join(root, 'static/js/main.12345678.js'), 'console.log(1)')
  await writeFile(path.join(root, 'LICENSE'), 'MIT')
  await writeFile(path.join(root, 'index.html'), '<script src="/chat-app/static/js/main.12345678.js"></script>')
  return root
}

test('Chat bundle preserves Rspack paths and license', async t => {
  await validateBundle(await fixture(t))
})

test('Chat bundle rejects missing index, license and referenced assets', async t => {
  for (const name of ['index.html', 'LICENSE', 'static/js/main.12345678.js']) {
    const root = await fixture(t)
    await rm(path.join(root, name))
    await assert.rejects(validateBundle(root))
  }
})

test('Chat bundle rejects incorrect publicPath and traversal references', async t => {
  const root = await fixture(t)
  await writeFile(path.join(root, 'index.html'), '<script src="/static/main.js"></script>')
  await assert.rejects(validateBundle(root), /publicPath/)
  await writeFile(path.join(root, 'index.html'), '<script src="/chat-app/%2e%2e/private.js"></script>')
  await assert.rejects(validateBundle(root), /referenced resource/)
})
