#!/usr/bin/env node
void async function main() {
  const process = await import('node:process')
  const args = process.argv.slice(2)
  if ((args.length === 0 && process.stdin.isTTY)
    || (args.length === 1 && ['-h', '--help'].includes(args[0]))) {
    printUsage()
    return
  }

  JSON.stringify = value => stringify(value)

  for await (const json of parseJson()) {
    // let json
    // if (['-r', '--raw'].includes(args[0])) {
    //   args.shift()
    //   json = input
    // } else try {
    //   json = JSON.parse(input)
    // } catch (err) {
    //   process.stderr.write(`Invalid JSON: ${err.message}\n`)
    //   return process.exitCode = 1
    // }
    let i, code, output = json
    for ([i, code] of args.entries()) try {
      output = await transform(output, code)
    } catch (err) {
      printErr(err)
      return 1
    }

    if (typeof output === 'undefined')
      process.stderr.write('undefined\n')
    else if (typeof output === 'string')
      console.log(output)
    else
      console.log(stringify(output, true))

    function printErr(err) {
      let pre = args.slice(0, i).join(' ')
      let post = args.slice(i + 1).join(' ')
      if (pre.length > 20) pre = '...' + pre.substring(pre.length - 20)
      if (post.length > 20) post = post.substring(0, 20) + '...'
      process.stderr.write(
        `\n  ${pre} ${code} ${post}\n` +
        `  ${' '.repeat(pre.length + 1)}${'^'.repeat(code.length)}\n` +
        `\n${err.stack || err}\n`
      )
    }
  }
}().then(exitCode => process.exitCode = exitCode)

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
}

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

async function* stdin() {
  const process = await import('node:process')
  process.stdin.setEncoding('utf8')
  for await (const chunk of process.stdin)
    for (const ch of chunk)
      yield ch
}

async function* parseJson() {
  const gen = stdin()
  let pos = 0, buffer = '', lastChar, done

  async function next() {
    ({value: lastChar, done: done} = await gen.next())
    pos++
    buffer = buffer.slice(-10) + lastChar
  }

  function printErrorSnippet() {
    const snippet = buffer + lastChar
    let nextChars = ''
    for (let i = 0; i < 10; i++) {
      const {value: ch} = gen.next()
      nextChars += ch || ''
    }
    return `Error snippet: ${snippet}${nextChars}`
  }

  await next()
  while (!done) {
    const value = await parseValue()
    expectValue(value)
    yield value
  }

  async function parseValue() {
    await skipWhitespace()
    const value =
      await parseString() ??
      await parseNumber() ??
      await parseObject() ??
      await parseArray() ??
      await parseKeyword('true', true) ??
      await parseKeyword('false', false) ??
      await parseKeyword('null', null)
    await skipWhitespace()
    return value
  }

  async function parseString() {
    if (lastChar !== '"') return
    let str = ''
    let escaped = false
    while (true) {
      await next()
      if (escaped) {
        if (lastChar === 'u') {
          let unicode = ''
          for (let i = 0; i < 4; i++) {
            await next()
            if (!isHexDigit(lastChar)) {
              throw new SyntaxError(`Invalid Unicode escape sequence '\\u${unicode}${lastChar}' at position ${pos}`)
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
            throw new SyntaxError(`Invalid escape sequence '\\${lastChar}' at position ${pos}`)
          }
          str += escapedChar
        }
        escaped = false
      } else if (lastChar === '\\') {
        escaped = true
      } else if (lastChar === '"') {
        break
      } else if (lastChar === undefined) {
        throw new SyntaxError(`Unterminated string literal at position ${pos}`)
      } else {
        str += lastChar
      }
    }
    await next()
    return str
  }

  async function parseNumber() {
    if (!isDigit(lastChar) && lastChar !== '-') return
    let numStr = ''
    if (lastChar === '-') {
      numStr += lastChar
      await next()
      if (!isDigit(lastChar)) {
        throw new SyntaxError(`Invalid number format at position ${pos}.`)
      }
    }
    if (lastChar === '0') {
      numStr += lastChar
      await next()
    } else {
      while (isDigit(lastChar)) {
        numStr += lastChar
        await next()
      }
    }
    if (lastChar === '.') {
      numStr += lastChar
      await next()
      if (!isDigit(lastChar)) {
        throw new SyntaxError(`Invalid number format at position ${pos}.`)
      }
      while (isDigit(lastChar)) {
        numStr += lastChar
        await next()
      }
    }
    if (lastChar === 'e' || lastChar === 'E') {
      numStr += lastChar
      await next()
      if (lastChar === '+' || lastChar === '-') {
        numStr += lastChar
        await next()
      }
      if (!isDigit(lastChar)) {
        throw new SyntaxError(`Invalid number format at position ${pos}.`)
      }
      while (isDigit(lastChar)) {
        numStr += lastChar
        await next()
      }
    }
    return isInteger(numStr) ? toSafeNumber(numStr) : parseFloat(numStr)
  }

  async function parseObject() {
    if (lastChar !== '{') return
    await next()
    await skipWhitespace()
    const obj = {}
    if (lastChar === '}') {
      await next()
      return obj
    }
    while (true) {
      if (lastChar !== '"') {
        throw new SyntaxError(`Unexpected character '${lastChar}' at position ${pos}. Expected a property name enclosed in double quotes.`)
      }
      const key = await parseString()
      await skipWhitespace()
      if (lastChar !== ':') {
        throw new SyntaxError(`Unexpected character '${lastChar}' at position ${pos}. Expected ':'.`)
      }
      await next()
      const value = await parseValue()
      expectValue(value)
      obj[key] = value
      await skipWhitespace()
      if (lastChar === '}') {
        await next()
        return obj
      } else if (lastChar === ',') {
        await next()
        await skipWhitespace()
      } else {
        throw new SyntaxError(`Unexpected character '${lastChar}' at position ${pos}. Expected ',' or '}'.`)
      }
    }
  }

  async function parseArray() {
    if (lastChar !== '[') return
    await next()
    await skipWhitespace()
    const array = []
    if (lastChar === ']') {
      await next()
      return array
    }
    while (true) {
      const value = await parseValue()
      expectValue(value)
      array.push(value)
      await skipWhitespace()
      if (lastChar === ']') {
        await next()
        return array
      } else if (lastChar === ',') {
        await next()
        await skipWhitespace()
      } else {
        throw new SyntaxError(`Unexpected character '${lastChar}' at position ${pos}. Expected ',' or ']'.`)
      }
    }
  }

  async function parseKeyword(name, value) {
    if (lastChar !== name[0]) return
    for (let i = 1; i < name.length; i++) {
      await next()
      if (lastChar !== name[i]) {
        throw new SyntaxError(`Unexpected character '${lastChar}' at position ${pos}.`)
      }
    }
    await next()
    if (isWhitespace(lastChar) || lastChar === ',' || lastChar === '}' || lastChar === ']' || lastChar === undefined) {
      return value
    }
    throw new SyntaxError(`Unexpected character '${lastChar}' at position ${pos}.`)
  }

  async function skipWhitespace() {
    while (isWhitespace(lastChar)) {
      await next()
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
      throw new SyntaxError(`JSON value expected but got '${value}' at position ${pos}. ${printErrorSnippet()}`)
    }
  }
}

function stringify(value, isPretty = false) {
  const colors = {
    string: isPretty ? '\x1b[32m' : '',
    number: isPretty ? '\x1b[33m' : '',
    boolean: isPretty ? '\x1b[35m' : '',
    null: isPretty ? '\x1b[36m' : '',
    reset: isPretty ? '\x1b[0m' : '',
    key: isPretty ? '\x1b[1m' : '',
    brace: isPretty ? '\x1b[1m' : '',
  }

  const indent = 2

  function getIndent(level) {
    return ' '.repeat(indent * level)
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
    } else if (value === null) {
      return `${colors.null}null${colors.reset}`
    } else if (Array.isArray(value)) {
      if (value.length === 0) {
        return `${colors.brace}[]${colors.reset}`
      }

      const items = value
        .map((v) => `${getIndent(level + 1)}${stringifyValue(v, level + 1)}`)
        .join(',\n')

      return `${colors.brace}[\n${items}\n${getIndent(level)}]${colors.reset}`
    } else if (typeof value === 'object') {
      const keys = Object.keys(value)

      if (keys.length === 0) {
        return `${colors.brace}{${colors.reset}}`
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

      return `${colors.brace}{\n${entries}\n${getIndent(level)}}${colors.reset}`
    }

    throw new Error(`Unsupported value type: ${typeof value}`)
  }

  return stringifyValue(value)
}

function printUsage() {
  const usage = `Usage
  fx [flags] [code...]

Flags
  -h, --help    Display this help message
  -r, --raw     Treat input as raw string`
  console.log(usage)
}
