// @vitest-environment jsdom
// Drift test: dashboard/docs/frontend.md CONTRACT:routes must equal the routes actually
// registered in src/router/index.js. Add/remove a route and the doc must change
// with it — the pre-commit hook (scripts/git-hooks/pre-commit) blocks otherwise.
import { describe, it, expect } from 'vitest'
import { existsSync, readFileSync } from 'node:fs'
import { resolve } from 'node:path'
import router from '../router/index.js'

// jsdom rewrites import.meta.url to a non-file URL, so resolve from cwd instead.
// Covers vitest run from dashboard/ (the hook's cwd) or from the repo root.
function findDoc() {
  for (const rel of ['docs/frontend.md', 'dashboard/docs/frontend.md']) {
    const abs = resolve(process.cwd(), rel)
    if (existsSync(abs)) return abs
  }
  throw new Error(`frontend.md not found from cwd ${process.cwd()}`)
}
const mdPath = findDoc()

// Extract the "- `item`" bullets inside <!-- CONTRACT:name --> ... <!-- /CONTRACT:name -->.
function parseContract(md, name) {
  const begin = `<!-- CONTRACT:${name}`
  const end = `<!-- /CONTRACT:${name} -->`
  const items = []
  let inBlock = false
  for (const line of md.split('\n')) {
    const t = line.trim()
    if (!inBlock) {
      if (t.startsWith(begin)) inBlock = true
      continue
    }
    if (t === end) return items
    if (t.startsWith('-')) {
      const item = t.replace(/^-/, '').trim().replace(/^`|`$/g, '').trim()
      if (item) items.push(item)
    }
  }
  throw new Error(`CONTRACT:${name} block not found or unterminated in frontend.md`)
}

describe('frontend.md routes contract', () => {
  it('matches the routes registered in the Vue router', () => {
    const documented = parseContract(readFileSync(mdPath, 'utf8'), 'routes')
    const actual = router
      .getRoutes()
      .map((r) => `${r.path} -> ${r.meta?.requiresRole || 'any'}`)

    const docSet = new Set(documented)
    const actSet = new Set(actual)
    const missing = actual.filter((x) => !docSet.has(x)) // in router, absent from doc
    const extra = documented.filter((x) => !actSet.has(x)) // in doc, absent from router

    if (missing.length || extra.length) {
      // Corrected block to paste back into frontend.md's CONTRACT:routes.
      const corrected = [...actual].sort().map((x) => `- \`${x}\``).join('\n')
      console.error(`\nfrontend.md CONTRACT:routes drifted.\nCorrected block:\n${corrected}\n`)
    }
    expect({ missing, extra }).toEqual({ missing: [], extra: [] })
  })
})
