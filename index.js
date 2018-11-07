#!/usr/bin/env node
'use strict'
const pretty = require('@medv/prettyjson')
const fs = require('fs')
const path = require('path')
const os = require('os')

const configFile = '.fxrc'
const readSyncSafe = file => {
  try {
    return fs.readFileSync(file, 'utf8')
  } catch (err) {}
}


const configFileContent =
  readSyncSafe(configFile) ||
  readSyncSafe(path.join(os.homedir(), configFile)) ||
  ''

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
  const { stdout, stderr } = process

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

const evalFn = code => eval(`
  ${configFileContent}
  function fn() {
    return ${code}
  }; fn
`)

function reduce(json, code) {
  if (/^\w+\s*=>/.test(code)) {
    const fx = evalFn(code)()
    return fx(json)
  }

  if (/yield/.test(code)) {
    const fx = evalFn(`[
      ...((function*(){
        ${code.replace(/\\\n/g, '')}
      }).call(this))
    ]`)
    return fx.call(json)
  }

  if (/^\?$/.test(code)) {
    return Object.keys(json)
  }

  if (/^\./.test(code)) {
    const fx = evalFn(code === '.' ? 'this' : 'this' + code)
    return fx.call(json)
  }

  const fx = evalFn(code)
  return fx.call(json)
}

function run() {
  const stdin = process.stdin
  stdin.setEncoding('utf8')

  let buff = ''

  if (stdin.isTTY) {
    main('')
  }

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
