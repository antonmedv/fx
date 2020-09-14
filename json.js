'use strict'
const LosslessJSON = require('lossless-json')
LosslessJSON.config({circularRefs: false})
module.exports.parse = LosslessJSON.parse
module.exports.stringify = LosslessJSON.stringify
