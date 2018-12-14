'use strict'
const indent = require('indent-string')
const config = require('./config')

function print(input, options = {}) {
  const {expanded, highlight} = options
  const index = new Map()
  let row = 0

  function format(text, style) {
    if (!highlight) {
      return style(text)
    }
    return text
      .replace(highlight, s => '<fx>' + s + '<fx>')
      .split(/<fx>/g)
      .map((s, i) => i % 2 !== 0 ? config.highlight(s) : style(s))
      .join('')
  }

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
      return format('null', config.null)
    }

    if (typeof v === 'number' && Number.isFinite(v)) {
      return format(v.toString(), config.number)
    }

    if (typeof v === 'boolean') {
      return format(v.toString(), config.boolean)

    }

    if (typeof v === 'string') {
      return format(JSON.stringify(v), config.string)
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
            const part = format(JSON.stringify(key), config.key) + config.colon(':') + ' ' + doPrint(value, path + '.' + key)
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

  return [doPrint(input), index]
}

module.exports = print
