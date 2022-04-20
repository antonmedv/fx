let goos = [
  'linux',
  'darwin',
  'windows',
]
let goarch = [
  'amd64',
  'arm64',
]

await Promise.all(
  goos.flatMap(GOOS =>
    goarch.map(GOARCH =>
      $`GOOS=${GOOS} GOARCH=${GOARCH} go build -o fx_${GOOS}_${GOARCH}`)))

await Promise.all(
  goos.flatMap(GOOS =>
    goarch.map(GOARCH =>
      $`gh release upload ${process.env.RELEASE_VERSION} fx_${GOOS}_${GOARCH}`)))
