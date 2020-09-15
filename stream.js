'use strict'
const JSON = require('lossless-json')

function apply(cb, input) {
  let json
  try {
    json = JSON.parse(input)
  } catch (e) {
    process.stderr.write(e.toString() + '\n')
    return
  }
  cb(json)
}

function stream(from, cb) {
  let buff = ''
  let lastChar = ''
  let len = 0
  let depth = 0
  let isString = false

  let count = 0
  let head = ''
  const check = (i) => {
    if (depth <= 0) {
      const input = buff.substring(0, len + i + 1)

      if (count > 0) {
        if (head !== '') {
          apply(cb, head)
          head = ''
        }

        apply(cb, input)
      } else {
        head = input
      }

      buff = buff.substring(len + i + 1)
      len = -i - 1
      count++
    }
  }

  return {
    isStream() {
      return count > 1
    },
    value() {
      return head + buff
    },
    read() {
      let chunk

      while ((chunk = from.read())) {
        len = buff.length
        buff += chunk

        for (let i = 0; i < chunk.length; i++) {
          if (isString) {
            if (chunk[i] === '"' && ((i === 0 && lastChar !== '\\') || (i > 0 && chunk[i - 1] !== '\\'))) {
              isString = false
              check(i)
            }
            continue
          }

          if (chunk[i] === '{' || chunk[i] === '[') {
            depth++
          } else if (chunk[i] === '}' || chunk[i] === ']') {
            depth--
            check(i)
          } else if (chunk[i] === '"') {
            isString = true
          }
        }

        lastChar = chunk[chunk.length - 1]
      }
    }
  }
}

module.exports = stream
