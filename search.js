'use strict'
const fs = require('fs')
const { walk, log } = require('./helpers')

function setup(options = {}) {
  const { blessed, program, screen, box } = options

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
  function backToBox() {
    const line = box.getScreenLine(box.childBase + boxLine)
    program.cursorPos(boxLine, line.search(/\S/))
    program.showCursor()
    box.focus()
    screen.render()
  }

  box.key('/', function () {
    log('box.key /')
    boxLine = program.y
    box.height = '100%-1'
    searchInput.show()
    searchInput.readInput()
    screen.render()
  })

  searchInput.on('submit', function () {
    box.height = '100%-2'
    searchResult.show()
    const hit = Math.random()
    if (hit > 0.5) {
      searchResult.content = 'found something'
      box.data.search = hit // this will tell fx.js to do something
      backToBox()
    }
    else {
      searchResult.content = 'nothing found'
      searchInput.readInput() // keep input so we can receive "cancel" event
      screen.render()
    }
  })

  searchInput.on('cancel', function () {
    log('searchInput.on cancel')
    box.height = '100%'
    searchInput.hide()
    searchResult.hide()
    backToBox()
  })

  searchInput.key('C-u', function () {
    searchInput.setValue('')
    screen.render()
  })

  searchInput.hide()
  searchResult.hide()
}

module.exports = { setup }
