<p align="center"><img src="https://medv.io/assets/fx-logo.png" height="100" alt="fx logo"></p>
<p align="center"><img src="https://medv.io/assets/fx.gif" width="562" alt="fx example"></p>

_* Function eXecution_

[![Build Status](https://travis-ci.org/antonmedv/fx.svg?branch=master)](https://travis-ci.org/antonmedv/fx)
[![Npm Version](https://img.shields.io/npm/v/fx.svg)](https://www.npmjs.com/package/fx)
[![Brew Version](https://img.shields.io/homebrew/v/fx.svg)](https://formulae.brew.sh/formula/fx)
![Nyancat Approved](https://img.shields.io/badge/nyancat-approved-ff69b4.svg)

Command-line JSON processing tool

## Features

* Easy to use
* Standalone binary
* Interactive mode 🎉
* Themes support 🎨
* Streaming support 🌊
* Bash completion

## Install

```
$ npm install -g fx
```
Or via Homebrew
```
$ brew install fx
```

Or download standalone binary from [releases](https://github.com/antonmedv/fx/releases) page.

Did you like **fx**? [Buy me a beer 🍺](https://paypal.me/antonmedv) or [send come ₿](https://www.wispay.io/t/ZQb)

## Usage

Start [interactive mode](https://github.com/antonmedv/fx/blob/master/DOCS.md#interactive-mode) without passing any arguments.
```
$ curl ... | fx
```

Or by passing filename as first argument.
```
$ fx data.json
```

Pipe into `fx` any JSON and anonymous function for reducing it.
```bash
$ curl ... | fx 'json => json.message'
```

Or same as above but short.
```bash
$ curl ... | fx this.message
$ curl ... | fx .message
```

Pass any numbers of arguments as code.
```bash
$ curl ... | fx 'json => json.message' 'json => json.filter(x => x.startsWith("a"))'
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

Pretty print JSON with dot.
```bash
$ curl ... | fx .
```

Stream JSON into fx.
```bash
$ kubectl logs ... -f | fx .message
```

Apply fx to a few JSON files.
```bash
$ cat *.json | fx .length
3
4
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

* [xx](https://github.com/antonmedv/xx) – fx-like JSON tool (*go*)
* [eat](https://github.com/antonmedv/eat) – converts anything into JSON
* [ymlx](https://github.com/matthewadams/ymlx) – fx-like YAML cli processor
* [fx-completion](https://github.com/antonmedv/fx-completion) – bash completion for fx
* [fx-theme-monokai](https://github.com/antonmedv/fx-theme-monokai) – monokai theme
* [fx-theme-night](https://github.com/antonmedv/fx-theme-night) – night theme


## License

[MIT](https://github.com/antonmedv/fx/blob/master/LICENSE)
