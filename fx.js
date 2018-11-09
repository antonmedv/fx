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

  const test = blessed.box({
    parent: screen,
    hidden: true,
    tags: false,
    width: '100%',
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
    // If exit program immediately, stdin may still receive mouse events which will be printed in stdout.
    program.disableMouse()
    setTimeout(() => process.exit(0), 10)
  })

  screen.on('resize', render)

  input.on('action', function (code) {
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
        if (typeof pretender !== 'undefined') {
          json = pretender
        }
      } catch (e) {
        // pass
      }
    }
    render()
  })


  box.key(':', function () {
    box.height = '100%-1'
    input.show()
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

    const pos = box.childBase + program.y
    const rest = [...index.keys()].filter(i => i < pos)
    if (rest.length > 0) {
      const next = Math.max(...rest)
      const y = next - box.childBase
      if (y <= 0) {
        box.scroll(-1)
        screen.render()
      }
      const line = box.getScreenLines()[next]
      const x = line.search(/\S/)
      program.cursorPos(y, x)
    }
  })

  box.key('down', function () {
    program.showCursor()

    const pos = box.childBase + program.y
    const rest = [...index.keys()].filter(i => i > pos)
    if (rest.length > 0) {
      const next = Math.min(...rest)
      const y = next - box.childBase
      if (y >= box.height) {
        box.scroll(1)
        screen.render()
      }
      const line = box.getScreenLines()[next]
      const x = line.search(/\S/)
      program.cursorPos(y, x)
    }
  })

  box.key('right', function () {
    program.showCursor()
    const pos = box.childBase + program.y
    const path = index.get(pos)
    if (!expanded.has(path)) {
      expanded.add(path)
      render()
    }
  })

  box.key('left', function () {
    program.showCursor()
    const pos = box.childBase + program.y
    const path = index.get(pos)
    if (expanded.has(path)) {
      expanded.delete(path)
      render()
    }
  })

  box.on('click', function (mouse) {
    program.hideCursor()

    const pos = box.childBase + mouse.y
    const line = box.getScreenLines()[pos]
    if (mouse.x >= stringWidth(line)) {
      return
    }

    const x = line.search(/\S/)
    program.cursorPos(mouse.y, x)

    const path = index.get(pos)
    if (expanded.has(path)) {
      expanded.delete(path)
    } else {
      expanded.add(path)
    }
    render()
  })

  function render() {
    let content
    [content, index] = print(json, expanded)

    if (typeof content === 'undefined') {
      content = 'undefined'
    }

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
