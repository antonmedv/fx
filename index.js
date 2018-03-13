#!/usr/bin/env node
'use strict'
const meow = require('meow')
const stdin = require('get-stdin')
const jsome = require('jsome')

const cli = meow(`
  Usage
    $ fx [code ...]

  Examples
    $ echo '{"key": "value"}' | fx 'x => x.key'
    "value"

    $ echo '[1,2,3]' | fx 'this.map(x => x * 2)'
    [2, 4, 6]

    $ echo '{"items": ["one", "two"]}' | fx 'this.items' 'this[1]'
    "two"

    $ echo '{"count": 0}' | fx '{...this, count: 1}'
    {"count": 1}
`)

async function main() {
  const text = await stdin()

  if (text === '') {
    cli.showHelp()
  }

  const json = JSON.parse(text)
  const result = cli.input.reduce(reduce, json)

  if (typeof result === 'undefined') {
    process.stderr.write('undefined\n')
  } else if (process.stdout.isTTY) {
    jsome(result)
  } else {
    console.log(JSON.stringify(result, null, 2))
  }
}

function reduce(json, code) {
  if (/^\w+\s*=>/.test(code)) {
    const fx = eval(code)
    return fx(json)
  } else if (/yield/.test(code)) {
    const fx = eval(`
      function fn() {
        const gen = (function*(){ 
          ${code.replace(/\\\n/g, '')} 
        }).call(this)
        return [...gen]
      }; fn
    `)
    return fx.call(json)
  } else {
    const fx = eval(`function fn() { return ${code} }; fn`)
    return fx.call(json)
  }
}

main()
