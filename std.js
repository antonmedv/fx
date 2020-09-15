'use strict'
const JSON = require('lossless-json')
const skip = Symbol('skip')

function select(cb) {
  return json => {
    if (!cb(json)) {
      throw skip
    }
    return json
  }
}

function filter(cb) {
  return json => {
    if (cb(json)) {
      throw skip
    }
    return json
  }
}

function save(json) {
  if (!global.FX_FILENAME) {
    throw "No filename provided.\nTo edit-in-place, specify JSON file as first argument."
  }
  require('fs').writeFileSync(global.FX_FILENAME, JSON.stringify(json, null, 2))
  return json
}

Object.assign(exports, {skip, select, filter, save})
Object.assign(global, exports)
global.std = exports
