import { test } from 'node:test'
import assert from 'node:assert/strict'
import { RECENT, atomFeed, parseReleases, recentChangelog, slugify } from './changelog.js'

const release = (v, d, dash = '-') => `## [${v}] ${dash} ${d}\n\nNotes for ${v}.\n\n---\n`
const SRC = '# Changelog\n\nIntro.\n\n---\n\n' +
  Array.from({ length: RECENT + 2 }, (_, i) => release(`1.${20 - i}.0`, `2026-01-${String(20 - i).padStart(2, '0')}`)).join('\n') +
  release('0.1.0', '2026-01-01', '—')

test('parses every release, including em dash headings, without the trailing rule', () => {
  const { releases } = parseReleases(SRC)
  assert.equal(releases.length, RECENT + 3)
  assert.deepEqual(releases.at(-1), { version: '0.1.0', date: '2026-01-01', heading: '[0.1.0] — 2026-01-01', body: 'Notes for 0.1.0.' })
})

test('refuses a heading it cannot read rather than dropping the release', () => {
  assert.throws(() => parseReleases(SRC + '\n## [9.9.9] someday\n'), /unrecognised heading/)
})

test('keeps the recent releases and lists the rest with archive and digest links', () => {
  const out = recentChangelog(SRC, (v) => v === '1.12.0')
  assert.match(out, /## \[1\.13\.0\]/)
  assert.doesNotMatch(out, /## \[1\.12\.0\]/)
  assert.match(out, /- \*\*1\.12\.0\*\* \(2026-01-12\): \[release notes\]\(\/changelog-archive#_1-12-0-2026-01-12\), \[dev digest\]\(\/digest\/v1\.12\.0\.html\)/)
  assert.match(out, /- \*\*1\.11\.0\*\* \(2026-01-11\): \[release notes\]\([^)]+\)$/m)
})

test('slugs match VitePress heading anchors', () => {
  assert.equal(slugify('[1.38.0-beta.2] - 2026-10-26'), '_1-38-0-beta-2-2026-10-26')
})

test('the feed carries the recent releases, newest first, escaped', () => {
  const xml = atomFeed(parseReleases(SRC).releases, (md) => `<p>${md}</p>`, 'https://lerd.sh')
  assert.equal(xml.match(/<entry>/g).length, RECENT)
  assert.match(xml, /<updated>2026-01-20T00:00:00Z<\/updated>/)
  assert.match(xml, /<link href="https:\/\/lerd\.sh\/changelog-archive#_1-20-0-2026-01-20"\/>/)
  assert.match(xml, /&lt;p&gt;Notes for 1\.20\.0\.&lt;\/p&gt;/)
})
