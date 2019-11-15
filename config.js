'use strict'
const chalk = require('chalk')
const noop = x => x
const list = {
  fg: 'cyan',
  bg: 'blue',
  selected: {
    fg: 'brightcyan',
    bg: 'brightblue'
  }
}

module.exports = {
  space:            global.FX_STYLE_SPACE             || 4,
  null:             global.FX_STYLE_NULL              || chalk.grey.bold,
  number:           global.FX_STYLE_NUMBER            || chalk.magenta,
  boolean:          global.FX_STYLE_BOOLEAN           || chalk.blue.bold,
  string:           global.FX_STYLE_STRING            || chalk.green,
  key:              global.FX_STYLE_KEY               || chalk.cyan,
  bracket:          global.FX_STYLE_BRACKET           || chalk.grey,
  comma:            global.FX_STYLE_COMMA             || chalk.grey,
  colon:            global.FX_STYLE_COLON             || noop,
  list:             global.FX_STYLE_LIST              || list,
  highlight:        global.FX_STYLE_HIGHLIGHT         || chalk.yellow.underline,
  highlightCurrent: global.FX_STYLE_HIGHLIGHT_CURRENT || chalk.yellow.bold.underline,
  statusBar:        global.FX_STYLE_STATUS_BAR        || chalk.inverse,
}
