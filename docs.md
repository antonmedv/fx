# Documentation

`fx` can work in two modes: cli and interactive. To start interactive mode pipe into `fx` any JSON:

```
$ curl ... | fx
```

Or you can pass file argument as first parameter:

```
$ fx my.json
```

If any argument was passed, `fx` will apply it and prints to stdout.

## Anonymous function

Use an anonymous function as reducer which gets JSON and processes it:
```
$ echo '{"foo": [{"bar": "value"}]}' | fx 'x => x.foo[0].bar'
value
```

## Binding

If you don't pass anonymous function `param => ...`, code will be automatically transformed into anonymous function.
And you can get access to JSON by `this` keyword:
```
$ echo '{"foo": [{"bar": "value"}]}' | fx 'this.foo[0].bar'
value
```

## Dot

It is possible to omit `this` keyword:
```
$ echo '{"foo": [{"bar": "value"}]}' | fx .foo[0].bar
value
```

If single dot is passed, JSON will be processed without modification:
```
$ echo '{"foo": "bar"}' | fx .
{
  "foo": "bar"
}
```

## Chain

You can pass any number of anonymous functions for reducing JSON:
```
$ echo '{"foo": [{"bar": "value"}]}' | fx 'x => x.foo' 'this[0]' 'this.bar'
value
```

## Generator

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

## Update

You can update existing JSON using spread operator:

```
$ echo '{"count": 0}' | fx '{...this, count: 1}'
{
  "count": 1
}
```

## Using packages

Use any npm package by installing it globally:
```
$ npm install -g lodash
$ cat package.json | fx 'require("lodash").keys(this.dependencies)'
```

## Using .fxrc

Create _.fxrc_ file in `$HOME` directory, and require any packages or define global functions.

For example, access all lodash methods without `_` prefix. Put in your `.fxrc` file:

```js
Object.assign(global, require('lodash'))
```

And now you will be able to call all lodash methods. For example, see who's been committing to react recently:

```
curl 'https://api.github.com/repos/facebook/react/commits?per_page=100' \
| fx 'mapValues(groupBy(this, "commit.committer.name"), size)'
```

> To be able require global modules make sure you have correct `NODE_PATH` env variable.
> ```bash
> export NODE_PATH=/usr/local/lib/node_modules
> ```

## Formatting

If you need something different then JSON (for example arguments for xargs) do not return anything from reducer.
`undefined` value is printed into stderr by default.
```
echo '[]' | fx 'void 0'
undefined
```

```
echo '[1,2,3]' | fx 'this.forEach(x => console.log(x))' 2>/dev/null | xargs echo
1 2 3
```

## Other examples

Convert object to array:
```
$ cat package.json | fx 'Object.keys(this.dependencies)'
[
  "@medv/prettyjson"
]
```

By the way, fx has shortcut for `Object.keys(this)`. Previous example can be rewritten as:

```
$ cat package.json | fx this.dependencies ?
``` 

## Interactive mode

Click on fields to expand or collapse JSON tree, use mouse wheel to scroll view.

Next commands available in interactive mode:

|             Key               |         Command         |
|-------------------------------|-------------------------|
| `q` or `Esc` or `Ctrl`+`c`    | Exit                    |
| `e`/`E`                       | Expand/Collapse all     |
| `g`/`G`                       | Goto top/bottom         |
| `up`/`down` or `k/j`          | Move cursor up/down     |
| `left`/`right` or `h/l`       | Expand/Collapse         |
| `.`                           | Edit filter             |

These commands are available when editing the filter:

|             Key               |         Command         |
|-------------------------------|-------------------------|
| `Enter`                       | Apply filter            |
| `Ctrl`+`u`                    | Clear filter            |
| `Ctrl`+`w`                    | Delete last part        |
| `up`/`down`                   | Select autocomplete     |

### Selecting text

You may found what you can't just select text in fx. This is due the fact that all mouse events redirected to stdin. To be able select again you need instruct your terminal not to do it. This can be done by holding special keys while selecting: 

|       Key        |   Terminal    |
|------------------|---------------|
| `Option`+`Mouse` | iTerm2, Hyper |
| `Fn`+`Mouse`     | Terminal.app  |
| `Shift`+`Mouse`  | Linux         |
