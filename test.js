'use strict'
const test = require('ava')
const {execSync} = require('child_process')
const stream = require('./stream')

function fx(json, code = '') {
  return execSync(`echo '${JSON.stringify(json)}' | node index.js ${code}`).toString('utf8')
}

test('pass', t => {
  const r = fx([{'greeting': 'hello world'}])
  t.deepEqual(JSON.parse(r), [{'greeting': 'hello world'}])
})

test('anon func', t => {
  const r = fx({'key': 'value'}, '\'function (x) { return x.key }\'')
  t.is(r, 'value\n')
})

test('arrow func', t => {
  const r = fx({'key': 'value'}, '\'x => x.key\'')
  t.is(r, 'value\n')
})

test('arrow func ()', t => {
  const r = fx({'key': 'value'}, '\'(x) => x.key\'')
  t.is(r, 'value\n')
})

test('this bind', t => {
  const r = fx([1, 2, 3, 4, 5], '\'this.map(x => x * this.length)\'')
  t.deepEqual(JSON.parse(r), [5, 10, 15, 20, 25])
})

test('chain', t => {
  const r = fx({'items': ['foo', 'bar']}, '\'this.items\' \'.\' \'x => x[1]\'')
  t.is(r, 'bar\n')
})

test('file argument', t => {
  const r = execSync(`node index.js package.json .name`).toString('utf8')
  t.is(r, 'fx\n')
})

test('stream', t => {
  const input = `
  {"index": 0} {"index": 1}
  {"index": 2, "quote": "\\""}
  {"index": 3} "Hello" "world"
  {"index": 6, "key": "one \\"two\\" three"}
  `
  t.plan(7 * (input.length - 1))

  for (let i = 0; i < input.length; i++) {
    const parts = [input.substring(0, i), input.substring(i)]

    const reader = stream(
      {
        read() {
          return parts.shift()
        }
      },
      json => {
        t.pass()
      }
    )

    reader.read()
  }
})

test('lossless number', t => {
  const r = execSync(`echo '{"long": 123456789012345678901}' | node index.js .long`).toString('utf8')
  t.is(r, '123456789012345678901\n')
})

test('value iterator', t => {
  const r = fx({master: {foo: [{bar: [{val: 1}]}]}}, '.master.foo[].bar[].val')
  t.deepEqual(JSON.parse(r), [1])
})

test('value iterator simple', t => {
  const r = fx([{val:1},{val:2}], '.[].val')
  t.deepEqual(JSON.parse(r), [1, 2])
})
