'use strict'
const fs = require('fs')
const config = require('./config')

function walk(v, cb, path = '') {
  if (!v) {
    return
  }

  if (Array.isArray(v)) {
    cb(path, v)
    let i = 0
    for (let item of v) {
      walk(item, cb, path + '[' + (i++) + ']')
    }
  }
  else if (typeof v === 'object' && v.constructor === Object) {
    cb(path, v)
    for (let [key, value] of Object.entries(v)) {
      walk(value, cb, path + '.' + key)
    }
  }
  else {
    cb(path, v)
  }
}

function log(...args) {
  if (config.log) {
    fs.appendFileSync(config.log, args.join(' ') + '\n')
  }
}

module.exports = { walk, log }
