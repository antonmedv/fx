<p align="center"><a href="http://fx.wtf"><img src="https://medv.io/assets/fx-logo.png" height="100" alt="fx logo"></a></p>
<p align="center"><img src="https://medv.io/assets/fx.gif" width="562" alt="fx example"></p>

_* Function eXecution_

[![Build Status](https://travis-ci.org/antonmedv/fx.svg?branch=master)](https://travis-ci.org/antonmedv/fx)
[![Npm Version](https://img.shields.io/npm/v/fx.svg)](https://www.npmjs.com/package/fx)
[![Brew Version](https://img.shields.io/homebrew/v/fx.svg)](https://formulae.brew.sh/formula/fx)

Command-line JSON processing tool

## Features

* Easy to use
* Standalone binary
* Interactive mode 🎉
* Streaming support 🌊

## Install

```bash
npm install -g fx
```
Or via Homebrew
```bash
brew install fx
```
Or download standalone binary from [releases](https://github.com/antonmedv/fx/releases)
```bash
bash <( curl https://fx.wtf )
```

## Usage

Start [interactive mode](https://github.com/antonmedv/fx/blob/master/DOCS.md#interactive-mode) without passing any arguments.
```bash
$ curl ... | fx
```

Or by passing filename as first argument.
```bash
$ fx data.json
```

Pass a few JSON files.
```bash
cat foo.json bar.json baz.json | fx .message
```

Use full power of JavaScript.
```bash
$ curl ... | fx '.filter(x => x.startsWith("a"))'
```

Access all lodash (or ramda, etc) methods by using [.fxrc](https://github.com/antonmedv/fx/blob/master/DOCS.md#using-fxrc) file.
```bash
$ curl ... | fx '_.groupBy("commit.committer.name")' '_.mapValues(_.size)'
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
$ fx commits.json | fx .[].author.name
```

Print formatted JSON to stdout.
```bash
$ curl ... | fx .
```

Pipe JSON logs stream into fx.
```bash
$ kubectl logs ... -f | fx .message
```

And try this:
```bash
$ fx --life
```

## Documentation

See full [documentation](https://github.com/antonmedv/fx/blob/master/DOCS.md).

## Links

* [Discover how to use fx effectively](https://medium.com/@antonmedv/discover-how-to-use-fx-effectively-668845d2a4ea)

## Related

* [gofx](https://github.com/antonmedv/gofx) – fx-like JSON tool (*go*)
* [eat](https://github.com/antonmedv/eat) – converts anything into JSON
* [ymlx](https://github.com/matthewadams/ymlx) – fx-like YAML cli processor
* [fx-completion](https://github.com/antonmedv/fx-completion) – bash completion for fx
* [fx-theme-monokai](https://github.com/antonmedv/fx-theme-monokai) – monokai theme
* [fx-theme-night](https://github.com/antonmedv/fx-theme-night) – night theme


## License

[MIT](https://github.com/antonmedv/fx/blob/master/LICENSE)
