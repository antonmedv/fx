#!/usr/bin/env zx

process.on('SIGPIPE', () => process.exit(0))
process.on('SIGINT', () => process.exit(0))
process.on('SIGTERM', () => process.exit(0))
process.stdout.on('error', (err) => {
  if (err.code === 'EPIPE') process.exit(0)
  throw err
})

const names = ['Alice', 'Bob', 'Charlie', 'Diana', 'Eve', 'Frank', 'Grace', 'Henry']
const cities = ['New York', 'London', 'Tokyo', 'Paris', 'Berlin', 'Sydney', 'Toronto', 'Mumbai']
const colors = ['red', 'green', 'blue', 'yellow', 'purple', 'orange', 'pink', 'cyan']
const departments = ['Engineering', 'Sales', 'Marketing', 'HR', 'Finance', 'Operations']
const skills = ['JavaScript', 'Python', 'Go', 'Rust', 'SQL', 'Docker', 'Kubernetes', 'AWS']

function randomItem(arr) {
  return arr[Math.floor(Math.random() * arr.length)]
}

function randomItems(arr, min = 1, max = 3) {
  const count = min + Math.floor(Math.random() * (max - min + 1))
  const shuffled = [...arr].sort(() => Math.random() - 0.5)
  return shuffled.slice(0, count)
}

function randomObject() {
  return {
    id: Math.floor(Math.random() * 1000000),
    name: randomItem(names),
    active: Math.random() > 0.5,
    timestamp: new Date().toISOString(),
    profile: {
      age: 20 + Math.floor(Math.random() * 40),
      city: randomItem(cities),
      preferences: {
        color: randomItem(colors),
        notifications: Math.random() > 0.5,
        theme: Math.random() > 0.5 ? 'dark' : 'light',
      },
    },
    work: {
      department: randomItem(departments),
      salary: Math.floor(50000 + Math.random() * 100000),
      skills: randomItems(skills, 2, 5),
    },
    scores: Array.from({length: 3}, () => Math.round(Math.random() * 100)),
    tags: randomItems(colors, 1, 3),
  }
}

const randomTexts = [
  'Processing records...',
  'Loading next batch',
  '--- checkpoint ---',
  'Fetching data from server',
  'INFO: Connection stable',
  'DEBUG: Buffer flushed',
  'Waiting for response...',
  '>> sync complete',
]

const count = parseInt(argv._[0]) || Infinity
const delay = parseInt(argv.delay) || 100
const withText = argv['with-text'] || argv.withText

for (let i = 0; i < count; i++) {
  if (withText && Math.random() < 0.15) {
    console.log(randomItem(randomTexts))
    if (delay > 0) await sleep(delay)
  }
  console.log(JSON.stringify(randomObject()))
  if (delay > 0 && i < count - 1) {
    await sleep(delay)
  }
}
