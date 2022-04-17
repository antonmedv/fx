# fx

<p align="center"><img src="https://medv.io/assets/fx.gif" width="562" alt="fx example"></p>

_* Function eXecution_

## Install

```bash
go install github.com/antonmedv/fx@latest
```

Or via Homebrew

```bash
TODO
```

Or download [pre-built binary](https://github.com/antonmedv/fx/releases).

## Usage

Start the interactive viewer via:

```bash
$ fx data.json
```

Or

```bash
$ curl ... | fx
```

Type `?` to see full list of key shortcuts.

### Reducers

```bash
$ fx data.json '.filter(x => x.startsWith("a"))'
```

Access all lodash (or ramda, etc) methods by
using [.fxrc](https://github.com/antonmedv/fx/blob/master/DOCS.md#using-fxrc)
file.

```bash
$ fx data.json 'groupBy("commit.committer.name")' 'mapValues(_.size)'
```

## Documentation


See full [documentation](https://github.com/antonmedv/fx/blob/master/DOCS.md).

## Themes

Theme can be configured by setting environment variable `FX_THEME` from `1` to `9`:

```bash
export FX_THEME=9
```

<img width="1214" alt="themes" src="docs/images/themes.png">

## License

[MIT](https://github.com/antonmedv/fx/blob/master/LICENSE)
