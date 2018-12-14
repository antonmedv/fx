'use strict'
const fs = require('fs')
const config = require('./config')

function walk(v, cb, path = '', paths = []) {
  if (!v) {
    return
  }

  paths.push(path)

  if (Array.isArray(v)) {
    cb(path, v, paths)
    let i = 0
    for (let item of v) {
      walk(item, cb, path + '[' + (i++) + ']', paths)
    }
  }
  else if (typeof v === 'object' && v.constructor === Object) {
    cb(path, v, paths)
    for (let [key, value] of Object.entries(v)) {
      walk(value, cb, path + '.' + key, paths)
    }
  }
  else {
    cb(path, v, paths)
  }

  paths.pop()
}

function reduce(json, code) {
  if (/^\./.test(code)) {
    const fx = eval(`function fn() { 
      return ${code === '.' ? 'this' : 'this' + code} 
    }; fn`)
    return fx.call(json)
  }

  if ('?' === code) {
    return Object.keys(json)
  }

  if (/yield\*?\s/.test(code)) {
    const fx = eval(`function fn() {
      const gen = (function*(){ 
        ${code.replace(/\\\n/g, '')} 
      }).call(this)
      return [...gen]
      }; fn`)
    return fx.call(json)
  }

  const fx = eval(`function fn() { 
    return ${code} 
  }; fn`)

  const fn = fx.call(json)
  if (typeof fn === 'function') {
    return fn(json)
  }
  return fn
}

module.exports = { walk, reduce }
