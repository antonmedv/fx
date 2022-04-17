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

```bash
$ fx data.json
```

Or

```bash
$ curl ... | fx
```

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

Update JSON using spread operator.

```bash
$ echo '{"count": 0}' | fx '{...this, count: 1}'
{
  "count": 1
}
```

Extract values from maps.

```bash
$ fx commits.json | fx .[].commit.author
```

Pretty print.

```bash
$ curl ... | fx .
```

## Documentation

See full [documentation](https://github.com/antonmedv/fx/blob/master/DOCS.md).

## Themes

| `FX_THEME=1` ![](docs/images/1.png) | `FX_THEME=2` ![](docs/images/2.png) | `FX_THEME=3` ![](docs/images/3.png) |
|:-----------------------------------:|:-----------------------------------:|:-----------------------------------:|
| `FX_THEME=4` ![](docs/images/4.png) | `FX_THEME=5` ![](docs/images/5.png) | `FX_THEME=6` ![](docs/images/6.png) |
| `FX_THEME=7` ![](docs/images/7.png) | `FX_THEME=8` ![](docs/images/8.png) | `FX_THEME=9` ![](docs/images/9.png) |

## License

[MIT](https://github.com/antonmedv/fx/blob/master/LICENSE)
