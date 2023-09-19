const goos = [
  'linux',
  'darwin',
  'windows',
]
const goarch = [
  'amd64',
  'arm64',
]

const name = (GOOS, GOARCH) => `fx_${GOOS}_${GOARCH}` + (GOOS === 'windows' ? '.exe' : '')

const resp = await fetch('https://api.github.com/repos/antonmedv/fx/releases/latest')
const {tag_name: latest} = await resp.json()

await $`go mod download`

await Promise.all(
  goos.flatMap(GOOS =>
    goarch.map(GOARCH =>
      $`GOOS=${GOOS} GOARCH=${GOARCH} go build -o ${name(GOOS, GOARCH)}`)))

await Promise.all(
  goos.flatMap(GOOS =>
    goarch.map(GOARCH =>
      $`gh release upload ${latest} ${name(GOOS, GOARCH)}`)))

await Promise.all(
  goos.flatMap(GOOS =>
    goarch.map(GOARCH =>
      $`rm ${name(GOOS, GOARCH)}`)))
