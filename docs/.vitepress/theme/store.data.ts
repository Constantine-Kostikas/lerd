import { defineLoader } from 'vitepress'
import { readdirSync } from 'node:fs'
import { fileURLToPath } from 'node:url'
import { COUNTS, countList } from './landing-data.js'

// The hero's counts at build time; the page refreshes them in the browser.
// A build with no network answers 0 and the page drops that stat rather than show a wrong one.
export interface Data {
  frameworks: number
  services: number
  plugins: number
  // The default stack ships inside the binary rather than the store, so the
  // browser adds this to the store's live count.
  builtin: number
}

declare const data: Data
export { data }

function builtinServices(): number {
  const dir = fileURLToPath(new URL('../../../internal/config/presets/', import.meta.url))
  return readdirSync(dir).filter((f) => f.endsWith('.yaml')).length
}

export default defineLoader({
  async load(): Promise<Data> {
    const [frameworks, store, plugins] = await Promise.all([countList(COUNTS.frameworks), countList(COUNTS.services), countList(COUNTS.plugins)])
    const builtin = builtinServices()
    return { frameworks, services: store ? store + builtin : 0, plugins, builtin }
  },
})
