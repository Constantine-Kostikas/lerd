import { test, afterEach } from 'node:test'
import assert from 'node:assert/strict'
import { COUNTS, countList, liveCounts } from './landing-data.js'

const realFetch = globalThis.fetch
afterEach(() => { globalThis.fetch = realFetch })

// Answers each count URL with a list of the given length, or the given failure.
function serve(answers) {
  globalThis.fetch = async (url) => {
    const [key, answer] = Object.entries(answers).find(([k]) => COUNTS[k][0] === url)
    if (answer instanceof Error) throw answer
    if (typeof answer === 'number') return new Response(JSON.stringify({ [COUNTS[key][1]]: Array(answer).fill({}) }))
    return answer
  }
}

test('counts the list under the index key', async () => {
  serve({ plugins: 3 })
  assert.equal(await countList(COUNTS.plugins), 3)
})

test('a failed request counts 0', async () => {
  serve({ plugins: new Error('offline') })
  assert.equal(await countList(COUNTS.plugins), 0)
})

test('an error status counts 0', async () => {
  serve({ plugins: new Response('nope', { status: 503 }) })
  assert.equal(await countList(COUNTS.plugins), 0)
})

test('an index without the list counts 0', async () => {
  serve({ plugins: new Response(JSON.stringify({ plugins: 'many' })) })
  assert.equal(await countList(COUNTS.plugins), 0)
})

test('the built-in stack is added to the services store', async () => {
  serve({ frameworks: 25, services: 34, plugins: 3 })
  assert.deepEqual(await liveCounts(6), { frameworks: 25, services: 40, plugins: 3 })
})

test('the built-in stack alone is never shown as a services count', async () => {
  serve({ frameworks: 25, services: new Error('offline'), plugins: 3 })
  assert.deepEqual(await liveCounts(6), { frameworks: 25, services: 0, plugins: 3 })
})
