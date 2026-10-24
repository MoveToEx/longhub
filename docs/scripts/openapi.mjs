import { readFile, writeFile } from 'node:fs/promises'
import { load } from 'js-yaml'

const source = new URL('../spec/openapi.yaml', import.meta.url)
const generated = new URL('../spec/openapi.json', import.meta.url)
const spec = load(await readFile(source, 'utf8'))
const operationIds = new Set()
for (const path of Object.values(spec.paths)) {
  for (const [method, operation] of Object.entries(path)) {
    if (!['get', 'post', 'put', 'patch', 'delete', 'options', 'head', 'trace'].includes(method)) continue
    if (!operation.operationId || operationIds.has(operation.operationId)) {
      throw new Error(`Missing or duplicate operationId: ${operation.operationId}`)
    }
    operationIds.add(operation.operationId)
  }
}

function checkReferences(value) {
  if (!value || typeof value !== 'object') return
  if (typeof value.$ref === 'string' && value.$ref.startsWith('#/')) {
    let target = spec
    for (const part of value.$ref.slice(2).split('/')) {
      target = target?.[part.replaceAll('~1', '/').replaceAll('~0', '~')]
    }
    if (target === undefined) throw new Error(`Unresolved OpenAPI reference: ${value.$ref}`)
  }
  for (const child of Object.values(value)) checkReferences(child)
}
checkReferences(spec)

const output = JSON.stringify(spec, null, 2) + '\n'
if (process.argv.includes('--check')) {
  const current = (await readFile(generated, 'utf8')).replaceAll('\r\n', '\n')
  if (current !== output) throw new Error('OpenAPI JSON is stale. Run yarn docs:spec in docs/.')
  console.log(`OpenAPI JSON and ${operationIds.size} operations verified`)
} else {
  await writeFile(generated, output)
  console.log(`Generated OpenAPI JSON with ${operationIds.size} operations`)
}
