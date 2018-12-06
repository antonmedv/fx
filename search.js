'use strict'
const config = require('./config')
const fs = require('fs')

function search(options = {}) {
  const { blessed, program, screen, box } = options

  function getLine(y) {
    return box.getScreenLine(box.childBase + y)
  }

  function log(...args) {
    if (config.log) {
      fs.appendFileSync(config.log, args.join(' ') + '\n')
    }
  }

  const searchInput = blessed.textbox({
    parent: screen,
    bottom: 0,
    left: 0,
    height: 1,
    width: '100%',
  })

  const searchResult = blessed.text({
    parent: screen,
    bottom: 1,
    left: 0,
    height: 1,
    width: '100%',
  })

  let boxLine = -1

  box.key('/', function () {
    log('box.key /')
    boxLine = program.y
    box.height = '100%-1'
    searchInput.show()
    searchInput.readInput()
    screen.render()
  })

  searchInput.on('submit', function () {
    log('searchInput.on submit')
    box.height = '100%-2'
    searchResult.show()
    searchResult.content = 'searched!'
    searchInput.readInput() // keep input so we can receive "cancel" event
    screen.render()
  })

  searchInput.on('cancel', function () {
    log('searchInput.on cancel')
    box.height = '100%'
    searchInput.hide()
    searchResult.hide()
    box.focus()

    const line = getLine(boxLine)
    program.cursorPos(boxLine, line.search(/\S/))
    program.showCursor()

    screen.render()
  })

  searchInput.key('C-u', function () {
    searchInput.setValue('')
    screen.render()
  })

  searchInput.hide()
  searchResult.hide()
}

module.exports = search
