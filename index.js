#!/usr/bin/env node
'use strict'
const os = require('os')
const path = require('path')
const pretty = require('@medv/prettyjson')
const {stdin, stdout, stderr} = process

try {
  require(path.join(os.homedir(), '.fxrc'))
} catch (err) {
  if (err.code !== 'MODULE_NOT_FOUND') {
    throw err
  }
}

// polyfill Object.entries
if (!Object.entries) {
  Object.entries = function(obj) {
    var ownProps = Object.keys(obj);
    var i = ownProps.length;
    var resArray = new Array(i); // preallocate the Array
    while (i--)
      resArray[i] = [ownProps[i], obj[ownProps[i]]];
    return resArray;
  };
}

const usage = `
  Usage
    $ fx [code ...]

  Examples
    $ echo '{"key": "value"}' | fx 'x => x.key'
    value

    $ echo '[1,2,3]' | fx 'this.map(x => x * 2)'
    [2, 4, 6]

    $ echo '{"items": ["one", "two"]}' | fx 'this.items' 'this[1]'
    two

    $ echo '{"count": 0}' | fx '{...this, count: 1}'
    {"count": 1}

    $ echo '{"foo": 1, "bar": 2}' | fx ?
    ["foo", "bar"]

    $ echo '{"key": "value"}' | fx .key
    value

`

function main(input) {
  if (input === '') {
    stderr.write(usage)
    process.exit(2)
  }

  const json = JSON.parse(input)
  const args = process.argv.slice(2)

  if (args.length === 0 && stdout.isTTY) {
    require('./fx')(json)
    return
  }

  const result = args.reduce(reduce, json)

  if (typeof result === 'undefined') {
    stderr.write('undefined\n')
  } else if (typeof result === 'string') {
    console.log(result)
  } else if (stdout.isTTY) {
    console.log(pretty(result))
  } else {
    console.log(JSON.stringify(result, null, 2))
  }
}

function reduce(json, code) {
  if (/^\w+\s*=>/.test(code)) {
    const fx = eval(code)
    return fx(json)
  }

  if (/yield/.test(code)) {
    const fx = eval(`
      function fn() {
        const gen = (function*(){
          ${code.replace(/\\\n/g, '')}
        }).call(this)
        return [...gen]
      }; fn
    `)
    return fx.call(json)
  }

  if (/^\?$/.test(code)) {
    return Object.keys(json)
  }

  if (/^\./.test(code)) {
    const fx = eval(`function fn() { return ${code === '.' ? 'this' : 'this' + code} }; fn`)
    return fx.call(json)
  }

  const fx = eval(`function fn() { return ${code} }; fn`)
  return fx.call(json)
}

function run() {
  stdin.setEncoding('utf8')

  if (stdin.isTTY) {
    main('')
  }

  let buff = ''
  stdin.on('readable', () => {
    let chunk

    while ((chunk = stdin.read())) {
      buff += chunk
    }
  })

  stdin.on('end', () => {
    main(buff)
  })
}

run()
