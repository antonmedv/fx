void async function main() {
  const process = await import('node:process')
  let input = ''
  process.stdin.setEncoding('utf8')
  for await (const chunk of process.stdin)
    input += chunk
  const args = process.argv.slice(2)

  let json
  try {
    json = JSON.parse(input)
  } catch (err) {
    try {
      json = eval(`(${input})`)
    } catch (_) {
      process.stderr.write(`Invalid JSON: ${err.message}\n`)
      process.exit(1)
    }
  }

  let i, code, output = json
  for ([i, code] of args.entries()) try {
    output = transform(output, code)
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

function transform(json, code) {
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

  if (/^map\(.+?\)$/.test(code))
    return eval(`(function () {
      return this.map(x => apply(x, ${code.substring(4, code.length - 1)}))
    })`).call(json)

  const fn = eval(`(function () {
    return ${code}
  })`).call(json)

  return apply(json, fn)
}

function apply(json, fn) {
  if (typeof fn === 'function') return fn(json)
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
