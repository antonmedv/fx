'use strict'
const fs = require('fs')
const tty = require('tty')
const blessed = require('@medv/blessed')
const stringWidth = require('string-width')
const { walk, reduce } = require('./helpers')
const print = require('./print')
const search = require('./search')
const config = require('./config')

module.exports = function start(filename, source) {
  // Current rendered object on a screen.
  let json = source

  // Contains map from row number to expand path.
  // Example: {0: '', 1: '.foo', 2: '.foo[0]'}
  let index = new Map()

  // Contains expanded paths. Example: ['', '.foo']
  // Empty string represents root path.
  const expanded = new Set()
  expanded.add('')

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

  box.on('focus', function () {
    if (box.data.searchHit) {
      const { hit, highlight } = box.data.searchHit
      box.data.searchHit = null
      expanded.clear()
      expanded.add('')
      hit.route.forEach(h => expanded.add(h))
      render({path:hit.path, highlight})
    }
  })

  screen.title = filename
  box.focus()
  input.hide()
  autocomplete.hide()
  search({blessed, program, screen, box, source})

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
    box.emit('hidesearch')
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

  box.key(['up', 'k'], function () {
    program.showCursor()
    let rest = [...index.keys()]

    const [n] = getLine(program.y)
    if (typeof n !== 'undefined') {
      rest = rest.filter(i => i < n)
    }

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

  box.key(['down', 'j'], function () {
    program.showCursor()
    let rest = [...index.keys()]

    const [n] = getLine(program.y)
    if (typeof n !== 'undefined') {
      rest = rest.filter(i => i > n)
    }

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

  box.key(['right', 'l'], function () {
    const [n, line] = getLine(program.y)
    program.showCursor()
    program.cursorPos(program.y, line.search(/\S/))
    const path = index.get(n)
    if (!expanded.has(path)) {
      expanded.add(path)
      render()
    }
  })

  box.key(['left', 'h'], function () {
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
    if (typeof line === 'undefined') {
      return [n, '']
    }
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
    box.emit('updatesearchsource', json)
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
      // NOTE: do not emit "updatesearchsource", this `json` is LOCAL scope
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
    box.emit('updatesearchsource', json)
    render()
  }

  // `searchInfo` is passed to us via:
  //   - searchInput.on("submit")
  //   - box.on("focus")
  function render(searchInfo = '') {
    const { path, highlight } = searchInfo
    let content
    [content, index] = print(json, {expanded, currentPath:path, highlight})

    if (typeof content === 'undefined') {
      content = 'undefined'
    }

    box.setContent(content)

    if (path) {
      // todo: make this more accurate: because of the difference in line/row
      // indexing between `content` and `box.getScreenLines()`, we cannot
      // exactly address the lines in `index`. So, we have to rely on the
      // imperfect method of relying on the printed-and-indented json lines
      // being unique enough that we can rely on this search method. The worst
      // that happens is the cursor does not land on the right line when
      // cycling through search results, so it's not the worst hack ever.
      const lines = box.getLines()
      const screenLines = box.getScreenLines()
      const match = [...index].find(pair => pair[1] === path)
      let screenLine
      if (match) {
        const showLine = screenLine = match[0]
        while(lines.length !== screenLines.length && screenLine < screenLines.length) {
          if (lines[showLine] === screenLines[screenLine]) {
            break;
          }
          if (lines[showLine].startsWith(screenLines[screenLine])) {
            break;
          }
          screenLine++
        }
      }
      else {
        // some internal error, just go to first line
        screenLine = 1
      }

      box.scrollTo(screenLine - 1)
      const line = screenLines[screenLine]
      program.cursorPos(screenLine - box.childBase, line ? line.search(/\S/) : 0)
    }

    screen.render()
  }

  render()
}
