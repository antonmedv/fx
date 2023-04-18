async function test(name, fn) {
  try {
    await fn(await import('node:assert/strict'))
    console.log(`✓ ${name}`)
  } catch (err) {
    console.error(`✗ ${name}`)
    throw err
  }
}

async function run(json, code = '') {
  const {spawnSync} = await import('node:child_process')
  return spawnSync(`echo '${typeof json === 'string' ? json : JSON.stringify(json)}' | node index.js ${code}`, {
    stdio: 'pipe',
    encoding: 'utf8',
    shell: true
  })
}

void async function main() {
  await test('properly formatted', async t => {
    const {stdout} = await run([{'greeting': 'hello world'}])
    t.deepEqual(stdout, '[\n  {\n    "greeting": "hello world"\n  }\n]\n')
  })


  await test('parseJson - valid json', async t => {
    const obj = {a: 2.3e100, b: 'str', c: null, d: false, e: [1, 2, 3]}
    const {stdout, stderr} = await run(obj)
    t.equal(stderr, '')
    t.equal(stdout, JSON.stringify(obj, null, 2) + '\n')
  })

  await test('parseJson - invalid json', async t => {
    const {stderr, status} = await run('{invalid}')
    t.equal(status, 1)
    t.ok(stderr.includes('SyntaxError'))
  })

  await test('parseJson - invalid number', async t => {
    const {stderr, status} = await run('{"num": 12.3.4}')
    t.equal(status, 1)
    t.ok(stderr.includes('SyntaxError'))
  })

  await test('parseJson - numbers', async t => {
    t.equal((await run('1.2e300')).stdout, '1.2e+300\n')
    t.equal((await run('123456789012345678901234567890')).stdout, '123456789012345678901234567890\n')
    t.equal((await run('23')).stdout, '23\n')
    t.equal((await run('0')).stdout, '0\n')
    t.equal((await run('0e+2')).stdout, '0\n')
    t.equal((await run('0e+2')).stdout, '0\n')
    t.equal((await run('0.0')).stdout, '0\n')
    t.equal((await run('-0')).stdout, '0\n')
    t.equal((await run('2.3')).stdout, '2.3\n')
    t.equal((await run('2300e3')).stdout, '2300000\n')
    t.equal((await run('2300e+3')).stdout, '2300000\n')
    t.equal((await run('-2')).stdout, '-2\n')
    t.equal((await run('2e-3')).stdout, '0.002\n')
    t.equal((await run('2.3e-3')).stdout, '0.0023\n')
  })

  await test('transform - anonymous function', async t => {
    const {stdout} = await run({'key': 'value'}, '\'function (x) { return x.key }\'')
    t.equal(stdout, 'value\n')
  })

  await test('transform - arrow function', async t => {
    const {stdout} = await run({'key': 'value'}, '\'x => x.key\'')
    t.equal(stdout, 'value\n')
  })

  await test('transform - arrow function with param brackets', async t => {
    const {stdout} = await run({'key': 'value'}, `'(x) => x.key'`)
    t.equal(stdout, 'value\n')
  })

  await test('transform - this is json', async t => {
    const {stdout} = await run([1, 2, 3, 4, 5], `'this.map(x => x * this.length)'`)
    t.deepEqual(JSON.parse(stdout), [5, 10, 15, 20, 25])
  })

  await test('transform - chain works', async t => {
    const {stdout} = await run({'items': ['foo', 'bar']}, `'this.items' '.' 'x => x[1]'`)
    t.equal(stdout, 'bar\n')
  })

  await test('transform - map works', async t => {
    const {stdout} = await run([1, 2, 3], `'map(x * 2)'`)
    t.deepEqual(JSON.parse(stdout), [2, 4, 6])
  })

  await test('transform - map works with dot', async t => {
    const {stdout} = await run([{foo: 'bar'}], `'map(.foo)'`)
    t.deepEqual(JSON.parse(stdout), ['bar'])
  })

  await test('transform - map works with func', async t => {
    const {stdout} = await run([{foo: 'bar'}], `'map(x => x.foo)'`)
    t.deepEqual(JSON.parse(stdout), ['bar'])
  })

  await test('transform - map passes index', async t => {
    const {stdout} = await run([1, 2, 3], `'map((x, i) => x * i)'`)
    t.deepEqual(JSON.parse(stdout), [0, 2, 6])
  })

  await test('transform - flat map works', async t => {
    const {stdout} = await run({master: {foo: [{bar: [{val: 1}]}]}}, '.master.foo[].bar[].val')
    t.deepEqual(JSON.parse(stdout), [1])
  })

  await test('transform - flat map works on the first level', async t => {
    const {stdout} = await run([{val: 1}, {val: 2}], '.[].val')
    t.deepEqual(JSON.parse(stdout), [1, 2])
  })

  await test('transform - sort & uniq', async t => {
    const {stdout} = await run([2, 2, 3, 1], `sort uniq`)
    t.deepEqual(JSON.parse(stdout), [1, 2, 3])
  })

  await test('transform - invalid code argument', async t => {
    const json = {foo: 'bar'}
    const code = '".foo.toUpperCase("'
    const {stderr, status} = await run(json, code)
    t.equal(status, 1)
    t.ok(stderr.includes(`SyntaxError: Unexpected token '}'`))
  })

  await test('stream - objects', async t => {
    const {stdout} = await run('{"foo": "bar"}\n{"foo": "baz"}')
    t.equal(stdout, '{\n  "foo": "bar"\n}\n{\n  "foo": "baz"\n}\n')
  })

  await test('stream - strings', async t => {
    const {stdout} = await run('"foo"\n"bar"')
    t.equal(stdout, 'foo\nbar\n')
  })

  await test('flags - raw flag', async t => {
    const {stdout} = await run(123, `-r 'x => typeof x'`)
    t.equal(stdout, 'string\n')
  })
}()
