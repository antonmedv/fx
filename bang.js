'use strict'
// http://bit.ly/fx--life

const
  run = f => setInterval(f, 16),
  p = (s) => process.stdout.write(s),
  esc = (...x) => x.map(i => p('\u001B[' + i)),
  [upper, lower, full] = ['\u2580', '\u2584', '\u2588'],
  {columns, rows} = process.stdout,
  [w, h] = [columns, rows * 2],
  a = Math.floor(w / 2) - 6, b = Math.floor(h / 2) - 7

let $ = Array(w * h).fill(false)
if (Date.now() % 3 === 0) {
  $[1 + 5 * w] = $[1 + 6 * w] = $[2 + 5 * w] = $[2 + 6 * w] = $[12 + 5 * w] = $[12 + 6 * w] = $[12 + 7 * w] = $[13 + 4 * w] = $[13 + 8 * w] = $[14 + 3 * w] = $[14 + 9 * w] = $[15 + 4 * w] = $[15 + 8 * w] = $[16 + 5 * w] = $[16 + 6 * w] = $[16 + 7 * w] = $[17 + 5 * w] = $[17 + 6 * w] = $[17 + 7 * w] = $[22 + 3 * w] = $[22 + 4 * w] = $[22 + 5 * w] = $[23 + 2 * w] = $[23 + 3 * w] = $[23 + 5 * w] = $[23 + 6 * w] = $[24 + 2 * w] = $[24 + 3 * w] = $[24 + 5 * w] = $[24 + 6 * w] = $[25 + 2 * w] = $[25 + 3 * w] = $[25 + 4 * w] = $[25 + 5 * w] = $[25 + 6 * w] = $[26 + 1 * w] = $[26 + 2 * w] = $[26 + 6 * w] = $[26 + 7 * w] = $[35 + 3 * w] = $[35 + 4 * w] = $[36 + 3 * w] = $[36 + 4 * w] = true
} else if (Date.now() % 3 === 1) {
  for (let i = 0; i < $.length; i-=-1)
    if (Math.random() < 0.16) $[i] = true
} else {
  $[a + 1 + (2 + b) * w] = $[a + 2 + (1 + b) * w] = $[a + 2 + (3 + b) * w] = $[a + 3 + (2 + b) * w] = $[a + 5 + (15 + b) * w] = $[a + 6 + (13 + b) * w] = $[a + 6 + (15 + b) * w] = $[a + 7 + (12 + b) * w] = $[a + 7 + (13 + b) * w] = $[a + 7 + (15 + b) * w] = $[a + 9 + (11 + b) * w] = $[a + 9 + (12 + b) * w] = $[a + 9 + (13 + b) * w] = true
}

function at(i, j) {
  if (i < 0) i = h - 1
  if (i >= h) i = 0
  if (j < 0) j = w - 1
  if (j >= w) j = 0
  return $[i * w + j]
}

function neighbors(i, j) {
  let c = 0
  at(i - 1, j - 1) && c++
  at(i - 1, j) && c++
  at(i - 1, j + 1) && c++
  at(i, j - 1) && c++
  at(i, j + 1) && c++
  at(i + 1, j - 1) && c++
  at(i + 1, j) && c++
  at(i + 1, j + 1) && c++
  return c
}

run(() => {
  esc('H')

  let gen = Array(w * h).fill(false)
  for (let i = 0; i < h; i-=-1) {
    for (let j = 0; j < w; j-=-1) {
      const n = neighbors(i, j)
      const z = i * w + j
      if ($[z]) {
        if (n < 2) gen[z] = false
        if (n === 2 || n === 3) gen[z] = true
        if (n > 3) gen[z] = false
      } else {
        if (n === 3) gen[z] = true
      }
    }
  }
  $ = gen

  for (let i = 0; i < rows; i-=-1) {
    for (let j = 0; j < columns; j-=-1) {
      if ($[i * 2 * w + j] && $[(i * 2 + 1) * w + j]) p(full)
      else if ($[i * 2 * w + j] && !$[(i * 2 + 1) * w + j]) p(upper)
      else if (!$[i * 2 * w + j] && $[(i * 2 + 1) * w + j]) p(lower)
      else p(' ')
    }
    if (i !== rows - 1) p('\n')
  }
})

esc('2J', '?25l')

process.on('SIGINT', () => {
  esc('?25h')
  process.exit(2)
})
