'use strict'
const indent = require('indent-string')
const config = require('./config')

function print(input, expanded = null) {
  const index = new Map()
  let row = 0

  function doPrint(v, path = '') {
    index.set(row, path)

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
      if (expanded && !expanded.has(path)) {
        return config.bracket('[') + '\u2026' + config.bracket(']')
      }

      let output = config.bracket('[') + eol()

      const len = v.length
      let i = 0

      for (let item of v) {
        const value = typeof item === 'undefined' ? null : item // JSON.stringify compatibility
        output += indent(doPrint(value, path + '[' + i + ']'))
        output += i++ < len - 1 ? config.comma(',') : ''
        output += eol()
      }

      return output + config.bracket(']')
    }

    if (typeof v === 'object' && v.constructor === Object) {
      if (expanded && !expanded.has(path)) {
        return config.bracket('{') + '\u2026' + config.bracket('}')
      }

      let output = config.bracket('{') + eol()

      const entries = Object.entries(v)
        .filter(([key, value]) => typeof value !== 'undefined') // JSON.stringify compatibility
      const len = entries.length

      let i = 0
      for (let [key, value] of entries) {
        const part = config.key(JSON.stringify(key)) + config.colon(':') + ' ' + doPrint(value, path + '.' + key)
        output += indent(part, config.space)
        output += i++ < len - 1 ? config.comma(',') : ''
        output += eol()
      }

      return output + config.bracket('}')
    }

    return v.toString()
  }

  return [doPrint(input), index]
}

module.exports = print
