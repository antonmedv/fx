#!/usr/bin/env node
'use strict'
const meow = require('meow')
const stdin = require('get-stdin')
const pretty = require('@medv/prettyjson')

const cli = meow(`
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
  } else if (typeof result === 'string') {
    console.log(result)
  } else if (process.stdout.isTTY) {
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

main()
