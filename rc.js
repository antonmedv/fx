"use strict";
const path = require('path')

module.exports = function load(rcPath) {
  try {
    require(path.join(rcPath, '.fxrc'));
  } catch (err) {
    if (err.code !== "MODULE_NOT_FOUND") {
      throw err;
    }
  }
};
