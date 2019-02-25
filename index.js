#!/usr/bin/env node
'use strict'
const os = require('os')
const fs = require('fs')
const path = require('path')

try {
  require(path.join(os.homedir(), '.fxrc')) // Should be required before config.js usage.
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

const {stdin, stdout, stderr} = process
const args = process.argv.slice(2)
const skip = Symbol('skip')
global.select = select


void function main() {
  stdin.setEncoding('utf8')

  if (stdin.isTTY) {
    handle('')
    return
  }

  const reader = stream()

  stdin.on('readable', reader.read)
  stdin.on('end', () => {
    if (!reader.isStream()) {
      handle(reader.value())
    } else if(reader.value().length > 0) {
      apply(JSON.parse(reader.value()))
    }
  })
}()


function handle(input) {
  let filename = 'fx'

  if (input === '') {
    if (args.length === 0 || (args.length === 1 && (args[0] === '-h' || args[0] === '--help'))) {
      stderr.write(usage)
      process.exit(2)
    }
    if (args.length === 1 && (args[0] === '-v' || args[0] === '--version')) {
      stderr.write(require('./package.json').version + '\n')
      process.exit(2)
    }
    if (args.length === 1 && args[0] === '--life') {
      require('./bang')
      return
    }

    input = fs.readFileSync(args[0])
    filename = path.basename(args[0])
    args.shift()
  }

  const json = JSON.parse(input)

  if (args.length === 0 && stdout.isTTY) {
    require('./fx')(filename, json)
    return
  }

  apply(json)
}


function apply(json) {
  let output

  try {
    output = args.reduce(reduce, json)
  } catch (e) {
    if (e !== skip) {
      throw e
    } else {
      return
    }
  }

  if (typeof output === 'undefined') {
    stderr.write('undefined\n')
  } else if (typeof output === 'string') {
    console.log(output)
  } else {
    const [text] = print(output)
    console.log(text)
  }
}


function select(cb) {
  return json => {
    if (!cb(json)) {
      throw skip
    }
    return json
  }
}


function stream() {
  let buff = ''

  let count = 0

  return {
    isStream() {
      return count > 0
    },
    value() {
      return buff
    },
    read() {
      let chunk
      while ((chunk = stdin.read())) {
        buff += chunk
        let parts = buff.split(os.EOL);
        if (parts.length > 1) {
          buff = parts.pop();
          for (let part of parts) {
            count++;
            apply(JSON.parse(part));
          }
        }
      }
    }
  }
}
