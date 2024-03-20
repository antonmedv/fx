const console = {
  log: function (...args) {
    const parts = []
    for (const arg of args) {
      parts.push(typeof arg === 'string' ? arg : JSON.stringify(arg, null, 2))
    }
    println(parts.join(' '))
  },
}
