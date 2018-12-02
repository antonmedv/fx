'use strict'
const fs = require('fs')
const tty = require('tty')
const blessed = require('@medv/blessed')
const stringWidth = require('string-width')
const reduce = require('./reduce')
const print = require('./print')
const config = require('./config')

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

  const autocomplete = blessed.list({
    parent: screen,
    width: 6,
    height: 7,
    left: 1,
    bottom: 1,
    style: config.list,
  })

  screen.title = filename
  box.focus()
  input.hide()
  autocomplete.hide()

  screen.key(['escape', 'q', 'C-c'], function () {
    program.disableMouse()                // If exit program immediately, stdin may still receive
    setTimeout(() => process.exit(0), 10) // mouse events which will be printed in stdout.
  })

  screen.on('resize', function () {
    render()
  })

  input.on('submit', function () {
    if (autocomplete.hidden) {
      apply()
    } else {
      // Autocomplete selected
      let code = input.getValue()
      let replace = autocomplete.getSelected()
      if (/^[a-z]\w+$/.test(replace)) {
        replace = '.' + replace
      } else {
        replace = `["${replace}"]`
      }
      code = code.replace(/\.\w*$/, replace)

      input.setValue(code)
      autocomplete.hide()
      update(code)

      // Keep editing code
      input.readInput()
    }
  })

  input.on('cancel', function () {
    if (autocomplete.hidden) {
      apply()
    } else {
      // Autocomplete not selected
      autocomplete.hide()
      screen.render()

      // Keep editing code
      input.readInput()
    }
  })

  input.on('update', function (code) {
    update(code)
    complete(code)
  })

  input.key('up', function () {
    if (!autocomplete.hidden) {
      autocomplete.up()
      screen.render()
    }
  })

  input.key('down', function () {
    if (!autocomplete.hidden) {
      autocomplete.down()
      screen.render()
    }
  })

  input.key('C-u', function () {
    input.setValue('')
    update('')
    render()
  })

  input.key('C-w', function () {
    let code = input.getValue()
    code = code.replace(/[\.\[][^\.\[]*$/, '')
    input.setValue(code)
    update(code)
    render()
  })

  box.key('.', function () {
    box.height = '100%-1'
    input.show()
    if (input.getValue() === '') {
      input.setValue('.')
      complete('.')
    }
    input.readInput()
    render()
  })

  box.key('e', function () {
    expanded.clear()
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
    autocomplete.hide()

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

  function apply() {
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
  }

  function complete(inputCode) {
    const match = inputCode.match(/\.(\w*)$/)
    const code = /^\.\w*$/.test(inputCode) ? '.' : inputCode.replace(/\.\w*$/, '')

    let json
    try {
      json = reduce(source, code)
    } catch (e) {
    }

    if (match) {
      if (typeof json === 'object' && json.constructor === Object) {
        const keys = Object.keys(json).filter(key => key.startsWith(match[1]))

        // Hide if there is nothing to show or
        // don't show if there is complete match.
        if (keys.length === 0 || (keys.length === 1 && keys[0] === match[1])) {
          autocomplete.hide()
          return
        }

        autocomplete.width = Math.max(...keys.map(key => key.length)) + 1
        autocomplete.height = Math.min(7, keys.length)
        autocomplete.left = Math.min(
          screen.width - autocomplete.width,
          code.length === 1 ? 1 : code.length + 1
        )

        let selectFirst = autocomplete.items.length !== keys.length
        autocomplete.setItems(keys)

        if (selectFirst) {
          autocomplete.select(autocomplete.items.length - 1)
        }
        if (autocomplete.hidden) {
          autocomplete.show()
        }
      } else {
        autocomplete.clearItems()
        autocomplete.hide()
      }
    }
  }

  function update(code) {
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
    if (code === '') {
      json = source
    }
    render()
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
