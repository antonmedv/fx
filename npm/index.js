#!/usr/bin/env node
'use strict'

void async function main() {
  const os = await import('node:os')
  const process = await import('node:process')
  let flagHelp = false
  let flagRaw = false
  let flagSlurp = false
  const args = []
  for (const arg of process.argv.slice(2)) {
    if (arg === '--help' || arg === '-h') flagHelp = true
    else if (arg === '--raw' || arg === '-r') flagRaw = true
    else if (arg === '--slurp' || arg === '-s') flagSlurp = true
    else if (arg === '-rs' || arg === '-sr') flagRaw = flagSlurp = true
    else args.push(arg)
  }
  if (flagHelp || (args.length === 0 && process.stdin.isTTY))
    return printUsage()
  await importFxrc(process.cwd())
  await importFxrc(os.homedir())
  const stdin = await readStdinGenerator()
  const input = flagRaw ? readLine(stdin) : parseJson(stdin)
  if (flagSlurp) {
    const array = []
    for (const json of input) {
      array.push(json)
    }
    await runTransforms(array, args)
  } else {
    for (const json of input) {
      await runTransforms(json, args)
    }
  }
}()

const skip = Symbol('skip')

async function runTransforms(json, args) {
  const process = await import('node:process')
  let i, code, output = json
  for ([i, code] of args.entries()) try {
    output = await transform(output, code)
  } catch (err) {
    printErr(err)
    process.exit(1)
  }

  if (typeof output === 'undefined')
    console.error('undefined')
  else if (typeof output === 'string')
    console.log(output)
  else if (output === skip)
    return
  else
    console.log(stringify(output, process.stdout.isTTY))

  function printErr(err) {
    let pre = args.slice(0, i).join(' ')
    let post = args.slice(i + 1).join(' ')
    if (pre.length > 20) pre = '...' + pre.substring(pre.length - 20)
    if (post.length > 20) post = post.substring(0, 20) + '...'
    console.error(
      `\n  ${pre} ${code} ${post}\n` +
      `  ${' '.repeat(pre.length + 1)}${'^'.repeat(code.length)}\n` +
      `\n${err.stack || err}`
    )
  }
}

async function transform(json, code) {
  if ('.' === code)
    return json

  if ('?' === code)
    return Object.keys(json)

  if (/^(\.\w*)+\[]/.test(code))
    return eval(`(function () {
      return (${fold(code.split('[]'))})(this)
    })`).call(json)

  if (/^\.\[/.test(code))
    return eval(`(function () {
      return this${code.substring(1)}
    })`).call(json)

  if (/^\./.test(code))
    return eval(`(function () {
      return this${code}
    })`).call(json)

  if (/^map\(.+?\)$/.test(code)) {
    let s = code.substring(4, code.length - 1)
    if (s[0] === '.') s = 'x' + s
    return eval(`(function () {
      return this.map((x, i) => apply(${s}, x, i))
    })`).call(json)
  }

  const fn = eval(`(function () {
    return ${code}
  })`).call(json)

  return apply(fn, json)

  function apply(fn, ...args) {
    if (typeof fn === 'function') return fn(...args)
    return fn
  }

  function fold(s) {
    if (s.length === 1)
      return 'x => x' + s[0]
    let obj = s.shift()
    obj = obj === '.' ? 'x' : 'x' + obj
    return `x => Object.values(${obj}).flatMap(${fold(s)})`
  }

  function uniq(array) {
    return [...new Set(array)]
  }

  function sort(array) {
    return array.sort()
  }

  function groupBy(keyOrFunction) {
    return array => {
      const grouped = {}
      for (const item of array) {
        const key = typeof keyOrFunction === 'function'
          ? keyOrFunction(item)
          : item[keyOrFunction]
        if (!grouped.hasOwnProperty(key))
          grouped[key] = []
        grouped[key].push(item)
      }
      return grouped
    }
  }

  function chunk(size) {
    return function (arr) {
      const res = []
      let i = 0
      while (i < arr.length) {
        res.push(arr.slice(i, i += size))
      }
      return res
    }
  }

  function zip(...arrays) {
    const length = Math.min(...arrays.map(a => a.length))
    const res = []
    for (let i = 0; i < length; i++) {
      res.push(arrays.map(a => a[i]))
    }
    return res
  }
}

async function readStdinGenerator() {
  const fs = await import('node:fs')
  const {Buffer} = await import('node:buffer')
  const {StringDecoder} = await import('node:string_decoder')
  const decoder = new StringDecoder('utf8')
  return function* () {
    while (true) {
      const buffer = Buffer.alloc(4_096)
      let bytesRead
      try {
        bytesRead = fs.readSync(0, buffer, 0, buffer.length, null)
      } catch (e) {
        if (e.code === 'EAGAIN' || e.code === 'EWOULDBLOCK') {
          sleepSync(10)
          continue
        }
        if (e.code === 'EOF') break
        throw e
      }
      if (bytesRead === 0) break
      for (const ch of decoder.write(buffer.subarray(0, bytesRead)))
        yield ch
    }
    for (const ch of decoder.end())
      yield ch
  }()
}

function sleepSync(ms) {
  Atomics.wait(new Int32Array(new SharedArrayBuffer(4)), 0, 0, ms)
}

function* readLine(stdin) {
  let buffer = ''
  for (const ch of stdin) {
    if (ch === '\n') {
      yield buffer
      buffer = ''
    } else {
      buffer += ch
    }
  }
  return buffer
}

function* parseJson(stdin) {
  let lineNumber = 1, buffer = '', lastChar, done = false

  function next() {
    ({value: lastChar, done} = stdin.next())
    if (lastChar === '\n') lineNumber++
    buffer += (lastChar || '')
    if (buffer.length > 100) buffer = buffer.slice(-40)
  }

  next()
  while (!done) {
    const value = parseValue()
    expectValue(value)
    yield value
  }

  function parseValue() {
    skipWhitespace()
    const value =
      parseString() ??
      parseNumber() ??
      parseObject() ??
      parseArray() ??
      parseKeyword('true', true) ??
      parseKeyword('false', false) ??
      parseKeyword('null', null)
    skipWhitespace()
    return value
  }

  function parseString() {
    if (lastChar !== '"') return
    let str = ''
    let escaped = false
    while (true) {
      next()
      if (escaped) {
        if (lastChar === 'u') {
          let unicode = ''
          for (let i = 0; i < 4; i++) {
            next()
            if (!isHexDigit(lastChar)) {
              throw new SyntaxError(errorSnippet(`Invalid Unicode escape sequence '\\u${unicode}${lastChar}'`))
            }
            unicode += lastChar
          }
          str += String.fromCharCode(parseInt(unicode, 16))
        } else {
          const escapedChar = {
            '"': '"',
            '\\': '\\',
            '/': '/',
            'b': '\b',
            'f': '\f',
            'n': '\n',
            'r': '\r',
            't': '\t'
          }[lastChar]
          if (!escapedChar) {
            throw new SyntaxError(errorSnippet())
          }
          str += escapedChar
        }
        escaped = false
      } else if (lastChar === '\\') {
        escaped = true
      } else if (lastChar === '"') {
        break
      } else if (lastChar === undefined) {
        throw new SyntaxError(errorSnippet())
      } else {
        str += lastChar
      }
    }
    next()
    return str
  }

  function parseNumber() {
    if (!isDigit(lastChar) && lastChar !== '-') return
    let numStr = ''
    if (lastChar === '-') {
      numStr += lastChar
      next()
      if (!isDigit(lastChar)) {
        throw new SyntaxError(errorSnippet())
      }
    }
    if (lastChar === '0') {
      numStr += lastChar
      next()
    } else {
      while (isDigit(lastChar)) {
        numStr += lastChar
        next()
      }
    }
    if (lastChar === '.') {
      numStr += lastChar
      next()
      if (!isDigit(lastChar)) {
        throw new SyntaxError(errorSnippet())
      }
      while (isDigit(lastChar)) {
        numStr += lastChar
        next()
      }
    }
    if (lastChar === 'e' || lastChar === 'E') {
      numStr += lastChar
      next()
      if (lastChar === '+' || lastChar === '-') {
        numStr += lastChar
        next()
      }
      if (!isDigit(lastChar)) {
        throw new SyntaxError(errorSnippet())
      }
      while (isDigit(lastChar)) {
        numStr += lastChar
        next()
      }
    }
    return isInteger(numStr) ? toSafeNumber(numStr) : parseFloat(numStr)
  }

  function parseObject() {
    if (lastChar !== '{') return
    next()
    skipWhitespace()
    const obj = {}
    if (lastChar === '}') {
      next()
      return obj
    }
    while (true) {
      if (lastChar !== '"') {
        throw new SyntaxError(errorSnippet())
      }
      const key = parseString()
      skipWhitespace()
      if (lastChar !== ':') {
        throw new SyntaxError(errorSnippet())
      }
      next()
      const value = parseValue()
      expectValue(value)
      obj[key] = value
      skipWhitespace()
      if (lastChar === '}') {
        next()
        return obj
      } else if (lastChar === ',') {
        next()
        skipWhitespace()
        if (lastChar === '}') {
          next()
          return obj
        }
      } else {
        throw new SyntaxError(errorSnippet())
      }
    }
  }

  function parseArray() {
    if (lastChar !== '[') return
    next()
    skipWhitespace()
    const array = []
    if (lastChar === ']') {
      next()
      return array
    }
    while (true) {
      const value = parseValue()
      expectValue(value)
      array.push(value)
      skipWhitespace()
      if (lastChar === ']') {
        next()
        return array
      } else if (lastChar === ',') {
        next()
        skipWhitespace()
        if (lastChar === ']') {
          next()
          return array
        }
      } else {
        throw new SyntaxError(errorSnippet())
      }
    }
  }

  function parseKeyword(name, value) {
    if (lastChar !== name[0]) return
    for (let i = 1; i < name.length; i++) {
      next()
      if (lastChar !== name[i]) {
        throw new SyntaxError(errorSnippet())
      }
    }
    next()
    if (isWhitespace(lastChar) || lastChar === ',' || lastChar === '}' || lastChar === ']' || lastChar === undefined) {
      return value
    }
    throw new SyntaxError(errorSnippet())
  }

  function skipWhitespace() {
    while (isWhitespace(lastChar)) {
      next()
    }
    skipComment()
  }

  function skipComment() {
    if (lastChar === '/') {
      next()
      if (lastChar === '/') {
        while (!done && lastChar !== '\n') {
          next()
        }
        skipWhitespace()
      } else if (lastChar === '*') {
        while (!done) {
          next()
          if (lastChar === '*') {
            next()
            if (lastChar === '/') {
              next()
              break
            }
          }
        }
        skipWhitespace()
      } else {
        throw new SyntaxError(errorSnippet())
      }
    }
  }

  function isWhitespace(ch) {
    return ch === ' ' || ch === '\n' || ch === '\t' || ch === '\r'
  }

  function isHexDigit(ch) {
    return (ch >= '0' && ch <= '9') || (ch >= 'a' && ch <= 'f') || (ch >= 'A' && ch <= 'F')
  }

  function isDigit(ch) {
    return ch >= '0' && ch <= '9'
  }

  function isInteger(value) {
    return /^-?[0-9]+$/.test(value)
  }

  function toSafeNumber(str) {
    const maxSafeInteger = Number.MAX_SAFE_INTEGER
    const minSafeInteger = Number.MIN_SAFE_INTEGER
    const num = BigInt(str)
    return num >= minSafeInteger && num <= maxSafeInteger ? Number(num) : num
  }

  function expectValue(value) {
    if (value === undefined) {
      throw new SyntaxError(errorSnippet(`JSON value expected`))
    }
  }

  function errorSnippet(message = `Unexpected character '${lastChar}'`) {
    if (!lastChar) {
      message = 'Unexpected end of input'
    }
    const lines = buffer.slice(-40).split('\n')
    const lastLine = lines.pop()
    const source =
      lines.map(line => `    ${line}\n`).join('')
      + `    ${lastLine}${readEOL()}\n`
    const p = `    ${'.'.repeat(Math.max(0, lastLine.length - 1))}^\n`
    return `${message} on line ${lineNumber}.\n\n${source}${p}`
  }

  function readEOL() {
    let line = ''
    for (const ch of stdin) {
      if (!ch || ch === '\n' || line.length >= 60) break
      line += ch
    }
    return line
  }
}

function stringify(value, isPretty = false) {
  const colors = {
    key: isPretty ? '\x1b[1;34m' : '',
    string: isPretty ? '\x1b[32m' : '',
    number: isPretty ? '\x1b[36m' : '',
    boolean: isPretty ? '\x1b[35m' : '',
    null: isPretty ? '\x1b[2m' : '',
    reset: isPretty ? '\x1b[0m' : '',
  }

  function getIndent(level) {
    return ' '.repeat(2 * level)
  }

  function stringifyValue(value, level = 0) {
    if (typeof value === 'string') {
      return `${colors.string}"${value}"${colors.reset}`
    } else if (typeof value === 'number') {
      return `${colors.number}${value}${colors.reset}`
    } else if (typeof value === 'bigint') {
      return `${colors.number}${value}${colors.reset}`
    } else if (typeof value === 'boolean') {
      return `${colors.boolean}${value}${colors.reset}`
    } else if (value === null || typeof value === 'undefined') {
      return `${colors.null}null${colors.reset}`
    } else if (Array.isArray(value)) {
      if (value.length === 0) {
        return `[]`
      }
      const items = value
        .map((v) => `${getIndent(level + 1)}${stringifyValue(v, level + 1)}`)
        .join(',\n')
      return `[\n${items}\n${getIndent(level)}]`
    } else if (typeof value === 'object') {
      const keys = Object.keys(value)
      if (keys.length === 0) {
        return `{}`
      }
      const entries = keys
        .map(
          (key) =>
            `${getIndent(level + 1)}${colors.key}"${key}"${colors.reset}: ${stringifyValue(
              value[key],
              level + 1
            )}`
        )
        .join(',\n')
      return `{\n${entries}\n${getIndent(level)}}`
    }
    throw new Error(`Unsupported value type: ${typeof value}`)
  }

  return stringifyValue(value)
}

async function importFxrc(path) {
  const {join} = await import('node:path')
  try {
    await import(join(path, '.fxrc.js'))
  } catch (err) {
    if (err.code !== 'ERR_MODULE_NOT_FOUND') throw err
  }
}

function printUsage() {
  const usage = `Usage
  fx [flags] [code...]

Flags
  -h, --help    Display this help message
  -r, --raw     Treat input as a raw string
  -s, --slurp   Read all inputs into an array`
  console.log(usage)
}
