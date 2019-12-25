'use strict'

function reduce(json, code) {
  if (process.env.FX_APPLY) {
    return global[process.env.FX_APPLY](code)(json)
  }

  if ('.' === code) {
    return json
  }

  if ('?' === code) {
    return Object.keys(json)
  }

  if (/^\.\[/.test(code)) {
    return eval(`function fn() { 
      return this${code.substring(1)} 
    }; fn`).call(json)
  }

  if (/^\./.test(code)) {
    return eval(`function fn() { 
      return this${code} 
    }; fn`).call(json)
  }

  const fn = eval(`function fn() { 
    return ${code} 
  }; fn`).call(json)

  if (typeof fn === 'function') {
    return fn(json)
  }
  return fn
}

module.exports = reduce
