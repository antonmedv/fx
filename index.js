#!/usr/bin/env node
'use strict'
const os = require('os')
const fs = require('fs')
const path = require('path')
const {stdin, stdout, stderr} = process
try {
  require(path.join(os.homedir(), '.fxrc'))
} catch (err) {
  if (err.code !== 'MODULE_NOT_FOUND') {
    throw err
  }
}
const print = require('./print')
const reduce = require('./reduce')

const usage = `
  Usage
    $ fx [code ...]

  Examples
    $ fx package.json
  
    $ echo '{"key": "value"}' | fx 'x => x.key'
    value
    
    $ echo '{"key": "value"}' | fx .key
    value    

    $ echo '[1,2,3]' | fx 'this.map(x => x * 2)'
    [2, 4, 6]

    $ echo '{"items": ["one", "two"]}' | fx 'this.items' 'this[1]'
    two

    $ echo '{"count": 0}' | fx '{...this, count: 1}'
    {"count": 1}
    
    $ echo '{"foo": 1, "bar": 2}' | fx ?
    ["foo", "bar"]
  
`

function main(input) {
  let args = process.argv.slice(2)
  let filename = 'fx'

  if (input === '') {
    if (args.length === 0) {
      stderr.write(usage)
      process.exit(2)
    }

    input = fs.readFileSync(args[0])
    filename = path.basename(args[0])
    args = args.slice(1)
  }

  const json = JSON.parse(input)

  if (args.length === 0 && stdout.isTTY) {
    require('./fx')(filename, json)
    return
  }

  const output = args.reduce(reduce, json)

  if (typeof output === 'undefined') {
    stderr.write('undefined\n')
  } else if (typeof output === 'string') {
    console.log(output)
  } else {
    const [text] = print(output)
    console.log(text)
  }
}

function run() {
  stdin.setEncoding('utf8')

  if (stdin.isTTY) {
    main('')
    return
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
