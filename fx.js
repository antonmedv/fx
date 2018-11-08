const fs = require('fs')
const tty = require('tty')
const blessed = require('neo-blessed')
const stringWidth = require('string-width')
const indent = require('indent-string')
const chalk = require('chalk')

module.exports = function start(input) {
  const ttyFd = fs.openSync('/dev/tty', 'r+')

  const program = blessed.program({
    input: tty.ReadStream(ttyFd),
    output: tty.WriteStream(ttyFd),
  })

  const screen = blessed.screen({
    program: program,
    smartCSR: true,
    fullUnicode: true,
  })
  screen.title = 'fx'
  screen.key(['escape', 'q', 'C-c'], function (ch, key) {
    return process.exit(0)
  })
  screen.on('resize', render)

  const box = blessed.box({
    parent: screen,
    tags: false,
    left: 0,
    top: 0,
    width: '100%',
    height: '100%',
    keys: true,
    vi: true,
    alwaysScroll: true,
    scrollable: true,
  })

  const test = blessed.box({
    parent: screen,
    hidden: true,
    tags: false,
    width: '100%',
  })

  const scrollSpeed = (() => {
    let prev = new Date()
    return () => {
      const now = new Date()
      const lines = now - prev < 20 ? 3 : 1 // TODO: Speed based on terminal.
      prev = now
      return lines
    }
  })()

  box.on('wheeldown', function () {
    box.scroll(scrollSpeed())
    screen.render()
  })
  box.on('wheelup', function () {
    box.scroll(-scrollSpeed())
    screen.render()
  })

  // TODO: fx input
  // const inputBar = blessed.textbox({
  //   parent: screen,
  //   bottom: 0,
  //   left: 0,
  //   height: 1,
  //   width: '100%',
  //   keys: true,
  //   mouse: true,
  //   inputOnFocus: true,
  // })

  const expanded = new Set()
  expanded.add('') // Root of JSON

  box.key('e', function () {
    walk(input, path => expanded.add(path))
    render()
  })
  box.key('S-e', function () {
    expanded.clear()
    expanded.add('')
    render()
  })

  function walk(v, cb, path = '') {
    if (!v) {
      return
    }

    if (Array.isArray(v)) {
      cb(path)
      let i = 0
      for (let item of v) {
        walk(item, cb, path + '[' + (i++) + ']')
      }
    }

    if (typeof v === 'object' && v.constructor === Object) {
      cb(path)
      let i = 0
      for (let [key, value] of Object.entries(v)) {
        walk(value, cb, path + '.' + key)
      }
    }
  }

  const space = 2

  let index = new Map()
  let row = 0

  function print(input) {
    row = 0
    index = new Map()
    return doPrint(input)
  }

  function doPrint(v, path = '') {
    const eol = () => {
      row++
      return '\n'
    }

    if (typeof v === 'undefined') {
      return void 0
    }

    if (v === null) {
      return chalk.grey.bold(v)
    }

    if (typeof v === 'number' && Number.isFinite(v)) {
      return chalk.cyan.bold(v)

    }

    if (typeof v === 'boolean') {
      return chalk.yellow.bold(v)

    }

    if (typeof v === 'string') {
      return chalk.green.bold(JSON.stringify(v))
    }

    if (Array.isArray(v)) {
      index.set(row, path)

      if (!expanded.has(path)) {
        return '[\u2026]'
      }

      let output = '[' + eol()

      const len = v.length
      let i = 0

      for (let item of v) {
        const value = typeof item === 'undefined' ? null : item // JSON.stringify compatibility
        output += indent(doPrint(value, path + '[' + i + ']'), space)
        output += i++ < len - 1 ? ',' : ''
        output += eol()
      }

      return output + ']'
    }

    if (typeof v === 'object' && v.constructor === Object) {
      index.set(row, path)

      if (!expanded.has(path)) {
        return '{\u2026}'
      }

      let output = '{' + eol()

      const entries = Object.entries(v).filter(noUndefined) // JSON.stringify compatibility
      const len = entries.length

      let i = 0
      for (let [key, value] of entries) {
        const part = chalk.blue.bold(JSON.stringify(key)) + ': ' + doPrint(value, path + '.' + key)
        output += indent(part, space)
        output += i++ < len - 1 ? ',' : ''
        output += eol()
      }

      return output + '}'
    }

    return JSON.stringify(v)
  }

  function noUndefined([key, value]) {
    return typeof value !== 'undefined'
  }

  box.on('click', function (mouse) {
    const pos = box.childBase + mouse.y
    const line = box.getScreenLines()[pos]
    if (mouse.x >= stringWidth(line)) {
      return
    }

    const path = index.get(pos)
    if (expanded.has(path)) {
      expanded.delete(path)
    } else {
      expanded.add(path)
    }
    render()
  })

  function render() {
    const content = print(input)

    // TODO: Move to own fork of blessed.
    let row = 0
    for (let line of content.split('\n')) {
      if (stringWidth(line) > box.width) {
        test.setContent(line)
        const pad = test.getScreenLines().length - 1

        const update = new Map()
        for (let [i, path] of index.entries()) {
          if (i > row) {
            index.delete(i)
            update.set(i + pad, path)
          }
        }

        row += pad

        for (let [i, path] of update.entries()) {
          index.set(i, path)
        }

      }
      row++
    }

    box.setContent(content)
    screen.render()
  }

  box.focus()
  render()
}
