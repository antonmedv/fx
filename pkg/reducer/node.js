import os from 'node:os'
import fs from 'node:fs'
import path from 'node:path'
import {createRequire} from 'node:module'

const cwd = process.env.FX_CWD ? process.env.FX_CWD : process.cwd()
const require = createRequire(cwd)

// .fxrc.js %v

void async function () {
  process.chdir(cwd)

  let buffer = ''
  process.stdin.setEncoding('utf8')
  for await (let chunk of process.stdin) {
    buffer += chunk
  }
  let x = JSON.parse(buffer)

  // Reducers %v

  // Adding a line break to easily differentiate the reduced object in the output.
  // No matter what reducers write to stdout, the reduced object will always be on the last line.
  process.stdout.write('\n')

  if (typeof x === 'undefined') {
    process.stderr.write('Reducer returned undefined')
  } else {
    process.stdout.write(JSON.stringify(x))
  }
}().catch(err => {
  console.error(err)
  process.exitCode = 1
})
