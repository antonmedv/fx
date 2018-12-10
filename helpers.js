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

function log(...args) {
  if (config.log) {
    fs.appendFileSync(config.log, args.join(' ') + '\n')
  }
}

module.exports = { walk, log }
