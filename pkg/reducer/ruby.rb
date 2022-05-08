require 'json'
x = JSON.parse(STDIN.read)

# Reducers %v

puts JSON.generate(x)
