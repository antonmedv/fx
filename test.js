'use strict'
const test = require('ava')
const {execSync} = require('child_process')

function fx(json, code = '') {
  return execSync(`echo '${JSON.stringify(json)}' | node index.js ${code}`).toString('utf8')
}

test('pass', t => {
  const r = fx([{"greeting": "hello world"}])
  t.deepEqual(JSON.parse(r), [{"greeting": "hello world"}])
})

test('anon func', t => {
  const r = fx({"key": "value"}, "'x => x.key'")
  t.deepEqual(r, 'value\n')
})

test('this bind', t => {
  const r = fx([1, 2, 3, 4, 5], "'this.map(x => x * this.length)'")
  t.deepEqual(JSON.parse(r), [5, 10, 15, 20, 25])
})

test('generator', t => {
  const r = fx([1, 2, 3, 4, 5], "'for (let i of this) if (i % 2 == 0) yield i'")
  t.deepEqual(JSON.parse(r), [2, 4])
})

test('chain', t => {
  const r = fx({"items": ["foo", "bar"]}, "'this.items' 'yield* this' 'x => x[1]'")
  t.deepEqual(r, 'bar\n')
})

test('.fxrc', t => {
  const r = fx({"foo": 1, "bar": 3}, "'_.mapValues(this, n => n * 3)'")
  t.deepEqual(JSON.parse(r), { foo: 3, bar: 9 })
})