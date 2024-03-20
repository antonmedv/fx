const __keys = new Set()

Object.prototype.__keys = function () {
  if (Array.isArray(this)) return
  if (typeof this === 'string') return
  if (this instanceof String) return
  if (typeof this === 'object' && this !== null)
    Object.keys(this).forEach(x => __keys.add(x))
}

