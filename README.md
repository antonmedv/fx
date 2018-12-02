<p align="center"><img src="https://user-images.githubusercontent.com/141232/35405308-4b41f446-0238-11e8-86c1-21f407cc8460.png" height="100" alt="fx"></p>
<p align="center"><img src="https://user-images.githubusercontent.com/141232/47933350-f0f22900-df06-11e8-9cf2-88492c1be774.gif" width="530" alt="fx example"></p>

_* Function eXecution_

[![Build Status](https://travis-ci.org/antonmedv/fx.svg?branch=master)](https://travis-ci.org/antonmedv/fx)
[![Npm Version](https://badgen.net/npm/v/fx)](https://www.npmjs.com/package/fx)
[![Brew Version](https://badgen.net/homebrew/v/fx)](https://formulae.brew.sh/formula/fx)

Command-line JSON processing tool

## Features

* Formatting and highlighting
* Standalone binary
* Interactive mode 🎉
* Themes support 🎨

## Install

```
$ npm install -g fx
```
Or via Homebrew
```
$ brew install fx
```

Or download standalone binary from [releases](https://github.com/antonmedv/fx/releases) page.

<a href="https://www.patreon.com/antonmedv">
	<img src="https://c5.patreon.com/external/logo/become_a_patron_button@2x.png" width="160">
</a>

## Usage

Start interactive mode without passing any arguments.
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

Access all lodash (or ramda, etc) methods by using [.fxrc](https://github.com/antonmedv/fx/blob/master/docs.md#using-fxrc) file.
```bash
$ curl ... | fx '_.mapValues(_.groupBy(this, "commit.committer.name"), _.size)'
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

## Documentation

See full [documentation](https://github.com/antonmedv/fx/blob/master/docs.md).

## Related

* [xx](https://github.com/antonmedv/xx) - fx-like JSON tool (*go*)
* [ymlx](https://github.com/matthewadams/ymlx) - fx-like YAML cli processor

## License

[MIT](https://github.com/antonmedv/fx/blob/master/LICENSE)  
