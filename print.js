'use strict'
const indent = require('indent-string')
const config = require('./config')

function print(input, options = {}) {
  const {expanded} = options
  const index = new Map()
  const pathIndex = new Map()
  let row = 0

  function doPrint(v, path = '') {
    index.set(row, path)
    pathIndex.set(path, row)

    const eol = () => {
      row++
      return '\n'
    }

    if (typeof v === 'undefined') {
      return void 0
    }

    if (v === null) {
      return config.null(v)
    }

    if (typeof v === 'number' && Number.isFinite(v)) {
      return config.number(v)
    }

    if (typeof v === 'boolean') {
      return config.boolean(v)

    }

    if (typeof v === 'string') {
      return config.string(JSON.stringify(v))
    }

    if (Array.isArray(v)) {
      let output = config.bracket('[')
      const len = v.length

      if (len > 0) {
        if (expanded && !expanded.has(path)) {
          output += '\u2026'
        } else {
          output += eol()
          let i = 0
          for (let item of v) {
            const value = typeof item === 'undefined' ? null : item // JSON.stringify compatibility
            output += indent(doPrint(value, path + '[' + i + ']'), config.space)
            output += i++ < len - 1 ? config.comma(',') : ''
            output += eol()
          }
        }
      }

      return output + config.bracket(']')
    }

    if (typeof v === 'object' && v.constructor === Object) {
      let output = config.bracket('{')

      const entries = Object.entries(v).filter(([key, value]) => typeof value !== 'undefined') // JSON.stringify compatibility
      const len = entries.length

      if (len > 0) {
        if (expanded && !expanded.has(path)) {
          output += '\u2026'
        } else {
          output += eol()
          let i = 0
          for (let [key, value] of entries) {
            const part = config.key(JSON.stringify(key)) + config.colon(':') + ' ' + doPrint(value, path + '.' + key)
            output += indent(part, config.space)
            output += i++ < len - 1 ? config.comma(',') : ''
            output += eol()
          }
        }
      }

      return output + config.bracket('}')
    }

    return JSON.stringify(v, null, config.space)
  }

  return [doPrint(input), index, pathIndex]
}

module.exports = print
