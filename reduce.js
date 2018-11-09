'use strict'

function reduce(json, code) {
  if (/^\w+\s*=>/.test(code)) {
    const fx = eval(code)
    return fx(json)
  }

  if (/yield/.test(code)) {
    const fx = eval(`
      function fn() {
        const gen = (function*(){ 
          ${code.replace(/\\\n/g, '')} 
        }).call(this)
        return [...gen]
      }; fn
    `)
    return fx.call(json)
  }

  if ('?' === code) {
    return Object.keys(json)
  }

  if (/^\./.test(code)) {
    const fx = eval(`function fn() { return ${code === '.' ? 'this' : 'this' + code} }; fn`)
    return fx.call(json)
  }

  const fx = eval(`function fn() { return ${code} }; fn`)
  return fx.call(json)
}

module.exports = reduce
