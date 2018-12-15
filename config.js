'use strict'
const chalk = require('chalk')
const noop = x => x
const list = {
  fg: 'black',
  bg: 'cyan',
  selected: {
    bg: 'magenta'
  }
}

module.exports = {
  space:            global.FX_STYLE_SPACE             || 2,
  null:             global.FX_STYLE_NULL              || chalk.grey.bold,
  number:           global.FX_STYLE_NUMBER            || chalk.cyan.bold,
  boolean:          global.FX_STYLE_BOOLEAN           || chalk.yellow.bold,
  string:           global.FX_STYLE_STRING            || chalk.green.bold,
  key:              global.FX_STYLE_KEY               || chalk.blue.bold,
  bracket:          global.FX_STYLE_BRACKET           || noop,
  comma:            global.FX_STYLE_COMMA             || noop,
  colon:            global.FX_STYLE_COLON             || noop,
  list:             global.FX_STYLE_LIST              || list,
  highlight:        global.FX_STYLE_HIGHLIGHT         || chalk.black.bgYellow,
  highlightCurrent: global.FX_STYLE_HIGHLIGHT_CURRENT || chalk.inverse,
  statusBar:        global.FX_STYLE_STATUS_BAR        || chalk.inverse,
}
