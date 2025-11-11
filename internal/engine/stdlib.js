'use strict'

const console = {
  log: function (...args) {
    const parts = []
    for (const arg of args) {
      if (typeof arg === 'undefined') {
        parts.push('undefined')
      } else if (typeof arg === 'string') {
        parts.push(arg)
      } else {
        parts.push(JSON.stringify(arg, null, 2))
      }
    }
    println(parts.join(' '))
  },
}

const YAML = {
  stringify: x => __yaml_stringify__(x),
  parse: x => JSON.parse(__yaml_parse__(x)),
}

const skip = Symbol('skip')

function apply(fn, ...args) {
  if (typeof fn === 'function') return fn(...args)
  return fn
}

function len(x) {
  if (Array.isArray(x)) return x.length
  if (typeof x === 'string') return x.length
  if (typeof x === 'object' && x !== null) return Object.keys(x).length
  throw new Error(`Cannot get length of ${typeof x}`)
}

function uniq(x) {
  if (Array.isArray(x)) return [...new Set(x)]
  throw new Error(`Cannot get unique values of ${typeof x}`)
}

function sort(x) {
  if (Array.isArray(x)) return x.sort()
  throw new Error(`Cannot sort ${typeof x}`)
}

function isFalsely(x) {
  return x === false || x === null || x === undefined
}

function filter(fn) {
  return function (x) {
    if (Array.isArray(x)) {
      return x.filter((v, i) => !isFalsely(fn(v, i)))
    }
    return isFalsely(fn(x))? skip : x
  }
}

function map(fn) {
  return function (x) {
    if (Array.isArray(x)) {
      return x.map((v, i) => fn(v, i))
    } else if (x !== null && typeof x === 'object') {
      const result = {}
      for (const [k, v] of Object.entries(x)) {
        result[k] = fn(v, k)
      }
      return result
    } else {
      throw new Error(`Cannot map over ${typeof x}`)
    }
  }
}

function walk(fn) {
  return function recurse(value, key = null) {
    if (Array.isArray(value)) {
      const mapped = value.map((v, i) => recurse(v, i))
      return fn(mapped, key)
    } else if (value !== null && typeof value === 'object') {
      const result = {}
      for (const [k, v] of Object.entries(value)) {
        result[k] = recurse(v, k)
      }
      return fn(result, key)
    } else {
      return fn(value, key)
    }
  }
}

function sortBy(fn) {
  return function (x) {
    if (Array.isArray(x)) return x.sort((a, b) => {
      const fa = fn(a)
      const fb = fn(b)
      return fa < fb ? -1 : fa > fb ? 1 : 0
    })
    throw new Error(`Cannot sort ${typeof x}`)
  }
}

function groupBy(keyFn) {
  return function (x) {
    const grouped = {}
    for (const item of x) {
      const key = typeof keyFn === 'function' ? keyFn(item) : item[keyFn]
      if (!grouped.hasOwnProperty(key)) grouped[key] = []
      grouped[key].push(item)
    }
    return grouped
  }
}

function chunk(size) {
  return function (x) {
    const res = []
    let i = 0
    while (i < x.length) {
      res.push(x.slice(i, i += size))
    }
    return res
  }
}

function zip(...x) {
  const length = Math.min(...x.map(a => a.length))
  const res = []
  for (let i = 0; i < length; i++) {
    res.push(x.map(a => a[i]))
  }
  return res
}

function flatten(x) {
  if (Array.isArray(x)) return x.flat()
  throw new Error(`Cannot flatten ${typeof x}`)
}

function reverse(x) {
  if (Array.isArray(x)) return x.reverse()
  throw new Error(`Cannot reverse ${typeof x}`)
}

function keys(x) {
  if (typeof x === 'object' && x !== null) return Object.keys(x)
  throw new Error(`Cannot get keys of ${typeof x}`)
}

function values(x) {
  if (typeof x === 'object' && x !== null) return Object.values(x)
  throw new Error(`Cannot get values of ${typeof x}`)
}

function list(x) {
  if (Array.isArray(x)) {
    for (const y of x) console.log(y)
    return skip
  }
  throw new Error(`Cannot list ${typeof x}`)
}

function save(x) {
  if (typeof x === 'undefined') throw new Error('Cannot save undefined')
  __save__(__stringify__(x, null, 2))
  return x
}

function toBase64(x) {
  return __toBase64__(x)
}

function fromBase64(x) {
  return __fromBase64__(x)
}
