'use strict'
const JSON = require('lossless-json') // override JSON for user's code

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

  if (/^(\.\w*)+\[]/.test(code)) {
    function fold(s) {
      if (s.length === 1) {
        return 'x => x' + s[0]
      }
      let obj = s.shift()
      obj = obj === '.' ? 'x' : 'x' + obj
      return `x => Object.values(${obj}).flatMap(${fold(s)})`
    }
    code = fold(code.split('[]'))
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
