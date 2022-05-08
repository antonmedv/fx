# Documentation

The **fx** can work in two modes: as a reducer or an interactive viewer.

To start the interactive mode pipe a JSON into **fx**:

```sh
$ curl ... | fx
```

Or you can pass a filename as the first parameter:

```sh
$ fx data.json
```

## Reducers

If any additional arguments was passed, **fx** converts it to a function which 
takes the JSON as an argument named `x`.

By default, **fx** uses builtin JavaScript VM ([goja](https://github.com/dop251/goja)), 
but **fx** also can be used with [node](#node), [python](#python), or [ruby](#ruby).

### JavaScript

```sh
export FX_LANG=js
```

An example of anonymous function used as a reducer:
```sh
$ echo '{"foo": [{"bar": "value"}]}' | fx 'x => x.foo[0].bar'
value
```

The same reducer function can be simplified to:

```sh
$ echo '{"foo": [{"bar": "value"}]}' | fx 'x.foo[0].bar'
value
```

Each argument treated as a reducer function.

```sh
$ echo '{"foo": [{"bar": "value"}]}' | fx 'x.foo' 'x[0]' 'x.bar'
value
```

Update JSON using the spread operator:

```sh
$ echo '{"name": "fx", "count": 0}' | fx '{...this, count: 1}'
{
  "name": "fx",
  "count": 1
}
```

Get the list 

### Dot

Fx supports simple JS-like syntax for accessing data, which can be used with any
`FX_LANG`.

```sh
$ echo '{"foo": [{"bar": "value"}]}' | fx .foo[0].bar
value
```

### .fxrc.js

Create _.fxrc.js_ file in `$HOME` directory, and define some useful functions.

```js
// .fxrc.js
function upper(s) {
  return s.toUpperCase()
}
```

```sh
$ cat data.json | fx .name upper
ANTON
```

### Node

```sh
export FX_LANG=node
```

### NPM Packages

Use any npm package by installing it globally. Create _.fxrc.js_ file in `$HOME` 
directory, and require any packages or define global functions.

For example, access all lodash methods without `_` prefix. 

Put next line your _.fxrc.js_ file:

```js
Object.assign(global, require('lodash/fp'))
```

And now you will be able to call all lodash methods. For example, see who's been committing to react recently:

```sh
curl 'https://api.github.com/repos/facebook/react/commits?per_page=100' \
| fx 'groupBy("commit.author.name")' 'mapValues(size)' toPairs 'sortBy(1)' reverse 'take(10)' fromPairs
```

> To be able to require global modules make sure you have correct `NODE_PATH` env variable.
> ```sh
> export NODE_PATH=`npm root -g`
> ```

The _.fxrc.js_ file supports both: import and require.

```js
import 'zx/globals'
const _ = require('lodash')
```

> With you want to use _.fxrc.js_ for both `FX_LANG=js` and `FX_LANG=node`,
> separate parts by `// nodejs:` comment:
> ```js
> function upper(s) {
>   return s.toUpperCase()
> }
> // nodejs:
> import 'zx/globals'
> const _ = require('lodash')
> ```

### Python

```sh
export FX_LANG=python
```
Or 
```sh
export FX_LANG=python3
```

Example:

```sh
fx data.json '[x["age"] + i for i in range(10)]'
```

### Ruby

```sh
export FX_LANG=ruby
```

Example:

```sh
fx data.json 'x.to_a.map {|x| x[1]}'
```

## Streaming mode

The **fx** supports line-delimited JSON streaming and concatenated JSON streaming.

```sh
$ kubectl logs ... | fx .message
```

## Interactive mode

Type `?` to see full list of available shortcuts while in interactive mode.

### Search

Press `/` and type regexp pattern to search in current JSON. 
Search is performed on internal representation of the JSON without newlines.

Type `n` to jump to next result, and `N` to previous.s

### Selecting text

You can't just select text in fx. This is due the fact that all mouse events are 
redirected to stdin. To be able to select again you need instruct your terminal 
not to do it. This can be done by holding special keys while selecting:

|       Key        |   Terminal    |
|------------------|---------------|
| `Option`+`Mouse` | iTerm2, Hyper |
| `Fn`+`Mouse`     | Terminal.app  |
| `Shift`+`Mouse`  | Linux         |


## Configs

Next configs available for **fx** via environment variables.

| Name           | Values                                             | Description                                           |
|----------------|----------------------------------------------------|-------------------------------------------------------|
| `FX_LANG`      | `js` (default), `node`, `python`, `python3`, `ruby` | Reducer type.                                         |
| `FX_THEME`     | `0` disable colors, `1` (default), `2..9`       | Color theme.                                          |
| `FX_SHOW_SIZE` | `true` or `false` (default)                        | Show size of arrays and object in collapsed previews. |
