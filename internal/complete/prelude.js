const __keys = new Set() // own properties
const __methods = new Set() // offered only if no own property matches

function __addOwn(set, x, skip = []) {
  for (const name of Object.getOwnPropertyNames(x)) {
    if (name === 'constructor' || name.startsWith('__') || skip.includes(name)) continue
    set.add(name)
  }
}

Object.prototype.__keys = function () {
  if (this === globalThis) return
  if (Array.isArray(this)) return __addOwn(__methods, Array.prototype)
  if (this instanceof String) return __addOwn(__methods, String.prototype)
  if (this instanceof Number) return __addOwn(__methods, Number.prototype)
  if (this instanceof Boolean) return
  if (typeof this === 'function') return __addOwn(__keys, this, ['length', 'name', 'prototype'])
  if (typeof this === 'object' && this !== null) __addOwn(__keys, this)
}

function __autocomplete() {
  const own = x => {
    const set = new Set()
    __addOwn(set, x)
    return Array.from(set)
  }
  return {
    stdlib: Object.keys(globalThis).filter(name => !name.startsWith('__')),
    string: own(String.prototype),
    number: own(Number.prototype),
    array: own(Array.prototype),
  }
}
