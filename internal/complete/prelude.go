package complete

const prelude = `
const __keys = new Set()

Object.prototype.__keys = function () {
  if (Array.isArray(this)) return
  if (typeof this === 'string') return
  if (this instanceof String) return
  if (typeof this === 'object' && this !== null)
    Object.keys(this).forEach(x => __keys.add(x))
}

function apply(fn, ...args) {
  if (typeof fn === 'function') return fn(...args)
  return fn
}

function len(x) {
  if (Array.isArray(x)) return x.length
  if (typeof x === 'string') return x.length
  if (typeof x === 'object' && x !== null) return Object.keys(x).length
  throw new Error()
}

function uniq(x) {
  if (Array.isArray(x)) return [...new Set(x)]
  throw new Error()
}

function sort(x) {
  if (Array.isArray(x)) return x.sort()
  throw new Error()
}

function map(fn) {
  return function (x) {
    if (Array.isArray(x)) return x.map((v, i) => fn(v, i))
    throw new Error()
  }
}

function sortBy(fn) {
  return function (x) {
    if (Array.isArray(x)) return x.sort((a, b) => {
      const fa = fn(a)
      const fb = fn(b)
      return fa < fb ? -1 : fa > fb ? 1 : 0
    })
    throw new Error()
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
  throw new Error()
}

function reverse(x) {
  if (Array.isArray(x)) return x.reverse()
  throw new Error()
}

function keys(x) {
  if (typeof x === 'object' && x !== null) return Object.keys(x)
  throw new Error()
}

function values(x) {
  if (typeof x === 'object' && x !== null) return Object.values(x)
  throw new Error()
}
`
