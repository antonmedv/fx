#!/usr/bin/env node
'use strict'
const pretty = require('@medv/prettyjson')

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
    console.log(usage)
    process.exit(2)
  }

  const json = JSON.parse(input)
  const result = process.argv.slice(2).reduce(reduce, json)

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

const stdin = process.stdin
let buff = ''

if (stdin.isTTY) {
  main(buff)
}

stdin.setEncoding('utf8')

stdin.on('readable', () => {
  let chunk

  while ((chunk = stdin.read())) {
    buff += chunk
  }
})

stdin.on('end', () => {
  main(buff)
})
