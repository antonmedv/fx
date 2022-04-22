import os from 'node:os'
import fs from 'node:fs'
import path from 'node:path'
import {createRequire} from 'node:module'
const require = createRequire(process.cwd())

// .fxrc.js %v

void async function () {
  if (process.env.FX_CWD) {
    process.chdir(process.env.FX_CWD)
  }

  let buffer = ''
  process.stdin.setEncoding('utf8')
  for await (let chunk of process.stdin) {
    buffer += chunk
  }
  let x = JSON.parse(buffer)

  // Reducers %v

  if (typeof x === 'undefined') {
    process.stderr.write('undefined')
  } else {
    process.stdout.write(JSON.stringify(x))
  }
}().catch(err => {
  console.error(err)
  process.exitCode = 1
})
