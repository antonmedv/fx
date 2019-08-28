# Documentation

* [Getting started](#getting-started)
* [Usage](#usage)
  + [Anonymous function](#anonymous-function)
  + [Binding](#binding)
  + [Dot](#dot)
  + [Chaining](#chaining)
  + [Generator](#generator)
  + [Updating](#updating)
  + [Using packages](#using-packages)
* [Using .fxrc](#using-fxrc)
  + [Edit in place](#edit-in-place)
* [Formatting](#formatting)
* [Other examples](#other-examples)
* [Streaming mode](#streaming-mode)
* [Interactive mode](#interactive-mode)
  + [Searching](#searching)
  + [Selecting text](#selecting-text)
* [Memory Usage](#memory-usage)

## Getting started

`fx` can work in two modes: cli and interactive. To start interactive mode pipe any JSON into `fx`:

```bash
$ curl ... | fx
```

Or you can pass a filename as the first parameter:

```bash
$ fx my.json
```

If any argument was passed, `fx` will apply it and prints to stdout.

## Usage

### Anonymous function

Use an anonymous function as reducer which gets JSON and processes it:
```bash
$ echo '{"foo": [{"bar": "value"}]}' | fx 'x => x.foo[0].bar'
value
```

### Binding

If you don't pass anonymous function `param => ...`, code will be automatically transformed into anonymous function.
And you can get access to JSON by `this` keyword:
```bash
$ echo '{"foo": [{"bar": "value"}]}' | fx 'this.foo[0].bar'
value
```

### Dot

It is possible to omit `this` keyword:
```bash
$ echo '{"foo": [{"bar": "value"}]}' | fx .foo[0].bar
value
```

If a single dot is passed, the input JSON will be formatted but otherwise unaltered:
```bash
$ echo '{"foo": "bar"}' | fx .
{
  "foo": "bar"
}
```

### Chaining

You can pass any number of anonymous functions for reducing JSON:
```bash
$ echo '{"foo": [{"bar": "value"}]}' | fx 'x => x.foo' 'this[0]' 'this.bar'
value
```

### Generator

If the passed code contains the `yield` keyword, [generator expression](https://github.com/sebmarkbage/ecmascript-generator-expression)
will be used:
```bash
$ curl ... | fx 'for (let user of this) if (user.login.startsWith("a")) yield user'
```

Access to JSON through `this` keyword:
```bash
$ echo '["a", "b"]' | fx 'yield* this'
[
  "a",
  "b"
]
```

```bash
$ echo '["a", "b"]' | fx 'yield* this; yield "c";'
[
  "a",
  "b",
  "c"
]
```

### Updating

You can update existing JSON using the spread operator:

```bash
$ echo '{"count": 0}' | fx '{...this, count: 1}'
{
  "count": 1
}
```

### Using packages

Use any npm package by installing it globally:
```bash
$ npm install -g lodash
$ cat package.json | fx 'require("lodash").keys(this.dependencies)'
```

## Using .fxrc

Create _.fxrc_ file in `$HOME` directory, and require any packages or define global functions.

For example, access all lodash methods without `_` prefix. Put in your `.fxrc` file:

```js
Object.assign(global, require('lodash/fp'))
```

And now you will be able to call all lodash methods. For example, see who's been committing to react recently:

```bash
curl 'https://api.github.com/repos/facebook/react/commits?per_page=100' \
| fx 'groupBy("commit.author.name")' 'mapValues(size)' toPairs 'sortBy(1)' reverse 'take(10)' fromPairs
```

> To be able require global modules make sure you have correct `NODE_PATH` env variable.
> ```bash
> export NODE_PATH=`npm root -g`
> ```

### Edit in place

Add next code to your _.fxrc_ file:

```js
const fs = require('fs')

global.save = json => {
  fs.writeFileSync(process.argv[2], JSON.stringify(json, null, 2))
  return json
}
```

Usage:

```bash
fx data.json '{...this, count: this.count+1}' save .count
```

## Formatting

If you need output other than JSON (for example arguments for xargs), do not return anything from the reducer.
`undefined` value is printed into stderr by default.
```bash
echo '[]' | fx 'void 0'
undefined
```

```bash
echo '[1,2,3]' | fx 'this.forEach(x => console.log(x))' 2>/dev/null | xargs echo
1 2 3
```

## Other examples

Convert object to array:
```bash
$ cat package.json | fx 'Object.keys(this.dependencies)'
```

Or by two functions:
```bash
$ cat package.json | fx .dependencies Object.keys
```

By the way, fx has shortcut for `Object.keys`. Previous example can be rewritten as:

```bash
$ cat package.json | fx .dependencies ?
``` 

## Streaming mode

`fx` supports line-delimited JSON and concatenated JSON streaming.

```bash
$ kubectl logs ... | fx .message
``` 

Sometimes it is necessary to omit some messages in JSON stream, or select only specified log messages.
For this purpose, `fx` has special helper `select`, pass function into it to select only some JSON messages.

```bash
$ kubectl logs ... | fx 'select(x => x.message.length > 40)' .message
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
| `/`                           | Search                  |
| `n`                           | Goto next found pattern |

These commands are available when editing the filter:

|             Key               |         Command         |
|-------------------------------|-------------------------|
| `Enter`                       | Apply filter            |
| `Ctrl`+`u`                    | Clear filter            |
| `Ctrl`+`w`                    | Delete last part        |
| `up`/`down`                   | Select autocomplete     |

### Searching

Press `/` and type regexp pattern to search in current JSON. Search work with currently applied filter.

Examples of pattern and corresponding regexp: 

|   Pattern  |    RegExp   |
|------------|-------------|
| `/apple`   | `/apple/ig` |
| `/apple/`  | `/apple/`   |
| `/apple/u` | `/apple/u`  |
| `/\w+`     | `/\w+/ig`   | 

### Selecting text

You may found what you can't just select text in fx. This is due the fact that all mouse events redirected to stdin. To be able select again you need instruct your terminal not to do it. This can be done by holding special keys while selecting: 

|       Key        |   Terminal    |
|------------------|---------------|
| `Option`+`Mouse` | iTerm2, Hyper |
| `Fn`+`Mouse`     | Terminal.app  |
| `Shift`+`Mouse`  | Linux         |

## Memory Usage

You may find that sometimes, on really big JSON files, fx prints an error message like this:

```
FATAL ERROR: JavaScript heap out of memory 
```

V8 limits memory usage to around 2 GB by default. You can increase the limit by putting this line in your _.profile_:

```bash
export NODE_OPTIONS='--max-old-space-size=8192'
``` 
