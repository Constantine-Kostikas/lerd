import { test } from 'node:test'
import assert from 'node:assert/strict'
import { existsSync, readdirSync } from 'node:fs'
import { fileURLToPath } from 'node:url'
import { SIDEBAR } from './sidebar.js'

const DOCS_DIR = fileURLToPath(new URL('..', import.meta.url))
const SECTIONS = ['getting-started', 'usage', 'features', 'reference', 'contributing']

// '/usage/sites#x' → 'usage/sites', '/features/' → 'features/index'
function page(link) {
  const path = link.split('#')[0].replace(/^\//, '')
  return path.endsWith('/') ? `${path}index` : path
}

function links(items) {
  return items.flatMap((item) => [...(item.link ? [page(item.link)] : []), ...links(item.items ?? [])])
}

const listed = new Set(Object.values(SIDEBAR).flatMap(links))

test('every docs page is in a sidebar', () => {
  const pages = SECTIONS.flatMap((dir) =>
    readdirSync(`${DOCS_DIR}/${dir}`).filter((f) => f.endsWith('.md')).map((f) => `${dir}/${f.slice(0, -3)}`),
  )
  assert.deepEqual(pages.filter((p) => !listed.has(p)), [])
})

test('every sidebar link points at a page', () => {
  assert.deepEqual([...listed].filter((p) => !existsSync(`${DOCS_DIR}/${p}.md`)), [])
})
