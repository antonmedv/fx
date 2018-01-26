'use strict'
const test = require('ava')
const {execSync} = require('child_process')

function fx(json, code = '') {
  const output = execSync(`echo '${JSON.stringify(json)}' | node index.js ${code}`).toString('utf8')
  return JSON.parse(output)
}

test('pass', t => {
  t.deepEqual(fx([{"greeting": "hello world"}]), [{"greeting": "hello world"}])
})

test('anon func', t => {
  t.deepEqual(fx({"key": "value"}, "'x => x.key'"), 'value')
})

test('this bind', t => {
  t.deepEqual(fx([1, 2, 3, 4, 5], "'this.map(x => x * this.length)'"), [5, 10, 15, 20, 25])
})

test('generator', t => {
  t.deepEqual(fx([1, 2, 3, 4, 5], "'for (let i of this) if (i % 2 == 0) yield i'"), [2, 4])
})

test('chain', t => {
  t.deepEqual(fx({"items": ["foo", "bar"]}, "'this.items' 'yield* this' 'x => x[1]'"), 'bar')
})
