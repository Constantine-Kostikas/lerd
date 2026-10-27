// /changelog renders only the latest releases. The full history stayed one
// 1 MB page that ranked for hundreds of error strings, so older releases move
// to a noindex archive. CHANGELOG.md itself keeps everything.
export const RECENT = 8

const HEADING = /^## \[([^\]]+)\]\s*[-—]\s*(\d{4}-\d{2}-\d{2})\s*$/

export function parseReleases(src) {
  const lines = src.split('\n')
  const starts = lines.flatMap((l, i) => (l.startsWith('## ') ? [i] : []))
  if (!starts.length || !HEADING.test(lines[starts[0]])) throw new Error('changelog: no release heading found')
  const releases = starts.map((start, n) => {
    const m = lines[start].match(HEADING)
    if (!m) throw new Error(`changelog: unrecognised heading "${lines[start]}"`)
    const body = lines.slice(start + 1, starts[n + 1] ?? lines.length).join('\n').replace(/\n-{3,}\s*$/, '').trim()
    return { version: m[1], date: m[2], heading: lines[start].slice(3), body }
  })
  return { header: lines.slice(0, starts[0]).join('\n'), releases }
}

// Same rule as VitePress's heading anchors, so archive links land on the heading.
export function slugify(s) {
  const slug = s
    .normalize('NFKD')
    .replace(/[\u0000-\u001f]/g, '')
    .replace(/[\s~`!@#$%^&*()\-_+=[\]{}|\\;:"'“”‘’<>,.?/]+/g, '-')
    .replace(/-{2,}/g, '-')
    .replace(/^-+|-+$/g, '')
    .replace(/^(\d)/, '_$1')
  return slug.toLowerCase()
}

export function recentChangelog(src, hasDigest) {
  const { header, releases } = parseReleases(src)
  const recent = releases.slice(0, RECENT).map((r) => `## ${r.heading}\n\n${r.body}\n`)
  const older = releases.slice(RECENT).map((r) => {
    const notes = `[release notes](/changelog-archive#${slugify(r.heading)})`
    const digest = hasDigest(r.version) ? `, [dev digest](/digest/v${r.version}.html)` : ''
    return `- **${r.version}** (${r.date}): ${notes}${digest}`
  })
  return [
    header.trimEnd() + '\n',
    'Subscribe to new releases with the [Atom feed](/feed.xml).\n',
    ...recent,
    '## Older releases\n',
    'Every earlier release, with its full notes in the [changelog archive](/changelog-archive).\n',
    ...older,
    '',
  ].join('\n')
}

const esc = (s) => s.replace(/&/g, '&amp;').replace(/</g, '&lt;').replace(/>/g, '&gt;')

export function atomFeed(releases, render, siteUrl) {
  const entries = releases.slice(0, RECENT).map((r) => {
    const link = `${siteUrl}/changelog#${slugify(r.heading)}`
    return `  <entry>
    <title>Lerd ${esc(r.version)}</title>
    <id>${link}</id>
    <link href="${link}"/>
    <updated>${r.date}T00:00:00Z</updated>
    <content type="html">${esc(render(r.body))}</content>
  </entry>`
  })
  return `<?xml version="1.0" encoding="utf-8"?>
<feed xmlns="http://www.w3.org/2005/Atom">
  <title>Lerd releases</title>
  <id>${siteUrl}/feed.xml</id>
  <link rel="self" href="${siteUrl}/feed.xml"/>
  <link href="${siteUrl}/changelog"/>
  <updated>${releases[0].date}T00:00:00Z</updated>
  <author><name>Lerd</name></author>
${entries.join('\n')}
</feed>
`
}
