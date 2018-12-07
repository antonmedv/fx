'use strict'
const fs = require('fs')
const { walk, log } = require('./helpers')

function setup(options = {}) {
  const { blessed, program, screen, box, source } = options

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
    const hits = find(source, searchInput.content)
    box.height = '100%-2'
    searchResult.show()
    if (hits.length) {
      searchResult.content = 'found something'
      box.data.search = hits // this will tell fx.js to do something
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

function find(source, query) {
  if (/^\s*$/.test(query)) {
    return []
  }

  const m = query.match(/^\/(.*)\/([gimuy]*)$/)
  const regex = new RegExp(m ? m[1] : query, m ? m[2] : '')
  let hits = []

  walk(source, function(path, v) {
    if (typeof v === 'object' && v.constructor === Object) {
      // walk already passes us `path` for:
      //   - scalars
      //   - array elements
      //   - object VALUES
      // ...but not object KEYS, which we have to check ourselves
      for (let key in Object.keys(v)) {
        if (regex.test(key)) {
          hits.push(path + '.' + key)
        }
      }
    }
    else if (typeof v === 'string' && regex.test(v)) {
      hits.push(path)
    }
  })

  return hits
}

module.exports = { setup, find }
