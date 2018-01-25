# fx [![Build Status](https://travis-ci.org/antonmedv/fx.svg?branch=master)](https://travis-ci.org/antonmedv/fx)

Command-line JSON processing tool

## Features

* Don't need to learn new syntax
* Plain JavaScript
* Formatting and highlighting

## Install

```
$ npm install -g fx
```

## Usage

Pipe into `fx` any JSON and anonymous function for reducing it.

```
$ fx [code ...]
```

Pretty print JSON:
```
$ echo '{"key":"value"}' | fx
{
    "key": "value"
}
```

Use anonymous function:
```
$ echo '{"foo": [{"bar": "value"}]}' | fx "x => x.foo[0].bar"
"value"
```

If you don't pass anonymous function `param => ...`, code will be automatically transformed into anonymous function.
And you can get access to JSON by `this` keyword:
```
$ echo '{"foo": [{"bar": "value"}]}' | fx "this.foo[0].bar"
"value"
```

You can pass any number of anonymous functions for reducing JSON:
```
$ echo '{"foo": [{"bar": "value"}]}' | fx "x => x.foo" "this[0]" "this.bar"
"value"
```

If passed code contains `yield` keyword, [generator expression](https://github.com/sebmarkbage/ecmascript-generator-expression)
will be used:
```
$ curl ... | fx "for (let user of this) if (user.login.startsWith('a')) yield user"
```

Access to JSON through `this` keyword:
```
$ echo '["a", "b"]' | fx "yield* this"
[
    "a",
    "b"
]
```

```
$ echo '["a", "b"]' | fx "yield* this; yield 'c';"
[
    "a",
    "b",
    "c"
]
```

You can update existing JSON using spread operator:

```
$ echo '{"count": 0}' | fx "{...this, count: 1}"
{
    "count": 1
}
```

Convert object to array:
```
$ cat package.json | fx "yield* Object.keys(this.dependencies)"
[
    "cardinal",
    "get-stdin",
    "meow"
]
```


## Related

* [jq](https://github.com/stedolan/jq) – cli JSON processor on C
* [jsawk](https://github.com/micha/jsawk) – like awk, but for JSON
* [json](https://github.com/trentm/json) – another JSON manipulating cli library

## License

MIT
