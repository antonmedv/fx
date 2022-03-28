const os = require('os')
const fs = require('fs')
const path = require('path')

try {
  require(path.join(os.homedir(), '.fxrc'))
} catch (err) {
  if (err.code !== 'MODULE_NOT_FOUND') throw err
}

void async function () {
  let buffer = ''
  process.stdin.setEncoding('utf8')
  for await (let chunk of process.stdin) {
    buffer += chunk
  }
  let json = JSON.parse(buffer)

  // Reducers %v

  if (typeof json === 'undefined') {
    process.stderr.write('undefined')
  } else {
    process.stdout.write(JSON.stringify(json))
  }
}().catch(err => {
  console.error(err)
  process.exitCode = 1
})
