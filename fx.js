'use strict'
const fs = require('fs')
const tty = require('tty')
const blessed = require('@medv/blessed')
const stringWidth = require('string-width')
const reduce = require('./reduce')
const print = require('./print')
const find = require('./find')
const config = require('./config')

module.exports = function start(filename, source, prev = {}) {
  // Current rendered object on a screen.
  let json = prev.json || source

  // Contains map from row number to expand path.
  // Example: {0: '', 1: '.foo', 2: '.foo[0]'}
  let index = new Map()

  // Contains expanded paths. Example: ['', '.foo']
  // Empty string represents root path.
  const expanded = prev.expanded || new Set()
  expanded.add('')

  // Current filter code.
  let currentCode = null

  // Current search regexp and generator.
  let highlight = null
  let findGen = null
  let currentPath = null

  let ttyReadStream, ttyWriteStream

  // Reopen tty
  if (process.platform === 'win32') {
    const cfs = process.binding('fs')
    ttyReadStream = tty.ReadStream(cfs.open('conin$', fs.constants.O_RDWR | fs.constants.O_EXCL, 0o666))
    ttyWriteStream = tty.WriteStream(cfs.open('conout$', fs.constants.O_RDWR | fs.constants.O_EXCL, 0o666))
  } else {
    const ttyFd = fs.openSync('/dev/tty', 'r+')
    ttyReadStream = tty.ReadStream(ttyFd)
    ttyWriteStream = tty.WriteStream(ttyFd)
  }

  const program = blessed.program({
    input: ttyReadStream,
    output: ttyWriteStream,
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

  const search = blessed.textbox({
    parent: screen,
    bottom: 0,
    left: 0,
    height: 1,
    width: '100%',
  })

  const statusBar = blessed.box({
    parent: screen,
    tags: false,
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
  search.hide()
  statusBar.hide()
  autocomplete.hide()

  process.stdout.on('resize', () => {
    // Blessed has a bug with resizing the terminal. I tried my best to fix it but was not succeeded.
    // For now exit and print seem like a reasonable alternative, as it not usable after resize.
    // If anyone can fix this bug it will be cool.
    printJson({expanded})
  })

  screen.key(['escape', 'q', 'C-c'], function () {
    exit()
  })

  input.on('submit', function () {
    if (autocomplete.hidden) {
      const code = input.getValue()
      if (/^\//.test(code)) {
        // Forgive a mistake to the user. This looks like user wanted to search something.
        apply('')
        applyPattern(code)
      } else {
        apply(code)
      }
    } else {
      // Autocomplete selected
      let code = input.getValue()
      let replace = autocomplete.getSelected()
      if (/^[a-z]\w*$/i.test(replace)) {
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
      const code = input.getValue()
      apply(code)
    } else {
      // Autocomplete not selected
      autocomplete.hide()
      screen.render()

      // Keep editing code
      input.readInput()
    }
  })

  input.on('update', function (code) {
    if (currentCode === code) {
      return
    }
    currentCode = code
    if (index.size < 10000) { // Don't live update in we have a big JSON file.
      update(code)
    }
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

  search.on('submit', function (pattern) {
    applyPattern(pattern)
  })

  search.on('cancel', function () {
    highlight = null
    currentPath = null

    search.hide()
    search.setValue('')

    box.height = '100%'
    box.focus()

    program.cursorPos(0, 0)
    render()
  })

  box.key('.', function () {
    hideStatusBar()
    box.height = '100%-1'
    input.show()
    if (input.getValue() === '') {
      input.setValue('.')
      complete('.')
    }
    input.readInput()
    screen.render()
  })

  box.key('/', function () {
    hideStatusBar()
    box.height = '100%-1'
    search.show()
    search.setValue('/')
    search.readInput()
    screen.render()
  })

  box.key('e', function () {
    hideStatusBar()
    expanded.clear()
    for (let path of dfs(json)) {
      if (expanded.size < 1000) {
        expanded.add(path)
      } else {
        break
      }
    }
    render()
  })

  box.key('S-e', function () {
    hideStatusBar()
    expanded.clear()
    expanded.add('')
    render()

    // Make sure cursor stay on JSON object.
    const [n] = getLine(program.y)
    if (typeof n === 'undefined' || !index.has(n)) {
      // No line under cursor
      let rest = [...index.keys()]
      if (rest.length > 0) {
        const next = Math.max(...rest)
        let y = box.getScreenNumber(next) - box.childBase
        if (y <= 0) {
          y = 0
        }
        const line = box.getScreenLine(y + box.childBase)
        program.cursorPos(y, line.search(/\S/))
      }
    }
  })

  box.key('n', function () {
    hideStatusBar()
    findNext()
  })

  box.key(['up', 'k'], function () {
    hideStatusBar()
    program.showCursor()
    const [n] = getLine(program.y)

    let next
    for (let [i,] of index) {
      if (i < n && (typeof next === 'undefined' || i > next)) {
        next = i
      }
    }

    if (typeof next !== 'undefined') {
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

  box.key(['down', 'j'], function () {
    hideStatusBar()
    program.showCursor()
    const [n] = getLine(program.y)

    let next
    for (let [i,] of index) {
      if (i > n && (typeof next === 'undefined' || i < next)) {
        next = i
      }
    }

    if (typeof next !== 'undefined') {
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

  box.key(['right', 'l'], function () {
    hideStatusBar()
    const [n, line] = getLine(program.y)
    program.showCursor()
    program.cursorPos(program.y, line.search(/\S/))
    const path = index.get(n)
    if (!expanded.has(path)) {
      expanded.add(path)
      render()
    }
  })

  // Expand everything under cursor.
  box.key(['S-right', 'S-l'], function () {
    hideStatusBar()
    const [n, line] = getLine(program.y)
    program.showCursor()
    program.cursorPos(program.y, line.search(/\S/))
    const path = index.get(n)
    const subJson = reduce(json, 'this' + path)
    for (let p of dfs(subJson, path)) {
      if (expanded.size < 1000) {
        expanded.add(p)
      } else {
        break
      }
    }
    render()
  })

  box.key(['left', 'h'], function () {
    hideStatusBar()
    const [n, line] = getLine(program.y)
    program.showCursor()
    program.cursorPos(program.y, line.search(/\S/))

    // Find path at current cursor position.
    const path = index.get(n)

    if (expanded.has(path)) {
      // Collapse current path.
      expanded.delete(path)
      render()
    } else {
      // If there is no expanded paths on current line,
      // collapse parent path of current location.
      if (typeof path === 'string') {
        // Trip last part (".foo", "[0]") to get parent path.
        const parentPath = path.replace(/(\.[^\[\].]+|\[\d+\])$/, '')
        if (expanded.has(parentPath)) {
          expanded.delete(parentPath)
          render()

          // Find line number of parent path, and if we able to find it,
          // move cursor to this position of just collapsed parent path.
          for (let y = program.y; y >= 0; --y) {
            const [n, line] = getLine(y)
            if (index.get(n) === parentPath) {
              program.cursorPos(y, line.search(/\S/))
              break
            }
          }
        }
      }
    }
  })

  box.on('click', function (mouse) {
    hideStatusBar()
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

  box.on('scroll', function () {
    hideStatusBar()
  })

  box.key('p', function () {
    printJson({expanded})
  })

  box.key('S-p', function () {
    printJson()
  })

  function printJson(options = {}) {
    screen.destroy()
    program.disableMouse()
    program.destroy()
    setTimeout(() => {
      const [text] = print(json, options)
      console.log(text)
      process.exit(0)
    }, 10)
  }

  function getLine(y) {
    const dy = box.childBase + y
    const n = box.getNumber(dy)
    const line = box.getScreenLine(dy)
    if (typeof line === 'undefined') {
      return [n, '']
    }
    return [n, line]
  }

  function apply(code) {
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
        const keys = Object.keys(json)
          .filter(key => key.startsWith(match[1]))
          .slice(0, 1000) // With lots of items, list takes forever to render.

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
        if (
          typeof pretender !== 'undefined'
          && typeof pretender !== 'function'
          && !(pretender instanceof RegExp)
        ) {
          json = pretender
        }
      } catch (e) {
        // pass
      }
    }
    if (code === '') {
      json = source
    }

    if (highlight) {
      findGen = find(json, highlight)
    }
    render()
  }

  function applyPattern(pattern) {
    let regex
    let m = pattern.match(/^\/(.*)\/([gimuy]*)$/)
    if (m) {
      try {
        regex = new RegExp(m[1], m[2])
      } catch (e) {
        // Wrong regexp.
      }
    } else {
      m = pattern.match(/^\/(.*)$/)
      if (m) {
        try {
          regex = new RegExp(m[1], 'gi')
        } catch (e) {
          // Wrong regexp.
        }
      }
    }
    highlight = regex

    search.hide()

    if (highlight) {
      findGen = find(json, highlight)
      findNext()
    } else {
      findGen = null
      currentPath = null
    }
    search.setValue('')

    box.height = '100%'
    box.focus()

    program.cursorPos(0, 0)
    render()
  }

  function findNext() {
    if (!findGen) {
      return
    }

    const {value: path, done} = findGen.next()

    if (done) {
      showStatusBar('Pattern not found')
    } else {

      currentPath = ''
      for (let p of path) {
        expanded.add(currentPath += p)
      }
      render()

      for (let [k, v] of index) {
        if (v === currentPath) {
          let y = box.getScreenNumber(k)

          // Scroll one line up for better view and make sure it's not negative.
          if (--y < 0) {
            y = 0
          }

          box.scrollTo(y)
          screen.render()
        }
      }

      // Set cursor to current path.
      // We need timeout here to give our terminal some time.
      // Without timeout first cursorPos call does not working,
      // it looks like an ugly hack and it is an ugly hack.
      setTimeout(() => {
        for (let [k, v] of index) {
          if (v === currentPath) {
            let y = box.getScreenNumber(k) - box.childBase
            if (y <= 0) {
              y = 0
            }
            const line = box.getScreenLine(y + box.childBase)
            program.cursorPos(y, line.search(/\S/))
          }
        }
      }, 100)
    }
  }

  function showStatusBar(status) {
    statusBar.show()
    statusBar.setContent(config.statusBar(` ${status} `))
    screen.render()
  }

  function hideStatusBar() {
    if (!statusBar.hidden) {
      statusBar.hide()
      statusBar.setContent('')
      screen.render()
    }
  }

  function render() {
    let content
    [content, index] = print(json, {expanded, highlight, currentPath})

    if (typeof content === 'undefined') {
      content = 'undefined'
    }

    box.setContent(content)
    screen.render()
  }

  function exit() {
    // If exit program immediately, stdin may still receive
    // mouse events which will be printed in stdout.
    program.disableMouse()
    setTimeout(() => process.exit(0), 10)
  }

  render()
}

function* bfs(json) {
  const queue = [[json, '']]

  while (queue.length > 0) {
    const [v, path] = queue.shift()

    if (!v) {
      continue
    }

    if (Array.isArray(v)) {
      yield path
      let i = 0
      for (let item of v) {
        const p = path + '[' + (i++) + ']'
        queue.push([item, p])
      }
    }

    if (typeof v === 'object' && v.constructor === Object) {
      yield path
      for (let [key, value] of Object.entries(v)) {
        const p = path + '.' + key
        queue.push([value, p])
      }
    }
  }
}

function* dfs(v, path = '') {
  if (!v) {
    return
  }

  if (Array.isArray(v)) {
    yield path
    let i = 0
    for (let item of v) {
      yield* dfs(item, path + '[' + (i++) + ']')
    }
  }

  if (typeof v === 'object' && v.constructor === Object) {
    yield path
    for (let [key, value] of Object.entries(v)) {
      yield* dfs(value, path + '.' + key)
    }
  }
}
