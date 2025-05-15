const __keys = new Set()

Object.prototype.__keys = function () {
  if (Array.isArray(this)) return
  if (typeof this === 'string') return
  if (this instanceof String) return
  if (typeof this === 'object' && this !== null)
    Object.keys(this).forEach(x => __keys.add(x))
}

function __autocomplete() {
  const keys = []
  for (const key of Object.keys(globalThis)) {
    if (key.startsWith('__')) continue
    keys.push(key)
  }
  keys.push(
    'JSON.stringify',
    'JSON.parse',
    'YAML.stringify',
    'YAML.parse',
    'Object.keys',
    'Object.values',
    'Object.entries',
    'Object.fromEntries',
    'Array.isArray',
    'Array.from',
    'console.log',
  )
  return keys
}
