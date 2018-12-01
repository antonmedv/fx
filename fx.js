'use strict'
const fs = require('fs')
const tty = require('tty')
const blessed = require('@medv/blessed')
const stringWidth = require('string-width')
const reduce = require('./reduce')
const print = require('./print')

module.exports = function start(filename, source) {
  let json = source
  let index = new Map()
  const expanded = new Set()
  expanded.add('') // Root of JSON

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

  const box = blessed.box({
    parent: screen,
    tags: false,
    left: 0,
    top: 0,
    width: '100%',
    height: '100%',
    mouse: true,
    keys: true,
    vi: true,
    ignoreArrows: true,
    alwaysScroll: true,
    scrollable: true,
  })

  const input = blessed.textbox({
    parent: screen,
    bottom: 0,
    left: 0,
    height: 1,
    width: '100%',
  })

  screen.title = filename
  input.hide()

  screen.key(['escape', 'q', 'C-c'], function (ch, key) {
    program.disableMouse()                // If exit program immediately, stdin may still receive
    setTimeout(() => process.exit(0), 10) // mouse events which will be printed in stdout.
  })

  screen.on('resize', function () {
    render()
  })

  input.on('action', function () {
    const code = input.getValue()

    if (code && code.length !== 0) {
      try {
        json = reduce(source, code)
      } catch (e) {
        // pass
      }
    } else {
      box.height = '100%'
      input.hide()
      json = source
    }
    box.focus()
    program.cursorPos(0, 0)
    render()
  })

  input.on('update', function (code) {
    if (code && code.length !== 0) {
      try {
        const pretender = reduce(source, code)
        if (typeof pretender !== 'undefined' && typeof pretender !== 'function') {
          json = pretender
        }
      } catch (e) {
        // pass
      }
    }
    render()
  })


  box.key('.', function () {
    box.height = '100%-1'
    input.show()
    if (input.getValue() === '') {
      input.setValue('.')
    }
    input.readInput()
    render()
  })

  box.key('e', function () {
    walk(json, path => expanded.size < 1000 && expanded.add(path))
    render()
  })

  box.key('S-e', function () {
    expanded.clear()
    expanded.add('')
    render()
  })

  box.key('up', function () {
    program.showCursor()

    const [n] = getLine(program.y)
    const rest = [...index.keys()].filter(i => i < n)
    if (rest.length > 0) {
      const next = Math.max(...rest)

      let y = box.getScreenNumber(next) - box.childBase
      if (y <= 0) {
        box.scroll(-1)
        screen.render()
        y = 0
      }

      const line = box.getScreenLine(y + box.childBase)
      program.cursorPos(y, line.search(/\S/))
    }
  })

  box.key('down', function () {
    program.showCursor()

    const [n] = getLine(program.y)
    const rest = [...index.keys()].filter(i => i > n)
    if (rest.length > 0) {
      const next = Math.min(...rest)

      let y = box.getScreenNumber(next) - box.childBase
      if (y >= box.height) {
        box.scroll(1)
        screen.render()
        y = box.height - 1
      }

      const line = box.getScreenLine(y + box.childBase)
      program.cursorPos(y, line.search(/\S/))
    }
  })

  box.key('right', function () {
    const [n, line] = getLine(program.y)
    program.showCursor()
    program.cursorPos(program.y, line.search(/\S/))
    const path = index.get(n)
    if (!expanded.has(path)) {
      expanded.add(path)
      render()
    }
  })

  box.key('left', function () {
    const [n, line] = getLine(program.y)
    program.showCursor()
    program.cursorPos(program.y, line.search(/\S/))
    const path = index.get(n)
    if (expanded.has(path)) {
      expanded.delete(path)
      render()
    }
  })

  box.on('click', function (mouse) {
    const [n, line] = getLine(mouse.y)
    if (mouse.x >= stringWidth(line)) {
      return
    }

    program.hideCursor()
    program.cursorPos(mouse.y, line.search(/\S/))

    const path = index.get(n)
    if (expanded.has(path)) {
      expanded.delete(path)
    } else {
      expanded.add(path)
    }
    render()
  })

  function getLine(y) {
    const dy = box.childBase + y
    const n = box.getNumber(dy)
    const line = box.getScreenLine(dy)
    return [n, line]
  }

  function render() {
    let content
    [content, index] = print(json, {expanded})

    if (typeof content === 'undefined') {
      content = 'undefined'
    }

    box.setContent(content)
    screen.render()
  }

  box.focus()
  render()
}

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
