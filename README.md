<p align="center"><a href="https://fx.wtf"><img src="https://medv.io/assets/fx/fx-preview.gif" width="500" alt="fx preview"></a></p>

_* Function eXecution_

## Features

- Mouse support
- Streaming support
- Preserves key order
- Preserves big numbers

## Install

```bash
brew install fx
```
```bash
snap install fx
```
```bash
scoop install fx
```
```bash
pacman -S fx
```
```bash
go install github.com/antonmedv/fx@latest
```

Or download [pre-built binary](https://github.com/antonmedv/fx/releases).

## Usage

Start the interactive viewer via:

```bash
fx data.json
```

Or

```bash
curl ... | fx
```

Type `?` to see full list of key shortcuts.

Pretty print:

```bash
curl ... | fx .
```

### Reducers

Write reducers in your favorite language: [JavaScript](docs/reducers.md#node) (default),
[Python](docs/reducers.md#python), or [Ruby](docs/reducers.md#ruby).

```bash
export FX_LANG=node
fx data.json '.filter(x => x.startsWith("a"))'
```

```bash
export FX_LANG=python
fx data.json '[x["age"] + i for i in range(10)]'
```

```bash
export FX_LANG=ruby
fx data.json 'x.to_a.map {|x| x[1]}'
```

## Documentation

See full [documentation](https://github.com/antonmedv/fx/blob/master/DOCS.md).

## Themes

Theme can be configured by setting environment variable `FX_THEME` from `1`
to `9`:

```bash
export FX_THEME=9
```

<img width="1214" alt="themes" src="docs/images/themes.png">

## License

[MIT](https://github.com/antonmedv/fx/blob/master/LICENSE)
