#!/usr/bin/env node
void async function main() {
  const process = await import('node:process')
  const args = process.argv.slice(2)
  if ((args.length === 0 && process.stdin.isTTY)
    || (args.length === 1 && ['-h', '--help'].includes(args[0]))) {
    printUsage()
    return
  }

  let input = ''
  process.stdin.setEncoding('utf8')
  for await (const chunk of process.stdin)
    input += chunk

  let json
  if (['-r', '--raw'].includes(args[0])) {
    args.shift()
    json = input
  } else try {
    json = JSON.parse(input)
  } catch (err) {
    process.stderr.write(`Invalid JSON: ${err.message}\n`)
    process.exit(1)
  }

  let i, code, output = json
  for ([i, code] of args.entries()) try {
    output = await transform(output, code)
  } catch (err) {
    printErr(err)
    process.exit(1)
  }

  if (typeof output === 'undefined')
    process.stderr.write('undefined\n')
  else if (typeof output === 'string')
    console.log(output)
  else
    console.log(JSON.stringify(output, null, 2))

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
}()

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

function printUsage() {
  const usage = `Usage
  fx [flags] [code...]

Flags
  -h, --help    Display this help message
  -r, --raw     Treat input as raw string`
  console.log(usage)
}
