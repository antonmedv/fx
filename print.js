'use strict'
const indent = require('indent-string')
const config = require('./config')

function print(input, options = {}) {
  const {expanded, highlight, currentPath} = options
  const index = new Map()
  let row = 0

  function format(text, style, path) {
    text = JSON.stringify(text)
    if (!highlight) {
      return style(text)
    }
    const highlightStyle = (currentPath === path) ? config.highlightCurrent : config.highlight
    return text
      .replace(highlight, s => '<fx>' + s + '<fx>')
      .split(/<fx>/g)
      .map((s, i) => i % 2 !== 0 ? highlightStyle(s) : style(s))
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
      return format(v, config.null, path)
    }

    if (typeof v === 'number' && Number.isFinite(v)) {
      return format(v, config.number, path)
    }

    if (typeof v === 'boolean') {
      return format(v, config.boolean, path)

    }

    if (typeof v === 'string') {
      return format(v, config.string, path)
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
            const part = format(key, config.key, path + '.' + key) + config.colon(':') + ' ' + doPrint(value, path + '.' + key)
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
