'use strict'
const fs = require('fs')
const { walk, log } = require('./helpers')

function setup(options = {}) {
  const { blessed, program, screen, box, source } = options

  const searchPrompt = blessed.text({
    parent: screen,
    bottom: 0,
    left: 0,
    height: 1,
    width: 8,
    content: 'search:',
  })

  const searchInput = blessed.textbox({
    parent: screen,
    bottom: 0,
    left: 8,
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
  let hits = []
  let hitIndex = 0
  function backToBox() {
    if (hits.length) {
      // update our result, and tell box to re-render the current hitIndex
      const hit = hits[hitIndex]
      searchResult.content = `${hitIndex + 1} of ${hits.length} found: ${hit.path}`
      box.data.searchHit = hit
    }
    else {
      // put the cursor back
      const line = box.getScreenLine(box.childBase + boxLine)
      program.cursorPos(boxLine, line.search(/\S/))
    }
    program.showCursor()
    screen.render()
    box.focus()
  }

  box.key('/', function () {
    boxLine = program.y
    box.height = '100%-2'
    if (!searchResult.getContent()) {
      searchResult.setContent('Enter a string or a /regex/')
    }
    searchResult.show()
    searchPrompt.show()
    searchInput.show()
    searchInput.readInput()
    screen.render()
  })

  box.key('n', function () {
    if (hitIndex + 1 > hits.length - 1) {
      return
    }
    hitIndex++
    backToBox()
  })

  box.key('p', function () {
    // todo: "N" would be prefereable, but `box.key('N', ...)` does not hook into "N" keypresses
    if (hitIndex - 1 < 0) {
      return
    }
    hitIndex--
    backToBox()
  })

  searchInput.on('submit', function () {
    box.height = '100%-2'
    searchResult.show()
    hits = find(source, searchInput.content)
    if (hits.length) {
      log('searchInput.on submit:', hits.length)
      hitIndex = 0
      backToBox()
    }
    else {
      searchResult.content = 'no hits found'
      searchInput.readInput() // keep input so we can receive "cancel" event
      screen.render()
    }
  })

  searchInput.on('cancel', function () {
    log('searchInput.on cancel')
    box.height = '100%'
    searchPrompt.hide()
    searchInput.hide()
    searchResult.hide()
    backToBox()
  })

  searchInput.key('C-u', function () {
    searchInput.setValue('')
    screen.render()
  })

  searchInput.key('C-c', function () {
    searchInput.emit('cancel')
  })

  searchResult.hide()
  searchPrompt.hide()
  searchInput.hide()
}

function find(source, query) {
  if (/^\s*$/.test(query)) {
    return []
  }

  const m = query.match(/^\/(.*)\/([gimuy]*)$/)
  const regex = new RegExp(m ? m[1] : query, m ? m[2] : '')
  let hits = []

  walk(source, function(path, v, paths) {
    if (typeof v === 'object' && v.constructor === Object) {
      // walk already passes us `path` for:
      //   - scalars
      //   - array elements
      //   - object VALUES
      // ...but not object KEYS, which we have to check ourselves
      for (let [key, value] of Object.entries(v)) {
        if (regex.test(key)) {
          path += '.' + key
          const route = paths.slice()
          route.push(path)
          hits.push({ path, route })
        }
      }
    }
    else if (typeof v === 'string' && regex.test(v)) {
      hits.push({
        path: path,
        route: paths.slice(),
      })
    }
  })

  return hits
}

module.exports = { setup, find }
