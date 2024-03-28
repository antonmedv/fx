const console = {
  log: function (...args) {
    const parts = []
    for (const arg of args) {
      if (typeof arg === 'undefined') {
        parts.push('undefined')
      } else if (typeof arg === 'string') {
        parts.push(arg)
      } else {
        parts.push(JSON.stringify(arg, null, 2))
      }
    }
    println(parts.join(' '))
  },
}
