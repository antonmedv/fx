// .fxrc.js %v

function reduce(input) {
  let x = JSON.parse(input)

  // Reducers %v
  if (typeof x === 'undefined') {
    return 'null'
  } else {
    return JSON.stringify(x)
  }
}
