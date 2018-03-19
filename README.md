<img src="https://user-images.githubusercontent.com/141232/35405308-4b41f446-0238-11e8-86c1-21f407cc8460.png" height="100" alt="fx">

# [![Build Status](https://travis-ci.org/antonmedv/fx.svg?branch=master)](https://travis-ci.org/antonmedv/fx)

Command-line JSON processing tool

## Features

* Don't need to learn new syntax
* Plain JavaScript
* Formatting and highlighting
* Standalone binary

## Install

```
$ npm install -g fx
```

Or download standalone binary from [releases](https://github.com/antonmedv/fx/releases) page.

## Usage

Pipe into `fx` any JSON and anonymous function for reducing it.

```
$ fx [code ...]
```

Pretty print JSON without passing any arguments:
```
$ echo '{"key":"value"}' | fx
{
    "key": "value"
}
```

### Anonymous function

Use an anonymous function as reducer which gets JSON and processes it:
```
$ echo '{"foo": [{"bar": "value"}]}' | fx 'x => x.foo[0].bar'
"value"
```

### This Binding

If you don't pass anonymous function `param => ...`, code will be automatically transformed into anonymous function.
And you can get access to JSON by `this` keyword:
```
$ echo '{"foo": [{"bar": "value"}]}' | fx 'this.foo[0].bar'
"value"
```

### Chain

You can pass any number of anonymous functions for reducing JSON:
```
$ echo '{"foo": [{"bar": "value"}]}' | fx 'x => x.foo' 'this[0]' 'this.bar'
"value"
```

### Generator

If passed code contains `yield` keyword, [generator expression](https://github.com/sebmarkbage/ecmascript-generator-expression)
will be used:
```
$ curl ... | fx 'for (let user of this) if (user.login.startsWith("a")) yield user'
```

Access to JSON through `this` keyword:
```
$ echo '["a", "b"]' | fx 'yield* this'
[
    "a",
    "b"
]
```

```
$ echo '["a", "b"]' | fx 'yield* this; yield "c";'
[
    "a",
    "b",
    "c"
]
```

### Update

You can update existing JSON using spread operator:

```
$ echo '{"count": 0}' | fx '{...this, count: 1}'
{
    "count": 1
}
```

### Use npm package

Use any npm package by installing it globally:
```
$ npm install -g lodash
$ cat package.json | fx 'require("lodash").keys(this.dependencies)'
```

### Formatting

If you need something different then JSON (for example arguments for xargs) do not return anything from reducer.
`undefined` value printed into stderr by default.
```
echo '[]' | fx 'void 0'
undefined
```

```
echo '[1,2,3]' | fx 'this.forEach(x => console.log(x))' 2>/dev/null | xargs echo
1 2 3
```

### Other examples

Convert object to array:
```
$ cat package.json | fx 'Object.keys(this.dependencies)'
[
    "get-stdin",
    "jsome",
    "meow"
]
```

By the way, fx has shortcut for `Object.keys(this)`. Previous example can be rewritten as:

```
$ cat package.json | fx this.dependencies ?
``` 


## Related

* [jq](https://github.com/stedolan/jq) – cli JSON processor on C
* [jsawk](https://github.com/micha/jsawk) – like awk, but for JSON
* [json](https://github.com/trentm/json) – another JSON manipulating cli library
* [jl](https://github.com/chrisdone/jl) – functional sed for JSON on Haskell
* [ymlx](https://github.com/matthewadams/ymlx) - `fx`-like YAML cli processor

## License

MIT
