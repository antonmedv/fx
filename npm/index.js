#!/usr/bin/env node
'use strict'

void async function main() {
  const os = await import('node:os')
  const fs = await import('node:fs')
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
  const theme = themes(process.stdout.isTTY ? (process.env.FX_THEME || '1') : '0')
  await importFxrc(process.cwd())
  await importFxrc(os.homedir())
  let fd = 0 // stdin
  if (args.length > 0) {
    if (await isFile(args[0])) {
      fd = fs.openSync(args.shift(), 'r')
    } else if (await isFile(args.at(-1))) {
      fd = fs.openSync(args.pop(), 'r')
    }
  }
  const gen = await read(fd)
  const input = flagRaw ? readLine(gen) : parseJson(gen)
  if (flagSlurp) {
    const array = []
    for (const json of input) {
      array.push(json)
    }
    await runTransforms(array, args, theme)
  } else {
    for (const json of input) {
      await runTransforms(json, args, theme)
    }
  }
}()

const skip = Symbol('skip')

async function runTransforms(json, args, theme) {
  const process = await import('node:process')
  let i, code, jsCode, output = json
  for ([i, code] of args.entries()) try {
    jsCode = transpile(code)
    const fn = `(function () {
      const x = this
      return ${jsCode}
    })`
    output = await run(output, fn)
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
    console.log(stringify(output, theme))

  function printErr(err) {
    let pre = args.slice(0, i).join(' ')
    let post = args.slice(i + 1).join(' ')
    if (pre.length > 20) pre = '...' + pre.substring(pre.length - 20)
    if (post.length > 20) post = post.substring(0, 20) + '...'
    console.error(
      `\n  ${pre} ${code} ${post}\n` +
      `  ${' '.repeat(pre.length + 1)}${'^'.repeat(code.length)}\n` +
      (jsCode !== code ? `\n${jsCode}\n` : ``) +
      `\n${err.stack || err}`,
    )
  }
}

function transpile(code) {
  if ('.' === code)
    return 'x'

  if (/^(\.\w*)+\[]/.test(code))
    return `(${fold(code.split('[]'))})(x)`

  function fold(s) {
    if (s.length === 1)
      return 'x => x' + s[0]
    let obj = s.shift()
    obj = obj === '.' ? 'x' : 'x' + obj
    return `x => ${obj}.flatMap(${fold(s)})`
  }

  if (/^\.\[/.test(code))
    return `x${code.substring(1)}`

  if (/^\./.test(code))
    return `x${code}`

  // deprecated
  if (/^map\(.+?\)$/i.test(code)) {
    let s = code.substring(4, code.length - 1)
    if (s[0] === '.') s = 'x' + s
    return `x.map((x, i) => apply(${s}, x, i))`
  }

  if (/^@/.test(code)) {
    const jsCode = transpile(code.substring(1))
    return `x.map((x, i) => apply(${jsCode}, x, i))`
  }

  return code
}

async function run(json, code) {
  const fn = eval(code).call(json)

  return apply(fn, json)

  function apply(fn, ...args) {
    if (typeof fn === 'function') return fn(...args)
    return fn
  }

  function len(x) {
    if (Array.isArray(x)) return x.length
    if (typeof x === 'string') return x.length
    if (typeof x === 'object' && x !== null) return Object.keys(x).length
    throw new Error(`Cannot get length of ${typeof x}`)
  }

  function uniq(x) {
    if (Array.isArray(x)) return [...new Set(x)]
    throw new Error(`Cannot get unique values of ${typeof x}`)
  }

  function sort(x) {
    if (Array.isArray(x)) return x.sort()
    throw new Error(`Cannot sort ${typeof x}`)
  }

  function map(fn) {
    return function (x) {
      if (Array.isArray(x)) return x.map((v, i) => fn(v, i))
      throw new Error(`Cannot map ${typeof x}`)
    }
  }

  function sortBy(fn) {
    return function (x) {
      if (Array.isArray(x)) return x.sort((a, b) => {
        const fa = fn(a)
        const fb = fn(b)
        return fa < fb ? -1 : fa > fb ? 1 : 0
      })
      throw new Error(`Cannot sort ${typeof x}`)
    }
  }

  function groupBy(keyFn) {
    return function (x) {
      const grouped = {}
      for (const item of x) {
        const key = typeof keyFn === 'function' ? keyFn(item) : item[keyFn]
        if (!grouped.hasOwnProperty(key)) grouped[key] = []
        grouped[key].push(item)
      }
      return grouped
    }
  }

  function chunk(size) {
    return function (x) {
      const res = []
      let i = 0
      while (i < x.length) {
        res.push(x.slice(i, i += size))
      }
      return res
    }
  }

  function zip(...x) {
    const length = Math.min(...x.map(a => a.length))
    const res = []
    for (let i = 0; i < length; i++) {
      res.push(x.map(a => a[i]))
    }
    return res
  }

  function flatten(x) {
    if (Array.isArray(x)) return x.flat()
    throw new Error(`Cannot flatten ${typeof x}`)
  }

  function reverse(x) {
    if (Array.isArray(x)) return x.reverse()
    throw new Error(`Cannot reverse ${typeof x}`)
  }

  function keys(x) {
    if (typeof x === 'object' && x !== null) return Object.keys(x)
    throw new Error(`Cannot get keys of ${typeof x}`)
  }

  function values(x) {
    if (typeof x === 'object' && x !== null) return Object.values(x)
    throw new Error(`Cannot get values of ${typeof x}`)
  }
}

async function read(fd = 0) {
  const fs = await import('node:fs')
  const {Buffer} = await import('node:buffer')
  const {StringDecoder} = await import('node:string_decoder')
  const decoder = new StringDecoder('utf8')
  return function* () {
    while (true) {
      const buffer = Buffer.alloc(4_096)
      let bytesRead
      try {
        bytesRead = fs.readSync(fd, buffer, 0, buffer.length, null)
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

async function isFile(path) {
  const fs = await import('node:fs')
  const stat = fs.statSync(path, {throwIfNoEntry: false})
  return stat !== undefined && stat.isFile()
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

function* parseJson(gen) {
  let lineNumber = 1, buffer = '', lastChar, done = false

  function next() {
    ({value: lastChar, done} = gen.next())
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
            't': '\t',
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
      } else if (lastChar < '\x1F') {
        throw new SyntaxError(errorSnippet(`Unescaped control character ${JSON.stringify(lastChar)}`))
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
    for (const ch of gen) {
      if (!ch || ch === '\n' || line.length >= 60) break
      line += ch
    }
    return line
  }
}

function stringify(value, theme) {
  function color(id, str) {
    if (theme[id] === '') return str
    return `\x1b[${theme[id]}m${str}\x1b[0m`
  }

  function getIndent(level) {
    return ' '.repeat(2 * level)
  }

  function stringifyValue(value, level = 0) {
    if (typeof value === 'string') {
      return color(2, JSON.stringify(value))
    } else if (typeof value === 'number') {
      return color(3, `${value}`)
    } else if (typeof value === 'bigint') {
      return color(3, `${value}`)
    } else if (typeof value === 'boolean') {
      return color(4, `${value}`)
    } else if (value === null || typeof value === 'undefined') {
      return color(5, `null`)
    } else if (Array.isArray(value)) {
      if (value.length === 0) {
        return color(0, `[]`)
      }
      const items = value
        .map((v) => getIndent(level + 1) + stringifyValue(v, level + 1))
        .join(color(0, ',') + '\n')
      return color(0, '[') + '\n' + items + '\n' + getIndent(level) + color(0, ']')
    } else if (typeof value === 'object') {
      const keys = Object.keys(value)
      if (keys.length === 0) {
        return color(0, '{}')
      }
      const entries = keys
        .map((key) =>
          getIndent(level + 1) + color(1, `"${key}"`) + color(0, ': ') +
          stringifyValue(value[key], level + 1),
        )
        .join(color(0, ',') + '\n')
      return color(0, '{') + '\n' + entries + '\n' + getIndent(level) + color(0, '}')
    }
    throw new Error(`Unsupported value type: ${typeof value}`)
  }

  return stringifyValue(value)
}

function themes(id) {
  const themes = {
    '0': ['', '', '', '', '', ''],
    '1': ['', '1;34', '32', '36', '35', '38;5;243'],
    '2': ['', '32', '34', '36', '35', '38;5;243'],
    '3': ['', '95', '93', '96', '31', '38;5;243'],
    '4': ['', '38;5;50', '38;5;39', '38;5;98', '38;5;205', '38;5;243'],
    '5': ['', '38;5;230', '38;5;221', '38;5;209', '38;5;209', '38;5;243'],
    '6': ['', '38;5;69', '38;5;78', '38;5;221', '38;5;203', '38;5;243'],
    '7': ['', '1;38;5;42', '1;38;5;213', '1;38;5;201', '1;38;5;201', '38;5;243'],
    '8': ['', '1;38;5;51', '38;5;195', '38;5;123', '38;5;50', '38;5;243'],
    '🔵': ['1;38;5;33', '38;5;33', '', '', '', ''],
    '🥝': ['38;5;179', '1;38;5;154', '38;5;82', '38;5;226', '38;5;226', '38;5;230'],
  }
  return themes[id] || themes['1']
}

async function importFxrc(path) {
  const {join} = await import('node:path')
  const {pathToFileURL} = await import('node:url')
  try {
    await import(pathToFileURL(join(path, '.fxrc.js')))
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
